package continuation

import (
	domainaijob "goserver/internal/domain/aijob"
	domaincommon "goserver/internal/domain/common"
	domainreview "goserver/internal/domain/review"
	domainworkflow "goserver/internal/domain/workflow"
	servicecommon "goserver/internal/service/common"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

func workflowBlocker(
	workflowRunID primitive.ObjectID,
	code string,
	message string,
	severity servicecommon.ValidationIssueSeverity,
	waitingOnStep domaincommon.WorkflowStepName,
) servicecommon.BlockingCondition {
	return servicecommon.BlockingCondition{
		Scope:         servicecommon.FailureScopeWorkflow,
		ID:            workflowRunID,
		WorkflowRunID: workflowRunID,
		Code:          code,
		Severity:      severity,
		Message:       message,
		WaitingOnStep: waitingOnStep,
	}
}

func stepBlocker(
	step *domainworkflow.WorkflowStepRun,
	code string,
	message string,
	severity servicecommon.ValidationIssueSeverity,
) servicecommon.BlockingCondition {
	condition := servicecommon.BlockingCondition{
		Scope:         servicecommon.FailureScopeWorkflow,
		Code:          code,
		Severity:      severity,
		Message:       message,
		WaitingOnStep: "",
	}
	if step != nil {
		condition.ID = step.ID
		condition.WorkflowRunID = step.WorkflowRunID
		condition.WaitingOnStep = step.StepName
	}
	return condition
}

func batchJobBlocker(
	job *domainaijob.AIBatchJob,
	code string,
	message string,
	severity servicecommon.ValidationIssueSeverity,
	waitingOnStep domaincommon.WorkflowStepName,
) servicecommon.BlockingCondition {
	condition := servicecommon.BlockingCondition{
		Scope:         servicecommon.FailureScopeJob,
		Code:          code,
		Severity:      severity,
		Message:       message,
		WaitingOnStep: waitingOnStep,
	}
	if job != nil {
		condition.ID = job.ID
		condition.WorkflowRunID = job.WorkflowRunID
		condition.BatchJobID = job.ID
		condition.Retry = retryHintForBatchJob(job)
	}
	return condition
}

func batchItemBlocker(
	item *domainaijob.AIBatchItem,
	code string,
	message string,
	severity servicecommon.ValidationIssueSeverity,
	waitingOnStep domaincommon.WorkflowStepName,
) servicecommon.BlockingCondition {
	condition := servicecommon.BlockingCondition{
		Scope:         servicecommon.FailureScopeItem,
		Code:          code,
		Severity:      severity,
		Message:       message,
		WaitingOnStep: waitingOnStep,
	}
	if item != nil {
		condition.ID = item.ID
		condition.WorkflowRunID = item.WorkflowRunID
		condition.BatchJobID = item.AIBatchJobID
		condition.BatchItemID = item.ID
		condition.ReviewID = item.TargetReviewID
		condition.Retry = retryHintForBatchItem(item)
	}
	return condition
}

func reviewBlocker(
	review *domainreview.CompanyReview,
	code string,
	message string,
	severity servicecommon.ValidationIssueSeverity,
	waitingOnStep domaincommon.WorkflowStepName,
) servicecommon.BlockingCondition {
	condition := servicecommon.BlockingCondition{
		Scope:         servicecommon.FailureScopeReview,
		Code:          code,
		Severity:      severity,
		Message:       message,
		WaitingOnStep: waitingOnStep,
	}
	if review != nil {
		condition.ID = review.ID
		condition.WorkflowRunID = review.WorkflowRunID
		condition.ReviewID = review.ID
	}
	return condition
}

func retryHintForBatchJob(job *domainaijob.AIBatchJob) servicecommon.RetryPolicyHint {
	if job == nil {
		return servicecommon.RetryPolicyHint{}
	}
	if job.CanRetry() {
		return servicecommon.RetryPolicyHint{
			Retryable:    true,
			RetryClass:   servicecommon.RetryClassProvider,
			AttemptsUsed: job.RetryCount,
			MaxAttempts:  job.MaxRetryCount,
			Reason:       "batch job is retryable",
		}
	}
	if job.Status == domaincommon.AIBatchJobStatusFailed || job.Status == domaincommon.AIBatchJobStatusTimedOut {
		return servicecommon.RetryPolicyHint{
			Retryable:    false,
			RetryClass:   servicecommon.RetryClassRetryExhausted,
			AttemptsUsed: job.RetryCount,
			MaxAttempts:  job.MaxRetryCount,
			Reason:       "batch job is terminal and has no retry attempts remaining",
		}
	}
	return servicecommon.RetryPolicyHint{}
}

func retryHintForBatchItem(item *domainaijob.AIBatchItem) servicecommon.RetryPolicyHint {
	if item == nil {
		return servicecommon.RetryPolicyHint{}
	}
	if item.CanRetry() {
		return servicecommon.RetryPolicyHint{
			Retryable:  true,
			RetryClass: servicecommon.RetryClassValidation,
			Reason:     "batch item can be retried",
		}
	}
	if item.Status == domaincommon.AIBatchItemStatusFailed || item.Status == domaincommon.AIBatchItemStatusInvalidOutput {
		return servicecommon.RetryPolicyHint{
			Retryable:  false,
			RetryClass: servicecommon.RetryClassManualReview,
			Reason:     "batch item is terminal and requires manual review or partial-success policy",
		}
	}
	return servicecommon.RetryPolicyHint{}
}

func appendOneBlocker(blockers []servicecommon.BlockingCondition, condition servicecommon.BlockingCondition) []servicecommon.BlockingCondition {
	if condition.Message == "" {
		return blockers
	}
	return append(blockers, condition)
}
