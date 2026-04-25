package validation

import (
	"context"
	"errors"
	"fmt"

	domainaijob "goserver/internal/domain/aijob"
	domaincommon "goserver/internal/domain/common"
	platformrepo "goserver/internal/platform/repository"
	servicecommon "goserver/internal/service/common"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

func (service *aiOutputValidationService) applyValidationResult(
	ctx context.Context,
	item *domainaijob.AIBatchItem,
	report outputValidationReport,
	options validationRequestOptions,
) (*domainaijob.AIBatchItem, error) {
	status := domaincommon.ValidationStatusValid
	validationErrors := []string(nil)
	if !report.IsValid() {
		status = domaincommon.ValidationStatusInvalid
		validationErrors = report.ErrorMessages()
		if len(validationErrors) == 0 {
			validationErrors = []string{"AI output failed validation"}
		}
	}
	if item.Status == domaincommon.AIBatchItemStatusInvalidOutput {
		status = domaincommon.ValidationStatusInvalid
		if len(validationErrors) == 0 {
			validationErrors = []string{"item is marked invalid_output"}
		}
	}

	return service.batchItems.SaveValidationResult(ctx, item.ID, platformrepo.AIBatchItemValidationPatch{
		ValidationStatus: status,
		ValidationErrors: validationErrors,
		ExpectedCurrentStatuses: []domaincommon.AIBatchItemStatus{
			item.Status,
		},
		Mutation: platformrepo.MutationMetadata{
			OccurredAt: service.now().UTC(),
			Actor:      options.InitiatedBy,
			Reason:     "AI output validation",
		},
	})
}

func buildSingleValidationResult(outcome validateOneOutcome) *ValidateBatchItemOutputResult {
	result := &ValidateBatchItemOutputResult{
		BatchItemID:      outcome.BatchItemID,
		ReviewID:         outcome.ReviewID,
		ValidationStatus: outcome.ValidationStatusAfter,
		ValidationIssues: outcome.Issues,
		FieldErrors:      outcome.FieldErrors,
		Summary: buildValidationSummary(
			"validate_batch_item_output",
			1,
			boolToInt(outcome.Valid),
			boolToInt(outcome.Invalid),
			0,
			len(outcome.Issues),
		),
	}
	if outcome.Valid {
		result.ValidItemIDs = []primitive.ObjectID{outcome.BatchItemID}
	}
	if outcome.Invalid {
		result.InvalidItemIDs = []primitive.ObjectID{outcome.BatchItemID}
	}
	return result
}

func mergeValidationOutcome(result *ValidatePendingAIOutputsResult, outcome validateOneOutcome) {
	if outcome.Valid {
		result.ValidItemIDs = append(result.ValidItemIDs, outcome.BatchItemID)
	}
	if outcome.Invalid {
		result.InvalidItemIDs = append(result.InvalidItemIDs, outcome.BatchItemID)
	}
	result.ValidationIssues = append(result.ValidationIssues, outcome.Issues...)
	if len(outcome.FieldErrors) > 0 {
		if result.FieldErrors == nil {
			result.FieldErrors = map[primitive.ObjectID][]servicecommon.FieldError{}
		}
		result.FieldErrors[outcome.BatchItemID] = append(result.FieldErrors[outcome.BatchItemID], outcome.FieldErrors...)
	}
}

func buildValidationSummary(
	operation string,
	attempted int,
	valid int,
	invalid int,
	failures int,
	issueCount int,
) servicecommon.ValidationSummary {
	outcome := servicecommon.ServiceOutcomeSuccess
	message := fmt.Sprintf("validated %d AI output item(s)", valid+invalid)
	switch {
	case attempted == 0:
		outcome = servicecommon.ServiceOutcomeNoop
		message = "no AI output items to validate"
	case failures > 0 && valid+invalid > 0:
		outcome = servicecommon.ServiceOutcomePartial
		message = fmt.Sprintf("validated %d AI output item(s) with %d operational failure(s)", valid+invalid, failures)
	case failures > 0:
		outcome = servicecommon.ServiceOutcomeFailed
		message = fmt.Sprintf("failed to validate %d AI output item(s)", failures)
	case invalid > 0:
		outcome = servicecommon.ServiceOutcomePartial
		message = fmt.Sprintf("validated %d AI output item(s); %d invalid", valid+invalid, invalid)
	}
	return servicecommon.ValidationSummary{
		OperationSummary: servicecommon.OperationSummary{
			Operation:      operation,
			Outcome:        outcome,
			AttemptedCount: attempted,
			SuccessCount:   valid,
			FailureCount:   failures,
			Message:        message,
		},
		ValidCount:   valid,
		InvalidCount: invalid,
		IssueCount:   issueCount,
	}
}

func validationPartialFailure(itemID primitive.ObjectID, err error) servicecommon.PartialFailure {
	retryClass := servicecommon.RetryClassTransient
	retryable := true
	if errors.Is(err, platformrepo.ErrNotFound) || isValidationSkip(err) {
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
		Scope:       servicecommon.FailureScopeItem,
		ID:          itemID,
		BatchItemID: itemID,
		Code:        "ai_output_validation_failed",
		Message:     err.Error(),
		Retry: servicecommon.RetryPolicyHint{
			Retryable:  retryable,
			RetryClass: retryClass,
			Reason:     "AI output validation failure",
		},
	}
}

func boolToInt(value bool) int {
	if value {
		return 1
	}
	return 0
}
