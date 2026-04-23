package repository

import (
	"context"
	"time"

	"goserver/internal/domain/aijob"
	"goserver/internal/domain/common"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

// AIBatchJobSortBy enumerates supported AI batch-job sort fields.
type AIBatchJobSortBy string

const (
	AIBatchJobSortBySubmittedAt  AIBatchJobSortBy = "submitted_at"
	AIBatchJobSortByLastPolledAt AIBatchJobSortBy = "last_polled_at"
	AIBatchJobSortByCreatedAt    AIBatchJobSortBy = "created_at"
	AIBatchJobSortByUpdatedAt    AIBatchJobSortBy = "updated_at"
)

// AIBatchJobSortOption controls AI batch-job ordering.
type AIBatchJobSortOption struct {
	By    AIBatchJobSortBy
	Order SortOrder
}

// AIBatchJobFilter captures repository query criteria for AI batch jobs.
type AIBatchJobFilter struct {
	IDs               []primitive.ObjectID
	WorkflowRunIDs    []primitive.ObjectID
	JobTypes          []common.AIBatchJobType
	BookTypes         []common.BookType
	Statuses          []common.AIBatchJobStatus
	ProviderName      string
	ProviderJobHandle string
	LocalJobHandle    string
	IdempotencyKey    string
	SubmittedAt       *TimeRange
	LastPolledAt      *TimeRange
	CompletedAt       *TimeRange
	FailedAt          *TimeRange
	CreatedAt         *TimeRange
	UpdatedAt         *TimeRange
	RetryableOnly     bool
	PollableOnly      bool
}

// AIBatchJobListOptions configures AI batch-job pagination and sort behavior.
type AIBatchJobListOptions struct {
	Pagination PageOptions
	Sort       AIBatchJobSortOption
}

// AIBatchJobStatusPatch performs a guarded status transition on a batch job.
type AIBatchJobStatusPatch struct {
	NextStatus   common.AIBatchJobStatus
	ErrorSummary *string

	ExpectedCurrentStatuses []common.AIBatchJobStatus
	Mutation                MutationMetadata
}

// AIBatchJobSubmissionPatch captures provider handles and payload references returned at submission time.
type AIBatchJobSubmissionPatch struct {
	NewStatus            common.AIBatchJobStatus
	ProviderJobHandle    string
	LocalJobHandle       string
	SubmissionPayloadRef *common.PayloadReference
	SubmittedAt          time.Time

	ExpectedCurrentStatuses []common.AIBatchJobStatus
	Mutation                MutationMetadata
}

// AIBatchJobPollingPatch records polling activity and optional in-flight status changes.
type AIBatchJobPollingPatch struct {
	LastPolledAt time.Time
	NextStatus   *common.AIBatchJobStatus
	ErrorSummary *string

	ExpectedCurrentStatuses []common.AIBatchJobStatus
	Mutation                MutationMetadata
}

// AIBatchJobCompletionPatch captures terminal completion details for a batch job.
type AIBatchJobCompletionPatch struct {
	ResultPayloadRef *common.PayloadReference
	CompletedAt      time.Time

	ExpectedCurrentStatuses []common.AIBatchJobStatus
	Mutation                MutationMetadata
}

// AIBatchJobFailurePatch captures terminal failure details for a batch job.
type AIBatchJobFailurePatch struct {
	FailedAt     time.Time
	ErrorSummary string

	ExpectedCurrentStatuses []common.AIBatchJobStatus
	Mutation                MutationMetadata
}

// AIBatchJobRetryPatch prepares a failed/timed-out job for a retry attempt.
type AIBatchJobRetryPatch struct {
	RetryAt time.Time

	ExpectedCurrentStatuses []common.AIBatchJobStatus
	Mutation                MutationMetadata
}

// AIBatchJobRepository stores provider-facing batch jobs and their submission/polling lifecycle.
type AIBatchJobRepository interface {
	Create(ctx context.Context, job *aijob.AIBatchJob) (*aijob.AIBatchJob, error)
	GetByID(ctx context.Context, id primitive.ObjectID) (*aijob.AIBatchJob, error)
	GetByProviderJobHandle(ctx context.Context, providerName, providerJobHandle string) (*aijob.AIBatchJob, error)
	GetByIdempotencyKey(ctx context.Context, idempotencyKey string) (*aijob.AIBatchJob, error)
	List(ctx context.Context, filter AIBatchJobFilter, options AIBatchJobListOptions) (*ListResult[*aijob.AIBatchJob], error)
	ListByWorkflowRunID(ctx context.Context, workflowRunID primitive.ObjectID, options AIBatchJobListOptions) (*ListResult[*aijob.AIBatchJob], error)
	FindPollableJobs(ctx context.Context, filter AIBatchJobFilter, options AIBatchJobListOptions) (*ListResult[*aijob.AIBatchJob], error)
	FindSubmittableJobs(ctx context.Context, filter AIBatchJobFilter, options AIBatchJobListOptions) (*ListResult[*aijob.AIBatchJob], error)
	UpdateStatus(ctx context.Context, jobID primitive.ObjectID, patch AIBatchJobStatusPatch) (*aijob.AIBatchJob, error)
	MarkSubmitted(ctx context.Context, jobID primitive.ObjectID, patch AIBatchJobSubmissionPatch) (*aijob.AIBatchJob, error)
	MarkPolled(ctx context.Context, jobID primitive.ObjectID, patch AIBatchJobPollingPatch) (*aijob.AIBatchJob, error)
	MarkCompleted(ctx context.Context, jobID primitive.ObjectID, patch AIBatchJobCompletionPatch) (*aijob.AIBatchJob, error)
	MarkFailed(ctx context.Context, jobID primitive.ObjectID, patch AIBatchJobFailurePatch) (*aijob.AIBatchJob, error)
	MarkTimedOut(ctx context.Context, jobID primitive.ObjectID, patch AIBatchJobFailurePatch) (*aijob.AIBatchJob, error)
	// PrepareRetry should increment retry visibility and return the job to a submittable state atomically.
	PrepareRetry(ctx context.Context, jobID primitive.ObjectID, patch AIBatchJobRetryPatch) (*aijob.AIBatchJob, error)
}

// AIBatchItemSortBy enumerates supported AI batch-item sort fields.
type AIBatchItemSortBy string

const (
	AIBatchItemSortByCreatedAt   AIBatchItemSortBy = "created_at"
	AIBatchItemSortByUpdatedAt   AIBatchItemSortBy = "updated_at"
	AIBatchItemSortByCompletedAt AIBatchItemSortBy = "completed_at"
	AIBatchItemSortBySymbol      AIBatchItemSortBy = "symbol"
)

// AIBatchItemSortOption controls AI batch-item ordering.
type AIBatchItemSortOption struct {
	By    AIBatchItemSortBy
	Order SortOrder
}

// AIBatchItemFilter captures repository query criteria for AI batch items.
type AIBatchItemFilter struct {
	IDs                   []primitive.ObjectID
	AIBatchJobIDs         []primitive.ObjectID
	WorkflowRunIDs        []primitive.ObjectID
	CompanyIDs            []primitive.ObjectID
	Symbols               []string
	BookTypes             []common.BookType
	ItemTypes             []common.AIBatchItemType
	Statuses              []common.AIBatchItemStatus
	ValidationStatuses    []common.ValidationStatus
	TargetReviewIDs       []primitive.ObjectID
	TargetThesisIDs       []primitive.ObjectID
	CreatedAt             *TimeRange
	UpdatedAt             *TimeRange
	CompletedAt           *TimeRange
	PendingValidationOnly bool
	RetryableOnly         bool
}

// AIBatchItemListOptions configures AI batch-item pagination and sort behavior.
type AIBatchItemListOptions struct {
	Pagination PageOptions
	Sort       AIBatchItemSortOption
}

// AIBatchItemStatusPatch performs a guarded status transition on a batch item.
type AIBatchItemStatusPatch struct {
	NextStatus   common.AIBatchItemStatus
	ErrorSummary *string

	ExpectedCurrentStatuses []common.AIBatchItemStatus
	Mutation                MutationMetadata
}

// AIBatchItemResultPatch stores the raw provider output for a batch item.
type AIBatchItemResultPatch struct {
	ResultPayload map[string]any
	NextStatus    *common.AIBatchItemStatus
	ErrorSummary  *string

	ExpectedCurrentStatuses []common.AIBatchItemStatus
	Mutation                MutationMetadata
}

// AIBatchItemValidationPatch stores structured validation outcomes for a completed item.
type AIBatchItemValidationPatch struct {
	ValidationStatus    common.ValidationStatus
	ValidationErrors    []string
	TargetReviewID      *primitive.ObjectID
	TargetThesisID      *primitive.ObjectID
	TargetEntityVersion *int

	ExpectedCurrentStatuses []common.AIBatchItemStatus
	Mutation                MutationMetadata
}

// AIBatchItemCompletionPatch marks an item as successfully completed.
type AIBatchItemCompletionPatch struct {
	CompletedAt time.Time

	ExpectedCurrentStatuses []common.AIBatchItemStatus
	Mutation                MutationMetadata
}

// AIBatchItemFailurePatch marks an item as failed.
type AIBatchItemFailurePatch struct {
	FailedAt     time.Time
	ErrorSummary string

	ExpectedCurrentStatuses []common.AIBatchItemStatus
	Mutation                MutationMetadata
}

// AIBatchItemInvalidOutputPatch marks an item as invalid due to output that could not be validated.
type AIBatchItemInvalidOutputPatch struct {
	InvalidAt        time.Time
	ErrorSummary     string
	ValidationErrors []string

	ExpectedCurrentStatuses []common.AIBatchItemStatus
	Mutation                MutationMetadata
}

// AIBatchItemSkipPatch marks an item as intentionally skipped.
type AIBatchItemSkipPatch struct {
	SkippedAt time.Time
	Reason    string

	ExpectedCurrentStatuses []common.AIBatchItemStatus
	Mutation                MutationMetadata
}

// AIBatchItemRetryPatch prepares a failed or invalid item for a retry attempt.
type AIBatchItemRetryPatch struct {
	RetryAt time.Time

	ExpectedCurrentStatuses []common.AIBatchItemStatus
	Mutation                MutationMetadata
}

// AIBatchItemRepository stores per-company/per-task batch item execution state.
// Implementations should support partial completion and item-level retries independently of job-level state.
type AIBatchItemRepository interface {
	Create(ctx context.Context, item *aijob.AIBatchItem) (*aijob.AIBatchItem, error)
	CreateMany(ctx context.Context, items []*aijob.AIBatchItem) ([]*aijob.AIBatchItem, error)
	GetByID(ctx context.Context, id primitive.ObjectID) (*aijob.AIBatchItem, error)
	ListByBatchJobID(ctx context.Context, batchJobID primitive.ObjectID, options AIBatchItemListOptions) (*ListResult[*aijob.AIBatchItem], error)
	List(ctx context.Context, filter AIBatchItemFilter, options AIBatchItemListOptions) (*ListResult[*aijob.AIBatchItem], error)
	FindPendingValidation(ctx context.Context, filter AIBatchItemFilter, options AIBatchItemListOptions) (*ListResult[*aijob.AIBatchItem], error)
	FindRetryableItems(ctx context.Context, filter AIBatchItemFilter, options AIBatchItemListOptions) (*ListResult[*aijob.AIBatchItem], error)
	UpdateStatus(ctx context.Context, itemID primitive.ObjectID, patch AIBatchItemStatusPatch) (*aijob.AIBatchItem, error)
	SaveResultPayload(ctx context.Context, itemID primitive.ObjectID, patch AIBatchItemResultPatch) (*aijob.AIBatchItem, error)
	SaveValidationResult(ctx context.Context, itemID primitive.ObjectID, patch AIBatchItemValidationPatch) (*aijob.AIBatchItem, error)
	MarkCompleted(ctx context.Context, itemID primitive.ObjectID, patch AIBatchItemCompletionPatch) (*aijob.AIBatchItem, error)
	MarkFailed(ctx context.Context, itemID primitive.ObjectID, patch AIBatchItemFailurePatch) (*aijob.AIBatchItem, error)
	MarkInvalidOutput(ctx context.Context, itemID primitive.ObjectID, patch AIBatchItemInvalidOutputPatch) (*aijob.AIBatchItem, error)
	MarkSkipped(ctx context.Context, itemID primitive.ObjectID, patch AIBatchItemSkipPatch) (*aijob.AIBatchItem, error)
	PrepareRetry(ctx context.Context, itemID primitive.ObjectID, patch AIBatchItemRetryPatch) (*aijob.AIBatchItem, error)
}
