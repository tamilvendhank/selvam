package async

import (
	"context"

	aijobsvc "goserver/internal/service/aijob"
	"goserver/internal/worker/framework"
)

type BatchPollingWorker struct {
	workerBase
	service aijobsvc.BatchJobPollingService
}

func NewBatchPollingWorker(
	service aijobsvc.BatchJobPollingService,
	options AsyncWorkerOptions,
	clock framework.Clock,
) *BatchPollingWorker {
	return &BatchPollingWorker{
		workerBase: newWorkerBase(BatchPollingWorkerName, options, clock),
		service:    service,
	}
}

func (worker *BatchPollingWorker) RunOnce(ctx context.Context) framework.WorkerRunResult {
	if worker == nil {
		return dependencyFailure(BatchPollingWorkerName, "batch polling worker")
	}
	if err := worker.validate(ctx); err != nil {
		return validationFailure(worker.Name(), err)
	}
	if worker.service == nil {
		return dependencyFailure(worker.Name(), "batch job polling service")
	}

	result, err := worker.service.PollPendingBatchJobs(ctx, aijobsvc.PollPendingBatchJobsRequest{
		BatchJobID:       worker.options.BatchJobID,
		WorkflowRunID:    worker.options.WorkflowRunID,
		BookType:         worker.options.BookType,
		JobType:          worker.options.JobType,
		MaxJobs:          worker.options.MaxJobsPerRun,
		PollOnlyStatuses: worker.options.PollOnlyStatuses,
		Force:            worker.options.Force,
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

	failedCount := len(result.FailedJobs)
	polledCount := len(result.PolledJobIDs)
	if polledCount == 0 {
		polledCount = result.Summary.PolledCount
	}
	counts := workerCounts{
		Processed: polledCount,
		Succeeded: maxInt(polledCount-failedCount, 0),
		Failed:    failedCount,
		Skipped:   result.Summary.SkippedCount,
	}

	return buildWorkerResult(worker.workerBase, counts, result.PartialFailures, result.HasFailures(), map[string]any{
		"polled_job_ids":                   objectIDHexes(result.PolledJobIDs),
		"completed_job_ids":                batchJobRefIDs(result.CompletedJobs),
		"still_running_job_ids":            batchJobRefIDs(result.StillRunningJobs),
		"failed_job_ids":                   batchJobRefIDs(result.FailedJobs),
		"ready_for_reconciliation_job_ids": batchJobRefIDs(result.CompletedJobs),
		"status_change_count":              result.Summary.StatusChangeCount,
	})
}
