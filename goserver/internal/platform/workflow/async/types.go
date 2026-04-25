package async

import (
	"fmt"
	"strings"
	"time"

	"goserver/internal/platform/domain"
	"goserver/internal/platform/workflow/common"
)

type ExternalDependencyKind string

const (
	ExternalDependencyKindBatchJob     ExternalDependencyKind = "ai_batch_job"
	ExternalDependencyKindBatchItem    ExternalDependencyKind = "ai_batch_item"
	ExternalDependencyKindProviderTask ExternalDependencyKind = "provider_task"
)

type BatchReference struct {
	WorkflowRunID     string                `json:"workflowRunId"`
	BatchJobID        string                `json:"batchJobId,omitempty"`
	JobType           domain.BatchJobType   `json:"jobType"`
	BookType          domain.BookType       `json:"bookType"`
	ProviderName      string                `json:"providerName,omitempty"`
	ProviderJobHandle string                `json:"providerJobHandle,omitempty"`
	LocalJobHandle    string                `json:"localJobHandle,omitempty"`
	Status            domain.BatchJobStatus `json:"status"`
	SubmittedAt       *time.Time            `json:"submittedAt,omitempty"`
	LastPolledAt      *time.Time            `json:"lastPolledAt,omitempty"`
	CompletedAt       *time.Time            `json:"completedAt,omitempty"`
	FailedAt          *time.Time            `json:"failedAt,omitempty"`
	Metadata          map[string]any        `json:"metadata,omitempty"`
}

func (reference BatchReference) IsTerminal() bool {
	return isTerminalBatchJobStatus(reference.Status)
}

type AsyncBatchSubmissionItem struct {
	CorrelationID   string               `json:"correlationId"`
	ReferenceID     string               `json:"referenceId"`
	CompanyID       string               `json:"companyId,omitempty"`
	Symbol          string               `json:"symbol,omitempty"`
	ItemType        domain.BatchItemType `json:"itemType"`
	TargetEntityID  string               `json:"targetEntityId,omitempty"`
	InputPayload    map[string]any       `json:"inputPayload,omitempty"`
	Prompt          string               `json:"prompt,omitempty"`
	TemplateRecord  map[string]any       `json:"templateRecord,omitempty"`
	Model           string               `json:"model,omitempty"`
	ReasoningEffort string               `json:"reasoningEffort,omitempty"`
	Metadata        map[string]any       `json:"metadata,omitempty"`
}

func (item AsyncBatchSubmissionItem) Validate() error {
	if strings.TrimSpace(item.CorrelationID) == "" {
		return fmt.Errorf("correlationId is required")
	}
	if strings.TrimSpace(item.ReferenceID) == "" {
		return fmt.Errorf("referenceId is required")
	}
	if !domain.IsValidBatchItemType(item.ItemType) {
		return fmt.Errorf("invalid batch item type %q", item.ItemType)
	}
	if len(item.InputPayload) == 0 && strings.TrimSpace(item.Prompt) == "" {
		return fmt.Errorf("batch submission item must contain inputPayload or prompt")
	}
	return nil
}

type AsyncBatchSubmissionRequest struct {
	WorkflowRunID        string                     `json:"workflowRunId"`
	BookType             domain.BookType            `json:"bookType"`
	JobType              domain.BatchJobType        `json:"jobType"`
	SubmissionStep       common.StepName            `json:"submissionStep"`
	ConfigSnapshotID     string                     `json:"configSnapshotId,omitempty"`
	IdempotencyKey       string                     `json:"idempotencyKey,omitempty"`
	PromptVersion        string                     `json:"promptVersion,omitempty"`
	ModelName            string                     `json:"modelName,omitempty"`
	ResponseInstructions string                     `json:"responseInstructions,omitempty"`
	ProviderMetadata     map[string]any             `json:"providerMetadata,omitempty"`
	Items                []AsyncBatchSubmissionItem `json:"items"`
	InitiatedBy          string                     `json:"initiatedBy,omitempty"`
	CorrelationID        string                     `json:"correlationId,omitempty"`
}

