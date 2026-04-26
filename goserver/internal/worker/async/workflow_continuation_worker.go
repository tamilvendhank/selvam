package async

import (
	"context"

	continuationsvc "goserver/internal/service/continuation"
	"goserver/internal/worker/framework"
)

type WorkflowContinuationWorker struct {
	workerBase
	service continuationsvc.WorkflowContinuationService
}

func NewWorkflowContinuationWorker(
	service continuationsvc.WorkflowContinuationService,
	options AsyncWorkerOptions,
	clock framework.Clock,
) *WorkflowContinuationWorker {
	return &WorkflowContinuationWorker{
		workerBase: newWorkerBase(WorkflowContinuationWorkerName, options, clock),
		service:    service,
	}
}

func (worker *WorkflowContinuationWorker) RunOnce(ctx context.Context) framework.WorkerRunResult {
	if worker == nil {
		return dependencyFailure(WorkflowContinuationWorkerName, "workflow continuation worker")
	}
	if err := worker.validate(ctx); err != nil {
		return validationFailure(worker.Name(), err)
	}
	if worker.service == nil {
		return dependencyFailure(worker.Name(), "workflow continuation service")
	}

	result, err := worker.service.ContinueEligibleWorkflows(ctx, continuationsvc.ContinueEligibleWorkflowsRequest{
		WorkflowRunID:    worker.options.WorkflowRunID,
		BookType:         worker.options.BookType,
		MaxWorkflows:     worker.options.MaxWorkflowsPerRun,
		DryRun:           worker.options.DryRun,
		Force:            worker.options.Force,
		AllowedStepRange: worker.options.AllowedStepRange,
		InitiatedBy:      worker.options.InitiatedBy,
		CorrelationID:    worker.options.CorrelationID,
	})
	if err != nil {
		if expectedNoopError(err) {
			return noopResult(worker.Name(), err, worker.options.metadata())
		}
		return framework.ResultFromError(worker.Name(), err)
	}
	if result == nil {
		return noopResult(worker.Name(), nil, worker.options.metadata())
	}

	continuedCount := len(result.ContinuedWorkflowRunIDs)
	completedCount := len(result.CompletedWorkflowRunIDs)
	blockedCount := len(result.StillBlockedWorkflowRunIDs)
	failedCount := len(result.FailedWorkflowRunIDs)
	counts := workerCounts{
		Processed: continuedCount + completedCount + blockedCount + failedCount,
		Succeeded: continuedCount + completedCount,
		Failed:    failedCount,
		Skipped:   blockedCount,
	}
	if counts.Processed == 0 {
		counts.Processed = result.Summary.AttemptedCount
	}

	return buildWorkerResult(worker.workerBase, counts, result.PartialFailures, result.HasFailures(), map[string]any{
		"continued_workflow_run_ids": objectIDHexes(result.ContinuedWorkflowRunIDs),
		"completed_workflow_run_ids": objectIDHexes(result.CompletedWorkflowRunIDs),
		"blocked_workflow_run_ids":   objectIDHexes(result.StillBlockedWorkflowRunIDs),
		"failed_workflow_run_ids":    objectIDHexes(result.FailedWorkflowRunIDs),
		"decision_count":             len(result.Decisions),
	})
}
