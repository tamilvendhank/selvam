package async

import (
	"context"

	finalizationsvc "goserver/internal/service/finalization"
	"goserver/internal/worker/framework"
)

type ReviewFinalizationWorker struct {
	workerBase
	service finalizationsvc.ReviewFinalizationService
}

func NewReviewFinalizationWorker(
	service finalizationsvc.ReviewFinalizationService,
	options AsyncWorkerOptions,
	clock framework.Clock,
) *ReviewFinalizationWorker {
	return &ReviewFinalizationWorker{
		workerBase: newWorkerBase(ReviewFinalizationWorkerName, options, clock),
		service:    service,
	}
}

func (worker *ReviewFinalizationWorker) RunOnce(ctx context.Context) framework.WorkerRunResult {
	if worker == nil {
		return dependencyFailure(ReviewFinalizationWorkerName, "review finalization worker")
	}
	if err := worker.validate(ctx); err != nil {
		return validationFailure(worker.Name(), err)
	}
	if worker.service == nil {
		return dependencyFailure(worker.Name(), "review finalization service")
	}

	result, err := worker.service.FinalizeEligibleReviews(ctx, finalizationsvc.FinalizeEligibleReviewsRequest{
		ReviewID:       worker.options.ReviewID,
		WorkflowRunID:  worker.options.WorkflowRunID,
		CompanyID:      worker.options.CompanyID,
		BookType:       worker.options.BookType,
		MaxReviews:     worker.options.MaxReviewsPerRun,
		Force:          worker.options.Force,
		SupersedePrior: worker.options.SupersedePrior,
		DryRun:         worker.options.DryRun,
		InitiatedBy:    worker.options.InitiatedBy,
		CorrelationID:  worker.options.CorrelationID,
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

	finalizedCount := len(result.FinalizedReviewIDs)
	failedCount := len(result.FailedReviewIDs)
	skippedCount := len(result.SkippedReviewIDs)
	counts := workerCounts{
		Processed: finalizedCount + failedCount + skippedCount,
		Succeeded: finalizedCount,
		Failed:    failedCount,
		Skipped:   skippedCount,
	}
	if counts.Processed == 0 {
		counts.Processed = result.Summary.AttemptedCount
	}

	return buildWorkerResult(worker.workerBase, counts, result.PartialFailures, result.HasFailures(), map[string]any{
		"finalized_review_ids":  objectIDHexes(result.FinalizedReviewIDs),
		"failed_review_ids":     objectIDHexes(result.FailedReviewIDs),
		"skipped_review_ids":    objectIDHexes(result.SkippedReviewIDs),
		"superseded_review_ids": objectIDHexes(result.SupersededReviewIDs),
		"review_ref_ids":        reviewRefIDs(result.ReviewRefs),
	})
}
