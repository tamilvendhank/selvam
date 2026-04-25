package aijob

import (
	"time"

	domaincommon "goserver/internal/domain/common"
	servicecommon "goserver/internal/service/common"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

type ProviderBatchHandle struct {
	BatchJobID        primitive.ObjectID `json:"batchJobId"`
	ProviderName      string             `json:"providerName,omitempty"`
	ProviderJobHandle string             `json:"providerJobHandle,omitempty"`
	LocalJobHandle    string             `json:"localJobHandle,omitempty"`
	SubmittedAt       *time.Time         `json:"submittedAt,omitempty"`
}

type BatchStatusUpdate struct {
	BatchJobID   primitive.ObjectID            `json:"batchJobId"`
	StatusBefore domaincommon.AIBatchJobStatus `json:"statusBefore,omitempty"`
	StatusAfter  domaincommon.AIBatchJobStatus `json:"statusAfter,omitempty"`
	PolledAt     *time.Time                    `json:"polledAt,omitempty"`
	Retryable    bool                          `json:"retryable,omitempty"`
}

type ReconciledBatchItemRef struct {
	servicecommon.BatchItemRef
	ProviderItemHandle string         `json:"providerItemHandle,omitempty"`
	ErrorSummary       string         `json:"errorSummary,omitempty"`
	OutputPayload      map[string]any `json:"outputPayload,omitempty"`
}

type SubmitBatchJobRequest struct {
	BatchJobID    primitive.ObjectID          `json:"batchJobId"`
	WorkflowRunID primitive.ObjectID          `json:"workflowRunId,omitempty"`
	BookType      domaincommon.BookType       `json:"bookType,omitempty"`
	JobType       domaincommon.AIBatchJobType `json:"jobType,omitempty"`
	DryRun        bool                        `json:"dryRun,omitempty"`
	Force         bool                        `json:"force,omitempty"`
	InitiatedBy   string                      `json:"initiatedBy,omitempty"`
	CorrelationID string                      `json:"correlationId,omitempty"`
}

func (request SubmitBatchJobRequest) Validate() error {
	if err := servicecommon.ValidateRequiredObjectID("batchJobId", request.BatchJobID); err != nil {
		return err
	}
	if err := servicecommon.ValidateOptionalBookType(request.BookType); err != nil {
		return err
	}
	if err := servicecommon.ValidateOptionalJobType(request.JobType); err != nil {
		return err
	}
	if err := servicecommon.ValidateOptionalText("initiatedBy", request.InitiatedBy); err != nil {
		return err
	}
	return servicecommon.ValidateOptionalText("correlationId", request.CorrelationID)
}

type SubmitBatchJobResult struct {
	BatchJobID      primitive.ObjectID                   `json:"batchJobId,omitempty"`
	SubmittedJobIDs []primitive.ObjectID                 `json:"submittedJobIds,omitempty"`
	SkippedJobIDs   []primitive.ObjectID                 `json:"skippedJobIds,omitempty"`
	FailedJobIDs    []primitive.ObjectID                 `json:"failedJobIds,omitempty"`
	SubmissionCount int                                  `json:"submissionCount,omitempty"`
	FailureCount    int                                  `json:"failureCount,omitempty"`
	PartialFailures []servicecommon.PartialFailure       `json:"partialFailures,omitempty"`
	ProviderHandles []ProviderBatchHandle                `json:"providerHandles,omitempty"`
	NeedsFollowUp   bool                                 `json:"needsFollowUp,omitempty"`
	Summary         servicecommon.BatchSubmissionSummary `json:"summary,omitempty"`
}

func (result SubmitBatchJobResult) HasFailures() bool {
	return result.FailureCount > 0 || len(result.PartialFailures) > 0 || result.Summary.HasFailures()
}

type SubmitPendingBatchJobsRequest struct {
	WorkflowRunID primitive.ObjectID          `json:"workflowRunId,omitempty"`
	BatchJobID    primitive.ObjectID          `json:"batchJobId,omitempty"`
	BookType      domaincommon.BookType       `json:"bookType,omitempty"`
	JobType       domaincommon.AIBatchJobType `json:"jobType,omitempty"`
	MaxJobs       int                         `json:"maxJobs,omitempty"`
	DryRun        bool                        `json:"dryRun,omitempty"`
	Force         bool                        `json:"force,omitempty"`
	InitiatedBy   string                      `json:"initiatedBy,omitempty"`
	CorrelationID string                      `json:"correlationId,omitempty"`
}

func (request SubmitPendingBatchJobsRequest) Validate() error {
	if err := servicecommon.ValidateOptionalMax("maxJobs", request.MaxJobs); err != nil {
		return err
	}
	if err := servicecommon.ValidateOptionalBookType(request.BookType); err != nil {
		return err
	}
	if err := servicecommon.ValidateOptionalJobType(request.JobType); err != nil {
		return err
	}
	if err := servicecommon.ValidateOptionalText("initiatedBy", request.InitiatedBy); err != nil {
		return err
	}
	return servicecommon.ValidateOptionalText("correlationId", request.CorrelationID)
}

type SubmitPendingBatchJobsResult struct {
	SubmittedJobIDs []primitive.ObjectID                 `json:"submittedJobIds,omitempty"`
	SkippedJobIDs   []primitive.ObjectID                 `json:"skippedJobIds,omitempty"`
	FailedJobIDs    []primitive.ObjectID                 `json:"failedJobIds,omitempty"`
	SubmissionCount int                                  `json:"submissionCount,omitempty"`
	FailureCount    int                                  `json:"failureCount,omitempty"`
	PartialFailures []servicecommon.PartialFailure       `json:"partialFailures,omitempty"`
	ProviderHandles []ProviderBatchHandle                `json:"providerHandles,omitempty"`
	NeedsFollowUp   bool                                 `json:"needsFollowUp,omitempty"`
	Summary         servicecommon.BatchSubmissionSummary `json:"summary,omitempty"`
}

func (result SubmitPendingBatchJobsResult) HasFailures() bool {
	return result.FailureCount > 0 || len(result.PartialFailures) > 0 || result.Summary.HasFailures()
}

type PollBatchJobRequest struct {
	BatchJobID       primitive.ObjectID              `json:"batchJobId"`
	WorkflowRunID    primitive.ObjectID              `json:"workflowRunId,omitempty"`
	PollOnlyStatuses []domaincommon.AIBatchJobStatus `json:"pollOnlyStatuses,omitempty"`
	Force            bool                            `json:"force,omitempty"`
	InitiatedBy      string                          `json:"initiatedBy,omitempty"`
	CorrelationID    string                          `json:"correlationId,omitempty"`
}

func (request PollBatchJobRequest) Validate() error {
	if err := servicecommon.ValidateRequiredObjectID("batchJobId", request.BatchJobID); err != nil {
		return err
	}
	if err := servicecommon.ValidateOptionalBatchJobStatuses(request.PollOnlyStatuses); err != nil {
		return err
	}
	if err := servicecommon.ValidateOptionalText("initiatedBy", request.InitiatedBy); err != nil {
		return err
	}
	return servicecommon.ValidateOptionalText("correlationId", request.CorrelationID)
}

type PollBatchJobResult struct {
	BatchJobID       primitive.ObjectID                `json:"batchJobId,omitempty"`
	PolledJobIDs     []primitive.ObjectID              `json:"polledJobIds,omitempty"`
	UpdatedStatuses  []BatchStatusUpdate               `json:"updatedStatuses,omitempty"`
	CompletedJobs    []servicecommon.BatchJobRef       `json:"completedJobs,omitempty"`
	StillRunningJobs []servicecommon.BatchJobRef       `json:"stillRunningJobs,omitempty"`
	FailedJobs       []servicecommon.BatchJobRef       `json:"failedJobs,omitempty"`
	PartialFailures  []servicecommon.PartialFailure    `json:"partialFailures,omitempty"`
	Summary          servicecommon.BatchPollingSummary `json:"summary,omitempty"`
}

func (result PollBatchJobResult) HasFailures() bool {
	return len(result.FailedJobs) > 0 || len(result.PartialFailures) > 0 || result.Summary.HasFailures()
}

type PollPendingBatchJobsRequest struct {
	BatchJobID       primitive.ObjectID              `json:"batchJobId,omitempty"`
	WorkflowRunID    primitive.ObjectID              `json:"workflowRunId,omitempty"`
	BookType         domaincommon.BookType           `json:"bookType,omitempty"`
	JobType          domaincommon.AIBatchJobType     `json:"jobType,omitempty"`
	MaxJobs          int                             `json:"maxJobs,omitempty"`
	PollOnlyStatuses []domaincommon.AIBatchJobStatus `json:"pollOnlyStatuses,omitempty"`
	Force            bool                            `json:"force,omitempty"`
	InitiatedBy      string                          `json:"initiatedBy,omitempty"`
	CorrelationID    string                          `json:"correlationId,omitempty"`
}

func (request PollPendingBatchJobsRequest) Validate() error {
	if err := servicecommon.ValidateOptionalMax("maxJobs", request.MaxJobs); err != nil {
		return err
	}
	if err := servicecommon.ValidateOptionalBookType(request.BookType); err != nil {
		return err
	}
	if err := servicecommon.ValidateOptionalJobType(request.JobType); err != nil {
		return err
	}
	if err := servicecommon.ValidateOptionalBatchJobStatuses(request.PollOnlyStatuses); err != nil {
		return err
	}
	if err := servicecommon.ValidateOptionalText("initiatedBy", request.InitiatedBy); err != nil {
		return err
	}
	return servicecommon.ValidateOptionalText("correlationId", request.CorrelationID)
}

type PollPendingBatchJobsResult struct {
	PolledJobIDs     []primitive.ObjectID              `json:"polledJobIds,omitempty"`
	UpdatedStatuses  []BatchStatusUpdate               `json:"updatedStatuses,omitempty"`
	CompletedJobs    []servicecommon.BatchJobRef       `json:"completedJobs,omitempty"`
	StillRunningJobs []servicecommon.BatchJobRef       `json:"stillRunningJobs,omitempty"`
	FailedJobs       []servicecommon.BatchJobRef       `json:"failedJobs,omitempty"`
	PartialFailures  []servicecommon.PartialFailure    `json:"partialFailures,omitempty"`
	Summary          servicecommon.BatchPollingSummary `json:"summary,omitempty"`
}

func (result PollPendingBatchJobsResult) HasFailures() bool {
	return len(result.FailedJobs) > 0 || len(result.PartialFailures) > 0 || result.Summary.HasFailures()
}

type ReconcileBatchJobRequest struct {
	BatchJobID            primitive.ObjectID `json:"batchJobId"`
	WorkflowRunID         primitive.ObjectID `json:"workflowRunId,omitempty"`
	Force                 bool               `json:"force,omitempty"`
	IncludeCompletedItems bool               `json:"includeCompletedItems,omitempty"`
	InitiatedBy           string             `json:"initiatedBy,omitempty"`
	CorrelationID         string             `json:"correlationId,omitempty"`
}

func (request ReconcileBatchJobRequest) Validate() error {
	if err := servicecommon.ValidateRequiredObjectID("batchJobId", request.BatchJobID); err != nil {
		return err
	}
	if err := servicecommon.ValidateOptionalText("initiatedBy", request.InitiatedBy); err != nil {
		return err
	}
	return servicecommon.ValidateOptionalText("correlationId", request.CorrelationID)
}

type ReconcileBatchJobResult struct {
	BatchJobID                         primitive.ObjectID                  `json:"batchJobId,omitempty"`
	ReconciledJobIDs                   []primitive.ObjectID                `json:"reconciledJobIds,omitempty"`
	CompletedItems                     []ReconciledBatchItemRef            `json:"completedItems,omitempty"`
	FailedItems                        []ReconciledBatchItemRef            `json:"failedItems,omitempty"`
	InvalidItems                       []ReconciledBatchItemRef            `json:"invalidItems,omitempty"`
	ItemsCompleted                     int                                 `json:"itemsCompleted,omitempty"`
	ItemsFailed                        int                                 `json:"itemsFailed,omitempty"`
	ItemsInvalid                       int                                 `json:"itemsInvalid,omitempty"`
	ItemsStillPending                  int                                 `json:"itemsStillPending,omitempty"`
	ReadyForValidationCount            int                                 `json:"readyForValidationCount,omitempty"`
	ReadyForContinuationWorkflowRunIDs []primitive.ObjectID                `json:"readyForContinuationWorkflowRunIds,omitempty"`
	PartialFailures                    []servicecommon.PartialFailure      `json:"partialFailures,omitempty"`
	Summary                            servicecommon.ReconciliationSummary `json:"summary,omitempty"`
}

func (result ReconcileBatchJobResult) HasFailures() bool {
	return result.ItemsFailed > 0 || result.ItemsInvalid > 0 || len(result.PartialFailures) > 0 || result.Summary.HasFailures()
}

type ReconcilePendingBatchJobsRequest struct {
	BatchJobID            primitive.ObjectID          `json:"batchJobId,omitempty"`
	WorkflowRunID         primitive.ObjectID          `json:"workflowRunId,omitempty"`
	BookType              domaincommon.BookType       `json:"bookType,omitempty"`
	JobType               domaincommon.AIBatchJobType `json:"jobType,omitempty"`
	MaxJobs               int                         `json:"maxJobs,omitempty"`
	Force                 bool                        `json:"force,omitempty"`
	IncludeCompletedItems bool                        `json:"includeCompletedItems,omitempty"`
	InitiatedBy           string                      `json:"initiatedBy,omitempty"`
	CorrelationID         string                      `json:"correlationId,omitempty"`
}

func (request ReconcilePendingBatchJobsRequest) Validate() error {
	if err := servicecommon.ValidateOptionalMax("maxJobs", request.MaxJobs); err != nil {
		return err
	}
	if err := servicecommon.ValidateOptionalBookType(request.BookType); err != nil {
		return err
	}
	if err := servicecommon.ValidateOptionalJobType(request.JobType); err != nil {
		return err
	}
	if err := servicecommon.ValidateOptionalText("initiatedBy", request.InitiatedBy); err != nil {
		return err
	}
	return servicecommon.ValidateOptionalText("correlationId", request.CorrelationID)
}

type ReconcilePendingBatchJobsResult struct {
	ReconciledJobIDs                   []primitive.ObjectID                `json:"reconciledJobIds,omitempty"`
	CompletedItems                     []ReconciledBatchItemRef            `json:"completedItems,omitempty"`
	FailedItems                        []ReconciledBatchItemRef            `json:"failedItems,omitempty"`
	InvalidItems                       []ReconciledBatchItemRef            `json:"invalidItems,omitempty"`
	ItemsCompleted                     int                                 `json:"itemsCompleted,omitempty"`
	ItemsFailed                        int                                 `json:"itemsFailed,omitempty"`
	ItemsInvalid                       int                                 `json:"itemsInvalid,omitempty"`
	ItemsStillPending                  int                                 `json:"itemsStillPending,omitempty"`
	ReadyForValidationCount            int                                 `json:"readyForValidationCount,omitempty"`
	ReadyForContinuationWorkflowRunIDs []primitive.ObjectID                `json:"readyForContinuationWorkflowRunIds,omitempty"`
	PartialFailures                    []servicecommon.PartialFailure      `json:"partialFailures,omitempty"`
	Summary                            servicecommon.ReconciliationSummary `json:"summary,omitempty"`
}

func (result ReconcilePendingBatchJobsResult) HasFailures() bool {
	return result.ItemsFailed > 0 || result.ItemsInvalid > 0 || len(result.PartialFailures) > 0 || result.Summary.HasFailures()
}

type BatchEngineSubmitItem struct {
	BatchItemID   primitive.ObjectID           `json:"batchItemId,omitempty"`
	CorrelationID string                       `json:"correlationId"`
	ItemType      domaincommon.AIBatchItemType `json:"itemType"`
	CompanyID     primitive.ObjectID           `json:"companyId,omitempty"`
	ReviewID      primitive.ObjectID           `json:"reviewId,omitempty"`
	Symbol        string                       `json:"symbol,omitempty"`
	InputPayload  map[string]any               `json:"inputPayload,omitempty"`
	Metadata      map[string]any               `json:"metadata,omitempty"`
}

type BatchEngineSubmitRequest struct {
	BatchJobID           primitive.ObjectID             `json:"batchJobId"`
	WorkflowRunID        primitive.ObjectID             `json:"workflowRunId"`
	BookType             domaincommon.BookType          `json:"bookType"`
	JobType              domaincommon.AIBatchJobType    `json:"jobType"`
	IdempotencyKey       string                         `json:"idempotencyKey,omitempty"`
	SubmissionPayloadRef *domaincommon.PayloadReference `json:"submissionPayloadRef,omitempty"`
	Items                []BatchEngineSubmitItem        `json:"items,omitempty"`
	ProviderMetadata     map[string]any                 `json:"providerMetadata,omitempty"`
}

func (request BatchEngineSubmitRequest) Validate() error {
	if err := servicecommon.ValidateRequiredObjectID("batchJobId", request.BatchJobID); err != nil {
		return err
	}
	if err := servicecommon.ValidateRequiredObjectID("workflowRunId", request.WorkflowRunID); err != nil {
		return err
	}
	if err := servicecommon.ValidateRequiredBookType(request.BookType); err != nil {
		return err
	}
	if err := servicecommon.ValidateRequiredJobType(request.JobType); err != nil {
		return err
	}
	if len(request.Items) == 0 {
		return servicecommon.ValidateRequiredText("items", "")
	}
	return nil
}

type BatchEngineSubmitResult struct {
	Handle           ProviderBatchHandle            `json:"handle"`
	Status           domaincommon.AIBatchJobStatus  `json:"status"`
	ResultPayloadRef *domaincommon.PayloadReference `json:"resultPayloadRef,omitempty"`
	AcceptedItemIDs  []primitive.ObjectID           `json:"acceptedItemIds,omitempty"`
	RejectedItemIDs  []primitive.ObjectID           `json:"rejectedItemIds,omitempty"`
	Metadata         map[string]any                 `json:"metadata,omitempty"`
}

type BatchEnginePollRequest struct {
	BatchJobID        primitive.ObjectID `json:"batchJobId"`
	ProviderJobHandle string             `json:"providerJobHandle"`
	ProviderName      string             `json:"providerName,omitempty"`
}

func (request BatchEnginePollRequest) Validate() error {
	if err := servicecommon.ValidateRequiredObjectID("batchJobId", request.BatchJobID); err != nil {
		return err
	}
	return servicecommon.ValidateRequiredText("providerJobHandle", request.ProviderJobHandle)
}

type BatchEnginePollResult struct {
	BatchJobID        primitive.ObjectID            `json:"batchJobId"`
	Status            domaincommon.AIBatchJobStatus `json:"status"`
	ResultAvailable   bool                          `json:"resultAvailable,omitempty"`
	Retryable         bool                          `json:"retryable,omitempty"`
	LastPolledAt      *time.Time                    `json:"lastPolledAt,omitempty"`
	CompletedAt       *time.Time                    `json:"completedAt,omitempty"`
	RawProviderStatus map[string]any                `json:"rawProviderStatus,omitempty"`
	ItemStatuses      []BatchEngineItemStatus       `json:"itemStatuses,omitempty"`
}

type BatchEngineItemStatus struct {
	BatchItemID    primitive.ObjectID             `json:"batchItemId,omitempty"`
	CorrelationID  string                         `json:"correlationId,omitempty"`
	Status         domaincommon.AIBatchItemStatus `json:"status"`
	ErrorSummary   string                         `json:"errorSummary,omitempty"`
	ProviderHandle string                         `json:"providerHandle,omitempty"`
	Metadata       map[string]any                 `json:"metadata,omitempty"`
}

type BatchEngineFetchResultsRequest struct {
	BatchJobID        primitive.ObjectID `json:"batchJobId"`
	ProviderJobHandle string             `json:"providerJobHandle"`
	ProviderName      string             `json:"providerName,omitempty"`
}

func (request BatchEngineFetchResultsRequest) Validate() error {
	if err := servicecommon.ValidateRequiredObjectID("batchJobId", request.BatchJobID); err != nil {
		return err
	}
	return servicecommon.ValidateRequiredText("providerJobHandle", request.ProviderJobHandle)
}

type BatchEngineFetchResultsResult struct {
	BatchJobID       primitive.ObjectID             `json:"batchJobId"`
	Status           domaincommon.AIBatchJobStatus  `json:"status"`
	ResultPayloadRef *domaincommon.PayloadReference `json:"resultPayloadRef,omitempty"`
	CompletedAt      *time.Time                     `json:"completedAt,omitempty"`
	RawPayload       map[string]any                 `json:"rawPayload,omitempty"`
	Items            []BatchEngineResultItem        `json:"items,omitempty"`
}

type BatchEngineResultItem struct {
	BatchItemID      primitive.ObjectID             `json:"batchItemId,omitempty"`
	CorrelationID    string                         `json:"correlationId,omitempty"`
	Status           domaincommon.AIBatchItemStatus `json:"status"`
	OutputPayload    map[string]any                 `json:"outputPayload,omitempty"`
	ErrorSummary     string                         `json:"errorSummary,omitempty"`
	Retryable        bool                           `json:"retryable,omitempty"`
	ProviderMetadata map[string]any                 `json:"providerMetadata,omitempty"`
}
