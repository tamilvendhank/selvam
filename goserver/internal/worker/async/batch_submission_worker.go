package async

import (
	"context"

	aijobsvc "goserver/internal/service/aijob"
	"goserver/internal/worker/framework"
)

type BatchSubmissionWorker struct {
	workerBase
	service aijobsvc.BatchJobSubmissionService
}

func NewBatchSubmissionWorker(
	service aijobsvc.BatchJobSubmissionService,
	options AsyncWorkerOptions,
	clock framework.Clock,
) *BatchSubmissionWorker {
	return &BatchSubmissionWorker{
		workerBase: newWorkerBase(BatchSubmissionWorkerName, options, clock),
		service:    service,
	}
}

func (worker *BatchSubmissionWorker) RunOnce(ctx context.Context) framework.WorkerRunResult {
	if worker == nil {
		return dependencyFailure(BatchSubmissionWorkerName, "batch submission worker")
	}
	if err := worker.validate(ctx); err != nil {
		return validationFailure(worker.Name(), err)
	}
	if worker.service == nil {
		return dependencyFailure(worker.Name(), "batch job submission service")
	}

	result, err := worker.service.SubmitPendingBatchJobs(ctx, aijobsvc.SubmitPendingBatchJobsRequest{
		WorkflowRunID: worker.options.WorkflowRunID,
		BatchJobID:    worker.options.BatchJobID,
		BookType:      worker.options.BookType,
		JobType:       worker.options.JobType,
		MaxJobs:       worker.options.MaxJobsPerRun,
		DryRun:        worker.options.DryRun,
		Force:         worker.options.Force,
		InitiatedBy:   worker.options.InitiatedBy,
		CorrelationID: worker.options.CorrelationID,
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

	submittedCount := result.SubmissionCount
	if submittedCount == 0 {
		submittedCount = len(result.SubmittedJobIDs)
	}
	failedCount := result.FailureCount
	if failedCount == 0 {
		failedCount = len(result.FailedJobIDs)
	}
	counts := workerCounts{
		Processed: submittedCount + len(result.SkippedJobIDs) + failedCount,
		Succeeded: submittedCount,
		Failed:    failedCount,
		Skipped:   len(result.SkippedJobIDs),
	}
	if counts.Processed == 0 {
		counts.Processed = result.Summary.AttemptedCount
	}

	return buildWorkerResult(worker.workerBase, counts, result.PartialFailures, result.HasFailures(), map[string]any{
		"submitted_job_ids": objectIDHexes(result.SubmittedJobIDs),
		"skipped_job_ids":   objectIDHexes(result.SkippedJobIDs),
		"failed_job_ids":    objectIDHexes(result.FailedJobIDs),
		"provider_handles":  len(result.ProviderHandles),
		"needs_follow_up":   result.NeedsFollowUp,
	})
}
