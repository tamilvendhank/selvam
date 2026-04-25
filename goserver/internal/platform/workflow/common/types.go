package common

import (
	"fmt"
	"strings"
	"time"

	"goserver/internal/platform/domain"
)

type ConfigSnapshotSelector struct {
	SnapshotID string         `json:"snapshotId,omitempty"`
	Override   map[string]any `json:"override,omitempty"`
}

func (selector ConfigSnapshotSelector) Validate() error {
	if selector.SnapshotID != "" && strings.TrimSpace(selector.SnapshotID) == "" {
		return fmt.Errorf("config snapshot id cannot be blank")
	}
	return nil
}

type WorkflowStartRequest struct {
	BookType         domain.BookType        `json:"bookType,omitempty"`
	RunType          domain.WorkflowRunType `json:"runType,omitempty"`
	DryRun           bool                   `json:"dryRun,omitempty"`
	AllowedStepRange *StepRange             `json:"allowedStepRange,omitempty"`
	Config           ConfigSnapshotSelector `json:"config,omitempty"`
	InitiatedBy      string                 `json:"initiatedBy,omitempty"`
	CorrelationID    string                 `json:"correlationId,omitempty"`
	IdempotencyKey   string                 `json:"idempotencyKey,omitempty"`
	Notes            string                 `json:"notes,omitempty"`
}

func (request WorkflowStartRequest) Validate() error {
	if request.BookType != "" && !domain.IsValidBookType(request.BookType) {
		return fmt.Errorf("invalid workflow book type %q", request.BookType)
	}
	if request.RunType != "" && !domain.IsValidWorkflowRunType(request.RunType) {
		return fmt.Errorf("invalid workflow run type %q", request.RunType)
	}
	if request.AllowedStepRange != nil {
		if err := request.AllowedStepRange.Validate(); err != nil {
			return err
		}
	}
	if err := request.Config.Validate(); err != nil {
		return err
	}
	if err := validateOptionalText("initiatedBy", request.InitiatedBy); err != nil {
		return err
	}
	if err := validateOptionalText("correlationId", request.CorrelationID); err != nil {
		return err
	}
	if err := validateOptionalText("idempotencyKey", request.IdempotencyKey); err != nil {
		return err
	}
	if err := validateOptionalText("notes", request.Notes); err != nil {
		return err
	}
	return nil
}

type WorkflowResumeRequest struct {
	WorkflowRunID    string     `json:"workflowRunId"`
	AllowedStepRange *StepRange `json:"allowedStepRange,omitempty"`
	InitiatedBy      string     `json:"initiatedBy,omitempty"`
	CorrelationID    string     `json:"correlationId,omitempty"`
}

func (request WorkflowResumeRequest) Validate() error {
	if strings.TrimSpace(request.WorkflowRunID) == "" {
		return fmt.Errorf("workflowRunId is required")
	}
	if request.AllowedStepRange != nil {
		if err := request.AllowedStepRange.Validate(); err != nil {
			return err
		}
	}
	if err := validateOptionalText("initiatedBy", request.InitiatedBy); err != nil {
		return err
	}
	if err := validateOptionalText("correlationId", request.CorrelationID); err != nil {
		return err
	}
	return nil
}

type WorkflowReconcileRequest struct {
	WorkflowRunID    string     `json:"workflowRunId"`
	ForceReconcile   bool       `json:"forceReconcile,omitempty"`
	AllowedStepRange *StepRange `json:"allowedStepRange,omitempty"`
	InitiatedBy      string     `json:"initiatedBy,omitempty"`
	CorrelationID    string     `json:"correlationId,omitempty"`
}

func (request WorkflowReconcileRequest) Validate() error {
	if strings.TrimSpace(request.WorkflowRunID) == "" {
		return fmt.Errorf("workflowRunId is required")
	}
	if request.AllowedStepRange != nil {
		if err := request.AllowedStepRange.Validate(); err != nil {
			return err
		}
	}
	if err := validateOptionalText("initiatedBy", request.InitiatedBy); err != nil {
		return err
	}
	if err := validateOptionalText("correlationId", request.CorrelationID); err != nil {
		return err
	}
	return nil
}

