package finalization

import (
	"errors"
	"fmt"

	platformrepo "goserver/internal/platform/repository"
	servicecommon "goserver/internal/service/common"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

func buildSingleFinalizationResult(outcome finalizeOneOutcome) *FinalizeReviewResult {
	result := &FinalizeReviewResult{
		ReviewID:            outcome.ReviewID,
		SupersededReviewIDs: uniqueObjectIDs(outcome.SupersededReviewIDs),
		PartialFailures:     outcome.PartialFailures,
		Summary: buildFinalizationSummary(
			"finalize_review",
			1,
			boolToInt(outcome.Finalized),
			boolToInt(outcome.Skipped),
			len(outcome.PartialFailures),
			len(uniqueObjectIDs(outcome.SupersededReviewIDs)),
			outcome.DryRun,
		),
	}
	if outcome.Finalized {
		result.FinalizedReviewIDs = []primitive.ObjectID{outcome.ReviewID}
	}
	if outcome.ReviewRef.ID == outcome.ReviewID && !outcome.ReviewID.IsZero() {
		result.ReviewRefs = []servicecommon.ReviewRef{outcome.ReviewRef}
	}
	return result
}

func mergeFinalizationOutcome(result *FinalizeEligibleReviewsResult, outcome finalizeOneOutcome) {
	if outcome.Finalized {
		result.FinalizedReviewIDs = append(result.FinalizedReviewIDs, outcome.ReviewID)
	}
	if outcome.Skipped {
		result.SkippedReviewIDs = append(result.SkippedReviewIDs, outcome.ReviewID)
	}
	if outcome.ReviewRef.ID == outcome.ReviewID && !outcome.ReviewID.IsZero() {
		result.ReviewRefs = append(result.ReviewRefs, outcome.ReviewRef)
	}
	result.SupersededReviewIDs = append(result.SupersededReviewIDs, outcome.SupersededReviewIDs...)
	result.PartialFailures = append(result.PartialFailures, outcome.PartialFailures...)
}

func buildFinalizationSummary(
	operation string,
	attempted int,
	finalized int,
	skipped int,
	failures int,
	superseded int,
	dryRun bool,
) servicecommon.FinalizationSummary {
	outcome := servicecommon.ServiceOutcomeSuccess
	message := fmt.Sprintf("finalized %d review(s)", finalized)
	switch {
	case attempted == 0:
		outcome = servicecommon.ServiceOutcomeNoop
		message = "no reviews to finalize"
	case dryRun:
		outcome = servicecommon.ServiceOutcomeDryRun
		message = fmt.Sprintf("dry run checked %d review(s) for finalization", attempted)
	case failures > 0 && finalized > 0:
		outcome = servicecommon.ServiceOutcomePartial
		message = fmt.Sprintf("finalized %d review(s) with %d failure(s)", finalized, failures)
	case failures > 0:
		outcome = servicecommon.ServiceOutcomeFailed
		message = fmt.Sprintf("failed to finalize %d review(s)", failures)
	case skipped > 0 && finalized > 0:
		outcome = servicecommon.ServiceOutcomePartial
		message = fmt.Sprintf("finalized %d review(s), skipped %d", finalized, skipped)
	case skipped > 0:
		outcome = servicecommon.ServiceOutcomeSkipped
		message = fmt.Sprintf("skipped %d review(s)", skipped)
	}
	return servicecommon.FinalizationSummary{
		OperationSummary: servicecommon.OperationSummary{
			Operation:      operation,
			Outcome:        outcome,
			AttemptedCount: attempted,
			SuccessCount:   finalized,
			SkippedCount:   skipped,
			FailureCount:   failures,
			DryRun:         dryRun,
			Message:        message,
		},
		FinalizedCount:   finalized,
		SupersededCount:  superseded,
		PreconditionMiss: skipped,
	}
}

func finalizationPartialFailure(reviewID primitive.ObjectID, err error) servicecommon.PartialFailure {
	retryClass := servicecommon.RetryClassTransient
	retryable := true
	if errors.Is(err, platformrepo.ErrNotFound) || isFinalizationSkip(err) {
		retryClass = servicecommon.RetryClassNone
		retryable = false
	}
	if errors.Is(err, platformrepo.ErrPreconditionFailed) || errors.Is(err, platformrepo.ErrConflict) {
		retryClass = servicecommon.RetryClassConflict
	}
	if errors.Is(err, platformrepo.ErrInvalidTransition) || errors.Is(err, platformrepo.ErrImmutableState) {
		retryClass = servicecommon.RetryClassManualReview
		retryable = false
	}
	return servicecommon.PartialFailure{
		Scope:    servicecommon.FailureScopeReview,
		ID:       reviewID,
		ReviewID: reviewID,
		Code:     "review_finalization_failed",
		Message:  err.Error(),
		Retry: servicecommon.RetryPolicyHint{
			Retryable:  retryable,
			RetryClass: retryClass,
			Reason:     "review finalization failure",
		},
	}
}

func boolToInt(value bool) int {
	if value {
		return 1
	}
	return 0
}
