package async

import (
	"context"

	aijobsvc "goserver/internal/service/aijob"
	"goserver/internal/worker/framework"
)

type BatchReconciliationWorker struct {
	workerBase
	service aijobsvc.BatchReconciliationService
}

func NewBatchReconciliationWorker(
	service aijobsvc.BatchReconciliationService,
	options AsyncWorkerOptions,
	clock framework.Clock,
) *BatchReconciliationWorker {
	return &BatchReconciliationWorker{
		workerBase: newWorkerBase(BatchReconciliationWorkerName, options, clock),
		service:    service,
	}
}

func (worker *BatchReconciliationWorker) RunOnce(ctx context.Context) framework.WorkerRunResult {
	if worker == nil {
		return dependencyFailure(BatchReconciliationWorkerName, "batch reconciliation worker")
	}
	if err := worker.validate(ctx); err != nil {
		return validationFailure(worker.Name(), err)
	}
	if worker.service == nil {
		return dependencyFailure(worker.Name(), "batch reconciliation service")
	}

	result, err := worker.service.ReconcilePendingBatchJobs(ctx, aijobsvc.ReconcilePendingBatchJobsRequest{
		BatchJobID:            worker.options.BatchJobID,
		WorkflowRunID:         worker.options.WorkflowRunID,
		BookType:              worker.options.BookType,
		JobType:               worker.options.JobType,
		MaxJobs:               worker.options.MaxJobsPerRun,
		Force:                 worker.options.Force,
		IncludeCompletedItems: worker.options.IncludeCompletedItems,
		InitiatedBy:           worker.options.InitiatedBy,
		CorrelationID:         worker.options.CorrelationID,
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

	reconciledCount := len(result.ReconciledJobIDs)
	if reconciledCount == 0 {
		reconciledCount = result.Summary.ReconciledJobCount
	}
	counts := workerCounts{
		Processed: reconciledCount,
		Succeeded: result.ItemsCompleted,
		Failed:    result.ItemsFailed + result.ItemsInvalid,
		Skipped:   result.ItemsStillPending,
	}

	return buildWorkerResult(worker.workerBase, counts, result.PartialFailures, result.HasFailures(), map[string]any{
		"reconciled_job_ids":                      objectIDHexes(result.ReconciledJobIDs),
		"items_completed":                         result.ItemsCompleted,
		"items_failed":                            result.ItemsFailed,
		"items_invalid":                           result.ItemsInvalid,
		"items_still_pending":                     result.ItemsStillPending,
		"ready_for_validation_count":              result.ReadyForValidationCount,
		"ready_for_continuation_workflow_run_ids": objectIDHexes(result.ReadyForContinuationWorkflowRunIDs),
	})
}