type WorkflowOperationResult struct {
	WorkflowRunID    string                   `json:"workflowRunId"`
	BookType         domain.BookType          `json:"bookType"`
	Status           WorkflowExecutionStatus  `json:"status"`
	PersistedStatus  domain.WorkflowRunStatus `json:"persistedStatus,omitempty"`
	AsyncBatchJobIDs []string                 `json:"asyncBatchJobIds,omitempty"`
	Progress         WorkflowProgressSummary  `json:"progress"`
	StepSummaries    []StepExecutionSummary   `json:"stepSummaries,omitempty"`
	ExternalWait     *ExternalWaitSummary     `json:"externalWait,omitempty"`
	PartialFailure   *PartialFailureSummary   `json:"partialFailure,omitempty"`
	ContinuationHint WorkflowContinuationHint `json:"continuationHint"`
	SummaryAvailable bool                     `json:"summaryAvailable"`
}

func (result WorkflowOperationResult) IsTerminal() bool {
	return result.Status.IsTerminal()
}

func (result WorkflowOperationResult) RequiresExternalWait() bool {
	if result.Status.RequiresExternalWait() {
		return true
	}
	return result.ExternalWait != nil && result.ExternalWait.HasPending()
}

func (result WorkflowOperationResult) CanResume() bool {
	return result.Status.CanResume() || result.ContinuationHint.CanResume()
}

func (result WorkflowOperationResult) CanReconcile() bool {
	return result.Status.CanReconcile() || result.ContinuationHint.CanReconcile()
}

type WorkflowStartResult = WorkflowOperationResult
type WorkflowResumeResult = WorkflowOperationResult
type WorkflowReconcileResult = WorkflowOperationResult

type WorkflowStatusView struct {
	WorkflowRunID    string                   `json:"workflowRunId"`
	BookType         domain.BookType          `json:"bookType"`
	RunType          domain.WorkflowRunType   `json:"runType,omitempty"`
	Mode             string                   `json:"mode,omitempty"`
	Status           WorkflowExecutionStatus  `json:"status"`
	PersistedStatus  domain.WorkflowRunStatus `json:"persistedStatus,omitempty"`
	CurrentStep      StepName                 `json:"currentStep,omitempty"`
	NextStep         StepName                 `json:"nextStep,omitempty"`
	StartedAt        *time.Time               `json:"startedAt,omitempty"`
	UpdatedAt        *time.Time               `json:"updatedAt,omitempty"`
	CompletedAt      *time.Time               `json:"completedAt,omitempty"`
	Progress         WorkflowProgressSummary  `json:"progress"`
	ExternalWait     *ExternalWaitSummary     `json:"externalWait,omitempty"`
	PartialFailure   *PartialFailureSummary   `json:"partialFailure,omitempty"`
	ContinuationHint WorkflowContinuationHint `json:"continuationHint"`
	Errors           []WorkflowErrorSummary   `json:"errors,omitempty"`
	StepSummaries    []StepExecutionSummary   `json:"stepSummaries,omitempty"`
}

func (view WorkflowStatusView) IsTerminal() bool {
	return view.Status.IsTerminal()
}

func (view WorkflowStatusView) RequiresExternalWait() bool {
	if view.Status.RequiresExternalWait() {
		return true
	}
	return view.ExternalWait != nil && view.ExternalWait.HasPending()
}

func (view WorkflowStatusView) CanResume() bool {
	return view.Status.CanResume() || view.ContinuationHint.CanResume()
}

func (view WorkflowStatusView) CanReconcile() bool {
	return view.Status.CanReconcile() || view.ContinuationHint.CanReconcile()
}

