package trading

import (
	"context"
	"time"

	platformconfig "goserver/internal/platform/config"
	"goserver/internal/platform/domain"
	"goserver/internal/platform/ports"
	platformservice "goserver/internal/platform/service"
	"goserver/internal/platform/workflow"
)

type Service struct {
	config        platformconfig.AppConfig
	workflowRuns  ports.WorkflowRunRepository
	configService ports.ConfigService
	timeProvider  ports.TimeProvider
}

func NewService(
	config platformconfig.AppConfig,
	workflowRuns ports.WorkflowRunRepository,
	configService ports.ConfigService,
	timeProvider ports.TimeProvider,
) *Service {
	return &Service{
		config:        config,
		workflowRuns:  workflowRuns,
		configService: configService,
		timeProvider:  platformservice.ResolveTimeProviderForWorkflow(timeProvider),
	}
}

func (service *Service) Start(ctx context.Context, request ports.StartTradingWorkflowRequest) (*domain.WorkflowRun, error) {
	if request.RunType == "" {
		request.RunType = domain.WorkflowRunTypeManual
	}
	if request.IdempotencyKey != "" {
		existing, err := service.workflowRuns.GetByIdempotencyKey(ctx, request.IdempotencyKey)
		if err != nil {
			return nil, err
		}
		if existing != nil {
			return existing, nil
		}
	}

	snapshot, err := service.configService.CreateSnapshot(ctx, domain.BookTypeTrading, "trend_momentum")
	if err != nil {
		return nil, err
	}

	now := service.timeProvider.Now()
	run := &domain.WorkflowRun{
		BookType:         domain.BookTypeTrading,
		RunType:          request.RunType,
		Mode:             "trend_momentum",
		Status:           domain.WorkflowRunStatusCompleted,
		StartedAt:        now,
		CompletedAt:      &now,
		ConfigSnapshotID: snapshot.ID,
		DryRun:           request.DryRun,
		IdempotencyKey:   request.IdempotencyKey,
		Notes:            request.Notes,
		RequestMetadata: map[string]any{
			"requestedBy": request.RequestedBy,
		},
		StepStatuses:  buildTradingSteps(now),
		SchemaVersion: service.config.SchemaVersion,
		CreatedAt:     now,
		UpdatedAt:     now,
	}

	return service.workflowRuns.Create(ctx, run)
}

func (service *Service) Resume(ctx context.Context, workflowRunID string) (*domain.WorkflowRun, error) {
	return service.workflowRuns.GetByID(ctx, workflowRunID)
}

func (service *Service) Reconcile(ctx context.Context, workflowRunID string) (*domain.WorkflowRun, error) {
	return service.workflowRuns.GetByID(ctx, workflowRunID)
}

func buildTradingSteps(now time.Time) []domain.WorkflowStepStatus {
	steps := make([]domain.WorkflowStepStatus, 0, len(workflow.TradingStepNames()))
	for _, name := range workflow.TradingStepNames() {
		steps = append(steps, domain.WorkflowStepStatus{
			StepName:       name,
			Status:         domain.WorkflowStepStatusCompleted,
			StartedAt:      &now,
			CompletedAt:    &now,
			OutputSnapshot: map[string]any{"placeholder": true},
		})
	}

	return steps
}
