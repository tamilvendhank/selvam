package app

import (
	"context"
	"net/http"

	legacyconfig "goserver/internal/config"
	legacyopenai "goserver/internal/openai"
	platformhttp "goserver/internal/platform/api/http"
	platformconfig "goserver/internal/platform/config"
	"goserver/internal/platform/ports"
	platformai "goserver/internal/platform/provider/ai"
	mongorepo "goserver/internal/platform/repository/mongo"
	platformservice "goserver/internal/platform/service"
	investingworkflow "goserver/internal/platform/workflow/investing"
	tradingworkflow "goserver/internal/platform/workflow/trading"
	legacyrepo "goserver/internal/repository"
	legacyservice "goserver/internal/service"

	"go.mongodb.org/mongo-driver/mongo"
)

type Application struct {
	Handler     http.Handler
	MongoClient *mongo.Client
	Database    *mongo.Database
	Close       func(context.Context) error
	Trading     *tradingworkflow.Service
	Investing   *investingworkflow.Service
	LegacyJobs  *legacyservice.JobsService
}

func Build(
	ctx context.Context,
	mongoClient *mongo.Client,
	database *mongo.Database,
	config platformconfig.AppConfig,
	legacyJobsService *legacyservice.JobsService,
) (*Application, error) {
	collections := mongorepo.NewCollections(database, config.Mongo.Collections)
	if err := mongorepo.EnsureIndexes(ctx, database, config.Mongo.Collections); err != nil {
		return nil, err
	}

	companyRepository := mongorepo.NewCompanyRepository(collections.Companies)
	reviewRepository := mongorepo.NewCompanyReviewRepository(collections.CompanyReviews)
	thesisRepository := mongorepo.NewThesisRepository(collections.InvestmentTheses)
	workflowRunRepository := mongorepo.NewWorkflowRunRepository(collections.WorkflowRuns)
	configSnapshotRepository := mongorepo.NewConfigSnapshotRepository(collections.ConfigSnapshots)
	capitalAllocationRepository := mongorepo.NewCapitalAllocationRepository(collections.CapitalAllocationRuns)
	overrideRepository := mongorepo.NewManualOverrideRepository(collections.ManualOverrides)
	positionRepository := mongorepo.NewPositionRepository(collections.CurrentPositions)

	configService := platformservice.NewConfigService(config, configSnapshotRepository, nil)
	scorecardService := platformservice.NewScorecardService(config)
	actionMappingService := platformservice.NewActionMappingService(config)
	changeDetectionService := platformservice.NewChangeDetectionService(config)
	reviewService := platformservice.NewReviewService(reviewRepository, thesisRepository, actionMappingService, changeDetectionService)
	companyService := platformservice.NewCompanyService(companyRepository, reviewRepository, thesisRepository, positionRepository)
	workflowService := platformservice.NewWorkflowService(workflowRunRepository)
	capitalAllocationService := platformservice.NewCapitalAllocationService(capitalAllocationRepository)
	overrideService := platformservice.NewOverrideService(overrideRepository, reviewRepository)
	projectionService := platformservice.NewProjectionService(positionRepository)

	var aiReviewEngine = ports.AIReviewEngine(&platformai.NoopAIReviewEngine{})

	if config.AsyncAI.Enabled && legacyJobsService != nil {
		legacyJobsRepository := legacyrepo.NewJobsRepository(database, config.Mongo.Collections.AIBatchJobs)
		aiReviewEngine = platformai.NewLegacyBatchAIReviewEngine(legacyJobsService, legacyJobsRepository)
	} else if config.AsyncAI.Enabled && config.AsyncAI.APIKey != "" {
		legacyCfg := legacyconfig.Config{
			MongoDB: legacyconfig.MongoConfig{
				JobsCollectionName:             config.Mongo.Collections.AIBatchJobs,
				SubmissionIterationsCollection: config.Mongo.Collections.AIBatchIterations,
			},
			OpenAI: legacyconfig.OpenAIConfig{
				APIKey:               config.AsyncAI.APIKey,
				Model:                config.AsyncAI.Model,
				ResponseInstructions: config.AsyncAI.ResponseInstructions,
				BatchEndpoint:        config.AsyncAI.BatchEndpoint,
				CompletionWindow:     config.AsyncAI.CompletionWindow,
				BaseURL:              config.AsyncAI.BaseURL,
			},
			Worker: legacyconfig.WorkerConfig{
				Enabled:                   config.AsyncAI.Worker.Enabled,
				SubmissionRefreshInterval: config.AsyncAI.Worker.RefreshInterval,
				MinBatchRefreshAge:        config.AsyncAI.Worker.MinBatchRefreshAge,
				FollowUpClaimTimeout:      config.AsyncAI.Worker.FollowUpClaimTimeout,
				MaxBatchesPerPass:         config.AsyncAI.Worker.MaxBatchesPerPass,
			},
		}
		legacyJobsRepository := legacyrepo.NewJobsRepository(database, config.Mongo.Collections.AIBatchJobs)
		legacyIterationsRepository := legacyrepo.NewSubmissionIterationsRepository(database, config.Mongo.Collections.AIBatchIterations)
		legacyOpenAIClient := legacyopenai.NewClient(legacyCfg.OpenAI)
		legacyJobsService = legacyservice.NewJobsService(legacyCfg, legacyJobsRepository, legacyIterationsRepository, legacyOpenAIClient, &legacyservice.UnconfiguredToolExecutor{})
		if config.AsyncAI.Worker.Enabled && config.AsyncAI.Worker.RefreshInterval > 0 {
			legacyservice.NewSubmissionRefreshWorker(legacyJobsService, nil, config.AsyncAI.Worker.RefreshInterval, nil).Start(ctx)
		}
		aiReviewEngine = platformai.NewLegacyBatchAIReviewEngine(legacyJobsService, legacyJobsRepository)
	}

	investingService := investingworkflow.NewService(
		config,
		companyRepository,
		workflowRunRepository,
		configService,
		scorecardService,
		aiReviewEngine,
		nil,
	)
	tradingService := tradingworkflow.NewService(config, workflowRunRepository, configService, nil)

	handler := platformhttp.NewAPI(
		companyService,
		reviewService,
		workflowService,
		investingService,
		capitalAllocationService,
		configService,
		overrideService,
		projectionService,
	)

	return &Application{
		Handler:     handler,
		MongoClient: mongoClient,
		Database:    database,
		Close: func(closeContext context.Context) error {
			return mongoClient.Disconnect(closeContext)
		},
		Trading:    tradingService,
		Investing:  investingService,
		LegacyJobs: legacyJobsService,
	}, nil
}