type WorkflowProgressSummary struct {
	TotalSteps      int                  `json:"totalSteps"`
	PendingSteps    int                  `json:"pendingSteps,omitempty"`
	RunningSteps    int                  `json:"runningSteps,omitempty"`
	WaitingSteps    int                  `json:"waitingSteps,omitempty"`
	CompletedSteps  int                  `json:"completedSteps,omitempty"`
	FailedSteps     int                  `json:"failedSteps,omitempty"`
	SkippedSteps    int                  `json:"skippedSteps,omitempty"`
	PercentComplete float64              `json:"percentComplete,omitempty"`
	CurrentStep     StepName             `json:"currentStep,omitempty"`
	NextStep        StepName             `json:"nextStep,omitempty"`
	Counts          WorkflowCounts       `json:"counts,omitempty"`
	Async           WorkflowAsyncSummary `json:"async,omitempty"`
}

func (summary WorkflowProgressSummary) HasFailures() bool {
	return summary.FailedSteps > 0 ||
		summary.Counts.ValidationFailed > 0 ||
		summary.Async.BatchJobsFailed > 0 ||
		summary.Async.ItemsFailed > 0 ||
		summary.Async.ItemsInvalid > 0
}

type WorkflowCounts struct {
	UniverseScanned   int `json:"universeScanned,omitempty"`
	Eligible          int `json:"eligible,omitempty"`
	Rejected          int `json:"rejected,omitempty"`
	InputsBuilt       int `json:"inputsBuilt,omitempty"`
	PendingRecords    int `json:"pendingRecords,omitempty"`
	BatchJobs         int `json:"batchJobs,omitempty"`
	BatchItems        int `json:"batchItems,omitempty"`
	BatchItemsDone    int `json:"batchItemsDone,omitempty"`
	BatchItemsFailed  int `json:"batchItemsFailed,omitempty"`
	BatchItemsInvalid int `json:"batchItemsInvalid,omitempty"`
	ValidationPassed  int `json:"validationPassed,omitempty"`
	ValidationFailed  int `json:"validationFailed,omitempty"`
	Materialized      int `json:"materialized,omitempty"`
	CandidatesBuilt   int `json:"candidatesBuilt,omitempty"`
	Approved          int `json:"approved,omitempty"`
	Allocated         int `json:"allocated,omitempty"`
	Persisted         int `json:"persisted,omitempty"`
	Published         int `json:"published,omitempty"`
}

type WorkflowAsyncSummary struct {
	BatchJobsTotal         int        `json:"batchJobsTotal,omitempty"`
	BatchJobsPending       int        `json:"batchJobsPending,omitempty"`
	BatchJobsRunning       int        `json:"batchJobsRunning,omitempty"`
	BatchJobsWaiting       int        `json:"batchJobsWaiting,omitempty"`
	BatchJobsCompleted     int        `json:"batchJobsCompleted,omitempty"`
	BatchJobsFailed        int        `json:"batchJobsFailed,omitempty"`
	BatchJobsCancelled     int        `json:"batchJobsCancelled,omitempty"`
	ItemsTotal             int        `json:"itemsTotal,omitempty"`
	ItemsPending           int        `json:"itemsPending,omitempty"`
	ItemsProcessing        int        `json:"itemsProcessing,omitempty"`
	ItemsCompleted         int        `json:"itemsCompleted,omitempty"`
	ItemsFailed            int        `json:"itemsFailed,omitempty"`
	ItemsInvalid           int        `json:"itemsInvalid,omitempty"`
	ItemsPendingValidation int        `json:"itemsPendingValidation,omitempty"`
	LastCheckedAt          *time.Time `json:"lastCheckedAt,omitempty"`
}

func (summary WorkflowAsyncSummary) HasPendingWork() bool {
	return summary.BatchJobsPending > 0 ||
		summary.BatchJobsRunning > 0 ||
		summary.BatchJobsWaiting > 0 ||
		summary.ItemsPending > 0 ||
		summary.ItemsProcessing > 0
}

