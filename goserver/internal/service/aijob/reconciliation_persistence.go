package aijob

import (
	"context"
	"fmt"
	"strings"
	"time"

	domainaijob "goserver/internal/domain/aijob"
	domaincommon "goserver/internal/domain/common"
	platformrepo "goserver/internal/platform/repository"
	servicecommon "goserver/internal/service/common"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

func (service *batchReconciliationService) applyItemReconciliation(
	ctx context.Context,
	item *domainaijob.AIBatchItem,
	result providerReconciliationItem,
	options reconciliationRequestOptions,
	completedAt time.Time,
) (itemReconciliationOutcome, error) {
	if item == nil {
		return itemReconciliationOutcome{}, fmt.Errorf("batch item is required")
	}
	if service.batchItems == nil {
		return itemReconciliationOutcome{}, fmt.Errorf("batch item repository is required")
	}
	if completedAt.IsZero() {
		completedAt = service.now().UTC()
	}
	completedAt = completedAt.UTC()

	if isItemAlreadyReconciled(item) {
		if !options.IncludeCompletedItems && !options.Force {
			return itemReconciliationOutcome{}, nil
		}
		return currentItemReconciliationOutcome(item, result), nil
	}

	switch result.Status {
	case domaincommon.AIBatchItemStatusCompleted:
		if resultPayloadEmpty(result.OutputPayload) {
			summary := "provider returned completed item without output payload"
			result.ErrorSummary = summary
			return service.persistInvalidItem(ctx, item, result, options, completedAt, []string{summary})
		}
		return service.persistCompletedItem(ctx, item, result, options, completedAt)
	case domaincommon.AIBatchItemStatusFailed:
		return service.persistFailedItem(ctx, item, result, options, completedAt)
	case domaincommon.AIBatchItemStatusInvalidOutput:
		summary := providerItemErrorSummary(result, "provider returned invalid item output")
		return service.persistInvalidItem(ctx, item, result, options, completedAt, []string{summary})
	case domaincommon.AIBatchItemStatusSkipped:
		return service.persistSkippedItem(ctx, item, result, options, completedAt)
	default:
		return itemReconciliationOutcome{StillPending: true}, nil
	}
}

func (service *batchReconciliationService) persistCompletedItem(
	ctx context.Context,
	item *domainaijob.AIBatchItem,
	result providerReconciliationItem,
	options reconciliationRequestOptions,
	completedAt time.Time,
) (itemReconciliationOutcome, error) {
	ready, err := service.ensureItemReadyForTerminalTransition(ctx, item, options, completedAt, "prepare item for completed reconciliation")
	if err != nil {
		return itemReconciliationOutcome{}, err
	}
	if ready.IsTerminal() && ready.Status != domaincommon.AIBatchItemStatusCompleted {
		return currentItemReconciliationOutcome(ready, result), nil
	}

	clearErrorSummary := ""
	saved, err := service.batchItems.SaveResultPayload(ctx, ready.ID, platformrepo.AIBatchItemResultPatch{
		ResultPayload: cloneStringAnyMap(result.OutputPayload),
		ErrorSummary:  &clearErrorSummary,
		ExpectedCurrentStatuses: []domaincommon.AIBatchItemStatus{
			ready.Status,
		},
		Mutation: platformrepo.MutationMetadata{
			OccurredAt: completedAt,
			Actor:      options.InitiatedBy,
			Reason:     "batch item result reconciliation",
		},
	})
	if err != nil {
		return itemReconciliationOutcome{}, err
	}

	completed := saved
	if saved.Status != domaincommon.AIBatchItemStatusCompleted {
		completed, err = service.batchItems.MarkCompleted(ctx, saved.ID, platformrepo.AIBatchItemCompletionPatch{
			CompletedAt: completedAt,
			ExpectedCurrentStatuses: []domaincommon.AIBatchItemStatus{
				saved.Status,
			},
			Mutation: platformrepo.MutationMetadata{
				OccurredAt: completedAt,
				Actor:      options.InitiatedBy,
				Reason:     "provider returned completed item result",
			},
		})
		if err != nil {
			return itemReconciliationOutcome{}, err
		}
	}

	ref := reconciledItemRef(completed, result)
	return itemReconciliationOutcome{
		Completed:          &ref,
		ReadyForValidation: completed.ValidationStatus == domaincommon.ValidationStatusNotValidated,
	}, nil
}

func (service *batchReconciliationService) persistFailedItem(
	ctx context.Context,
	item *domainaijob.AIBatchItem,
	result providerReconciliationItem,
	options reconciliationRequestOptions,
	failedAt time.Time,
) (itemReconciliationOutcome, error) {
	ready, err := service.ensureItemReadyForTerminalTransition(ctx, item, options, failedAt, "prepare item for failed reconciliation")
	if err != nil {
		return itemReconciliationOutcome{}, err
	}
	if ready.IsTerminal() {
		return currentItemReconciliationOutcome(ready, result), nil
	}
	failed, err := service.batchItems.MarkFailed(ctx, ready.ID, platformrepo.AIBatchItemFailurePatch{
		FailedAt:     failedAt,
		ErrorSummary: providerItemErrorSummary(result, "provider returned item failure"),
		ExpectedCurrentStatuses: []domaincommon.AIBatchItemStatus{
			ready.Status,
		},
		Mutation: platformrepo.MutationMetadata{
			OccurredAt: failedAt,
			Actor:      options.InitiatedBy,
			Reason:     "provider returned failed item result",
		},
	})
	if err != nil {
		return itemReconciliationOutcome{}, err
	}
	ref := reconciledItemRef(failed, result)
	return itemReconciliationOutcome{Failed: &ref}, nil
}

func (service *batchReconciliationService) persistInvalidItem(
	ctx context.Context,
	item *domainaijob.AIBatchItem,
	result providerReconciliationItem,
	options reconciliationRequestOptions,
	invalidAt time.Time,
	validationErrors []string,
) (itemReconciliationOutcome, error) {
	ready, err := service.ensureItemReadyForTerminalTransition(ctx, item, options, invalidAt, "prepare item for invalid-output reconciliation")
	if err != nil {
		return itemReconciliationOutcome{}, err
	}
	if ready.IsTerminal() {
		return currentItemReconciliationOutcome(ready, result), nil
	}
	summary := providerItemErrorSummary(result, "provider returned unusable item output")
	if len(validationErrors) == 0 {
		validationErrors = []string{summary}
	}
	invalid, err := service.batchItems.MarkInvalidOutput(ctx, ready.ID, platformrepo.AIBatchItemInvalidOutputPatch{
		InvalidAt:        invalidAt,
		ErrorSummary:     summary,
		ValidationErrors: validationErrors,
		ExpectedCurrentStatuses: []domaincommon.AIBatchItemStatus{
			ready.Status,
		},
		Mutation: platformrepo.MutationMetadata{
			OccurredAt: invalidAt,
			Actor:      options.InitiatedBy,
			Reason:     "provider returned invalid item result",
		},
	})
	if err != nil {
		return itemReconciliationOutcome{}, err
	}
	ref := reconciledItemRef(invalid, result)
	return itemReconciliationOutcome{Invalid: &ref}, nil
}

func (service *batchReconciliationService) persistSkippedItem(
	ctx context.Context,
	item *domainaijob.AIBatchItem,
	result providerReconciliationItem,
	options reconciliationRequestOptions,
	skippedAt time.Time,
) (itemReconciliationOutcome, error) {
	ready, err := service.ensureItemReadyForTerminalTransition(ctx, item, options, skippedAt, "prepare item for skipped reconciliation")
	if err != nil {
		return itemReconciliationOutcome{}, err
	}
	if ready.IsTerminal() {
		return currentItemReconciliationOutcome(ready, result), nil
	}
	skipped, err := service.batchItems.MarkSkipped(ctx, ready.ID, platformrepo.AIBatchItemSkipPatch{
		SkippedAt: skippedAt,
		Reason:    providerItemErrorSummary(result, "provider skipped item result"),
		ExpectedCurrentStatuses: []domaincommon.AIBatchItemStatus{
			ready.Status,
		},
		Mutation: platformrepo.MutationMetadata{
			OccurredAt: skippedAt,
			Actor:      options.InitiatedBy,
			Reason:     "provider returned skipped item result",
		},
	})
	if err != nil {
		return itemReconciliationOutcome{}, err
	}
	ref := reconciledItemRef(skipped, result)
	return itemReconciliationOutcome{Failed: &ref}, nil
}

func (service *batchReconciliationService) ensureItemReadyForTerminalTransition(
	ctx context.Context,
	item *domainaijob.AIBatchItem,
	options reconciliationRequestOptions,
	at time.Time,
	reason string,
) (*domainaijob.AIBatchItem, error) {
	if item.Status != domaincommon.AIBatchItemStatusPending {
		return item, nil
	}
	return service.batchItems.UpdateStatus(ctx, item.ID, platformrepo.AIBatchItemStatusPatch{
		NextStatus: domaincommon.AIBatchItemStatusSubmitted,
		ExpectedCurrentStatuses: []domaincommon.AIBatchItemStatus{
			domaincommon.AIBatchItemStatusPending,
		},
		Mutation: platformrepo.MutationMetadata{
			OccurredAt: at,
			Actor:      options.InitiatedBy,
			Reason:     reason,
		},
	})
}

func (service *batchReconciliationService) applyJobReconciliationSummary(
	ctx context.Context,
	job *domainaijob.AIBatchJob,
	results providerReconciliationResults,
	options reconciliationRequestOptions,
) error {
	if job == nil || service.batchJobs == nil {
		return nil
	}
	if job.Status == results.Status {
		return nil
	}
	if !job.CanTransitionTo(results.Status) {
		return nil
	}

	switch results.Status {
	case domaincommon.AIBatchJobStatusCompleted:
		_, err := service.batchJobs.MarkCompleted(ctx, job.ID, platformrepo.AIBatchJobCompletionPatch{
			CompletedAt: results.CompletedAt,
			ExpectedCurrentStatuses: []domaincommon.AIBatchJobStatus{
				job.Status,
			},
			Mutation: platformrepo.MutationMetadata{
				OccurredAt: results.CompletedAt,
				Actor:      options.InitiatedBy,
				Reason:     "provider results indicate batch completion",
			},
		})
		return err
	case domaincommon.AIBatchJobStatusFailed:
		_, err := service.batchJobs.MarkFailed(ctx, job.ID, platformrepo.AIBatchJobFailurePatch{
			FailedAt:     results.CompletedAt,
			ErrorSummary: providerResultsErrorSummary(results, "provider results indicate batch failure"),
			ExpectedCurrentStatuses: []domaincommon.AIBatchJobStatus{
				job.Status,
			},
			Mutation: platformrepo.MutationMetadata{
				OccurredAt: results.CompletedAt,
				Actor:      options.InitiatedBy,
				Reason:     "provider results indicate batch failure",
			},
		})
		return err
	case domaincommon.AIBatchJobStatusTimedOut:
		_, err := service.batchJobs.MarkTimedOut(ctx, job.ID, platformrepo.AIBatchJobFailurePatch{
			FailedAt:     results.CompletedAt,
			ErrorSummary: providerResultsErrorSummary(results, "provider results indicate batch timeout"),
			ExpectedCurrentStatuses: []domaincommon.AIBatchJobStatus{
				job.Status,
			},
			Mutation: platformrepo.MutationMetadata{
				OccurredAt: results.CompletedAt,
				Actor:      options.InitiatedBy,
				Reason:     "provider results indicate batch timeout",
			},
		})
		return err
	case domaincommon.AIBatchJobStatusCancelled,
		domaincommon.AIBatchJobStatusPartiallyCompleted,
		domaincommon.AIBatchJobStatusRunning:
		_, err := service.batchJobs.UpdateStatus(ctx, job.ID, platformrepo.AIBatchJobStatusPatch{
			NextStatus: results.Status,
			ExpectedCurrentStatuses: []domaincommon.AIBatchJobStatus{
				job.Status,
			},
			Mutation: platformrepo.MutationMetadata{
				OccurredAt: results.CompletedAt,
				Actor:      options.InitiatedBy,
				Reason:     "provider results indicate batch status update",
			},
		})
		return err
	default:
		return nil
	}
}

func currentItemReconciliationOutcome(
	item *domainaijob.AIBatchItem,
	result providerReconciliationItem,
) itemReconciliationOutcome {
	ref := reconciledItemRef(item, result)
	switch item.Status {
	case domaincommon.AIBatchItemStatusCompleted:
		return itemReconciliationOutcome{
			Completed:          &ref,
			ReadyForValidation: item.ValidationStatus == domaincommon.ValidationStatusNotValidated,
		}
	case domaincommon.AIBatchItemStatusFailed, domaincommon.AIBatchItemStatusSkipped:
		return itemReconciliationOutcome{Failed: &ref}
	case domaincommon.AIBatchItemStatusInvalidOutput:
		return itemReconciliationOutcome{Invalid: &ref}
	default:
		return itemReconciliationOutcome{StillPending: true}
	}
}

func reconciledItemRef(
	item *domainaijob.AIBatchItem,
	result providerReconciliationItem,
) ReconciledBatchItemRef {
	ref := ReconciledBatchItemRef{
		BatchItemRef: servicecommon.BatchItemRef{
			ID:               item.ID,
			BatchJobID:       item.AIBatchJobID,
			WorkflowRunID:    item.WorkflowRunID,
			CompanyID:        item.CompanyID,
			ReviewID:         item.TargetReviewID,
			BookType:         item.BookType,
			ItemType:         item.ItemType,
			Status:           item.Status,
			ValidationStatus: item.ValidationStatus,
			Symbol:           item.Symbol,
		},
		ProviderItemHandle: result.ProviderItemHandle,
		ErrorSummary:       strings.TrimSpace(result.ErrorSummary),
		OutputPayload:      cloneStringAnyMap(result.OutputPayload),
	}
	if ref.ErrorSummary == "" {
		ref.ErrorSummary = strings.TrimSpace(item.ErrorSummary)
	}
	if len(ref.OutputPayload) == 0 {
		ref.OutputPayload = cloneStringAnyMap(item.ResultPayload)
	}
	return ref
}

func providerItemErrorSummary(result providerReconciliationItem, fallback string) string {
	if text := strings.TrimSpace(result.ErrorSummary); text != "" {
		return text
	}
	if text := strings.TrimSpace(result.ProviderItemHandle); text != "" {
		return fmt.Sprintf("%s: %s", fallback, text)
	}
	if text := strings.TrimSpace(result.CorrelationID); text != "" {
		return fmt.Sprintf("%s: %s", fallback, text)
	}
	return fallback
}

func providerResultsErrorSummary(results providerReconciliationResults, fallback string) string {
	for _, key := range []string{"error", "message", "errorSummary", "statusReason"} {
		if value, ok := results.RawPayload[key]; ok {
			if text := strings.TrimSpace(fmt.Sprint(value)); text != "" {
				return text
			}
		}
	}
	return fallback
}

func resultPayloadEmpty(payload map[string]any) bool {
	return len(payload) == 0
}

func uniqueObjectIDs(ids []primitive.ObjectID) []primitive.ObjectID {
	seen := make(map[primitive.ObjectID]struct{}, len(ids))
	unique := make([]primitive.ObjectID, 0, len(ids))
	for _, id := range ids {
		if id.IsZero() {
			continue
		}
		if _, exists := seen[id]; exists {
			continue
		}
		seen[id] = struct{}{}
		unique = append(unique, id)
	}
	return unique
}
