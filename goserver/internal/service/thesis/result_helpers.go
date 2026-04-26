package thesis

import (
	"errors"
	"fmt"
	"strings"
	"time"

	domaincommon "goserver/internal/domain/common"
	domainreview "goserver/internal/domain/review"
	platformrepo "goserver/internal/platform/repository"
	servicecommon "goserver/internal/service/common"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

func buildSingleThesisResult(outcome thesisOneOutcome) *EvaluateThesisResult {
	updatedIDs := []primitive.ObjectID(nil)
	if !outcome.ThesisID.IsZero() && (outcome.Created || outcome.Updated) {
		updatedIDs = []primitive.ObjectID{outcome.ThesisID}
	}
	return &EvaluateThesisResult{
		ReviewID:          outcome.ReviewID,
		CompanyID:         outcome.CompanyID,
		ThesisCreated:     outcome.Created,
		ThesisUpdated:     outcome.Updated,
		ThesisBroken:      outcome.Broken,
		ThesisUnderReview: outcome.UnderReview,
		UpdatedThesisIDs:  updatedIDs,
		Summary: buildSingleThesisSummary(
			outcome,
			time.Time{},
			time.Time{},
		),
	}
}

func mergeThesisOutcome(result *EvaluateThesisForWorkflowResult, outcome thesisOneOutcome) {
	if outcome.Created {
		result.Summary.CreatedCount++
	}
	if outcome.Updated {
		result.Summary.UpdatedCount++
	}
	if outcome.Broken {
		result.Summary.BrokenCount++
	}
	if outcome.UnderReview {
		result.Summary.UnderReviewCount++
	}
	if outcome.Skipped {
		result.Summary.SkippedCount++
	}
	if !outcome.ThesisID.IsZero() && (outcome.Created || outcome.Updated) {
		result.UpdatedThesisIDs = append(result.UpdatedThesisIDs, outcome.ThesisID)
	}
}

func buildSingleThesisSummary(
	outcome thesisOneOutcome,
	startedAt time.Time,
	completedAt time.Time,
) servicecommon.ThesisSummary {
	created := boolToInt(outcome.Created)
	updated := boolToInt(outcome.Updated)
	broken := boolToInt(outcome.Broken)
	underReview := boolToInt(outcome.UnderReview)
	skipped := boolToInt(outcome.Skipped)
	success := created + updated
	if outcome.DryRun && !outcome.Skipped {
		success = 0
	}

	summary := buildThesisSummaryCounts(
		"evaluate_thesis",
		1,
		success,
		skipped,
		0,
		created,
		updated,
		broken,
		underReview,
		outcome.DryRun,
		false,
	)
	if !startedAt.IsZero() {
		started := startedAt.UTC()
		summary.StartedAt = &started
	}
	if !completedAt.IsZero() {
		completed := completedAt.UTC()
		summary.CompletedAt = &completed
	}
	if outcome.Summary != "" {
		summary.Message = outcome.Summary
	}
	return summary
}

func buildWorkflowThesisSummary(
	attempted int,
	created int,
	updated int,
	broken int,
	underReview int,
	skipped int,
	failures int,
	dryRun bool,
	hasMore bool,
	startedAt time.Time,
	completedAt time.Time,
) servicecommon.ThesisSummary {
	success := created + updated
	summary := buildThesisSummaryCounts(
		"evaluate_thesis_for_workflow",
		attempted,
		success,
		skipped,
		failures,
		created,
		updated,
		broken,
		underReview,
		dryRun,
		hasMore,
	)
	if !startedAt.IsZero() {
		started := startedAt.UTC()
		summary.StartedAt = &started
	}
	if !completedAt.IsZero() {
		completed := completedAt.UTC()
		summary.CompletedAt = &completed
	}
	return summary
}

func buildThesisSummaryCounts(
	operation string,
	attempted int,
	success int,
	skipped int,
	failures int,
	created int,
	updated int,
	broken int,
	underReview int,
	dryRun bool,
	hasMore bool,
) servicecommon.ThesisSummary {
	outcome := servicecommon.ServiceOutcomeSuccess
	message := fmt.Sprintf("updated %d thesis/theses", success)
	switch {
	case attempted == 0:
		outcome = servicecommon.ServiceOutcomeNoop
		message = "no reviews to evaluate for thesis updates"
	case dryRun:
		outcome = servicecommon.ServiceOutcomeDryRun
		message = fmt.Sprintf("dry run evaluated %d review(s) for thesis updates", attempted)
	case failures > 0 && success > 0:
		outcome = servicecommon.ServiceOutcomePartial
		message = fmt.Sprintf("updated %d thesis/theses with %d failure(s)", success, failures)
	case failures > 0:
		outcome = servicecommon.ServiceOutcomeFailed
		message = fmt.Sprintf("failed to evaluate %d thesis review(s)", failures)
	case skipped > 0 && success > 0:
		outcome = servicecommon.ServiceOutcomePartial
		message = fmt.Sprintf("updated %d thesis/theses, skipped %d review(s)", success, skipped)
	case skipped > 0:
		outcome = servicecommon.ServiceOutcomeSkipped
		message = fmt.Sprintf("skipped %d review(s)", skipped)
	}
	if hasMore {
		message += "; more reviews may be available"
	}
	return servicecommon.ThesisSummary{
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
		CreatedCount:     created,
		UpdatedCount:     updated,
		BrokenCount:      broken,
		UnderReviewCount: underReview,
	}
}

func thesisPartialFailure(
	review *domainreview.CompanyReview,
	workflowRunID primitive.ObjectID,
	err error,
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
	case errors.Is(err, platformrepo.ErrNotFound), isThesisSkip(err), errors.Is(err, servicecommon.ErrInvalidServiceRequest):
		retryClass = servicecommon.RetryClassNone
		retryable = false
	case errors.Is(err, platformrepo.ErrPreconditionFailed), errors.Is(err, platformrepo.ErrConflict), errors.Is(err, platformrepo.ErrAlreadyExists):
		retryClass = servicecommon.RetryClassConflict
	case errors.Is(err, platformrepo.ErrInvalidTransition), errors.Is(err, platformrepo.ErrImmutableState):
		retryClass = servicecommon.RetryClassManualReview
		retryable = false
	}
	return servicecommon.PartialFailure{
		Scope:         servicecommon.FailureScopeThesis,
		ID:            reviewID,
		WorkflowRunID: workflowRunID,
		ReviewID:      reviewID,
		CompanyID:     companyID,
		Code:          "thesis_evaluation_failed",
		Message:       err.Error(),
		Retry: servicecommon.RetryPolicyHint{
			Retryable:  retryable,
			RetryClass: retryClass,
			Reason:     "thesis evaluation failure",
		},
	}
}

