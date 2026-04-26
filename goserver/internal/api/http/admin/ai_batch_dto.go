package admin

import (
	"time"

	domaincommon "goserver/internal/domain/common"
	servicecommon "goserver/internal/service/common"
)

type AIBatchJobListItemDTO struct {
	AIBatchJobID      string                         `json:"aiBatchJobId"`
	WorkflowRunID     string                         `json:"workflowRunId,omitempty"`
	BookType          domaincommon.BookType          `json:"bookType,omitempty"`
	JobType           domaincommon.AIBatchJobType    `json:"jobType,omitempty"`
	ProviderName      string                         `json:"providerName,omitempty"`
	ProviderJobHandle string                         `json:"providerJobHandle,omitempty"`
	LocalJobHandle    string                         `json:"localJobHandle,omitempty"`
	Status            domaincommon.AIBatchJobStatus  `json:"status,omitempty"`
	SubmittedAt       *time.Time                     `json:"submittedAt,omitempty"`
	LastPolledAt      *time.Time                     `json:"lastPolledAt,omitempty"`
	CompletedAt       *time.Time                     `json:"completedAt,omitempty"`
	FailedAt          *time.Time                     `json:"failedAt,omitempty"`
	RetryCount        int                            `json:"retryCount,omitempty"`
	MaxRetryCount     int                            `json:"maxRetryCount,omitempty"`
	ErrorSummary      string                         `json:"errorSummary,omitempty"`
	CreatedAt         time.Time                      `json:"createdAt"`
	UpdatedAt         time.Time                      `json:"updatedAt"`
}

type AIBatchJobDetailDTO struct {
	AIBatchJobListItemDTO
	SubmissionPayloadRef *domaincommon.PayloadReference `json:"submissionPayloadRef,omitempty"`
	ResultPayloadRef     *domaincommon.PayloadReference `json:"resultPayloadRef,omitempty"`
	ItemSummary          StatusCountsDTO                `json:"itemSummary,omitempty"`
	ValidationSummary    ValidationCountsDTO            `json:"validationSummary,omitempty"`
	RecentErrors         []string                       `json:"recentErrors,omitempty"`
	CountsPartial        bool                           `json:"countsPartial,omitempty"`
}

type AIBatchItemListItemDTO struct {
	AIBatchItemID        string                          `json:"aiBatchItemId"`
	AIBatchJobID         string                          `json:"aiBatchJobId,omitempty"`
	WorkflowRunID        string                          `json:"workflowRunId,omitempty"`
	CompanyID            string                          `json:"companyId,omitempty"`
	Symbol               string                          `json:"symbol,omitempty"`
	BookType             domaincommon.BookType           `json:"bookType,omitempty"`
	ItemType             domaincommon.AIBatchItemType     `json:"itemType,omitempty"`
	Status               domaincommon.AIBatchItemStatus   `json:"status,omitempty"`
	ValidationStatus     domaincommon.ValidationStatus    `json:"validationStatus,omitempty"`
	ValidationErrorCount int                             `json:"validationErrorsCount,omitempty"`
	TargetReviewID       string                          `json:"targetReviewId,omitempty"`
	TargetThesisID       string                          `json:"targetThesisId,omitempty"`
	ErrorSummary         string                          `json:"errorSummary,omitempty"`
	CreatedAt            time.Time                       `json:"createdAt"`
	UpdatedAt            time.Time                       `json:"updatedAt"`
	CompletedAt          *time.Time                      `json:"completedAt,omitempty"`
}

type AIBatchItemDetailDTO struct {
	AIBatchItemListItemDTO
	InputHash              string   `json:"inputHash,omitempty"`
	TargetEntityVersion    int      `json:"targetEntityVersion,omitempty"`
	InputPayloadIncluded    bool     `json:"inputPayloadIncluded"`
	ResultPayloadIncluded   bool     `json:"resultPayloadIncluded"`
	InputPayload            map[string]any `json:"inputPayload,omitempty"`
	ResultPayload           map[string]any `json:"resultPayload,omitempty"`
	ValidationErrors        []string `json:"validationErrors,omitempty"`
	SchemaVersion           int      `json:"schemaVersion,omitempty"`
}

type ValidationFailureDTO struct {
	AIBatchItemID     string                         `json:"aiBatchItemId"`
	AIBatchJobID      string                         `json:"aiBatchJobId,omitempty"`
	WorkflowRunID     string                         `json:"workflowRunId,omitempty"`
	CompanyID         string                         `json:"companyId,omitempty"`
	Symbol            string                         `json:"symbol,omitempty"`
	ItemType          domaincommon.AIBatchItemType   `json:"itemType,omitempty"`
	ValidationStatus  domaincommon.ValidationStatus  `json:"validationStatus,omitempty"`
	ValidationErrors  []ValidationErrorDTO           `json:"validationErrors,omitempty"`
	ErrorSummary      string                         `json:"errorSummary,omitempty"`
	UpdatedAt         time.Time                      `json:"updatedAt"`
}

type ValidationErrorDTO struct {
	Severity  servicecommon.ValidationIssueSeverity `json:"severity,omitempty"`
	Code      string                                `json:"code,omitempty"`
	Message   string                                `json:"message"`
	FieldPath string                                `json:"fieldPath,omitempty"`
}

type AdminFailuresDTO struct {
	FailedJobs   []AIBatchJobListItemDTO   `json:"failedJobs,omitempty"`
	FailedItems  []AIBatchItemListItemDTO  `json:"failedItems,omitempty"`
	FailedSteps  []WorkflowStepDTO         `json:"failedWorkflowSteps,omitempty"`
	InvalidItems []ValidationFailureDTO    `json:"invalidItems,omitempty"`
	Page         PageDTO                    `json:"page"`
}
