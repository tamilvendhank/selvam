package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"time"

	platformapp "goserver/internal/platform/app"
	platformconfig "goserver/internal/platform/config"
	"goserver/internal/web"

	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

func main() {
	configPath := os.Getenv("PLATFORM_CONFIG_FILE")
	if configPath == "" {
		configPath = "configs/platform.example.yaml"
	}

	cfg, err := loadConfig(configPath)
	if err != nil {
		log.Fatalf("load config: %v", err)
	}

	rootContext, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	mongoClient, err := mongo.Connect(rootContext, options.Client().ApplyURI(cfg.Mongo.URI))
	if err != nil {
		log.Fatalf("connect mongodb: %v", err)
	}

	application, err := platformapp.Build(rootContext, mongoClient, mongoClient.Database(cfg.Mongo.Database), cfg, nil)
	if err != nil {
		log.Fatalf("build application: %v", err)
	}

	handler := application.Handler
	frontend, err := resolveFrontend(cfg.Server.FrontendRootDir)
	if err != nil {
		log.Printf("frontend assets unavailable, continuing with API-only mode: %v", err)
	} else {
		handler = wrapWithFrontend(frontend, handler)
	}

	server := &http.Server{
		Addr:              ":" + strconv.Itoa(cfg.Server.Port),
		Handler:           handler,
		ReadHeaderTimeout: cfg.Server.ReadHeaderTimeout,
	}

	go func() {
		log.Printf("platform server listening on http://localhost:%d", cfg.Server.Port)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("listen: %v", err)
		}
	}()

	<-rootContext.Done()

	shutdownContext, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutdownCancel()
	_ = server.Shutdown(shutdownContext)
	_ = application.Close(shutdownContext)
}

func loadConfig(path string) (platformconfig.AppConfig, error) {
	if _, err := os.Stat(path); err == nil {
		return platformconfig.LoadFromFile(path)
	}

	config := platformconfig.Default()
	return config, config.Validate()
}

func resolveFrontend(configuredRoot string) (*web.Frontend, error) {
	candidates := make([]string, 0, 3)
	if strings.TrimSpace(configuredRoot) != "" {
		candidates = append(candidates, configuredRoot)
	} else {
		candidates = append(candidates, "../webapp", "webapp")
	}

	var errorsByPath []string
	for _, candidate := range candidates {
		root := filepath.Clean(candidate)
		frontend := web.NewFrontend(root)
		if err := frontend.Validate(); err == nil {
			return frontend, nil
		} else {
			errorsByPath = append(errorsByPath, fmt.Sprintf("%s: %v", root, err))
		}
	}

	return nil, fmt.Errorf("no valid frontend root found (%s)", strings.Join(errorsByPath, "; "))
}

func wrapWithFrontend(frontend *web.Frontend, apiHandler http.Handler) http.Handler {
	return http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		if frontend.TryServeStatic(writer, request) {
			return
		}

		if strings.HasPrefix(request.URL.Path, "/api/") {
			apiHandler.ServeHTTP(writer, request)
			return
		}

		if request.Method == http.MethodGet {
			frontend.ServeIndex(writer, http.StatusOK)
			return
		}

		apiHandler.ServeHTTP(writer, request)
	})
}
