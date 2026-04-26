package continuation

import (
	"context"
	"errors"
	"fmt"

	domaincommon "goserver/internal/domain/common"
	domainworkflow "goserver/internal/domain/workflow"
	platformrepo "goserver/internal/platform/repository"
	servicecommon "goserver/internal/service/common"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

// Step/run writes use repository preconditions as the concurrency guard. If an
// admin path or worker advances the same workflow concurrently, stale writes are
// treated as resumable conflicts and the caller can safely retry.
func (service *workflowContinuationService) executeContinuationStep(
	ctx context.Context,
	execution *continuationExecutionContext,
	stepName domaincommon.WorkflowStepName,
	request ContinueWorkflowRequest,
) (stepExecutionOutcome, error) {
	switch execution.run.BookType {
	case domaincommon.BookTypeInvesting:
		return service.executeInvestingStep(ctx, execution, stepName, request)
	case domaincommon.BookTypeTrading:
		return service.executeTradingStep(ctx, execution, stepName, request)
	default:
		return stepExecutionOutcome{StepName: stepName}, fmt.Errorf("unsupported workflow bookType %q", execution.run.BookType)
	}
}

func (service *workflowContinuationService) executeStep(
	ctx context.Context,
	execution *continuationExecutionContext,
	stepName domaincommon.WorkflowStepName,
	request ContinueWorkflowRequest,
	fn stepExecutionFunc,
) (stepExecutionOutcome, error) {
	if fn == nil {
		return service.skipStep(ctx, execution, stepName, request, "step has no configured executor")
	}
	current := execution.steps[stepName]
	if shouldSkipCompletedStep(current, request.Force) {
		return stepExecutionOutcome{StepName: stepName, Skipped: true}, nil
	}
	if current != nil && isStepComplete(current) {
		return stepExecutionOutcome{StepName: stepName, Skipped: true}, nil
	}
	if current != nil && current.Status == domaincommon.WorkflowStepStatusFailed {
		return stepExecutionOutcome{StepName: stepName, Failed: true}, fmt.Errorf("workflow step %q previously failed and cannot be retried without resetting the terminal step run", stepName)
	}

	step, err := service.ensureStepRun(ctx, execution.run, stepName, current)
	if err != nil {
		return stepExecutionOutcome{StepName: stepName, Failed: true}, err
	}
	started, err := service.markStepRunning(ctx, step, request)
	if err != nil {
		return stepExecutionOutcome{StepName: stepName, Failed: true}, err
	}
	execution.steps[stepName] = started

	outcome, err := fn(ctx, execution, request)
	outcome.StepName = stepName
	if err != nil {
		failed, markErr := service.markStepFailed(ctx, started, stepName, request, err)
		if markErr == nil && failed != nil {
			execution.steps[stepName] = failed
		}
		if markErr != nil {
			outcome.PartialFailures = append(outcome.PartialFailures, stepPartialFailure(execution.run.ID, stepName, markErr))
		}
		outcome.Failed = true
		return outcome, err
	}

	completed, err := service.markStepCompleted(ctx, started, stepName, request, outcome.Metadata)
	if err != nil {
		outcome.Failed = true
		return outcome, err
	}
	execution.steps[stepName] = completed
	outcome.Executed = true
	return outcome, nil
}

func (service *workflowContinuationService) skipStep(
	ctx context.Context,
	execution *continuationExecutionContext,
	stepName domaincommon.WorkflowStepName,
	request ContinueWorkflowRequest,
	reason string,
) (stepExecutionOutcome, error) {
	current := execution.steps[stepName]
	if isStepComplete(current) {
		return stepExecutionOutcome{StepName: stepName, Skipped: true}, nil
	}
	if current != nil && current.Status == domaincommon.WorkflowStepStatusFailed {
		return stepExecutionOutcome{StepName: stepName, Failed: true}, fmt.Errorf("workflow step %q previously failed and cannot be skipped without resetting the terminal step run", stepName)
	}

	step, err := service.ensureStepRun(ctx, execution.run, stepName, current)
	if err != nil {
		return stepExecutionOutcome{StepName: stepName, Failed: true}, err
	}
	if step.Status == domaincommon.WorkflowStepStatusPending {
		skipped, err := service.workflowSteps.MarkSkipped(ctx, step.ID, platformrepo.WorkflowStepSkipPatch{
			SkippedAt: service.now().UTC(),
			Reason:    reason,
			Metadata: &platformrepo.MetadataPatch{
				Values: map[string]any{
					"skipped": true,
					"reason":  reason,
				},
			},
			ExpectedCurrentStatuses: []domaincommon.WorkflowStepStatus{domaincommon.WorkflowStepStatusPending},
			Mutation:                mutationMetadata(service.now().UTC(), request.InitiatedBy, reason),
		})
		if err != nil {
			return stepExecutionOutcome{StepName: stepName, Failed: true}, err
		}
		execution.steps[stepName] = skipped
		return stepExecutionOutcome{StepName: stepName, Skipped: true}, nil
	}

	completed, err := service.markStepCompleted(ctx, step, stepName, request, map[string]any{
		"skipped": true,
		"reason":  reason,
	})
	if err != nil {
		return stepExecutionOutcome{StepName: stepName, Failed: true}, err
	}
	execution.steps[stepName] = completed
	return stepExecutionOutcome{StepName: stepName, Skipped: true}, nil
}

func (service *workflowContinuationService) ensureStepRun(
	ctx context.Context,
	run *domainworkflow.WorkflowRun,
	stepName domaincommon.WorkflowStepName,
	current *domainworkflow.WorkflowStepRun,
) (*domainworkflow.WorkflowStepRun, error) {
	if service.workflowSteps == nil {
		return nil, fmt.Errorf("workflow step repository is required")
	}
	if run == nil {
		return nil, fmt.Errorf("workflow run is required")
	}
	if current != nil {
		return current, nil
	}

	now := service.now().UTC()
	created, err := service.workflowSteps.Create(ctx, &domainworkflow.WorkflowStepRun{
		WorkflowRunID: run.ID,
		StepName:      stepName,
		Status:        domaincommon.WorkflowStepStatusPending,
		CreatedAt:     now,
		UpdatedAt:     now,
		SchemaVersion: domaincommon.SchemaVersion1,
	})
	if err != nil {
		if !errors.Is(err, platformrepo.ErrAlreadyExists) {
			return nil, fmt.Errorf("create workflow step %q: %w", stepName, err)
		}
		existing, getErr := service.workflowSteps.GetByWorkflowRunAndStepName(ctx, run.ID, stepName)
		if getErr != nil {
			return nil, fmt.Errorf("load existing workflow step %q after create conflict: %w", stepName, getErr)
		}
		return existing, nil
	}
	return created, nil
}

func (service *workflowContinuationService) markStepRunning(
	ctx context.Context,
	step *domainworkflow.WorkflowStepRun,
	request ContinueWorkflowRequest,
) (*domainworkflow.WorkflowStepRun, error) {
	if step == nil {
		return nil, fmt.Errorf("workflow step is required")
	}
	switch step.Status {
	case domaincommon.WorkflowStepStatusRunning:
		return step, nil
	case domaincommon.WorkflowStepStatusPending, domaincommon.WorkflowStepStatusWaitingExternal:
		updated, err := service.workflowSteps.MarkStarted(ctx, step.ID, platformrepo.WorkflowStepStartPatch{
			StartedAt: service.now().UTC(),
			Metadata: &platformrepo.MetadataPatch{
				Values: map[string]any{
					"continuedBy": request.InitiatedBy,
				},
			},
			ExpectedCurrentStatuses: []domaincommon.WorkflowStepStatus{
				domaincommon.WorkflowStepStatusPending,
				domaincommon.WorkflowStepStatusWaitingExternal,
			},
			Mutation: mutationMetadata(service.now().UTC(), request.InitiatedBy, "workflow continuation step started"),
		})
		if err != nil {
			return nil, fmt.Errorf("mark step %q running: %w", step.StepName, err)
		}
		return updated, nil
	default:
		return nil, fmt.Errorf("workflow step %q cannot run from status %q", step.StepName, step.Status)
	}
}

func (service *workflowContinuationService) markStepCompleted(
	ctx context.Context,
	step *domainworkflow.WorkflowStepRun,
	stepName domaincommon.WorkflowStepName,
	request ContinueWorkflowRequest,
	metadata map[string]any,
) (*domainworkflow.WorkflowStepRun, error) {
	if step == nil {
		return nil, fmt.Errorf("workflow step is required")
	}
	values := map[string]any{
		"continuedBy": request.InitiatedBy,
	}
	for key, value := range metadata {
		values[key] = value
	}
	updated, err := service.workflowSteps.MarkCompleted(ctx, step.ID, platformrepo.WorkflowStepCompletionPatch{
		CompletedAt: service.now().UTC(),
		Metadata: &platformrepo.MetadataPatch{
			Values: values,
		},
		ExpectedCurrentStatuses: []domaincommon.WorkflowStepStatus{
			domaincommon.WorkflowStepStatusRunning,
			domaincommon.WorkflowStepStatusWaitingExternal,
		},
		Mutation: mutationMetadata(service.now().UTC(), request.InitiatedBy, "workflow continuation step completed"),
	})
	if err != nil {
		return nil, fmt.Errorf("mark step %q completed: %w", stepName, err)
	}
	return updated, nil
}

func (service *workflowContinuationService) markStepFailed(
	ctx context.Context,
	step *domainworkflow.WorkflowStepRun,
	stepName domaincommon.WorkflowStepName,
	request ContinueWorkflowRequest,
	cause error,
) (*domainworkflow.WorkflowStepRun, error) {
	if step == nil {
		return nil, fmt.Errorf("workflow step is required")
	}
	errorSummary := cause.Error()
	updated, err := service.workflowSteps.MarkFailed(ctx, step.ID, platformrepo.WorkflowStepFailurePatch{
		FailedAt:     service.now().UTC(),
		ErrorSummary: errorSummary,
		Metadata: &platformrepo.MetadataPatch{
			Values: map[string]any{
				"continuedBy": request.InitiatedBy,
				"error":       errorSummary,
			},
		},
		ExpectedCurrentStatuses: []domaincommon.WorkflowStepStatus{
			domaincommon.WorkflowStepStatusRunning,
			domaincommon.WorkflowStepStatusWaitingExternal,
		},
		Mutation: mutationMetadata(service.now().UTC(), request.InitiatedBy, "workflow continuation step failed"),
	})
	if err != nil {
		return nil, fmt.Errorf("mark step %q failed: %w", stepName, err)
	}
	return updated, nil
}

func stepPartialFailure(
	workflowRunID primitive.ObjectID,
	stepName domaincommon.WorkflowStepName,
	err error,
) servicecommon.PartialFailure {
	return servicecommon.PartialFailure{
		Scope:         servicecommon.FailureScopeContinuation,
		WorkflowRunID: workflowRunID,
		ID:            workflowRunID,
		Code:          fmt.Sprintf("%s_failed", stepName),
		Message:       err.Error(),
	}
}
