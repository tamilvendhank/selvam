package async

import (
	"context"

	materializationsvc "goserver/internal/service/materialization"
	"goserver/internal/worker/framework"
)

type ReviewMaterializationWorker struct {
	workerBase
	service materializationsvc.ReviewMaterializationService
}

func NewReviewMaterializationWorker(
	service materializationsvc.ReviewMaterializationService,
	options AsyncWorkerOptions,
	clock framework.Clock,
) *ReviewMaterializationWorker {
	return &ReviewMaterializationWorker{
		workerBase: newWorkerBase(ReviewMaterializationWorkerName, options, clock),
		service:    service,
	}
}

func (worker *ReviewMaterializationWorker) RunOnce(ctx context.Context) framework.WorkerRunResult {
	if worker == nil {
		return dependencyFailure(ReviewMaterializationWorkerName, "review materialization worker")
	}
	if err := worker.validate(ctx); err != nil {
		return validationFailure(worker.Name(), err)
	}
	if worker.service == nil {
		return dependencyFailure(worker.Name(), "review materialization service")
	}

	result, err := worker.service.MaterializePendingReviews(ctx, materializationsvc.MaterializePendingReviewsRequest{
		ReviewID:      worker.options.ReviewID,
		BatchItemID:   worker.options.BatchItemID,
		WorkflowRunID: worker.options.WorkflowRunID,
		BookType:      worker.options.BookType,
		MaxItems:      effectiveMaxItems(worker.options),
		Force:         worker.options.Force,
		DryRun:        worker.options.DryRun,
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

	materializedCount := len(result.MaterializedReviewIDs)
	failedCount := len(result.FailedReviewIDs)
	skippedCount := len(result.SkippedReviewIDs)
	counts := workerCounts{
		Processed: materializedCount + failedCount + skippedCount,
		Succeeded: materializedCount,
		Failed:    failedCount,
		Skipped:   skippedCount,
	}
	if counts.Processed == 0 {
		counts.Processed = result.Summary.AttemptedCount
	}

	return buildWorkerResult(worker.workerBase, counts, result.PartialFailures, result.HasFailures(), map[string]any{
		"materialized_review_ids":           objectIDHexes(result.MaterializedReviewIDs),
		"failed_review_ids":                 objectIDHexes(result.FailedReviewIDs),
		"skipped_review_ids":                objectIDHexes(result.SkippedReviewIDs),
		"review_ref_ids":                    reviewRefIDs(result.ReviewRefs),
		"ready_for_finalization_review_ids": objectIDHexes(result.MaterializedReviewIDs),
	})
}
