package worker

import (
	"time"

	domaincommon "goserver/internal/domain/common"
	servicecommon "goserver/internal/service/common"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

type WorkerKind string

const (
	WorkerKindSubmission      WorkerKind = "submission"
	WorkerKindPolling         WorkerKind = "polling"
	WorkerKindReconciliation  WorkerKind = "reconciliation"
	WorkerKindValidation      WorkerKind = "validation"
	WorkerKindMaterialization WorkerKind = "materialization"
	WorkerKindFinalization    WorkerKind = "finalization"
	WorkerKindContinuation    WorkerKind = "continuation"
	WorkerKindProjection      WorkerKind = "projection"
)

type DiscoveryRequestBase struct {
	WorkflowRunID  primitive.ObjectID           `json:"workflowRunId,omitempty"`
	BookType       domaincommon.BookType        `json:"bookType,omitempty"`
	JobType        domaincommon.AIBatchJobType  `json:"jobType,omitempty"`
	ItemType       domaincommon.AIBatchItemType `json:"itemType,omitempty"`
	MaxItems       int                          `json:"maxItems,omitempty"`
	IncludeClaimed bool                         `json:"includeClaimed,omitempty"`
	WorkerID       string                       `json:"workerId,omitempty"`
	CorrelationID  string                       `json:"correlationId,omitempty"`
}

func (request DiscoveryRequestBase) Validate() error {
	if err := servicecommon.ValidateOptionalMax("maxItems", request.MaxItems); err != nil {
		return err
	}
	if err := servicecommon.ValidateOptionalBookType(request.BookType); err != nil {
		return err
	}
	if err := servicecommon.ValidateOptionalJobType(request.JobType); err != nil {
		return err
	}
	if err := servicecommon.ValidateOptionalItemType(request.ItemType); err != nil {
		return err
	}
	if err := servicecommon.ValidateOptionalText("workerId", request.WorkerID); err != nil {
		return err
	}
	return servicecommon.ValidateOptionalText("correlationId", request.CorrelationID)
}

type DiscoverSubmittableBatchJobsRequest struct {
	DiscoveryRequestBase
}

func (request DiscoverSubmittableBatchJobsRequest) Validate() error {
	return request.DiscoveryRequestBase.Validate()
}

type DiscoverSubmittableBatchJobsResult struct {
	BatchJobs []servicecommon.BatchJobRef          `json:"batchJobs,omitempty"`
	WorkItems []servicecommon.WorkItemRef          `json:"workItems,omitempty"`
	HasMore   bool                                 `json:"hasMore,omitempty"`
	Summary   servicecommon.WorkerOperationSummary `json:"summary,omitempty"`
}

func (result DiscoverSubmittableBatchJobsResult) HasWork() bool {
	return len(result.BatchJobs) > 0 || len(result.WorkItems) > 0
}

type DiscoverPollableBatchJobsRequest struct {
	DiscoveryRequestBase
	PollOnlyStatuses []domaincommon.AIBatchJobStatus `json:"pollOnlyStatuses,omitempty"`
}

func (request DiscoverPollableBatchJobsRequest) Validate() error {
	if err := request.DiscoveryRequestBase.Validate(); err != nil {
		return err
	}
	return servicecommon.ValidateOptionalBatchJobStatuses(request.PollOnlyStatuses)
}

type DiscoverPollableBatchJobsResult struct {
	BatchJobs []servicecommon.BatchJobRef          `json:"batchJobs,omitempty"`
	WorkItems []servicecommon.WorkItemRef          `json:"workItems,omitempty"`
	HasMore   bool                                 `json:"hasMore,omitempty"`
	Summary   servicecommon.WorkerOperationSummary `json:"summary,omitempty"`
}

func (result DiscoverPollableBatchJobsResult) HasWork() bool {
	return len(result.BatchJobs) > 0 || len(result.WorkItems) > 0
}

type DiscoverReconciliableBatchJobsRequest struct {
	DiscoveryRequestBase
	CompletedOnly bool `json:"completedOnly,omitempty"`
}

func (request DiscoverReconciliableBatchJobsRequest) Validate() error {
	return request.DiscoveryRequestBase.Validate()
}

type DiscoverReconciliableBatchJobsResult struct {
	BatchJobs []servicecommon.BatchJobRef          `json:"batchJobs,omitempty"`
	WorkItems []servicecommon.WorkItemRef          `json:"workItems,omitempty"`
	HasMore   bool                                 `json:"hasMore,omitempty"`
	Summary   servicecommon.WorkerOperationSummary `json:"summary,omitempty"`
}

func (result DiscoverReconciliableBatchJobsResult) HasWork() bool {
	return len(result.BatchJobs) > 0 || len(result.WorkItems) > 0
}

type DiscoverValidatableItemsRequest struct {
	DiscoveryRequestBase
	StrictMode bool `json:"strictMode,omitempty"`
	Revalidate bool `json:"revalidate,omitempty"`
}

func (request DiscoverValidatableItemsRequest) Validate() error {
	return request.DiscoveryRequestBase.Validate()
}

type DiscoverValidatableItemsResult struct {
	BatchItems []servicecommon.BatchItemRef         `json:"batchItems,omitempty"`
	WorkItems  []servicecommon.WorkItemRef          `json:"workItems,omitempty"`
	HasMore    bool                                 `json:"hasMore,omitempty"`
	Summary    servicecommon.WorkerOperationSummary `json:"summary,omitempty"`
}

func (result DiscoverValidatableItemsResult) HasWork() bool {
	return len(result.BatchItems) > 0 || len(result.WorkItems) > 0
}

type DiscoverMaterializableReviewsRequest struct {
	DiscoveryRequestBase
	Force bool `json:"force,omitempty"`
}

func (request DiscoverMaterializableReviewsRequest) Validate() error {
	return request.DiscoveryRequestBase.Validate()
}

type DiscoverMaterializableReviewsResult struct {
	Reviews    []servicecommon.ReviewRef            `json:"reviews,omitempty"`
	BatchItems []servicecommon.BatchItemRef         `json:"batchItems,omitempty"`
	WorkItems  []servicecommon.WorkItemRef          `json:"workItems,omitempty"`
	HasMore    bool                                 `json:"hasMore,omitempty"`
	Summary    servicecommon.WorkerOperationSummary `json:"summary,omitempty"`
}

func (result DiscoverMaterializableReviewsResult) HasWork() bool {
	return len(result.Reviews) > 0 || len(result.BatchItems) > 0 || len(result.WorkItems) > 0
}

type DiscoverFinalizableReviewsRequest struct {
	DiscoveryRequestBase
	CompanyID primitive.ObjectID `json:"companyId,omitempty"`
	Force     bool               `json:"force,omitempty"`
}

func (request DiscoverFinalizableReviewsRequest) Validate() error {
	return request.DiscoveryRequestBase.Validate()
}

type DiscoverFinalizableReviewsResult struct {
	Reviews   []servicecommon.ReviewRef            `json:"reviews,omitempty"`
	WorkItems []servicecommon.WorkItemRef          `json:"workItems,omitempty"`
	HasMore   bool                                 `json:"hasMore,omitempty"`
	Summary   servicecommon.WorkerOperationSummary `json:"summary,omitempty"`
}

func (result DiscoverFinalizableReviewsResult) HasWork() bool {
	return len(result.Reviews) > 0 || len(result.WorkItems) > 0
}

type DiscoverContinuableWorkflowsRequest struct {
	DiscoveryRequestBase
	Force bool `json:"force,omitempty"`
}

func (request DiscoverContinuableWorkflowsRequest) Validate() error {
	return request.DiscoveryRequestBase.Validate()
}

type DiscoverContinuableWorkflowsResult struct {
	Continuations []servicecommon.ContinuationRef      `json:"continuations,omitempty"`
	WorkItems     []servicecommon.WorkItemRef          `json:"workItems,omitempty"`
	HasMore       bool                                 `json:"hasMore,omitempty"`
	Summary       servicecommon.WorkerOperationSummary `json:"summary,omitempty"`
}

func (result DiscoverContinuableWorkflowsResult) HasWork() bool {
	return len(result.Continuations) > 0 || len(result.WorkItems) > 0
}

type WorkLeaseState string

const (
	WorkLeaseStateClaimed   WorkLeaseState = "claimed"
	WorkLeaseStateReleased  WorkLeaseState = "released"
	WorkLeaseStateCompleted WorkLeaseState = "completed"
	WorkLeaseStateFailed    WorkLeaseState = "failed"
	WorkLeaseStateExpired   WorkLeaseState = "expired"
)

type WorkLeaseRef struct {
	LeaseID    string                    `json:"leaseId"`
	WorkerID   string                    `json:"workerId"`
	WorkerKind WorkerKind                `json:"workerKind,omitempty"`
	State      WorkLeaseState            `json:"state"`
	WorkItem   servicecommon.WorkItemRef `json:"workItem"`
	ClaimedAt  time.Time                 `json:"claimedAt"`
	ExpiresAt  time.Time                 `json:"expiresAt"`
}

func (lease WorkLeaseRef) IsTerminal() bool {
	return lease.State == WorkLeaseStateReleased ||
		lease.State == WorkLeaseStateCompleted ||
		lease.State == WorkLeaseStateFailed ||
		lease.State == WorkLeaseStateExpired
}

type ClaimWorkRequest struct {
	WorkerID      string                      `json:"workerId"`
	WorkerKind    WorkerKind                  `json:"workerKind"`
	WorkItems     []servicecommon.WorkItemRef `json:"workItems,omitempty"`
	MaxItems      int                         `json:"maxItems,omitempty"`
	LeaseDuration time.Duration               `json:"leaseDuration,omitempty"`
	CorrelationID string                      `json:"correlationId,omitempty"`
}

func (request ClaimWorkRequest) Validate() error {
	if err := servicecommon.ValidateRequiredText("workerId", request.WorkerID); err != nil {
		return err
	}
	if err := servicecommon.ValidateRequiredText("workerKind", string(request.WorkerKind)); err != nil {
		return err
	}
	if err := servicecommon.ValidateOptionalMax("maxItems", request.MaxItems); err != nil {
		return err
	}
	if request.LeaseDuration < 0 {
		return servicecommon.ValidateOptionalMax("leaseDuration", -1)
	}
	return servicecommon.ValidateOptionalText("correlationId", request.CorrelationID)
}

type ClaimWorkResult struct {
	Leases           []WorkLeaseRef                       `json:"leases,omitempty"`
	ClaimedWorkItems []servicecommon.WorkItemRef          `json:"claimedWorkItems,omitempty"`
	SkippedWorkItems []servicecommon.WorkItemRef          `json:"skippedWorkItems,omitempty"`
	PartialFailures  []servicecommon.PartialFailure       `json:"partialFailures,omitempty"`
	Summary          servicecommon.WorkerOperationSummary `json:"summary,omitempty"`
}

type HeartbeatWorkRequest struct {
	LeaseID       string        `json:"leaseId"`
	WorkerID      string        `json:"workerId"`
	ExtendBy      time.Duration `json:"extendBy,omitempty"`
	CorrelationID string        `json:"correlationId,omitempty"`
}

func (request HeartbeatWorkRequest) Validate() error {
	if err := servicecommon.ValidateRequiredText("leaseId", request.LeaseID); err != nil {
		return err
	}
	if err := servicecommon.ValidateRequiredText("workerId", request.WorkerID); err != nil {
		return err
	}
	if request.ExtendBy < 0 {
		return servicecommon.ValidateOptionalMax("extendBy", -1)
	}
	return servicecommon.ValidateOptionalText("correlationId", request.CorrelationID)
}

type HeartbeatWorkResult struct {
	Lease   WorkLeaseRef                         `json:"lease"`
	Summary servicecommon.WorkerOperationSummary `json:"summary,omitempty"`
}

type ReleaseWorkRequest struct {
	LeaseID       string `json:"leaseId"`
	WorkerID      string `json:"workerId"`
	Reason        string `json:"reason,omitempty"`
	CorrelationID string `json:"correlationId,omitempty"`
}

func (request ReleaseWorkRequest) Validate() error {
	if err := servicecommon.ValidateRequiredText("leaseId", request.LeaseID); err != nil {
		return err
	}
	if err := servicecommon.ValidateRequiredText("workerId", request.WorkerID); err != nil {
		return err
	}
	return servicecommon.ValidateOptionalText("correlationId", request.CorrelationID)
}

type ReleaseWorkResult struct {
	Lease   WorkLeaseRef                         `json:"lease"`
	Summary servicecommon.WorkerOperationSummary `json:"summary,omitempty"`
}

type CompleteWorkRequest struct {
	LeaseID       string                       `json:"leaseId"`
	WorkerID      string                       `json:"workerId"`
	Outcome       servicecommon.ServiceOutcome `json:"outcome,omitempty"`
	CorrelationID string                       `json:"correlationId,omitempty"`
}

func (request CompleteWorkRequest) Validate() error {
	if err := servicecommon.ValidateRequiredText("leaseId", request.LeaseID); err != nil {
		return err
	}
	if err := servicecommon.ValidateRequiredText("workerId", request.WorkerID); err != nil {
		return err
	}
	return servicecommon.ValidateOptionalText("correlationId", request.CorrelationID)
}

type CompleteWorkResult struct {
	Lease   WorkLeaseRef                         `json:"lease"`
	Summary servicecommon.WorkerOperationSummary `json:"summary,omitempty"`
}

type FailWorkRequest struct {
	LeaseID       string                       `json:"leaseId"`
	WorkerID      string                       `json:"workerId"`
	Failure       servicecommon.PartialFailure `json:"failure"`
	RetryDecision servicecommon.RetryDecision  `json:"retryDecision,omitempty"`
	CorrelationID string                       `json:"correlationId,omitempty"`
}

func (request FailWorkRequest) Validate() error {
	if err := servicecommon.ValidateRequiredText("leaseId", request.LeaseID); err != nil {
		return err
	}
	if err := servicecommon.ValidateRequiredText("workerId", request.WorkerID); err != nil {
		return err
	}
	return servicecommon.ValidateOptionalText("correlationId", request.CorrelationID)
}

type FailWorkResult struct {
	Lease   WorkLeaseRef                         `json:"lease"`
	Retry   servicecommon.RetryDecision          `json:"retry,omitempty"`
	Summary servicecommon.WorkerOperationSummary `json:"summary,omitempty"`
}
