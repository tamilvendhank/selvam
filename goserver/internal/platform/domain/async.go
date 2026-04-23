package domain

import (
	"fmt"
	"strings"
	"time"
)

type AIBatchJob struct {
	ID                   string         `json:"id" bson:"-"`
	JobType              BatchJobType   `json:"jobType" bson:"jobType"`
	WorkflowRunID        string         `json:"workflowRunId" bson:"workflowRunId"`
	BookType             BookType       `json:"bookType" bson:"bookType"`
	ProviderName         string         `json:"providerName" bson:"providerName"`
	ProviderJobHandle    string         `json:"providerJobHandle,omitempty" bson:"providerJobHandle,omitempty"`
	LocalJobHandle       string         `json:"localJobHandle,omitempty" bson:"localJobHandle,omitempty"`
	Status               BatchJobStatus `json:"status" bson:"status"`
	SubmissionPayloadRef map[string]any `json:"submissionPayloadRef,omitempty" bson:"submissionPayloadRef,omitempty"`
	ResultPayloadRef     map[string]any `json:"resultPayloadRef,omitempty" bson:"resultPayloadRef,omitempty"`
	SubmittedAt          *time.Time     `json:"submittedAt,omitempty" bson:"submittedAt,omitempty"`
	LastPolledAt         *time.Time     `json:"lastPolledAt,omitempty" bson:"lastPolledAt,omitempty"`
	CompletedAt          *time.Time     `json:"completedAt,omitempty" bson:"completedAt,omitempty"`
	FailedAt             *time.Time     `json:"failedAt,omitempty" bson:"failedAt,omitempty"`
	ErrorSummary         string         `json:"errorSummary,omitempty" bson:"errorSummary,omitempty"`
	RetryCount           int            `json:"retryCount" bson:"retryCount"`
	MaxRetryCount        int            `json:"maxRetryCount" bson:"maxRetryCount"`
	IdempotencyKey       string         `json:"idempotencyKey,omitempty" bson:"idempotencyKey,omitempty"`
	SchemaVersion        string         `json:"schemaVersion" bson:"schemaVersion"`
	CreatedAt            time.Time      `json:"createdAt" bson:"createdAt"`
	UpdatedAt            time.Time      `json:"updatedAt" bson:"updatedAt"`
}

func (job *AIBatchJob) Validate() error {
	if job == nil {
		return fmt.Errorf("ai batch job is required")
	}
	if !IsValidBatchJobType(job.JobType) {
		return fmt.Errorf("invalid ai batch job type %q", job.JobType)
	}
	if strings.TrimSpace(job.WorkflowRunID) == "" {
		return fmt.Errorf("ai batch job workflowRunId is required")
	}
	if !IsValidBookType(job.BookType) {
		return fmt.Errorf("invalid ai batch job book type %q", job.BookType)
	}
	if strings.TrimSpace(job.ProviderName) == "" {
		return fmt.Errorf("ai batch job providerName is required")
	}
	if !IsValidBatchJobStatus(job.Status) {
		return fmt.Errorf("invalid ai batch job status %q", job.Status)
	}
	if job.RetryCount < 0 {
		return fmt.Errorf("ai batch job retryCount must be zero or greater")
	}
	if job.MaxRetryCount < 0 {
		return fmt.Errorf("ai batch job maxRetryCount must be zero or greater")
	}
	if job.RetryCount > job.MaxRetryCount && job.MaxRetryCount > 0 {
		return fmt.Errorf("ai batch job retryCount cannot exceed maxRetryCount")
	}
	if strings.TrimSpace(job.SchemaVersion) == "" {
		return fmt.Errorf("ai batch job schemaVersion is required")
	}
	if err := ValidateNonZeroTime("ai batch job createdAt", job.CreatedAt); err != nil {
		return err
	}
	if err := ValidateNonZeroTime("ai batch job updatedAt", job.UpdatedAt); err != nil {
		return err
	}

	return nil
}

type AIBatchItem struct {
	ID                  string           `json:"id" bson:"-"`
	AIBatchJobID        string           `json:"aiBatchJobId" bson:"aiBatchJobId"`
	WorkflowRunID       string           `json:"workflowRunId" bson:"workflowRunId"`
	CompanyID           string           `json:"companyId,omitempty" bson:"companyId,omitempty"`
	Symbol              string           `json:"symbol,omitempty" bson:"symbol,omitempty"`
	BookType            BookType         `json:"bookType" bson:"bookType"`
	ItemType            BatchItemType    `json:"itemType" bson:"itemType"`
	InputPayload        map[string]any   `json:"inputPayload,omitempty" bson:"inputPayload,omitempty"`
	InputHash           string           `json:"inputHash,omitempty" bson:"inputHash,omitempty"`
	Status              BatchItemStatus  `json:"status" bson:"status"`
	ResultPayload       map[string]any   `json:"resultPayload,omitempty" bson:"resultPayload,omitempty"`
	ValidationStatus    ValidationStatus `json:"validationStatus" bson:"validationStatus"`
	ValidationErrors    []string         `json:"validationErrors,omitempty" bson:"validationErrors,omitempty"`
	TargetReviewID      string           `json:"targetReviewId,omitempty" bson:"targetReviewId,omitempty"`
	TargetThesisID      string           `json:"targetThesisId,omitempty" bson:"targetThesisId,omitempty"`
	TargetEntityVersion int              `json:"targetEntityVersion,omitempty" bson:"targetEntityVersion,omitempty"`
	ErrorSummary        string           `json:"errorSummary,omitempty" bson:"errorSummary,omitempty"`
	CreatedAt           time.Time        `json:"createdAt" bson:"createdAt"`
	UpdatedAt           time.Time        `json:"updatedAt" bson:"updatedAt"`
	CompletedAt         *time.Time       `json:"completedAt,omitempty" bson:"completedAt,omitempty"`
}

