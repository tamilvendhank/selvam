package async

import (
	"fmt"

	servicecommon "goserver/internal/service/common"
	"goserver/internal/worker/framework"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

type workerCounts struct {
	Processed int
	Succeeded int
	Failed    int
	Skipped   int
}

func buildWorkerResult(
	base workerBase,
	counts workerCounts,
	failures []servicecommon.PartialFailure,
	hasServiceFailures bool,
	metadata map[string]any,
) framework.WorkerRunResult {
	result := base.base.NewResult()
	result.ProcessedCount = counts.Processed
	result.SucceededCount = counts.Succeeded
	result.FailedCount = counts.Failed
	result.SkippedCount = counts.Skipped
	result.PartialFailures = mapPartialFailures(failures)
	result.Metadata = mergeMetadata(base.options.metadata(), metadata)
	if hasServiceFailures {
		result.Success = false
		if result.ErrorSummary == "" {
			result.ErrorSummary = "service reported failures"
		}
	} else {
		result.Success = true
	}
	if len(result.PartialFailures) > 0 && result.ErrorSummary == "" {
		result.ErrorSummary = fmt.Sprintf("%d partial failures", len(result.PartialFailures))
	}
	return base.base.FinishResult(result)
}

func mapPartialFailures(failures []servicecommon.PartialFailure) []framework.WorkerFailure {
	if len(failures) == 0 {
		return nil
	}
	mapped := make([]framework.WorkerFailure, 0, len(failures))
	for _, failure := range failures {
		mapped = append(mapped, framework.WorkerFailure{
			Scope:     string(failure.Scope),
			EntityID:  partialFailureEntityID(failure),
			Code:      failure.Code,
			Message:   failure.Message,
			Retryable: failure.Retry.IsRetryable(),
		})
	}
	return mapped
}

func partialFailureEntityID(failure servicecommon.PartialFailure) string {
	for _, id := range []primitive.ObjectID{
		failure.ID,
		failure.BatchJobID,
		failure.BatchItemID,
		failure.ReviewID,
		failure.CompanyID,
		failure.WorkflowRunID,
	} {
		if !id.IsZero() {
			return id.Hex()
		}
	}
	return failure.ExternalID
}

func objectIDHexes(ids []primitive.ObjectID) []string {
	if len(ids) == 0 {
		return nil
	}
	values := make([]string, 0, len(ids))
	for _, id := range ids {
		if !id.IsZero() {
			values = append(values, id.Hex())
		}
	}
	return values
}

func batchJobRefIDs(refs []servicecommon.BatchJobRef) []string {
	if len(refs) == 0 {
		return nil
	}
	ids := make([]string, 0, len(refs))
	for _, ref := range refs {
		if !ref.ID.IsZero() {
			ids = append(ids, ref.ID.Hex())
		}
	}
	return ids
}

func reviewRefIDs(refs []servicecommon.ReviewRef) []string {
	if len(refs) == 0 {
		return nil
	}
	ids := make([]string, 0, len(refs))
	for _, ref := range refs {
		if !ref.ID.IsZero() {
			ids = append(ids, ref.ID.Hex())
		}
	}
	return ids
}
