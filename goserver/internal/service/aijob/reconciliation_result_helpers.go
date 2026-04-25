package aijob

import (
	"errors"
	"fmt"
	"strings"

	platformrepo "goserver/internal/platform/repository"
	servicecommon "goserver/internal/service/common"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

func buildSingleReconciliationResult(outcome reconcileOneOutcome) *ReconcileBatchJobResult {
	readyValidationIDs := uniqueObjectIDs(outcome.ReadyValidationIDs)
	readyWorkflowRunIDs := uniqueObjectIDs(outcome.ReadyWorkflowRunIDs)
	reconciledJobIDs := []primitive.ObjectID{}
	if !outcome.BatchJobID.IsZero() {
		reconciledJobIDs = append(reconciledJobIDs, outcome.BatchJobID)
	}
	return &ReconcileBatchJobResult{
		BatchJobID:                         outcome.BatchJobID,
		ReconciledJobIDs:                   reconciledJobIDs,
		CompletedItems:                     outcome.CompletedItems,
		FailedItems:                        outcome.FailedItems,
		InvalidItems:                       outcome.InvalidItems,
		ItemsCompleted:                     outcome.ItemsCompleted,
		ItemsFailed:                        outcome.ItemsFailed,
		ItemsInvalid:                       outcome.ItemsInvalid,
		ItemsStillPending:                  outcome.ItemsStillPending,
		ReadyForValidationCount:            len(readyValidationIDs),
		ReadyForContinuationWorkflowRunIDs: readyWorkflowRunIDs,
		PartialFailures:                    outcome.PartialFailures,
		Summary: buildReconciliationSummary(
			"reconcile_batch_job",
			1,
			len(reconciledJobIDs),
			len(outcome.PartialFailures),
			outcome.ItemsCompleted,
			outcome.ItemsFailed,
			outcome.ItemsInvalid,
			outcome.ItemsStillPending,
		),
	}
}

func mergeReconciliationOutcome(result *ReconcilePendingBatchJobsResult, outcome reconcileOneOutcome) {
	if !outcome.BatchJobID.IsZero() {
		result.ReconciledJobIDs = append(result.ReconciledJobIDs, outcome.BatchJobID)
	}
	result.CompletedItems = append(result.CompletedItems, outcome.CompletedItems...)
	result.FailedItems = append(result.FailedItems, outcome.FailedItems...)
	result.InvalidItems = append(result.InvalidItems, outcome.InvalidItems...)
	result.ItemsCompleted += outcome.ItemsCompleted
	result.ItemsFailed += outcome.ItemsFailed
	result.ItemsInvalid += outcome.ItemsInvalid
	result.ItemsStillPending += outcome.ItemsStillPending
	result.ReadyForValidationCount += len(uniqueObjectIDs(outcome.ReadyValidationIDs))
	result.ReadyForContinuationWorkflowRunIDs = uniqueObjectIDs(append(result.ReadyForContinuationWorkflowRunIDs, outcome.ReadyWorkflowRunIDs...))
	result.PartialFailures = append(result.PartialFailures, outcome.PartialFailures...)
}

func buildReconciliationSummary(
	operation string,
	attempted int,
	reconciled int,
	failures int,
	itemsCompleted int,
	itemsFailed int,
	itemsInvalid int,
	itemsStillPending int,
) servicecommon.ReconciliationSummary {
	outcome := servicecommon.ServiceOutcomeSuccess
	message := fmt.Sprintf("reconciled %d batch job(s)", reconciled)
	switch {
	case attempted == 0:
		outcome = servicecommon.ServiceOutcomeNoop
		message = "no batch jobs to reconcile"
	case failures > 0 && reconciled > 0:
		outcome = servicecommon.ServiceOutcomePartial
		message = fmt.Sprintf("reconciled %d batch job(s) with %d failure(s)", reconciled, failures)
	case failures > 0:
		outcome = servicecommon.ServiceOutcomeFailed
		message = fmt.Sprintf("failed to reconcile %d batch job(s)", failures)
	case itemsFailed > 0 || itemsInvalid > 0 || itemsStillPending > 0:
		outcome = servicecommon.ServiceOutcomePartial
		message = fmt.Sprintf(
			"reconciled %d batch job(s); %d completed, %d failed, %d invalid, %d pending item(s)",
			reconciled,
			itemsCompleted,
			itemsFailed,
			itemsInvalid,
			itemsStillPending,
		)
	}

	return servicecommon.ReconciliationSummary{
		OperationSummary: servicecommon.OperationSummary{
			Operation:      operation,
			Outcome:        outcome,
			AttemptedCount: attempted,
			SuccessCount:   reconciled,
			FailureCount:   failures,
			Message:        message,
		},
		ReconciledJobCount: reconciled,
		ItemsCompleted:     itemsCompleted,
		ItemsFailed:        itemsFailed,
		ItemsInvalid:       itemsInvalid,
		ItemsStillPending:  itemsStillPending,
	}
}

func reconciliationPartialFailure(jobID primitive.ObjectID, err error) servicecommon.PartialFailure {
	retryClass := servicecommon.RetryClassTransient
	retryable := true
	if errors.Is(err, platformrepo.ErrNotFound) || isReconciliationSkip(err) {
		retryClass = servicecommon.RetryClassNone
		retryable = false
	}
	if errors.Is(err, platformrepo.ErrPreconditionFailed) || errors.Is(err, platformrepo.ErrConflict) {
		retryClass = servicecommon.RetryClassConflict
		retryable = true
	}
	if errors.Is(err, platformrepo.ErrInvalidTransition) || errors.Is(err, platformrepo.ErrImmutableState) {
		retryClass = servicecommon.RetryClassManualReview
		retryable = false
	}
	return servicecommon.PartialFailure{
		Scope:      servicecommon.FailureScopeJob,
		ID:         jobID,
		BatchJobID: jobID,
		Code:       "batch_reconcile_failed",
		Message:    err.Error(),
		Retry: servicecommon.RetryPolicyHint{
			Retryable:  retryable,
			RetryClass: retryClass,
			Reason:     "batch reconciliation failure",
		},
	}
}

func itemReconciliationPartialFailure(
	jobID primitive.ObjectID,
	itemID primitive.ObjectID,
	err error,
) servicecommon.PartialFailure {
	retryClass := servicecommon.RetryClassTransient
	retryable := true
	if errors.Is(err, platformrepo.ErrPreconditionFailed) || errors.Is(err, platformrepo.ErrConflict) {
		retryClass = servicecommon.RetryClassConflict
	}
	if errors.Is(err, platformrepo.ErrInvalidTransition) || errors.Is(err, platformrepo.ErrImmutableState) {
		retryClass = servicecommon.RetryClassManualReview
		retryable = false
	}
	return servicecommon.PartialFailure{
		Scope:       servicecommon.FailureScopeItem,
		ID:          itemID,
		BatchJobID:  jobID,
		BatchItemID: itemID,
		Code:        "batch_item_reconcile_failed",
		Message:     err.Error(),
		Retry: servicecommon.RetryPolicyHint{
			Retryable:  retryable,
			RetryClass: retryClass,
			Reason:     "batch item reconciliation failure",
		},
	}
}

func providerResultPartialFailure(
	jobID primitive.ObjectID,
	result providerReconciliationItem,
	code string,
	message string,
) servicecommon.PartialFailure {
	return servicecommon.PartialFailure{
		Scope:      servicecommon.FailureScopeItem,
		BatchJobID: jobID,
		ExternalID: providerResultExternalID(result),
		Code:       code,
		Message:    message,
		Retry: servicecommon.RetryPolicyHint{
			Retryable:  false,
			RetryClass: servicecommon.RetryClassManualReview,
			Reason:     "provider result correlation issue",
		},
	}
}

func providerResultExternalID(result providerReconciliationItem) string {
	for _, candidate := range []string{result.ProviderItemHandle, result.CorrelationID} {
		if text := strings.TrimSpace(candidate); text != "" {
			return text
		}
	}
	if result.ResultIndex >= 0 {
		return fmt.Sprintf("result_index:%d", result.ResultIndex)
	}
	return ""
}
