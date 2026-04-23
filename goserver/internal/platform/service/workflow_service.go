package service

import (
	"context"

	"goserver/internal/platform/domain"
	"goserver/internal/platform/ports"
)

type DefaultWorkflowService struct {
	repository ports.WorkflowRunRepository
}

func NewWorkflowService(repository ports.WorkflowRunRepository) *DefaultWorkflowService {
	return &DefaultWorkflowService{repository: repository}
}

func (service *DefaultWorkflowService) ListWorkflowRuns(ctx context.Context, filter ports.WorkflowRunListFilter) ([]*domain.WorkflowRun, error) {
	return service.repository.List(ctx, filter)
}

func (service *DefaultWorkflowService) GetWorkflowRun(ctx context.Context, id string) (*domain.WorkflowRun, error) {
	run, err := service.repository.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if run == nil {
		return nil, ErrNotFound
	}

	return run, nil
}

func (service *DefaultWorkflowService) GetWorkflowSummary(ctx context.Context, id string) (map[string]any, error) {
	run, err := service.GetWorkflowRun(ctx, id)
	if err != nil {
		return nil, err
	}

	completedSteps := 0
	waitingSteps := 0
	failedSteps := 0
	for _, step := range run.StepStatuses {
		switch step.Status {
		case domain.WorkflowStepStatusCompleted:
			completedSteps++
		case domain.WorkflowStepStatusWaitingAsync:
			waitingSteps++
		case domain.WorkflowStepStatusFailed:
			failedSteps++
		}
	}

	return map[string]any{
		"id":                    run.ID,
		"bookType":              run.BookType,
		"runType":               run.RunType,
		"status":                run.Status,
		"companiesScannedCount": run.CompaniesScannedCount,
		"reviewsCreatedCount":   run.ReviewsCreatedCount,
		"errorsCount":           run.ErrorsCount,
		"completedSteps":        completedSteps,
		"waitingSteps":          waitingSteps,
		"failedSteps":           failedSteps,
		"dryRun":                run.DryRun,
	}, nil
}
