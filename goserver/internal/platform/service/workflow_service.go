package service

import (
	"context"

	"goserver/internal/platform/domain"
	"goserver/internal/platform/ports"
)

type DefaultWorkflowService struct {
	repository        ports.WorkflowRunRepository
	stepRuns          ports.WorkflowStepRunRepository
	investingWorkflow ports.InvestingWorkflowService
	tradingWorkflow   ports.TradingWorkflowService
}

func NewWorkflowService(
	repository ports.WorkflowRunRepository,
	stepRuns ports.WorkflowStepRunRepository,
	investingWorkflow ports.InvestingWorkflowService,
	tradingWorkflow ports.TradingWorkflowService,
) *DefaultWorkflowService {
	return &DefaultWorkflowService{
		repository:        repository,
		stepRuns:          stepRuns,
		investingWorkflow: investingWorkflow,
		tradingWorkflow:   tradingWorkflow,
	}
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

	steps, err := service.ListWorkflowSteps(ctx, id)
	if err != nil {
		return nil, err
	}

	completedSteps := 0
	waitingSteps := 0
	failedSteps := 0
	for _, step := range steps {
		switch step.Status {
		case domain.WorkflowStepStatusCompleted:
			completedSteps++
		case domain.WorkflowStepStatusWaitingAsync, domain.WorkflowStepStatusWaitingExternal:
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

func (service *DefaultWorkflowService) GetWorkflowStatus(ctx context.Context, id string) (map[string]any, error) {
	run, err := service.GetWorkflowRun(ctx, id)
	if err != nil {
		return nil, err
	}

	steps, err := service.ListWorkflowSteps(ctx, id)
	if err != nil {
		return nil, err
	}
	return map[string]any{
		"id":          run.ID,
		"bookType":    run.BookType,
		"status":      run.Status,
		"startedAt":   run.StartedAt,
		"completedAt": run.CompletedAt,
		"steps":       steps,
	}, nil
}

func (service *DefaultWorkflowService) ListWorkflowSteps(ctx context.Context, workflowRunID string) ([]*domain.WorkflowStepRun, error) {
	if service.stepRuns == nil {
		return []*domain.WorkflowStepRun{}, nil
	}

	return service.stepRuns.List(ctx, ports.WorkflowStepRunListFilter{
		WorkflowRunID: workflowRunID,
		Limit:         200,
	})
}

func (service *DefaultWorkflowService) ResumeWorkflow(ctx context.Context, id string) (*domain.WorkflowRun, error) {
	run, err := service.GetWorkflowRun(ctx, id)
	if err != nil {
		return nil, err
	}
	return service.delegateRun(ctx, run, true)
}

func (service *DefaultWorkflowService) ReconcileWorkflow(ctx context.Context, id string) (*domain.WorkflowRun, error) {
	run, err := service.GetWorkflowRun(ctx, id)
	if err != nil {
		return nil, err
	}
	return service.delegateRun(ctx, run, false)
}

func (service *DefaultWorkflowService) delegateRun(ctx context.Context, run *domain.WorkflowRun, resume bool) (*domain.WorkflowRun, error) {
	switch run.BookType {
	case domain.BookTypeInvesting:
		if service.investingWorkflow == nil {
			return run, nil
		}
		if resume {
			return service.investingWorkflow.Resume(ctx, run.ID)
		}
		return service.investingWorkflow.Reconcile(ctx, run.ID)
	case domain.BookTypeTrading:
		if service.tradingWorkflow == nil {
			return run, nil
		}
		if resume {
			return service.tradingWorkflow.Resume(ctx, run.ID)
		}
		return service.tradingWorkflow.Reconcile(ctx, run.ID)
	default:
		return run, nil
	}
}
