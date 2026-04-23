package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"goserver/internal/config"
	"goserver/internal/db"
	"goserver/internal/httpapi"
	"goserver/internal/logging"
	openaiapi "goserver/internal/openai"
	platformapp "goserver/internal/platform/app"
	"goserver/internal/repository"
	"goserver/internal/service"
	"goserver/internal/web"

	"go.uber.org/zap"
)

func main() {
	bootstrapLogger := logging.NewBootstrap()
	defer logging.Sync(bootstrapLogger)

	cfg, err := config.Load()
	if err != nil {
		bootstrapLogger.Fatal("failed to load config", zap.Error(err))
	}

	logger, err := logging.New(cfg.Logging)
	if err != nil {
		bootstrapLogger.Fatal("failed to configure logger", zap.Error(err))
	}
	defer logging.Sync(logger)

	serverLogger := logger.Named("goserver")

	frontend := web.NewFrontend(cfg.Frontend.RootDir)
	if err := frontend.Validate(); err != nil {
		serverLogger.Fatal("failed to validate frontend assets", zap.Error(err))
	}

	rootContext, cancelRoot := context.WithCancel(context.Background())
	defer cancelRoot()

	mongoClient, err := db.Connect(rootContext, cfg)
	if err != nil {
		serverLogger.Fatal("failed to connect to mongodb", zap.Error(err))
	}
	defer mongoClient.Close(rootContext)

	openaiClient := openaiapi.NewClient(cfg.OpenAI)
	jobsRepository := repository.NewJobsRepository(mongoClient.Database(), cfg.MongoDB.JobsCollectionName)
	submissionIterationsRepository := repository.NewSubmissionIterationsRepository(mongoClient.Database(), cfg.MongoDB.SubmissionIterationsCollection)
	proceduresRepository := repository.NewProceduresRepository(mongoClient.Database(), cfg.MongoDB.ProceduresCollectionName)
	procedureExecutionsRepository := repository.NewProcedureExecutionsRepository(mongoClient.Database(), cfg.MongoDB.ProcedureExecutionsCollection)

	jobsService := service.NewJobsService(cfg, jobsRepository, submissionIterationsRepository, openaiClient, &service.UnconfiguredToolExecutor{})
	proceduresService := service.NewProceduresService(proceduresRepository)
	procedureExecutionsService := service.NewProcedureExecutionsService(procedureExecutionsRepository, proceduresRepository, jobsService)
	jobsService.RegisterRefreshObserver(procedureExecutionsService)
	if cfg.Worker.Enabled {
		service.NewSubmissionRefreshWorker(
			jobsService,
			procedureExecutionsService,
			cfg.Worker.SubmissionRefreshInterval,
			logger.Named("submission_refresh_worker"),
		).Start(rootContext)
	}

	platformApplication, err := platformapp.Build(rootContext, mongoClient.MongoClient(), mongoClient.Database(), cfg.Platform, jobsService)
	if err != nil {
		serverLogger.Fatal("failed to build platform application", zap.Error(err))
	}

	handler := httpapi.NewHandler(
		frontend,
		jobsService,
		proceduresService,
		procedureExecutionsService,
		platformApplication.Handler,
		logger.Named("httpapi"),
	)
	server := &http.Server{
		Addr:              ":" + strconv.Itoa(cfg.Port),
		Handler:           handler,
		ReadHeaderTimeout: 10 * time.Second,
	}

	go func() {
		serverLogger.Info("go server started", zap.String("url", fmt.Sprintf("http://localhost:%d", cfg.Port)))
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			serverLogger.Fatal("server stopped unexpectedly", zap.Error(err))
		}
	}()

	signalChan := make(chan os.Signal, 1)
	signal.Notify(signalChan, syscall.SIGINT, syscall.SIGTERM)
	<-signalChan
	cancelRoot()

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := server.Shutdown(shutdownCtx); err != nil {
		serverLogger.Error("failed to shut down server cleanly", zap.Error(err))
	}
	if err := mongoClient.Close(shutdownCtx); err != nil {
		serverLogger.Error("failed to close mongodb cleanly", zap.Error(err))
	}
}
