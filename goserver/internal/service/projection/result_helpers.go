package projection

import (
	"fmt"
	"time"

	domaincommon "goserver/internal/domain/common"
	servicecommon "goserver/internal/service/common"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

func buildProjectionSummary(
	attemptedReviews int,
	updatedRefs []ProjectionUpdateRef,
	skippedRefs []ProjectionUpdateRef,
	failures []servicecommon.PartialFailure,
	dryRun bool,
	startedAt time.Time,
	completedAt time.Time,
) servicecommon.ProjectionUpdateSummary {
	updatedPositionCount := countUpdatedTarget(updatedRefs, ProjectionTargetPosition)
	updatedCompanyCount := countUpdatedTarget(updatedRefs, ProjectionTargetCompanyState)
	updatedReviewCount := countUpdatedTarget(updatedRefs, ProjectionTargetReview)
	updatedTotal := len(updatedRefs)

	outcome := servicecommon.ServiceOutcomeSuccess
	switch {
	case dryRun:
		outcome = servicecommon.ServiceOutcomeDryRun
	case len(failures) > 0 && updatedTotal > 0:
		outcome = servicecommon.ServiceOutcomePartial
	case len(failures) > 0:
		outcome = servicecommon.ServiceOutcomeFailed
	case updatedTotal == 0:
		outcome = servicecommon.ServiceOutcomeNoop
	}

	return servicecommon.ProjectionUpdateSummary{
		OperationSummary: servicecommon.OperationSummary{
			Operation:      "update_projections",
			Outcome:        outcome,
			AttemptedCount: attemptedReviews,
			SuccessCount:   updatedTotal,
			SkippedCount:   len(skippedRefs),
			FailureCount:   len(failures),
			DryRun:         dryRun,
			StartedAt:      projectionTimePtr(startedAt),
			CompletedAt:    projectionTimePtr(completedAt),
			Message:        projectionSummaryMessage(updatedTotal, len(skippedRefs), len(failures)),
		},
		UpdatedCompanyCount:  updatedCompanyCount,
		UpdatedPositionCount: updatedPositionCount,
		UpdatedReviewCount:   updatedReviewCount,
	}
}

func projectionPartialFailure(
	workflowRunID primitive.ObjectID,
	reviewID primitive.ObjectID,
	companyID primitive.ObjectID,
	bookType domaincommon.BookType,
	code string,
	err error,
) servicecommon.PartialFailure {
	message := code
	cause := ""
	if err != nil {
		message = err.Error()
		cause = err.Error()
	}
	return servicecommon.PartialFailure{
		Scope:         servicecommon.FailureScopeProjection,
		WorkflowRunID: workflowRunID,
		ReviewID:      reviewID,
		CompanyID:     companyID,
		Code:          code,
		Message:       message,
		Cause:         cause,
		Retry: servicecommon.RetryPolicyHint{
			Retryable:  false,
			RetryClass: servicecommon.RetryClassDependency,
			Reason:     fmt.Sprintf("%s book=%s", code, bookType),
		},
	}
}

func countUpdatedTarget(refs []ProjectionUpdateRef, target ProjectionTarget) int {
	count := 0
	for _, ref := range refs {
		if ref.Target == target && ref.Updated {
			count++
		}
	}
	return count
}

func projectionSummaryMessage(updatedCount int, skippedCount int, failureCount int) string {
	switch {
	case failureCount > 0:
		return fmt.Sprintf("updated %d projections with %d skipped and %d partial failures", updatedCount, skippedCount, failureCount)
	case updatedCount == 0:
		return fmt.Sprintf("no projections updated; skipped %d", skippedCount)
	default:
		return fmt.Sprintf("updated %d projections; skipped %d", updatedCount, skippedCount)
	}
}

func appendProjectionSummaryMessage(current string, addition string) string {
	if current == "" {
		return addition
	}
	return current + "; " + addition
}

func projectionTimePtr(value time.Time) *time.Time {
	if value.IsZero() {
		return nil
	}
	copy := value.UTC()
	return &copy
}