type StepExecutionSummary struct {
	Descriptor           WorkflowStepDescriptor `json:"descriptor"`
	Status               StepExecutionStatus    `json:"status"`
	Outcome              StepOutcome            `json:"outcome,omitempty"`
	StartedAt            *time.Time             `json:"startedAt,omitempty"`
	CompletedAt          *time.Time             `json:"completedAt,omitempty"`
	DurationMs           int64                  `json:"durationMs,omitempty"`
	Retryable            bool                   `json:"retryable,omitempty"`
	RequiresExternalWait bool                   `json:"requiresExternalWait,omitempty"`
	Counts               WorkflowCounts         `json:"counts,omitempty"`
	Error                *WorkflowErrorSummary  `json:"error,omitempty"`
	Metadata             map[string]any         `json:"metadata,omitempty"`
}

func (summary StepExecutionSummary) IsTerminal() bool {
	return summary.Status.IsTerminal()
}

type WorkflowErrorSummary struct {
	Code        string         `json:"code,omitempty"`
	Message     string         `json:"message,omitempty"`
	Step        StepName       `json:"step,omitempty"`
	Retryable   bool           `json:"retryable,omitempty"`
	Recoverable bool           `json:"recoverable,omitempty"`
	Metadata    map[string]any `json:"metadata,omitempty"`
}

type PartialFailureSummary struct {
	Count             int        `json:"count,omitempty"`
	RetryableCount    int        `json:"retryableCount,omitempty"`
	NonRetryableCount int        `json:"nonRetryableCount,omitempty"`
	Steps             []StepName `json:"steps,omitempty"`
	References        []string   `json:"references,omitempty"`
	Message           string     `json:"message,omitempty"`
}

type WorkflowContinuationHint struct {
	Readiness             ContinuationReadiness  `json:"readiness"`
	Reason                ContinuationReason     `json:"reason"`
	Recommendation        WorkflowRecommendation `json:"recommendation"`
	NextStep              StepName               `json:"nextStep,omitempty"`
	Message               string                 `json:"message,omitempty"`
	Retryable             bool                   `json:"retryable,omitempty"`
	EarliestNextAttemptAt *time.Time             `json:"earliestNextAttemptAt,omitempty"`
}

func (hint WorkflowContinuationHint) ReadyNow() bool {
	return hint.Readiness.ReadyNow()
}

func (hint WorkflowContinuationHint) CanResume() bool {
	return hint.Readiness == ContinuationReadinessReadyToResume || hint.Recommendation == WorkflowRecommendationResume
}

func (hint WorkflowContinuationHint) CanReconcile() bool {
	return hint.Readiness == ContinuationReadinessReadyToReconcile || hint.Recommendation == WorkflowRecommendationReconcile
}

type ExternalDependencyRef struct {
	Kind              string     `json:"kind,omitempty"`
	ReferenceID       string     `json:"referenceId,omitempty"`
	ExternalHandle    string     `json:"externalHandle,omitempty"`
	ProviderName      string     `json:"providerName,omitempty"`
	Status            string     `json:"status,omitempty"`
	ReadyForReconcile bool       `json:"readyForReconcile,omitempty"`
	LastObservedAt    *time.Time `json:"lastObservedAt,omitempty"`
}

type ExternalWaitSummary struct {
	Dependencies       []ExternalDependencyRef `json:"dependencies,omitempty"`
	PendingCount       int                     `json:"pendingCount,omitempty"`
	CompletedCount     int                     `json:"completedCount,omitempty"`
	FailedCount        int                     `json:"failedCount,omitempty"`
	ProviderNames      []string                `json:"providerNames,omitempty"`
	SuggestedPollAfter *time.Time              `json:"suggestedPollAfter,omitempty"`
	Message            string                  `json:"message,omitempty"`
}

func (summary ExternalWaitSummary) HasPending() bool {
	return summary.PendingCount > 0
}

func validateOptionalText(fieldName string, value string) error {
	if value != "" && strings.TrimSpace(value) == "" {
		return fmt.Errorf("%s cannot be blank", fieldName)
	}
	return nil
}
