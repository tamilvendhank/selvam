package review

import (
	"errors"
	"fmt"
	"math"
	"strings"
	"time"

	domaincommon "goserver/internal/domain/common"
	domainreview "goserver/internal/domain/review"
	platformrepo "goserver/internal/platform/repository"
	servicecommon "goserver/internal/service/common"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

func buildMapReviewActionResult(outcome actionMappingOutcome) *MapReviewActionResult {
	summary := buildSingleActionSummary(outcome)
	return &MapReviewActionResult{
		ReviewID:          outcome.ReviewID,
		WorkflowRunID:     outcome.WorkflowRunID,
		ActionType:        outcome.ActionType,
		BucketAfterAction: outcome.BucketAfterAction,
		Constraints:       outcome.Constraints,
		CapitalEligible:   outcome.CapitalEligible,
		PriorityScore:     outcome.PriorityScore,
		Summary:           summary,
	}
}

func buildSingleActionSummary(outcome actionMappingOutcome) servicecommon.ActionMappingSummary {
	mapped := boolToInt(outcome.Mapped)
	skipped := boolToInt(outcome.Skipped)
	success := mapped
	if outcome.DryRun && !outcome.Skipped {
		success = 0
	}
	summary := buildActionSummaryCounts(
		"map_review_action",
		1,
		success,
		skipped,
		0,
		boolToInt(outcome.CapitalEligible),
		len(outcome.Constraints),
		outcome.DryRun,
		false,
		time.Time{},
		time.Time{},
	)
	if outcome.Message != "" {
		summary.Message = outcome.Message
	}
	return summary
}

func buildWorkflowActionSummary(
	attempted int,
	mapped int,
	skipped int,
	failures int,
	capitalEligible int,
	constraintHits int,
	dryRun bool,
	hasMore bool,
	startedAt time.Time,
	completedAt time.Time,
) servicecommon.ActionMappingSummary {
	success := mapped
	if dryRun {
		success = 0
	}
	return buildActionSummaryCounts(
		"map_workflow_actions",
		attempted,
		success,
		skipped,
		failures,
		capitalEligible,
		constraintHits,
		dryRun,
		hasMore,
		startedAt,
		completedAt,
	)
}

func buildActionSummaryCounts(
	operation string,
	attempted int,
	success int,
	skipped int,
	failures int,
	capitalEligible int,
	constraintHits int,
	dryRun bool,
	hasMore bool,
	startedAt time.Time,
	completedAt time.Time,
) servicecommon.ActionMappingSummary {
	outcome := servicecommon.ServiceOutcomeSuccess
	message := fmt.Sprintf("mapped %d review action(s)", success)
	switch {
	case attempted == 0:
		outcome = servicecommon.ServiceOutcomeNoop
		message = "no reviews to map"
	case dryRun:
		outcome = servicecommon.ServiceOutcomeDryRun
		message = fmt.Sprintf("dry run mapped %d review action(s)", attempted-skipped-failures)
	case failures > 0 && success > 0:
		outcome = servicecommon.ServiceOutcomePartial
		message = fmt.Sprintf("mapped %d review action(s) with %d failure(s)", success, failures)
	case failures > 0:
		outcome = servicecommon.ServiceOutcomeFailed
		message = fmt.Sprintf("failed to map %d review action(s)", failures)
	case skipped > 0 && success > 0:
		outcome = servicecommon.ServiceOutcomePartial
		message = fmt.Sprintf("mapped %d review action(s), skipped %d", success, skipped)
	case skipped > 0:
		outcome = servicecommon.ServiceOutcomeSkipped
		message = fmt.Sprintf("skipped %d review(s)", skipped)
	}
	if hasMore {
		message += "; more reviews may be available"
	}
	summary := servicecommon.ActionMappingSummary{
		OperationSummary: servicecommon.OperationSummary{
			Operation:      operation,
			Outcome:        outcome,
			AttemptedCount: attempted,
			SuccessCount:   success,
			SkippedCount:   skipped,
			FailureCount:   failures,
			DryRun:         dryRun,
			Message:        message,
		},
		MappedCount:        success,
		CapitalEligible:    capitalEligible,
		ConstraintHitCount: constraintHits,
	}
	attachTimes(&summary.OperationSummary, startedAt, completedAt)
	return summary
}

func buildAssignBucketResult(outcome bucketAssignmentOutcome) *AssignBucketResult {
	return &AssignBucketResult{
		ReviewID:      outcome.ReviewID,
		CompanyID:     outcome.CompanyID,
		BucketBefore:  outcome.BucketBefore,
		BucketAfter:   outcome.BucketAfter,
		BucketChanged: outcome.BucketChanged,
		Summary:       buildSingleBucketSummary(outcome),
	}
}

func buildSingleBucketSummary(outcome bucketAssignmentOutcome) servicecommon.BucketAssignmentSummary {
	assigned := boolToInt(outcome.Assigned)
	skipped := boolToInt(outcome.Skipped)
	success := assigned
	if outcome.DryRun && !outcome.Skipped {
		success = 0
	}
	summary := buildBucketSummaryCounts(
		"assign_bucket",
		1,
		success,
		boolToInt(outcome.BucketChanged),
		skipped,
		0,
		outcome.DryRun,
		false,
		time.Time{},
		time.Time{},
	)
	if outcome.Message != "" {
		summary.Message = outcome.Message
	}
	return summary
}

func buildWorkflowBucketSummary(
	attempted int,
	assigned int,
	changed int,
	skipped int,
	failures int,
	dryRun bool,
	hasMore bool,
	startedAt time.Time,
	completedAt time.Time,
) servicecommon.BucketAssignmentSummary {
	success := assigned
	if dryRun {
		success = 0
	}
	return buildBucketSummaryCounts(
		"assign_buckets_for_workflow",
		attempted,
		success,
		changed,
		skipped,
		failures,
		dryRun,
		hasMore,
		startedAt,
		completedAt,
	)
}

func buildBucketSummaryCounts(
	operation string,
	attempted int,
	success int,
	changed int,
	skipped int,
	failures int,
	dryRun bool,
	hasMore bool,
	startedAt time.Time,
	completedAt time.Time,
) servicecommon.BucketAssignmentSummary {
	outcome := servicecommon.ServiceOutcomeSuccess
	message := fmt.Sprintf("assigned %d bucket(s)", success)
	switch {
	case attempted == 0:
		outcome = servicecommon.ServiceOutcomeNoop
		message = "no reviews to assign buckets"
	case dryRun:
		outcome = servicecommon.ServiceOutcomeDryRun
		message = fmt.Sprintf("dry run assigned %d bucket(s)", attempted-skipped-failures)
	case failures > 0 && success > 0:
		outcome = servicecommon.ServiceOutcomePartial
		message = fmt.Sprintf("assigned %d bucket(s) with %d failure(s)", success, failures)
	case failures > 0:
		outcome = servicecommon.ServiceOutcomeFailed
		message = fmt.Sprintf("failed to assign %d bucket(s)", failures)
	case skipped > 0 && success > 0:
		outcome = servicecommon.ServiceOutcomePartial
		message = fmt.Sprintf("assigned %d bucket(s), skipped %d", success, skipped)
	case skipped > 0:
		outcome = servicecommon.ServiceOutcomeSkipped
		message = fmt.Sprintf("skipped %d review(s)", skipped)
	}
	if hasMore {
		message += "; more reviews may be available"
	}
	summary := servicecommon.BucketAssignmentSummary{
		OperationSummary: servicecommon.OperationSummary{
			Operation:      operation,
			Outcome:        outcome,
			AttemptedCount: attempted,
			SuccessCount:   success,
			SkippedCount:   skipped,
			FailureCount:   failures,
			DryRun:         dryRun,
			Message:        message,
		},
		AssignedCount: success,
		ChangedCount:  changed,
	}
	attachTimes(&summary.OperationSummary, startedAt, completedAt)
	return summary
}

func actionPartialFailure(
	review *domainreview.CompanyReview,
	workflowRunID primitive.ObjectID,
	err error,
) servicecommon.PartialFailure {
	return reviewPartialFailure(review, workflowRunID, err, "action_mapping_failed")
}

func bucketPartialFailure(
	review *domainreview.CompanyReview,
	workflowRunID primitive.ObjectID,
	err error,
) servicecommon.PartialFailure {
	return reviewPartialFailure(review, workflowRunID, err, "bucket_assignment_failed")
}

func reviewPartialFailure(
	review *domainreview.CompanyReview,
	workflowRunID primitive.ObjectID,
	err error,
	code string,
) servicecommon.PartialFailure {
	reviewID := primitive.ObjectID{}
	companyID := primitive.ObjectID{}
	if review != nil {
		reviewID = review.ID
		companyID = review.CompanyID
	}
	retryClass := servicecommon.RetryClassTransient
	retryable := true
	switch {
	case errors.Is(err, platformrepo.ErrNotFound), isReviewServiceSkip(err), errors.Is(err, servicecommon.ErrInvalidServiceRequest):
		retryClass = servicecommon.RetryClassNone
		retryable = false
	case errors.Is(err, platformrepo.ErrPreconditionFailed), errors.Is(err, platformrepo.ErrConflict), errors.Is(err, platformrepo.ErrAlreadyExists):
		retryClass = servicecommon.RetryClassConflict
	case errors.Is(err, platformrepo.ErrInvalidTransition), errors.Is(err, platformrepo.ErrImmutableState):
		retryClass = servicecommon.RetryClassManualReview
		retryable = false
	}
	return servicecommon.PartialFailure{
		Scope:         servicecommon.FailureScopeReview,
		ID:            reviewID,
		WorkflowRunID: workflowRunID,
		ReviewID:      reviewID,
		CompanyID:     companyID,
		Code:          code,
		Message:       err.Error(),
		Retry: servicecommon.RetryPolicyHint{
			Retryable:  retryable,
			RetryClass: retryClass,
			Reason:     code,
		},
	}
}

func countCapitalEligible(results []MapReviewActionResult) int {
	count := 0
	for _, result := range results {
		if result.CapitalEligible {
			count++
		}
	}
	return count
}

func countActionConstraints(results []MapReviewActionResult) int {
	count := 0
	for _, result := range results {
		count += len(result.Constraints)
	}
	return count
}

func countChangedBuckets(results []AssignBucketResult) int {
	count := 0
	for _, result := range results {
		if result.BucketChanged {
			count++
		}
	}
	return count
}

func attachTimes(summary *servicecommon.OperationSummary, startedAt time.Time, completedAt time.Time) {
	if summary == nil {
		return
	}
	if !startedAt.IsZero() {
		started := startedAt.UTC()
		summary.StartedAt = &started
	}
	if !completedAt.IsZero() {
		completed := completedAt.UTC()
		summary.CompletedAt = &completed
	}
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

func nonBlankStrings(values []string) []string {
	clean := make([]string, 0, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value != "" {
			clean = append(clean, value)
		}
	}
	return clean
}

func limitStrings(values []string, limit int) []string {
	values = nonBlankStrings(values)
	if limit <= 0 || len(values) <= limit {
		return values
	}
	return values[:limit]
}

func normalizeKey(value string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	var builder strings.Builder
	lastUnderscore := false
	for _, r := range value {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') {
			builder.WriteRune(r)
			lastUnderscore = false
			continue
		}
		if !lastUnderscore {
			builder.WriteByte('_')
			lastUnderscore = true
		}
	}
	return strings.Trim(builder.String(), "_")
}

func humanizeSectionName(name domaincommon.SectionName) string {
	value := strings.ReplaceAll(string(name), "_", " ")
	parts := strings.Fields(value)
	for index, part := range parts {
		if part == "" {
			continue
		}
		parts[index] = strings.ToUpper(part[:1]) + part[1:]
	}
	return strings.Join(parts, " ")
}

func clampScore(value float64) float64 {
	if value < 1 {
		return 1
	}
	if value > 10 {
		return 10
	}
	return value
}

func roundToTenth(value float64) float64 {
	return math.Round(value*10) / 10
}

func boolToInt(value bool) int {
	if value {
		return 1
	}
	return 0
}

func applyPositiveFloat(target *float64, value float64) {
	if value > 0 {
		*target = value
	}
}

func applyDefaultFloat(target *float64, fallback float64) {
	if *target <= 0 {
		*target = fallback
	}
}

func applyPositiveInt(target *int, value int) {
	if value > 0 {
		*target = value
	}
}

func applyDefaultInt(target *int, fallback int) {
	if *target <= 0 {
		*target = fallback
	}
}
