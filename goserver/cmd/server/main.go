package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	platformapp "goserver/internal/platform/app"
	platformconfig "goserver/internal/platform/config"

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

	application, err := platformapp.Build(rootContext, mongoClient, mongoClient.Database(cfg.Mongo.Database), cfg)
	if err != nil {
		log.Fatalf("build application: %v", err)
	}

	server := &http.Server{
		Addr:              ":" + strconv.Itoa(cfg.Server.Port),
		Handler:           application.Handler,
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
