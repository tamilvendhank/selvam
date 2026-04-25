package common

import (
	"fmt"
	"strings"
	"time"

	domaincommon "goserver/internal/domain/common"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

type ServiceOutcome string

const (
	ServiceOutcomeUnknown  ServiceOutcome = ""
	ServiceOutcomeNoop     ServiceOutcome = "noop"
	ServiceOutcomeDryRun   ServiceOutcome = "dry_run"
	ServiceOutcomeSkipped  ServiceOutcome = "skipped"
	ServiceOutcomeSuccess  ServiceOutcome = "success"
	ServiceOutcomePartial  ServiceOutcome = "partial_success"
	ServiceOutcomeFailed   ServiceOutcome = "failed"
	ServiceOutcomeBlocked  ServiceOutcome = "blocked"
	ServiceOutcomeDeferred ServiceOutcome = "deferred"
)

func (outcome ServiceOutcome) IsTerminal() bool {
	switch outcome {
	case ServiceOutcomeNoop, ServiceOutcomeDryRun, ServiceOutcomeSkipped, ServiceOutcomeSuccess, ServiceOutcomePartial, ServiceOutcomeFailed:
		return true
	default:
		return false
	}
}

func (outcome ServiceOutcome) HasFailures() bool {
	return outcome == ServiceOutcomeFailed || outcome == ServiceOutcomePartial
}

type OperationSummary struct {
	Operation      string         `json:"operation,omitempty"`
	Outcome        ServiceOutcome `json:"outcome,omitempty"`
	AttemptedCount int            `json:"attemptedCount,omitempty"`
	SuccessCount   int            `json:"successCount,omitempty"`
	SkippedCount   int            `json:"skippedCount,omitempty"`
	FailureCount   int            `json:"failureCount,omitempty"`
	RetryableCount int            `json:"retryableCount,omitempty"`
	DryRun         bool           `json:"dryRun,omitempty"`
	StartedAt      *time.Time     `json:"startedAt,omitempty"`
	CompletedAt    *time.Time     `json:"completedAt,omitempty"`
	Message        string         `json:"message,omitempty"`
}

func (summary OperationSummary) HasFailures() bool {
	return summary.FailureCount > 0 || summary.Outcome.HasFailures()
}

func (summary OperationSummary) IsTerminal() bool {
	return summary.Outcome.IsTerminal()
}

type WorkerOperationSummary struct {
	OperationSummary
	DiscoveredCount int `json:"discoveredCount,omitempty"`
	ClaimedCount    int `json:"claimedCount,omitempty"`
	ReleasedCount   int `json:"releasedCount,omitempty"`
}

type BatchSubmissionSummary struct {
	OperationSummary
	SubmittedCount int `json:"submittedCount,omitempty"`
}

type BatchPollingSummary struct {
	OperationSummary
	PolledCount       int `json:"polledCount,omitempty"`
	StatusChangeCount int `json:"statusChangeCount,omitempty"`
}

type ReconciliationSummary struct {
	OperationSummary
	ReconciledJobCount int `json:"reconciledJobCount,omitempty"`
	ItemsCompleted     int `json:"itemsCompleted,omitempty"`
	ItemsFailed        int `json:"itemsFailed,omitempty"`
	ItemsInvalid       int `json:"itemsInvalid,omitempty"`
	ItemsStillPending  int `json:"itemsStillPending,omitempty"`
}

type ValidationSummary struct {
	OperationSummary
	ValidCount   int `json:"validCount,omitempty"`
	InvalidCount int `json:"invalidCount,omitempty"`
	IssueCount   int `json:"issueCount,omitempty"`
}

type MaterializationSummary struct {
	OperationSummary
	MaterializedCount int `json:"materializedCount,omitempty"`
}

type FinalizationSummary struct {
	OperationSummary
	FinalizedCount   int `json:"finalizedCount,omitempty"`
	SupersededCount  int `json:"supersededCount,omitempty"`
	PreconditionMiss int `json:"preconditionMissCount,omitempty"`
}

type ContinuationSummary struct {
	OperationSummary
	ReadyCount     int `json:"readyCount,omitempty"`
	BlockedCount   int `json:"blockedCount,omitempty"`
	ContinuedCount int `json:"continuedCount,omitempty"`
	CompletedCount int `json:"completedCount,omitempty"`
}

type ThesisSummary struct {
	OperationSummary
	CreatedCount     int `json:"createdCount,omitempty"`
	UpdatedCount     int `json:"updatedCount,omitempty"`
	BrokenCount      int `json:"brokenCount,omitempty"`
	UnderReviewCount int `json:"underReviewCount,omitempty"`
}

type ActionMappingSummary struct {
	OperationSummary
	MappedCount        int `json:"mappedCount,omitempty"`
	CapitalEligible    int `json:"capitalEligible,omitempty"`
	ConstraintHitCount int `json:"constraintHitCount,omitempty"`
}

type BucketAssignmentSummary struct {
	OperationSummary
	AssignedCount int `json:"assignedCount,omitempty"`
	ChangedCount  int `json:"changedCount,omitempty"`
}

type AllocationSummary struct {
	OperationSummary
	CandidateCount       int     `json:"candidateCount,omitempty"`
	AllocatedCount       int     `json:"allocatedCount,omitempty"`
	BlockedCount         int     `json:"blockedCount,omitempty"`
	AllocatedCashTotal   float64 `json:"allocatedCashTotal,omitempty"`
	UnallocatedCashTotal float64 `json:"unallocatedCashTotal,omitempty"`
}

type ProjectionUpdateSummary struct {
	OperationSummary
	UpdatedCompanyCount  int `json:"updatedCompanyCount,omitempty"`
	UpdatedPositionCount int `json:"updatedPositionCount,omitempty"`
	UpdatedReviewCount   int `json:"updatedReviewCount,omitempty"`
}

type FailureScope string

const (
	FailureScopeOperation    FailureScope = "operation"
	FailureScopeJob          FailureScope = "job"
	FailureScopeItem         FailureScope = "item"
	FailureScopeReview       FailureScope = "review"
	FailureScopeWorkflow     FailureScope = "workflow"
	FailureScopeThesis       FailureScope = "thesis"
	FailureScopeCandidate    FailureScope = "candidate"
	FailureScopeAllocation   FailureScope = "allocation"
	FailureScopeProjection   FailureScope = "projection"
	FailureScopeContinuation FailureScope = "continuation"
)

type RetryClass string

const (
	RetryClassUnknown        RetryClass = ""
	RetryClassNone           RetryClass = "none"
	RetryClassTransient      RetryClass = "transient"
	RetryClassRateLimited    RetryClass = "rate_limited"
	RetryClassProvider       RetryClass = "provider"
	RetryClassConflict       RetryClass = "conflict"
	RetryClassDependency     RetryClass = "dependency"
	RetryClassValidation     RetryClass = "validation"
	RetryClassPermanent      RetryClass = "permanent"
	RetryClassManualReview   RetryClass = "manual_review"
	RetryClassRetryExhausted RetryClass = "retry_exhausted"
)

type RetryPolicyHint struct {
	Retryable      bool       `json:"retryable,omitempty"`
	RetryClass     RetryClass `json:"retryClass,omitempty"`
	RetryAfter     *time.Time `json:"retryAfter,omitempty"`
	AttemptsUsed   int        `json:"attemptsUsed,omitempty"`
	MaxAttempts    int        `json:"maxAttempts,omitempty"`
	BackoffSeconds int        `json:"backoffSeconds,omitempty"`
	Reason         string     `json:"reason,omitempty"`
}

func (hint RetryPolicyHint) IsRetryable() bool {
	return hint.Retryable && hint.RetryClass != RetryClassNone && hint.RetryClass != RetryClassPermanent && hint.RetryClass != RetryClassRetryExhausted
}

type RetryDecision struct {
	Retry       bool            `json:"retry,omitempty"`
	RetryClass  RetryClass      `json:"retryClass,omitempty"`
	RetryAfter  *time.Time      `json:"retryAfter,omitempty"`
	PolicyHint  RetryPolicyHint `json:"policyHint,omitempty"`
	Reason      string          `json:"reason,omitempty"`
	ManualOnly  bool            `json:"manualOnly,omitempty"`
	Exhausted   bool            `json:"exhausted,omitempty"`
	NeedsReplan bool            `json:"needsReplan,omitempty"`
}

func (decision RetryDecision) IsRetryable() bool {
	if decision.Exhausted || decision.ManualOnly {
		return false
	}
	if decision.PolicyHint.Retryable {
		return decision.PolicyHint.IsRetryable()
	}
	return decision.Retry && decision.RetryClass != RetryClassPermanent && decision.RetryClass != RetryClassRetryExhausted
}

type PartialFailure struct {
	Scope         FailureScope       `json:"scope"`
	ID            primitive.ObjectID `json:"id,omitempty"`
	ExternalID    string             `json:"externalId,omitempty"`
	WorkflowRunID primitive.ObjectID `json:"workflowRunId,omitempty"`
	BatchJobID    primitive.ObjectID `json:"batchJobId,omitempty"`
	BatchItemID   primitive.ObjectID `json:"batchItemId,omitempty"`
	ReviewID      primitive.ObjectID `json:"reviewId,omitempty"`
	CompanyID     primitive.ObjectID `json:"companyId,omitempty"`
	Code          string             `json:"code,omitempty"`
	Message       string             `json:"message"`
	Retry         RetryPolicyHint    `json:"retry,omitempty"`
	Cause         string             `json:"cause,omitempty"`
}

func (failure PartialFailure) IsRetryable() bool {
	return failure.Retry.IsRetryable()
}

type OperationFailure struct {
	Operation string           `json:"operation,omitempty"`
	Code      string           `json:"code,omitempty"`
	Message   string           `json:"message"`
	Retry     RetryPolicyHint  `json:"retry,omitempty"`
	Failures  []PartialFailure `json:"failures,omitempty"`
}

func (failure OperationFailure) HasFailures() bool {
	return failure.Message != "" || len(failure.Failures) > 0
}

func (failure OperationFailure) IsRetryable() bool {
	if failure.Retry.IsRetryable() {
		return true
	}
	for _, partial := range failure.Failures {
		if partial.IsRetryable() {
			return true
		}
	}
	return false
}

type WorkItemKind string

const (
	WorkItemKindBatchSubmission      WorkItemKind = "batch_submission"
	WorkItemKindBatchPolling         WorkItemKind = "batch_polling"
	WorkItemKindBatchReconciliation  WorkItemKind = "batch_reconciliation"
	WorkItemKindAIOutputValidation   WorkItemKind = "ai_output_validation"
	WorkItemKindReviewMaterialize    WorkItemKind = "review_materialization"
	WorkItemKindReviewFinalize       WorkItemKind = "review_finalization"
	WorkItemKindWorkflowContinuation WorkItemKind = "workflow_continuation"
	WorkItemKindProjectionUpdate     WorkItemKind = "projection_update"
)

type WorkItemRef struct {
	Kind          WorkItemKind          `json:"kind"`
	ID            primitive.ObjectID    `json:"id,omitempty"`
	WorkflowRunID primitive.ObjectID    `json:"workflowRunId,omitempty"`
	BookType      domaincommon.BookType `json:"bookType,omitempty"`
	BatchJobID    primitive.ObjectID    `json:"batchJobId,omitempty"`
	BatchItemID   primitive.ObjectID    `json:"batchItemId,omitempty"`
	ReviewID      primitive.ObjectID    `json:"reviewId,omitempty"`
	CompanyID     primitive.ObjectID    `json:"companyId,omitempty"`
	Priority      int                   `json:"priority,omitempty"`
	Reason        string                `json:"reason,omitempty"`
	NotBeforeAt   *time.Time            `json:"notBeforeAt,omitempty"`
	Metadata      map[string]any        `json:"metadata,omitempty"`
}

type WorkDiscoveryResult struct {
	WorkItems []WorkItemRef          `json:"workItems,omitempty"`
	Limit     int                    `json:"limit,omitempty"`
	HasMore   bool                   `json:"hasMore,omitempty"`
	Summary   WorkerOperationSummary `json:"summary,omitempty"`
}

func (result WorkDiscoveryResult) HasWork() bool {
	return len(result.WorkItems) > 0
}

type BatchJobRef struct {
	ID                primitive.ObjectID            `json:"id"`
	WorkflowRunID     primitive.ObjectID            `json:"workflowRunId,omitempty"`
	BookType          domaincommon.BookType         `json:"bookType,omitempty"`
	JobType           domaincommon.AIBatchJobType   `json:"jobType,omitempty"`
	Status            domaincommon.AIBatchJobStatus `json:"status,omitempty"`
	ProviderName      string                        `json:"providerName,omitempty"`
	ProviderJobHandle string                        `json:"providerJobHandle,omitempty"`
	LocalJobHandle    string                        `json:"localJobHandle,omitempty"`
	RetryCount        int                           `json:"retryCount,omitempty"`
	MaxRetryCount     int                           `json:"maxRetryCount,omitempty"`
	SubmittedAt       *time.Time                    `json:"submittedAt,omitempty"`
	LastPolledAt      *time.Time                    `json:"lastPolledAt,omitempty"`
	CompletedAt       *time.Time                    `json:"completedAt,omitempty"`
	FailedAt          *time.Time                    `json:"failedAt,omitempty"`
}

func (reference BatchJobRef) IsTerminal() bool {
	switch reference.Status {
	case domaincommon.AIBatchJobStatusCompleted,
		domaincommon.AIBatchJobStatusFailed,
		domaincommon.AIBatchJobStatusCancelled,
		domaincommon.AIBatchJobStatusTimedOut:
		return true
	default:
		return false
	}
}

func (reference BatchJobRef) IsRetryable() bool {
	switch reference.Status {
	case domaincommon.AIBatchJobStatusFailed, domaincommon.AIBatchJobStatusTimedOut:
		return reference.MaxRetryCount == 0 || reference.RetryCount < reference.MaxRetryCount
	default:
		return false
	}
}

type BatchItemRef struct {
	ID               primitive.ObjectID             `json:"id"`
	BatchJobID       primitive.ObjectID             `json:"batchJobId,omitempty"`
	WorkflowRunID    primitive.ObjectID             `json:"workflowRunId,omitempty"`
	CompanyID        primitive.ObjectID             `json:"companyId,omitempty"`
	ReviewID         primitive.ObjectID             `json:"reviewId,omitempty"`
	BookType         domaincommon.BookType          `json:"bookType,omitempty"`
	ItemType         domaincommon.AIBatchItemType   `json:"itemType,omitempty"`
	Status           domaincommon.AIBatchItemStatus `json:"status,omitempty"`
	ValidationStatus domaincommon.ValidationStatus  `json:"validationStatus,omitempty"`
	Symbol           string                         `json:"symbol,omitempty"`
}

func (reference BatchItemRef) IsTerminal() bool {
	switch reference.Status {
	case domaincommon.AIBatchItemStatusCompleted,
		domaincommon.AIBatchItemStatusFailed,
		domaincommon.AIBatchItemStatusInvalidOutput,
		domaincommon.AIBatchItemStatusSkipped:
		return true
	default:
		return false
	}
}

type ReviewRef struct {
	ID             primitive.ObjectID                `json:"id"`
	CompanyID      primitive.ObjectID                `json:"companyId,omitempty"`
	WorkflowRunID  primitive.ObjectID                `json:"workflowRunId,omitempty"`
	BookType       domaincommon.BookType             `json:"bookType,omitempty"`
	Status         domaincommon.ReviewStatus         `json:"status,omitempty"`
	LifecycleState domaincommon.ReviewLifecycleState `json:"lifecycleState,omitempty"`
	Symbol         string                            `json:"symbol,omitempty"`
}

func (reference ReviewRef) IsFinalized() bool {
	return reference.LifecycleState == domaincommon.ReviewLifecycleStateFinalized ||
		reference.LifecycleState == domaincommon.ReviewLifecycleStateSuperseded
}

type ContinuationRef struct {
	WorkflowRunID     primitive.ObjectID             `json:"workflowRunId"`
	BookType          domaincommon.BookType          `json:"bookType,omitempty"`
	CurrentStatus     domaincommon.WorkflowRunStatus `json:"currentStatus,omitempty"`
	NextSuggestedStep domaincommon.WorkflowStepName  `json:"nextSuggestedStep,omitempty"`
	Ready             bool                           `json:"ready"`
	Blockers          []BlockingCondition            `json:"blockers,omitempty"`
}

func (reference ContinuationRef) ReadyToContinue() bool {
	return reference.Ready && len(reference.Blockers) == 0
}

type ValidationIssueSeverity string

const (
	ValidationIssueSeverityInfo    ValidationIssueSeverity = "info"
	ValidationIssueSeverityWarning ValidationIssueSeverity = "warning"
	ValidationIssueSeverityError   ValidationIssueSeverity = "error"
)

type ValidationIssue struct {
	Severity      ValidationIssueSeverity `json:"severity"`
	Code          string                  `json:"code,omitempty"`
	Message       string                  `json:"message"`
	FieldPath     string                  `json:"fieldPath,omitempty"`
	BatchItemID   primitive.ObjectID      `json:"batchItemId,omitempty"`
	ReviewID      primitive.ObjectID      `json:"reviewId,omitempty"`
	CompanyID     primitive.ObjectID      `json:"companyId,omitempty"`
	Retryable     bool                    `json:"retryable,omitempty"`
	RawValue      any                     `json:"rawValue,omitempty"`
	ProviderTrace string                  `json:"providerTrace,omitempty"`
}

func (issue ValidationIssue) IsFailure() bool {
	return issue.Severity == ValidationIssueSeverityError
}

type FieldError struct {
	FieldPath string `json:"fieldPath"`
	Code      string `json:"code,omitempty"`
	Message   string `json:"message"`
}

type BlockingCondition struct {
	Scope         FailureScope                  `json:"scope"`
	ID            primitive.ObjectID            `json:"id,omitempty"`
	WorkflowRunID primitive.ObjectID            `json:"workflowRunId,omitempty"`
	BatchJobID    primitive.ObjectID            `json:"batchJobId,omitempty"`
	BatchItemID   primitive.ObjectID            `json:"batchItemId,omitempty"`
	ReviewID      primitive.ObjectID            `json:"reviewId,omitempty"`
	Code          string                        `json:"code,omitempty"`
	Message       string                        `json:"message"`
	Retry         RetryPolicyHint               `json:"retry,omitempty"`
	WaitingOnStep domaincommon.WorkflowStepName `json:"waitingOnStep,omitempty"`
}

func (condition BlockingCondition) IsRetryable() bool {
	return condition.Retry.IsRetryable()
}

type StepRange struct {
	Start domaincommon.WorkflowStepName `json:"start,omitempty"`
	End   domaincommon.WorkflowStepName `json:"end,omitempty"`
}

func (stepRange StepRange) Validate() error {
	if stepRange.Start != "" && !stepRange.Start.IsValid() {
		return invalidRequestf("invalid start step %q", stepRange.Start)
	}
	if stepRange.End != "" && !stepRange.End.IsValid() {
		return invalidRequestf("invalid end step %q", stepRange.End)
	}
	return nil
}

func ValidateOptionalBookType(bookType domaincommon.BookType) error {
	if bookType != "" && !bookType.IsValid() {
		return invalidRequestf("invalid bookType %q", bookType)
	}
	return nil
}

func ValidateRequiredBookType(bookType domaincommon.BookType) error {
	if !bookType.IsValid() {
		return invalidRequestf("bookType is required")
	}
	return nil
}

func ValidateOptionalJobType(jobType domaincommon.AIBatchJobType) error {
	if jobType != "" && !jobType.IsValid() {
		return invalidRequestf("invalid jobType %q", jobType)
	}
	return nil
}

func ValidateRequiredJobType(jobType domaincommon.AIBatchJobType) error {
	if !jobType.IsValid() {
		return invalidRequestf("jobType is required")
	}
	return nil
}

func ValidateOptionalItemType(itemType domaincommon.AIBatchItemType) error {
	if itemType != "" && !itemType.IsValid() {
		return invalidRequestf("invalid itemType %q", itemType)
	}
	return nil
}

func ValidateRequiredItemType(itemType domaincommon.AIBatchItemType) error {
	if !itemType.IsValid() {
		return invalidRequestf("itemType is required")
	}
	return nil
}

func ValidateOptionalActionType(actionType domaincommon.InvestingActionType) error {
	if actionType != "" && !actionType.IsValid() {
		return invalidRequestf("invalid actionType %q", actionType)
	}
	return nil
}

func ValidateOptionalBucket(bucket domaincommon.WatchlistBucket) error {
	if bucket != "" && !bucket.IsValid() {
		return invalidRequestf("invalid bucket %q", bucket)
	}
	return nil
}

func ValidateOptionalWorkflowStatus(status domaincommon.WorkflowRunStatus) error {
	if status != "" && !status.IsValid() {
		return invalidRequestf("invalid workflow status %q", status)
	}
	return nil
}

func ValidateOptionalBatchJobStatuses(statuses []domaincommon.AIBatchJobStatus) error {
	for _, status := range statuses {
		if !status.IsValid() {
			return invalidRequestf("invalid batch job status %q", status)
		}
	}
	return nil
}

func ValidateRequiredObjectID(field string, id primitive.ObjectID) error {
	if id.IsZero() {
		return invalidRequestf("%s is required", field)
	}
	return nil
}

func ValidateAtLeastOneObjectID(fields map[string]primitive.ObjectID) error {
	for _, id := range fields {
		if !id.IsZero() {
			return nil
		}
	}
	names := make([]string, 0, len(fields))
	for name := range fields {
		names = append(names, name)
	}
	return invalidRequestf("at least one of %s is required", strings.Join(names, ", "))
}

func ValidateOptionalMax(field string, value int) error {
	if value < 0 {
		return invalidRequestf("%s must be zero or greater", field)
	}
	return nil
}

func ValidatePositiveMoney(field string, value float64) error {
	if value < 0 {
		return invalidRequestf("%s must be zero or greater", field)
	}
	return nil
}

func ValidateOptionalText(field string, value string) error {
	if value != "" && strings.TrimSpace(value) == "" {
		return invalidRequestf("%s cannot be blank", field)
	}
	return nil
}

func ValidateRequiredText(field string, value string) error {
	if strings.TrimSpace(value) == "" {
		return invalidRequestf("%s is required", field)
	}
	return nil
}

func ValidateRequiredTime(field string, value time.Time) error {
	if value.IsZero() {
		return invalidRequestf("%s is required", field)
	}
	return nil
}

func ValidateOptionalTimeYear(field string, value time.Time, minYear int) error {
	if value.IsZero() {
		return nil
	}
	if value.Year() < minYear {
		return invalidRequestf("%s must be in or after year %d", field, minYear)
	}
	return nil
}

func invalidRequestf(format string, args ...any) error {
	return fmt.Errorf("%w: %s", ErrInvalidServiceRequest, fmt.Sprintf(format, args...))
}