func (request AsyncBatchSubmissionRequest) Validate() error {
	if strings.TrimSpace(request.WorkflowRunID) == "" {
		return fmt.Errorf("workflowRunId is required")
	}
	if !domain.IsValidBookType(request.BookType) {
		return fmt.Errorf("invalid book type %q", request.BookType)
	}
	if !domain.IsValidBatchJobType(request.JobType) {
		return fmt.Errorf("invalid batch job type %q", request.JobType)
	}
	if strings.TrimSpace(string(request.SubmissionStep)) == "" {
		return fmt.Errorf("submissionStep is required")
	}
	if len(request.Items) == 0 {
		return fmt.Errorf("at least one batch item is required")
	}
	for index := range request.Items {
		if err := request.Items[index].Validate(); err != nil {
			return fmt.Errorf("items[%d]: %w", index, err)
		}
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
	return nil
}

type AsyncBatchSubmittedItem struct {
	BatchItemID        string                 `json:"batchItemId,omitempty"`
	CorrelationID      string                 `json:"correlationId"`
	ProviderItemHandle string                 `json:"providerItemHandle,omitempty"`
	Status             domain.BatchItemStatus `json:"status"`
	Metadata           map[string]any         `json:"metadata,omitempty"`
}

type AsyncBatchSubmissionResult struct {
	Batch             BatchReference               `json:"batch"`
	SubmittedItems    []AsyncBatchSubmittedItem    `json:"submittedItems,omitempty"`
	AcceptedItemCount int                          `json:"acceptedItemCount,omitempty"`
	RejectedItemCount int                          `json:"rejectedItemCount,omitempty"`
	ExternalWait      common.ExternalWaitSummary   `json:"externalWait,omitempty"`
	Decision          WorkflowContinuationDecision `json:"decision"`
}

func (result AsyncBatchSubmissionResult) RequiresExternalWait() bool {
	return result.ExternalWait.HasPending() || !result.Batch.IsTerminal()
}

type AsyncBatchStatusRequest struct {
	WorkflowRunID string   `json:"workflowRunId,omitempty"`
	BatchJobIDs   []string `json:"batchJobIds,omitempty"`
	IncludeItems  bool     `json:"includeItems,omitempty"`
}

func (request AsyncBatchStatusRequest) Validate() error {
	if strings.TrimSpace(request.WorkflowRunID) == "" && len(request.BatchJobIDs) == 0 {
		return fmt.Errorf("workflowRunId or batchJobIds is required")
	}
	if err := validateIDs("batchJobIds", request.BatchJobIDs); err != nil {
		return err
	}
	return nil
}

type BatchStatusSnapshot struct {
	Batch             BatchReference `json:"batch"`
	ItemsTotal        int            `json:"itemsTotal,omitempty"`
	ItemsPending      int            `json:"itemsPending,omitempty"`
	ItemsProcessing   int            `json:"itemsProcessing,omitempty"`
	ItemsCompleted    int            `json:"itemsCompleted,omitempty"`
	ItemsFailed       int            `json:"itemsFailed,omitempty"`
	ItemsInvalid      int            `json:"itemsInvalid,omitempty"`
	ResultAvailable   bool           `json:"resultAvailable,omitempty"`
	Retryable         bool           `json:"retryable,omitempty"`
	RawProviderStatus map[string]any `json:"rawProviderStatus,omitempty"`
}

func (snapshot BatchStatusSnapshot) IsTerminal() bool {
	return snapshot.Batch.IsTerminal()
}

type PendingExternalDependency struct {
	common.ExternalDependencyRef
	WorkflowRunID   string              `json:"workflowRunId"`
	BatchJobID      string              `json:"batchJobId,omitempty"`
	JobType         domain.BatchJobType `json:"jobType,omitempty"`
	PendingItems    int                 `json:"pendingItems,omitempty"`
	CompletedItems  int                 `json:"completedItems,omitempty"`
	FailedItems     int                 `json:"failedItems,omitempty"`
	InvalidItems    int                 `json:"invalidItems,omitempty"`
	ResultAvailable bool                `json:"resultAvailable,omitempty"`
}

func (dependency PendingExternalDependency) RequiresWait() bool {
	return !dependency.ReadyForReconcile
}

func (dependency PendingExternalDependency) ReadyToReconcile() bool {
	return dependency.ReadyForReconcile || dependency.ResultAvailable
}

type AsyncBatchStatusResult struct {
	WorkflowRunID       string                       `json:"workflowRunId"`
	Batches             []BatchStatusSnapshot        `json:"batches,omitempty"`
	AsyncSummary        common.WorkflowAsyncSummary  `json:"asyncSummary,omitempty"`
	PendingDependencies []PendingExternalDependency  `json:"pendingDependencies,omitempty"`
	Decision            WorkflowContinuationDecision `json:"decision"`
}

func (result AsyncBatchStatusResult) RequiresExternalWait() bool {
	return len(result.PendingDependencies) > 0 || result.AsyncSummary.HasPendingWork()
}

type AsyncBatchReconciliationRequest struct {
	WorkflowRunID         string   `json:"workflowRunId,omitempty"`
	BatchJobIDs           []string `json:"batchJobIds,omitempty"`
	Force                 bool     `json:"force,omitempty"`
	IncludeCompletedItems bool     `json:"includeCompletedItems,omitempty"`
	InitiatedBy           string   `json:"initiatedBy,omitempty"`
	CorrelationID         string   `json:"correlationId,omitempty"`
}

func (request AsyncBatchReconciliationRequest) Validate() error {
	if strings.TrimSpace(request.WorkflowRunID) == "" && len(request.BatchJobIDs) == 0 {
		return fmt.Errorf("workflowRunId or batchJobIds is required")
	}
	if err := validateIDs("batchJobIds", request.BatchJobIDs); err != nil {
		return err
	}
	if err := validateOptionalText("initiatedBy", request.InitiatedBy); err != nil {
		return err
	}
	if err := validateOptionalText("correlationId", request.CorrelationID); err != nil {
		return err
	}
	return nil
}

type ReconciledBatchItem struct {
	BatchItemID      string                  `json:"batchItemId,omitempty"`
	CorrelationID    string                  `json:"correlationId,omitempty"`
	CompanyID        string                  `json:"companyId,omitempty"`
	Symbol           string                  `json:"symbol,omitempty"`
	ItemType         domain.BatchItemType    `json:"itemType,omitempty"`
	TargetEntityID   string                  `json:"targetEntityId,omitempty"`
	Status           domain.BatchItemStatus  `json:"status"`
	ValidationStatus domain.ValidationStatus `json:"validationStatus,omitempty"`
	Retryable        bool                    `json:"retryable,omitempty"`
	ErrorSummary     string                  `json:"errorSummary,omitempty"`
	OutputPayload    map[string]any          `json:"outputPayload,omitempty"`
	ProviderMetadata map[string]any          `json:"providerMetadata,omitempty"`
}

type BatchReconciliationOutcome struct {
	Batch          BatchReference        `json:"batch"`
	StatusBefore   domain.BatchJobStatus `json:"statusBefore,omitempty"`
	StatusAfter    domain.BatchJobStatus `json:"statusAfter,omitempty"`
	ItemsCompleted int                   `json:"itemsCompleted,omitempty"`
	ItemsFailed    int                   `json:"itemsFailed,omitempty"`
	ItemsInvalid   int                   `json:"itemsInvalid,omitempty"`
	CompletedItems []ReconciledBatchItem `json:"completedItems,omitempty"`
	FailedItems    []ReconciledBatchItem `json:"failedItems,omitempty"`
	InvalidItems   []ReconciledBatchItem `json:"invalidItems,omitempty"`
	ErrorSummary   string                `json:"errorSummary,omitempty"`
}

type ResumeEligibility struct {
	WorkflowRunID string                    `json:"workflowRunId"`
	Eligible      bool                      `json:"eligible"`
	Reason        common.ContinuationReason `json:"reason"`
	NextStep      common.StepName           `json:"nextStep,omitempty"`
}

type ReadyForFinalization struct {
	WorkflowRunID       string `json:"workflowRunId"`
	Ready               bool   `json:"ready"`
	ValidCompletedItems int    `json:"validCompletedItems,omitempty"`
	InvalidItems        int    `json:"invalidItems,omitempty"`
	FailedItems         int    `json:"failedItems,omitempty"`
	PendingItems        int    `json:"pendingItems,omitempty"`
}

type WorkflowContinuationDecision struct {
	WorkflowRunID       string                        `json:"workflowRunId"`
	BookType            domain.BookType               `json:"bookType"`
	Readiness           common.ContinuationReadiness  `json:"readiness"`
	Reason              common.ContinuationReason     `json:"reason"`
	Recommendation      common.WorkflowRecommendation `json:"recommendation"`
	NextStep            common.StepName               `json:"nextStep,omitempty"`
	ResumeEligibility   ResumeEligibility             `json:"resumeEligibility"`
	Finalization        ReadyForFinalization          `json:"finalization"`
	PendingDependencies []PendingExternalDependency   `json:"pendingDependencies,omitempty"`
	ExternalWait        *common.ExternalWaitSummary   `json:"externalWait,omitempty"`
	Message             string                        `json:"message,omitempty"`
}

func (decision WorkflowContinuationDecision) ReadyNow() bool {
	return decision.Readiness.ReadyNow()
}

func (decision WorkflowContinuationDecision) CanResume() bool {
	return decision.ResumeEligibility.Eligible || decision.Recommendation == common.WorkflowRecommendationResume
}

func (decision WorkflowContinuationDecision) CanReconcile() bool {
	return decision.Readiness == common.ContinuationReadinessReadyToReconcile ||
		decision.Recommendation == common.WorkflowRecommendationReconcile
}

type ReconciliationOutcome struct {
	WorkflowRunID      string                        `json:"workflowRunId"`
	AsyncSummary       common.WorkflowAsyncSummary   `json:"asyncSummary,omitempty"`
	ReadyForValidation bool                          `json:"readyForValidation,omitempty"`
	ReadyForResume     bool                          `json:"readyForResume,omitempty"`
	Finalization       ReadyForFinalization          `json:"finalization"`
	PartialFailure     *common.PartialFailureSummary `json:"partialFailure,omitempty"`
}

type AsyncBatchReconciliationResult struct {
	WorkflowRunID string                       `json:"workflowRunId"`
	JobOutcomes   []BatchReconciliationOutcome `json:"jobOutcomes,omitempty"`
	Outcome       ReconciliationOutcome        `json:"outcome"`
	Decision      WorkflowContinuationDecision `json:"decision"`
}

func (result AsyncBatchReconciliationResult) ReadyNow() bool {
	return result.Decision.ReadyNow()
}

type CollectCompletedBatchItemsRequest struct {
	WorkflowRunID  string   `json:"workflowRunId,omitempty"`
	BatchJobIDs    []string `json:"batchJobIds,omitempty"`
	IncludeInvalid bool     `json:"includeInvalid,omitempty"`
}

func (request CollectCompletedBatchItemsRequest) Validate() error {
	if strings.TrimSpace(request.WorkflowRunID) == "" && len(request.BatchJobIDs) == 0 {
		return fmt.Errorf("workflowRunId or batchJobIds is required")
	}
	if err := validateIDs("batchJobIds", request.BatchJobIDs); err != nil {
		return err
	}
	return nil
}

type CollectCompletedBatchItemsResult struct {
	WorkflowRunID string                       `json:"workflowRunId"`
	Items         []ReconciledBatchItem        `json:"items,omitempty"`
	Outcome       ReadyForFinalization         `json:"outcome"`
	Decision      WorkflowContinuationDecision `json:"decision"`
}

type WorkflowContinuationAssessmentRequest struct {
	WorkflowRunID       string                         `json:"workflowRunId"`
	BookType            domain.BookType                `json:"bookType"`
	CurrentStatus       common.WorkflowExecutionStatus `json:"currentStatus"`
	CurrentStep         common.StepName                `json:"currentStep,omitempty"`
	AsyncSummary        common.WorkflowAsyncSummary    `json:"asyncSummary,omitempty"`
	PendingDependencies []PendingExternalDependency    `json:"pendingDependencies,omitempty"`
}

func (request WorkflowContinuationAssessmentRequest) Validate() error {
	if strings.TrimSpace(request.WorkflowRunID) == "" {
		return fmt.Errorf("workflowRunId is required")
	}
	if !domain.IsValidBookType(request.BookType) {
		return fmt.Errorf("invalid book type %q", request.BookType)
	}
	return nil
}

func isTerminalBatchJobStatus(status domain.BatchJobStatus) bool {
	switch status {
	case domain.BatchJobStatusCompleted, domain.BatchJobStatusFailed, domain.BatchJobStatusCancelled, domain.BatchJobStatusTimedOut:
		return true
	default:
		return false
	}
}

func validateOptionalText(fieldName string, value string) error {
	if value != "" && strings.TrimSpace(value) == "" {
		return fmt.Errorf("%s cannot be blank", fieldName)
	}
	return nil
}

func validateIDs(fieldName string, values []string) error {
	seen := make(map[string]struct{}, len(values))
	for _, value := range values {
		trimmed := strings.TrimSpace(value)
		if trimmed == "" {
			return fmt.Errorf("%s cannot contain blank values", fieldName)
		}
		if _, exists := seen[trimmed]; exists {
			return fmt.Errorf("%s cannot contain duplicates", fieldName)
		}
		seen[trimmed] = struct{}{}
	}
	return nil
}
