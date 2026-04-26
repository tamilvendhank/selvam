package continuation

import (
	"fmt"
	"time"

	domainaijob "goserver/internal/domain/aijob"
	domaincommon "goserver/internal/domain/common"
	domainreview "goserver/internal/domain/review"
	domainworkflow "goserver/internal/domain/workflow"
	servicecommon "goserver/internal/service/common"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

func summarizeStepRuns(steps []*domainworkflow.WorkflowStepRun) WorkflowContinuationStepCounts {
	counts := WorkflowContinuationStepCounts{Total: len(steps)}
	for _, step := range steps {
		if step == nil {
			continue
		}
		switch step.Status {
		case domaincommon.WorkflowStepStatusPending:
			counts.Pending++
		case domaincommon.WorkflowStepStatusRunning:
			counts.Running++
		case domaincommon.WorkflowStepStatusWaitingExternal:
			counts.Waiting++
		case domaincommon.WorkflowStepStatusCompleted:
			counts.Completed++
		case domaincommon.WorkflowStepStatusFailed:
			counts.Failed++
		case domaincommon.WorkflowStepStatusSkipped:
			counts.Skipped++
		}
	}
	return counts
}

func summarizeBatchJobs(jobs []*domainaijob.AIBatchJob) WorkflowContinuationBatchJobCounts {
	counts := WorkflowContinuationBatchJobCounts{Total: len(jobs)}
	for _, job := range jobs {
		if job == nil {
			continue
		}
		switch job.Status {
		case domaincommon.AIBatchJobStatusCreated:
			counts.Created++
		case domaincommon.AIBatchJobStatusSubmitted:
			counts.Submitted++
		case domaincommon.AIBatchJobStatusRunning:
			counts.Running++
		case domaincommon.AIBatchJobStatusPartiallyCompleted:
			counts.PartiallyCompleted++
		case domaincommon.AIBatchJobStatusCompleted:
			counts.Completed++
		case domaincommon.AIBatchJobStatusFailed:
			counts.Failed++
		case domaincommon.AIBatchJobStatusCancelled:
			counts.Cancelled++
		case domaincommon.AIBatchJobStatusTimedOut:
			counts.TimedOut++
		}
	}
	return counts
}

func summarizeBatchItems(items []*domainaijob.AIBatchItem) WorkflowContinuationBatchItemCounts {
	counts := WorkflowContinuationBatchItemCounts{Total: len(items)}
	for _, item := range items {
		if item == nil {
			continue
		}
		switch item.Status {
		case domaincommon.AIBatchItemStatusPending:
			counts.Pending++
			counts.Unreconciled++
		case domaincommon.AIBatchItemStatusSubmitted:
			counts.Submitted++
			counts.Unreconciled++
		case domaincommon.AIBatchItemStatusProcessing:
			counts.Processing++
			counts.Unreconciled++
		case domaincommon.AIBatchItemStatusCompleted:
			counts.Completed++
			counts.TerminalSuccessful++
		case domaincommon.AIBatchItemStatusFailed:
			counts.Failed++
			counts.TerminalFailures++
		case domaincommon.AIBatchItemStatusInvalidOutput:
			counts.InvalidOutput++
			counts.TerminalFailures++
		case domaincommon.AIBatchItemStatusSkipped:
			counts.Skipped++
		}

		switch item.ValidationStatus {
		case domaincommon.ValidationStatusValid:
			counts.Valid++
		case domaincommon.ValidationStatusInvalid:
			counts.Invalid++
		case domaincommon.ValidationStatusNotValidated:
			counts.NotValidated++
			if item.Status == domaincommon.AIBatchItemStatusCompleted {
				counts.PendingValidation++
			}
		}

		if isMaterializableBatchItem(item) {
			counts.Materializable++
		}
	}
	return counts
}

func summarizeReviews(reviews []*domainreview.CompanyReview) WorkflowContinuationReviewCounts {
	counts := WorkflowContinuationReviewCounts{Total: len(reviews)}
	for _, review := range reviews {
		if review == nil {
			continue
		}
		switch review.ReviewLifecycleState {
		case domaincommon.ReviewLifecycleStatePendingInput:
			counts.PendingInput++
			counts.Pending++
		case domaincommon.ReviewLifecycleStatePendingAI:
			counts.PendingAI++
			counts.Pending++
		case domaincommon.ReviewLifecycleStateAICompletedUnvalidated:
			counts.AICompletedUnvalidated++
			counts.MaterializationIncomplete++
			counts.Pending++
		case domaincommon.ReviewLifecycleStateValidationFailed:
			counts.ValidationFailed++
			counts.MaterializationIncomplete++
			counts.Pending++
		case domaincommon.ReviewLifecycleStateAIValidated:
			counts.AIValidated++
			counts.Materialized++
			counts.FinalizationIncomplete++
			counts.Pending++
			if review.CanFinalize() {
				counts.Finalizable++
			}
		case domaincommon.ReviewLifecycleStateFinalized:
			counts.Finalized++
			counts.Materialized++
		case domaincommon.ReviewLifecycleStateSuperseded:
			counts.Superseded++
			counts.Materialized++
		}
	}
	return counts
}

func buildContinuationCounts(snapshot continuationContext) WorkflowContinuationCounts {
	return WorkflowContinuationCounts{
		WorkflowSteps: summarizeStepRuns(snapshot.steps),
		BatchJobs:     summarizeBatchJobs(snapshot.batchJobs),
		BatchItems:    summarizeBatchItems(snapshot.batchItems),
		Reviews:       summarizeReviews(snapshot.reviews),
	}
}

func buildSingleContinuationSummary(
	result *EvaluateWorkflowContinuationResult,
	startedAt time.Time,
	completedAt time.Time,
) servicecommon.ContinuationSummary {
	outcome := servicecommon.ServiceOutcomeBlocked
	message := "workflow continuation is blocked"
	successCount := 0
	blockedCount := 1
	if result.ReadyToContinueNow() {
		outcome = servicecommon.ServiceOutcomeSuccess
		message = "workflow is ready to continue"
		successCount = 1
		blockedCount = 0
	} else if isTerminalReadiness(result.Readiness) {
		outcome = servicecommon.ServiceOutcomeNoop
		message = "workflow is terminal"
		blockedCount = 0
	}

	return servicecommon.ContinuationSummary{
		OperationSummary: servicecommon.OperationSummary{
			Operation:      "evaluate_workflow_continuation",
			Outcome:        outcome,
			AttemptedCount: 1,
			SuccessCount:   successCount,
			FailureCount:   failureCountForReadiness(result.Readiness),
			StartedAt:      &startedAt,
			CompletedAt:    &completedAt,
			Message:        message,
		},
		ReadyCount:   successCount,
		BlockedCount: blockedCount,
	}
}

func buildBulkContinuationSummary(
	attempted int,
	ready int,
	blocked int,
	terminal int,
	failures int,
	hasMore bool,
	startedAt time.Time,
	completedAt time.Time,
) servicecommon.ContinuationSummary {
	outcome := servicecommon.ServiceOutcomeSuccess
	switch {
	case failures > 0 && ready > 0:
		outcome = servicecommon.ServiceOutcomePartial
	case failures > 0:
		outcome = servicecommon.ServiceOutcomeFailed
	case ready == 0 && blocked+terminal > 0:
		outcome = servicecommon.ServiceOutcomeNoop
	case ready == 0:
		outcome = servicecommon.ServiceOutcomeNoop
	}

	message := fmt.Sprintf("evaluated %d workflow continuation candidate(s); %d ready, %d blocked, %d terminal", attempted, ready, blocked, terminal)
	if hasMore {
		message = fmt.Sprintf("%s; more candidates may be available", message)
	}

	return servicecommon.ContinuationSummary{
		OperationSummary: servicecommon.OperationSummary{
			Operation:      "evaluate_many_workflow_continuations",
			Outcome:        outcome,
			AttemptedCount: attempted,
			SuccessCount:   ready,
			SkippedCount:   terminal,
			FailureCount:   failures,
			StartedAt:      &startedAt,
			CompletedAt:    &completedAt,
			Message:        message,
		},
		ReadyCount:   ready,
		BlockedCount: blocked,
	}
}

func failureCountForReadiness(readiness WorkflowContinuationReadiness) int {
	switch readiness {
	case WorkflowContinuationReadinessBlockedByFailures,
		WorkflowContinuationReadinessFailedTerminal,
		WorkflowContinuationReadinessInvalidState:
		return 1
	default:
		return 0
	}
}

func isTerminalReadiness(readiness WorkflowContinuationReadiness) bool {
	return readiness == WorkflowContinuationReadinessAlreadyCompleted ||
		readiness == WorkflowContinuationReadinessFailedTerminal
}

func appendUniqueObjectID(ids []primitive.ObjectID, id primitive.ObjectID) []primitive.ObjectID {
	if id.IsZero() {
		return ids
	}
	for _, existing := range ids {
		if existing == id {
			return ids
		}
	}
	return append(ids, id)
}

func uniqueObjectIDs(ids []primitive.ObjectID) []primitive.ObjectID {
	unique := make([]primitive.ObjectID, 0, len(ids))
	for _, id := range ids {
		unique = appendUniqueObjectID(unique, id)
	}
	return unique
}

func isMaterializableBatchItem(item *domainaijob.AIBatchItem) bool {
	return item != nil &&
		item.Status == domaincommon.AIBatchItemStatusCompleted &&
		item.ValidationStatus == domaincommon.ValidationStatusValid &&
		!item.TargetReviewID.IsZero()
}

func buildSingleContinuationExecutionSummary(
	result *ContinueWorkflowResult,
	startedAt time.Time,
	completedAt time.Time,
) servicecommon.ContinuationSummary {
	outcome := servicecommon.ServiceOutcomeSuccess
	message := "workflow continuation executed"
	successCount := 0
	blockedCount := 0
	continuedCount := 0
	completedCount := 0

	switch {
	case result.DryRun:
		outcome = servicecommon.ServiceOutcomeDryRun
		message = "workflow continuation dry run completed"
	case result.Blocked:
		outcome = servicecommon.ServiceOutcomeBlocked
		message = "workflow continuation is blocked"
		blockedCount = 1
	case result.Failed:
		outcome = servicecommon.ServiceOutcomeFailed
		message = "workflow continuation failed"
	case result.Completed:
		outcome = servicecommon.ServiceOutcomeSuccess
		message = "workflow continuation completed workflow"
	case !result.Continued && len(result.SkippedSteps) == 0:
		outcome = servicecommon.ServiceOutcomeNoop
		message = "no continuation steps executed"
	}

	if result.Continued || result.DryRun {
		successCount = 1
		continuedCount = 1
	}
	if result.Completed {
		completedCount = 1
	}
	if len(result.PartialFailures) > 0 && outcome == servicecommon.ServiceOutcomeSuccess {
		outcome = servicecommon.ServiceOutcomePartial
	}

	return servicecommon.ContinuationSummary{
		OperationSummary: servicecommon.OperationSummary{
			Operation:      "continue_workflow",
			Outcome:        outcome,
			AttemptedCount: 1,
			SuccessCount:   successCount,
			SkippedCount:   len(result.SkippedSteps),
			FailureCount:   len(result.FailedSteps),
			DryRun:         result.DryRun,
			StartedAt:      &startedAt,
			CompletedAt:    &completedAt,
			Message:        message,
		},
		ReadyCount:     successCount,
		BlockedCount:   blockedCount,
		ContinuedCount: continuedCount,
		CompletedCount: completedCount,
	}
}

func buildBulkContinuationExecutionSummary(
	attempted int,
	continued int,
	completed int,
	blocked int,
	failed int,
	partialFailures int,
	dryRun bool,
	hasMore bool,
	startedAt time.Time,
	completedAt time.Time,
) servicecommon.ContinuationSummary {
	outcome := servicecommon.ServiceOutcomeSuccess
	switch {
	case dryRun:
		outcome = servicecommon.ServiceOutcomeDryRun
	case failed > 0 && continued > 0:
		outcome = servicecommon.ServiceOutcomePartial
	case failed > 0:
		outcome = servicecommon.ServiceOutcomeFailed
	case continued == 0 && blocked > 0:
		outcome = servicecommon.ServiceOutcomeBlocked
	case continued == 0:
		outcome = servicecommon.ServiceOutcomeNoop
	case partialFailures > 0:
		outcome = servicecommon.ServiceOutcomePartial
	}

	message := fmt.Sprintf("continued %d workflow(s), completed %d, blocked %d, failed %d", continued, completed, blocked, failed)
	if dryRun {
		message = fmt.Sprintf("dry run: %s", message)
	}
	if hasMore {
		message = fmt.Sprintf("%s; more eligible workflows may be available", message)
	}

	return servicecommon.ContinuationSummary{
		OperationSummary: servicecommon.OperationSummary{
			Operation:      "continue_eligible_workflows",
			Outcome:        outcome,
			AttemptedCount: attempted,
			SuccessCount:   continued,
			SkippedCount:   blocked,
			FailureCount:   failed + partialFailures,
			DryRun:         dryRun,
			StartedAt:      &startedAt,
			CompletedAt:    &completedAt,
			Message:        message,
		},
		ReadyCount:     continued,
		BlockedCount:   blocked,
		ContinuedCount: continued,
		CompletedCount: completed,
	}
}

func appendDecisionIfMissing(
	decisions []EvaluateWorkflowContinuationResult,
	result *ContinueWorkflowResult,
) []EvaluateWorkflowContinuationResult {
	if result == nil || result.WorkflowRunID.IsZero() {
		return decisions
	}
	for _, decision := range decisions {
		if decision.WorkflowRunID == result.WorkflowRunID {
			return decisions
		}
	}
	return append(decisions, EvaluateWorkflowContinuationResult{
		WorkflowRunID:      result.WorkflowRunID,
		BookType:           result.BookType,
		CurrentStatus:      result.CurrentStatus,
		Readiness:          result.Readiness,
		ReadyToContinue:    !result.Blocked && !result.Failed,
		NextSuggestedStep:  result.NextSuggestedStep,
		Blockers:           result.Blockers,
		ContinuationReason: continuationReasonForReadiness(result.Readiness),
	})
}
