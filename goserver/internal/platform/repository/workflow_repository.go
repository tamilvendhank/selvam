package repository

import (
	"context"
	"time"

	"goserver/internal/domain/common"
	"goserver/internal/domain/workflow"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

// WorkflowRunSortBy enumerates supported workflow-run sort fields.
type WorkflowRunSortBy string

const (
	WorkflowRunSortByStartedAt   WorkflowRunSortBy = "started_at"
	WorkflowRunSortByCompletedAt WorkflowRunSortBy = "completed_at"
	WorkflowRunSortByCreatedAt   WorkflowRunSortBy = "created_at"
	WorkflowRunSortByUpdatedAt   WorkflowRunSortBy = "updated_at"
)

// WorkflowRunSortOption controls workflow-run ordering.
type WorkflowRunSortOption struct {
	By    WorkflowRunSortBy
	Order SortOrder
}

// WorkflowRunFilter captures repository query criteria for workflow runs.
type WorkflowRunFilter struct {
	IDs               []primitive.ObjectID
	BookTypes         []common.BookType
	RunTypes          []common.WorkflowRunType
	Statuses          []common.WorkflowRunStatus
	ConfigSnapshotIDs []primitive.ObjectID
	StartedAt         *TimeRange
	CompletedAt       *TimeRange
	CreatedAt         *TimeRange
	UpdatedAt         *TimeRange
	ActiveOnly        bool
	TerminalOnly      bool
}

// WorkflowRunListOptions configures workflow-run pagination and sort behavior.
type WorkflowRunListOptions struct {
	Pagination PageOptions
	Sort       WorkflowRunSortOption
}

// WorkflowRunStatusPatch performs a guarded workflow-run status transition.
type WorkflowRunStatusPatch struct {
	NextStatus common.WorkflowRunStatus
	Notes      *string

	ExpectedCurrentStatuses []common.WorkflowRunStatus
	Mutation                MutationMetadata
}

// WorkflowRunProgressPatch increments persisted workflow counters without requiring a full-document rewrite.
type WorkflowRunProgressPatch struct {
	CompaniesScannedDelta int
	ReviewsCreatedDelta   int
	ErrorsDelta           int
	Notes                 *string

	ExpectedCurrentStatuses []common.WorkflowRunStatus
	Mutation                MutationMetadata
}

// WorkflowRunCompletionPatch marks a workflow run as completed.
type WorkflowRunCompletionPatch struct {
	CompletedAt time.Time
	Notes       *string

	ExpectedCurrentStatuses []common.WorkflowRunStatus
	Mutation                MutationMetadata
}

// WorkflowRunFailurePatch marks a workflow run as failed.
type WorkflowRunFailurePatch struct {
	FailedAt      time.Time
	FailureReason string
	Notes         *string

	ExpectedCurrentStatuses []common.WorkflowRunStatus
	Mutation                MutationMetadata
}

// WorkflowRunRepository stores orchestration-level workflow state and resumability metadata.
type WorkflowRunRepository interface {
	Create(ctx context.Context, run *workflow.WorkflowRun) (*workflow.WorkflowRun, error)
	GetByID(ctx context.Context, id primitive.ObjectID) (*workflow.WorkflowRun, error)
	List(ctx context.Context, filter WorkflowRunFilter, options WorkflowRunListOptions) (*ListResult[*workflow.WorkflowRun], error)
	FindResumable(ctx context.Context, filter WorkflowRunFilter, options WorkflowRunListOptions) (*ListResult[*workflow.WorkflowRun], error)
	UpdateStatus(ctx context.Context, workflowRunID primitive.ObjectID, patch WorkflowRunStatusPatch) (*workflow.WorkflowRun, error)
	UpdateProgressCounters(ctx context.Context, workflowRunID primitive.ObjectID, patch WorkflowRunProgressPatch) (*workflow.WorkflowRun, error)
	MarkCompleted(ctx context.Context, workflowRunID primitive.ObjectID, patch WorkflowRunCompletionPatch) (*workflow.WorkflowRun, error)
	MarkFailed(ctx context.Context, workflowRunID primitive.ObjectID, patch WorkflowRunFailurePatch) (*workflow.WorkflowRun, error)
}

// WorkflowStepRunSortBy enumerates supported workflow-step sort fields.
type WorkflowStepRunSortBy string

const (
	WorkflowStepRunSortByStepName    WorkflowStepRunSortBy = "step_name"
	WorkflowStepRunSortByCreatedAt   WorkflowStepRunSortBy = "created_at"
	WorkflowStepRunSortByUpdatedAt   WorkflowStepRunSortBy = "updated_at"
	WorkflowStepRunSortByStartedAt   WorkflowStepRunSortBy = "started_at"
	WorkflowStepRunSortByCompletedAt WorkflowStepRunSortBy = "completed_at"
)

// WorkflowStepRunSortOption controls workflow-step ordering.
type WorkflowStepRunSortOption struct {
	By    WorkflowStepRunSortBy
	Order SortOrder
}

// WorkflowStepRunFilter captures repository query criteria for workflow step runs.
type WorkflowStepRunFilter struct {
	IDs            []primitive.ObjectID
	WorkflowRunIDs []primitive.ObjectID
	StepNames      []common.WorkflowStepName
	Statuses       []common.WorkflowStepStatus
	StartedAt      *TimeRange
	CompletedAt    *TimeRange
	CreatedAt      *TimeRange
	UpdatedAt      *TimeRange
	ActiveOnly     bool
	TerminalOnly   bool
}

// WorkflowStepRunListOptions configures workflow-step pagination and sort behavior.
type WorkflowStepRunListOptions struct {
	Pagination PageOptions
	Sort       WorkflowStepRunSortOption
}

// WorkflowStepStatusPatch performs a guarded workflow-step status transition.
type WorkflowStepStatusPatch struct {
	NextStatus   common.WorkflowStepStatus
	ErrorSummary *string

	ExpectedCurrentStatuses []common.WorkflowStepStatus
	Mutation                MutationMetadata
}

// WorkflowStepStartPatch marks a workflow step as started.
type WorkflowStepStartPatch struct {
	StartedAt time.Time
	Metadata  *MetadataPatch

	ExpectedCurrentStatuses []common.WorkflowStepStatus
	Mutation                MutationMetadata
}

// WorkflowStepCompletionPatch marks a workflow step as completed.
type WorkflowStepCompletionPatch struct {
	CompletedAt time.Time
	Metadata    *MetadataPatch

	ExpectedCurrentStatuses []common.WorkflowStepStatus
	Mutation                MutationMetadata
}

// WorkflowStepFailurePatch marks a workflow step as failed.
type WorkflowStepFailurePatch struct {
	FailedAt     time.Time
	ErrorSummary string
	Metadata     *MetadataPatch

	ExpectedCurrentStatuses []common.WorkflowStepStatus
	Mutation                MutationMetadata
}

// WorkflowStepSkipPatch marks a workflow step as skipped.
type WorkflowStepSkipPatch struct {
	SkippedAt time.Time
	Reason    string
	Metadata  *MetadataPatch

	ExpectedCurrentStatuses []common.WorkflowStepStatus
	Mutation                MutationMetadata
}

// WorkflowStepMetadataPatch updates step metadata without requiring a lifecycle change.
type WorkflowStepMetadataPatch struct {
	Metadata MetadataPatch

	ExpectedCurrentStatuses []common.WorkflowStepStatus
	Mutation                MutationMetadata
}

// WorkflowStepRunRepository stores per-step execution state for resumable workflow orchestration.
type WorkflowStepRunRepository interface {
	Create(ctx context.Context, stepRun *workflow.WorkflowStepRun) (*workflow.WorkflowStepRun, error)
	GetByID(ctx context.Context, id primitive.ObjectID) (*workflow.WorkflowStepRun, error)
	GetByWorkflowRunAndStepName(ctx context.Context, workflowRunID primitive.ObjectID, stepName common.WorkflowStepName) (*workflow.WorkflowStepRun, error)
	ListByWorkflowRunID(ctx context.Context, workflowRunID primitive.ObjectID, options WorkflowStepRunListOptions) (*ListResult[*workflow.WorkflowStepRun], error)
	List(ctx context.Context, filter WorkflowStepRunFilter, options WorkflowStepRunListOptions) (*ListResult[*workflow.WorkflowStepRun], error)
	UpdateStatus(ctx context.Context, stepRunID primitive.ObjectID, patch WorkflowStepStatusPatch) (*workflow.WorkflowStepRun, error)
	MarkStarted(ctx context.Context, stepRunID primitive.ObjectID, patch WorkflowStepStartPatch) (*workflow.WorkflowStepRun, error)
	MarkCompleted(ctx context.Context, stepRunID primitive.ObjectID, patch WorkflowStepCompletionPatch) (*workflow.WorkflowStepRun, error)
	MarkFailed(ctx context.Context, stepRunID primitive.ObjectID, patch WorkflowStepFailurePatch) (*workflow.WorkflowStepRun, error)
	MarkSkipped(ctx context.Context, stepRunID primitive.ObjectID, patch WorkflowStepSkipPatch) (*workflow.WorkflowStepRun, error)
	UpdateMetadata(ctx context.Context, stepRunID primitive.ObjectID, patch WorkflowStepMetadataPatch) (*workflow.WorkflowStepRun, error)
}