func (item *AIBatchItem) Validate() error {
	if item == nil {
		return fmt.Errorf("ai batch item is required")
	}
	if strings.TrimSpace(item.AIBatchJobID) == "" {
		return fmt.Errorf("ai batch item aiBatchJobId is required")
	}
	if strings.TrimSpace(item.WorkflowRunID) == "" {
		return fmt.Errorf("ai batch item workflowRunId is required")
	}
	if !IsValidBookType(item.BookType) {
		return fmt.Errorf("invalid ai batch item book type %q", item.BookType)
	}
	if !IsValidBatchItemType(item.ItemType) {
		return fmt.Errorf("invalid ai batch item type %q", item.ItemType)
	}
	if !IsValidBatchItemStatus(item.Status) {
		return fmt.Errorf("invalid ai batch item status %q", item.Status)
	}
	if !IsValidValidationStatus(item.ValidationStatus) {
		return fmt.Errorf("invalid ai batch item validation status %q", item.ValidationStatus)
	}
	if err := ValidateNonZeroTime("ai batch item createdAt", item.CreatedAt); err != nil {
		return err
	}
	if err := ValidateNonZeroTime("ai batch item updatedAt", item.UpdatedAt); err != nil {
		return err
	}

	return nil
}

type WorkflowStepRun struct {
	ID            string                 `json:"id" bson:"-"`
	WorkflowRunID string                 `json:"workflowRunId" bson:"workflowRunId"`
	StepName      string                 `json:"stepName" bson:"stepName"`
	Status        WorkflowStepStatusType `json:"status" bson:"status"`
	StartedAt     *time.Time             `json:"startedAt,omitempty" bson:"startedAt,omitempty"`
	CompletedAt   *time.Time             `json:"completedAt,omitempty" bson:"completedAt,omitempty"`
	ErrorSummary  string                 `json:"errorSummary,omitempty" bson:"errorSummary,omitempty"`
	Metadata      map[string]any         `json:"metadata,omitempty" bson:"metadata,omitempty"`
	CreatedAt     time.Time              `json:"createdAt" bson:"createdAt"`
	UpdatedAt     time.Time              `json:"updatedAt" bson:"updatedAt"`
}

func (run *WorkflowStepRun) Validate() error {
	if run == nil {
		return fmt.Errorf("workflow step run is required")
	}
	if strings.TrimSpace(run.WorkflowRunID) == "" {
		return fmt.Errorf("workflow step run workflowRunId is required")
	}
	if strings.TrimSpace(run.StepName) == "" {
		return fmt.Errorf("workflow step run stepName is required")
	}
	if !IsValidWorkflowStepStatus(run.Status) {
		return fmt.Errorf("invalid workflow step run status %q", run.Status)
	}
	if err := ValidateNonZeroTime("workflow step run createdAt", run.CreatedAt); err != nil {
		return err
	}
	if err := ValidateNonZeroTime("workflow step run updatedAt", run.UpdatedAt); err != nil {
		return err
	}

	return nil
}

type JobReconciliationLog struct {
	ID                       string         `json:"id" bson:"-"`
	AIBatchJobID             string         `json:"aiBatchJobId" bson:"aiBatchJobId"`
	PolledAt                 time.Time      `json:"polledAt" bson:"polledAt"`
	StatusBefore             BatchJobStatus `json:"statusBefore,omitempty" bson:"statusBefore,omitempty"`
	StatusAfter              BatchJobStatus `json:"statusAfter,omitempty" bson:"statusAfter,omitempty"`
	ItemsCompletedDelta      int            `json:"itemsCompletedDelta,omitempty" bson:"itemsCompletedDelta,omitempty"`
	ItemsFailedDelta         int            `json:"itemsFailedDelta,omitempty" bson:"itemsFailedDelta,omitempty"`
	RawProviderStatusSummary map[string]any `json:"rawProviderStatusSummary,omitempty" bson:"rawProviderStatusSummary,omitempty"`
	ErrorSummary             string         `json:"errorSummary,omitempty" bson:"errorSummary,omitempty"`
	CreatedAt                time.Time      `json:"createdAt" bson:"createdAt"`
}

func (log *JobReconciliationLog) Validate() error {
	if log == nil {
		return fmt.Errorf("job reconciliation log is required")
	}
	if strings.TrimSpace(log.AIBatchJobID) == "" {
		return fmt.Errorf("job reconciliation log aiBatchJobId is required")
	}
	if err := ValidateNonZeroTime("job reconciliation log polledAt", log.PolledAt); err != nil {
		return err
	}
	if log.StatusBefore != "" && !IsValidBatchJobStatus(log.StatusBefore) {
		return fmt.Errorf("invalid job reconciliation log statusBefore %q", log.StatusBefore)
	}
	if log.StatusAfter != "" && !IsValidBatchJobStatus(log.StatusAfter) {
		return fmt.Errorf("invalid job reconciliation log statusAfter %q", log.StatusAfter)
	}
	if err := ValidateNonZeroTime("job reconciliation log createdAt", log.CreatedAt); err != nil {
		return err
	}

	return nil
}