func thesisSkipSummary(err error) string {
	message := strings.TrimSpace(err.Error())
	message = strings.TrimPrefix(message, errThesisSkipped.Error()+": ")
	if message == "" {
		return "review skipped for thesis evaluation"
	}
	return message
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

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}

func firstString(values []string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
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

func uniqueStrings(values []string) []string {
	seen := make(map[string]struct{}, len(values))
	unique := make([]string, 0, len(values))
	for _, value := range nonBlankStrings(values) {
		key := strings.ToLower(value)
		if _, exists := seen[key]; exists {
			continue
		}
		seen[key] = struct{}{}
		unique = append(unique, value)
	}
	return unique
}

func mergeStringSlices(existing []string, next []string, limit int) []string {
	values := make([]string, 0, len(existing)+len(next))
	values = append(values, existing...)
	values = append(values, next...)
	return limitStrings(uniqueStrings(values), limit)
}

func limitStrings(values []string, limit int) []string {
	values = nonBlankStrings(values)
	if limit <= 0 || len(values) <= limit {
		return values
	}
	return values[:limit]
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

func clampUnit(value float64) float64 {
	if value < 0 {
		return 0
	}
	if value > 1 {
		return 1
	}
	return value
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
	return mathRound(value*10) / 10
}

func mathRound(value float64) float64 {
	if value < 0 {
		return float64(int(value - 0.5))
	}
	return float64(int(value + 0.5))
}

func boolToInt(value bool) int {
	if value {
		return 1
	}
	return 0
}
