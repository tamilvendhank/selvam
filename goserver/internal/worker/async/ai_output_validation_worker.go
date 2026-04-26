package async

import (
	"context"

	validationsvc "goserver/internal/service/validation"
	"goserver/internal/worker/framework"
)

type AIOutputValidationWorker struct {
	workerBase
	service validationsvc.AIOutputValidationService
}

func NewAIOutputValidationWorker(
	service validationsvc.AIOutputValidationService,
	options AsyncWorkerOptions,
	clock framework.Clock,
) *AIOutputValidationWorker {
	return &AIOutputValidationWorker{
		workerBase: newWorkerBase(AIOutputValidationWorkerName, options, clock),
		service:    service,
	}
}

func (worker *AIOutputValidationWorker) RunOnce(ctx context.Context) framework.WorkerRunResult {
	if worker == nil {
		return dependencyFailure(AIOutputValidationWorkerName, "ai output validation worker")
	}
	if err := worker.validate(ctx); err != nil {
		return validationFailure(worker.Name(), err)
	}
	if worker.service == nil {
		return dependencyFailure(worker.Name(), "ai output validation service")
	}

	result, err := worker.service.ValidatePendingAIOutputs(ctx, validationsvc.ValidatePendingAIOutputsRequest{
		BatchItemID:   worker.options.BatchItemID,
		WorkflowRunID: worker.options.WorkflowRunID,
		BookType:      worker.options.BookType,
		ItemType:      worker.options.ItemType,
		MaxItems:      worker.options.MaxItemsPerRun,
		StrictMode:    worker.options.StrictMode,
		Revalidate:    worker.options.Revalidate || worker.options.Force,
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

	validCount := len(result.ValidItemIDs)
	invalidCount := len(result.InvalidItemIDs)
	skippedCount := len(result.SkippedItemIDs)
	counts := workerCounts{
		Processed: validCount + invalidCount + skippedCount,
		Succeeded: validCount,
		Failed:    invalidCount,
		Skipped:   skippedCount,
	}
	if counts.Processed == 0 {
		counts.Processed = result.Summary.AttemptedCount
	}

	return buildWorkerResult(worker.workerBase, counts, result.PartialFailures, result.HasFailures(), map[string]any{
		"valid_item_ids":                     objectIDHexes(result.ValidItemIDs),
		"invalid_item_ids":                   objectIDHexes(result.InvalidItemIDs),
		"skipped_item_ids":                   objectIDHexes(result.SkippedItemIDs),
		"ready_for_materialization_item_ids": objectIDHexes(result.ValidItemIDs),
		"validation_issue_count":             len(result.ValidationIssues),
		"field_error_item_count":             len(result.FieldErrors),
	})
}
