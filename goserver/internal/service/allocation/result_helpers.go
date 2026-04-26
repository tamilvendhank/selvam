package allocation

import (
	"fmt"
	"strings"
	"time"

	servicecommon "goserver/internal/service/common"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

func buildCandidateSummary(
	operation string,
	attempted int,
	candidateCount int,
	skippedCount int,
	failureCount int,
	dryRun bool,
	startedAt time.Time,
	completedAt time.Time,
) servicecommon.AllocationSummary {
	outcome := servicecommon.ServiceOutcomeSuccess
	switch {
	case dryRun:
		outcome = servicecommon.ServiceOutcomeDryRun
	case failureCount > 0 && candidateCount > 0:
		outcome = servicecommon.ServiceOutcomePartial
	case failureCount > 0:
		outcome = servicecommon.ServiceOutcomeFailed
	case candidateCount == 0:
		outcome = servicecommon.ServiceOutcomeNoop
	}
	return servicecommon.AllocationSummary{
		OperationSummary: servicecommon.OperationSummary{
			Operation:      operation,
			Outcome:        outcome,
			AttemptedCount: attempted,
			SuccessCount:   candidateCount,
			SkippedCount:   skippedCount,
			FailureCount:   failureCount,
			DryRun:         dryRun,
			StartedAt:      timePtr(startedAt),
			CompletedAt:    timePtr(completedAt),
			Message:        candidateSummaryMessage(candidateCount, skippedCount, failureCount),
		},
		CandidateCount: candidateCount,
		BlockedCount:   skippedCount,
	}
}

func buildCapitalAllocationSummary(
	operation string,
	candidateCount int,
	allocatedCount int,
	blockedCount int,
	failureCount int,
	allocatedCashTotal float64,
	unallocatedCashTotal float64,
	dryRun bool,
	startedAt time.Time,
	completedAt time.Time,
) servicecommon.AllocationSummary {
	outcome := servicecommon.ServiceOutcomeSuccess
	switch {
	case dryRun:
		outcome = servicecommon.ServiceOutcomeDryRun
	case failureCount > 0 && allocatedCount > 0:
		outcome = servicecommon.ServiceOutcomePartial
	case failureCount > 0:
		outcome = servicecommon.ServiceOutcomeFailed
	case allocatedCount == 0:
		outcome = servicecommon.ServiceOutcomeNoop
	}
	return servicecommon.AllocationSummary{
		OperationSummary: servicecommon.OperationSummary{
			Operation:      operation,
			Outcome:        outcome,
			AttemptedCount: candidateCount,
			SuccessCount:   allocatedCount,
			SkippedCount:   blockedCount,
			FailureCount:   failureCount,
			DryRun:         dryRun,
			StartedAt:      timePtr(startedAt),
			CompletedAt:    timePtr(completedAt),
			Message:        allocationSummaryMessage(allocatedCount, blockedCount, allocatedCashTotal, unallocatedCashTotal),
		},
		CandidateCount:       candidateCount,
		AllocatedCount:       allocatedCount,
		BlockedCount:         blockedCount,
		AllocatedCashTotal:   allocatedCashTotal,
		UnallocatedCashTotal: unallocatedCashTotal,
	}
}

func candidatePartialFailure(
	workflowRunID primitive.ObjectID,
	reviewID primitive.ObjectID,
	companyID primitive.ObjectID,
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
		Scope:         servicecommon.FailureScopeCandidate,
		WorkflowRunID: workflowRunID,
		ReviewID:      reviewID,
		CompanyID:     companyID,
		Code:          code,
		Message:       message,
		Cause:         cause,
		Retry: servicecommon.RetryPolicyHint{
			Retryable:  false,
			RetryClass: servicecommon.RetryClassDependency,
			Reason:     code,
		},
	}
}

func candidateSummaryMessage(candidateCount int, skippedCount int, failureCount int) string {
	switch {
	case candidateCount == 0 && failureCount == 0:
		return "no eligible capital candidates"
	case failureCount > 0:
		return fmt.Sprintf("built %d capital candidates with %d skipped and %d partial failures", candidateCount, skippedCount, failureCount)
	default:
		return fmt.Sprintf("built %d ranked capital candidates; skipped %d", candidateCount, skippedCount)
	}
}

func allocationSummaryMessage(allocatedCount int, blockedCount int, allocatedCashTotal float64, unallocatedCashTotal float64) string {
	if allocatedCount == 0 {
		return fmt.Sprintf("no capital allocated; blocked %d candidates; unallocated %.2f", blockedCount, unallocatedCashTotal)
	}
	return fmt.Sprintf("allocated %.2f across %d candidates; blocked %d; unallocated %.2f", allocatedCashTotal, allocatedCount, blockedCount, unallocatedCashTotal)
}

func appendSummaryMessage(current string, addition string) string {
	if strings.TrimSpace(current) == "" {
		return addition
	}
	return current + "; " + addition
}

func timePtr(value time.Time) *time.Time {
	if value.IsZero() {
		return nil
	}
	copy := value.UTC()
	return &copy
}
