package common

type WorkflowExecutionStatus string

const (
	WorkflowExecutionStatusCreated            WorkflowExecutionStatus = "created"
	WorkflowExecutionStatusRunning            WorkflowExecutionStatus = "running"
	WorkflowExecutionStatusWaitingExternal    WorkflowExecutionStatus = "waiting_external"
	WorkflowExecutionStatusPartiallyCompleted WorkflowExecutionStatus = "partially_completed"
	WorkflowExecutionStatusCompleted          WorkflowExecutionStatus = "completed"
	WorkflowExecutionStatusFailed             WorkflowExecutionStatus = "failed"
	WorkflowExecutionStatusCancelled          WorkflowExecutionStatus = "cancelled"
)

func (status WorkflowExecutionStatus) IsTerminal() bool {
	switch status {
	case WorkflowExecutionStatusCompleted, WorkflowExecutionStatusFailed, WorkflowExecutionStatusCancelled:
		return true
	default:
		return false
	}
}

func (status WorkflowExecutionStatus) RequiresExternalWait() bool {
	return status == WorkflowExecutionStatusWaitingExternal
}

func (status WorkflowExecutionStatus) CanResume() bool {
	switch status {
	case WorkflowExecutionStatusCreated, WorkflowExecutionStatusRunning, WorkflowExecutionStatusPartiallyCompleted:
		return true
	default:
		return false
	}
}

func (status WorkflowExecutionStatus) CanReconcile() bool {
	switch status {
	case WorkflowExecutionStatusRunning, WorkflowExecutionStatusWaitingExternal, WorkflowExecutionStatusPartiallyCompleted:
		return true
	default:
		return false
	}
}

type StepExecutionStatus string

const (
	StepExecutionStatusPending         StepExecutionStatus = "pending"
	StepExecutionStatusRunning         StepExecutionStatus = "running"
	StepExecutionStatusWaitingAsync    StepExecutionStatus = "waiting_async"
	StepExecutionStatusWaitingExternal StepExecutionStatus = "waiting_external"
	StepExecutionStatusCompleted       StepExecutionStatus = "completed"
	StepExecutionStatusFailed          StepExecutionStatus = "failed"
	StepExecutionStatusSkipped         StepExecutionStatus = "skipped"
)

func (status StepExecutionStatus) IsTerminal() bool {
	switch status {
	case StepExecutionStatusCompleted, StepExecutionStatusFailed, StepExecutionStatusSkipped:
		return true
	default:
		return false
	}
}

func (status StepExecutionStatus) RequiresExternalWait() bool {
	return status == StepExecutionStatusWaitingAsync || status == StepExecutionStatusWaitingExternal
}

type StepOutcome string

const (
	StepOutcomeUnknown            StepOutcome = "unknown"
	StepOutcomeSucceeded          StepOutcome = "succeeded"
	StepOutcomePartiallySucceeded StepOutcome = "partially_succeeded"
	StepOutcomeFailed             StepOutcome = "failed"
	StepOutcomeSkipped            StepOutcome = "skipped"
)

type WorkflowRecommendation string

const (
	WorkflowRecommendationNone            WorkflowRecommendation = "none"
	WorkflowRecommendationResume          WorkflowRecommendation = "resume"
	WorkflowRecommendationReconcile       WorkflowRecommendation = "reconcile"
	WorkflowRecommendationWait            WorkflowRecommendation = "wait"
	WorkflowRecommendationRetrySubmission WorkflowRecommendation = "retry_submission"
	WorkflowRecommendationManualReview    WorkflowRecommendation = "manual_review"
	WorkflowRecommendationViewSummary     WorkflowRecommendation = "view_summary"
)

type ContinuationReadiness string

const (
	ContinuationReadinessUnknown              ContinuationReadiness = "unknown"
	ContinuationReadinessWaitingExternal      ContinuationReadiness = "waiting_external"
	ContinuationReadinessReadyToReconcile     ContinuationReadiness = "ready_to_reconcile"
	ContinuationReadinessReadyToResume        ContinuationReadiness = "ready_to_resume"
	ContinuationReadinessReadyForFinalization ContinuationReadiness = "ready_for_finalization"
	ContinuationReadinessBlocked              ContinuationReadiness = "blocked"
	ContinuationReadinessTerminal             ContinuationReadiness = "terminal"
)

func (readiness ContinuationReadiness) ReadyNow() bool {
	switch readiness {
	case ContinuationReadinessReadyToReconcile, ContinuationReadinessReadyToResume, ContinuationReadinessReadyForFinalization:
		return true
	default:
		return false
	}
}

type ContinuationReason string

const (
	ContinuationReasonUnknown                    ContinuationReason = "unknown"
	ContinuationReasonExternalDependencyPending  ContinuationReason = "external_dependency_pending"
	ContinuationReasonResultsAvailable           ContinuationReason = "results_available"
	ContinuationReasonPartialResultsAvailable    ContinuationReason = "partial_results_available"
	ContinuationReasonAwaitingValidation         ContinuationReason = "awaiting_validation"
	ContinuationReasonAwaitingMaterialization    ContinuationReason = "awaiting_materialization"
	ContinuationReasonManualInterventionRequired ContinuationReason = "manual_intervention_required"
	ContinuationReasonAlreadyTerminal            ContinuationReason = "already_terminal"
)
