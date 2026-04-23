package repository

import (
	"context"
	"time"

	"goserver/internal/domain/common"
	"goserver/internal/domain/review"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

// CompanyReviewSortBy enumerates supported review list sort fields.
type CompanyReviewSortBy string

const (
	CompanyReviewSortByReviewDate  CompanyReviewSortBy = "review_date"
	CompanyReviewSortByCreatedAt   CompanyReviewSortBy = "created_at"
	CompanyReviewSortByUpdatedAt   CompanyReviewSortBy = "updated_at"
	CompanyReviewSortByFinalizedAt CompanyReviewSortBy = "finalized_at"
	CompanyReviewSortByScore       CompanyReviewSortBy = "weighted_total_score"
	CompanyReviewSortBySymbol      CompanyReviewSortBy = "symbol"
)

// CompanyReviewSortOption controls company review ordering.
type CompanyReviewSortOption struct {
	By    CompanyReviewSortBy
	Order SortOrder
}

// CompanyReviewFilter captures list/query constraints for review shells and finalized review history.
type CompanyReviewFilter struct {
	IDs               []primitive.ObjectID
	CompanyIDs        []primitive.ObjectID
	Symbols           []string
	BookTypes         []common.BookType
	WorkflowRunIDs    []primitive.ObjectID
	ReviewStatuses    []common.ReviewStatus
	LifecycleStates   []common.ReviewLifecycleState
	FinalActions      []common.InvestingActionType
	FinalBuckets      []common.WatchlistBucket
	OwnedBeforeReview *bool
	ReviewerTypes     []common.ReviewerType
	ReviewDate        *TimeRange
	CreatedAt         *TimeRange
	UpdatedAt         *TimeRange
	FinalizedAt       *TimeRange

	IncludeSuperseded bool
	FinalizedOnly     bool
	PendingOnly       bool
}

// CompanyReviewListOptions configures review list pagination and sort behavior.
type CompanyReviewListOptions struct {
	Pagination PageOptions
	Sort       CompanyReviewSortOption
}

// LatestCompanyReviewOptions controls "latest review" lookup behavior.
type LatestCompanyReviewOptions struct {
	FinalizedOnly     bool
	IncludeSuperseded bool
}

// PreviousFinalizedReviewLookup constrains previous finalized-review lookups for change detection.
type PreviousFinalizedReviewLookup struct {
	ExcludeReviewID   primitive.ObjectID
	BeforeReviewDate  *time.Time
	BeforeFinalizedAt *time.Time
	IncludeSuperseded bool
}

// CompanyReviewSummary is a UI-friendly projection for review tables and workflow dashboards.
type CompanyReviewSummary struct {
	ID                     primitive.ObjectID          `json:"id"`
	CompanyID              primitive.ObjectID          `json:"companyId"`
	Symbol                 string                      `json:"symbol"`
	BookType               common.BookType             `json:"bookType"`
	WorkflowRunID          primitive.ObjectID          `json:"workflowRunId,omitempty"`
	ReviewDate             time.Time                   `json:"reviewDate"`
	ReviewStatus           common.ReviewStatus         `json:"reviewStatus"`
	ReviewLifecycleState   common.ReviewLifecycleState `json:"reviewLifecycleState"`
	WeightedTotalScore     float64                     `json:"weightedTotalScore,omitempty"`
	FinalActionAfterReview common.InvestingActionType  `json:"finalActionAfterReview,omitempty"`
	FinalBucketAfterReview common.WatchlistBucket      `json:"finalBucketAfterReview,omitempty"`
	ReviewerType           common.ReviewerType         `json:"reviewerType"`
	UpdatedAt              time.Time                   `json:"updatedAt"`
	FinalizedAt            *time.Time                  `json:"finalizedAt,omitempty"`
}

// ReviewLifecycleUpdatePatch performs a guarded lifecycle transition on an existing review shell.
type ReviewLifecycleUpdatePatch struct {
	NextLifecycleState common.ReviewLifecycleState

	ExpectedCurrentLifecycleStates []common.ReviewLifecycleState
	ExpectedCurrentStatuses        []common.ReviewStatus
	Mutation                       MutationMetadata
}

// ReviewAIResultPatch attaches AI output references and related model metadata to a mutable review shell.
type ReviewAIResultPatch struct {
	RawAIResultRef  *common.PayloadReference
	AIModelName     *string
	AIPromptVersion *string
	ReviewMetadata  *MetadataPatch

	ExpectedCurrentLifecycleStates []common.ReviewLifecycleState
	ExpectedCurrentStatuses        []common.ReviewStatus
	Mutation                       MutationMetadata
}

// ReviewValidatedContentPatch replaces the validated review payload before finalization.
type ReviewValidatedContentPatch struct {
	Sections               []review.SectionScore
	DecisionAction         *review.DecisionAction
	PositionSnapshot       *review.PositionSnapshot
	ChangeLog              *review.ReviewChangeLog
	WeightedTotalScore     float64
	HardGateFailed         bool
	HardGateFailureReasons []string
	ConfidenceScore        float64
	FinalBucketAfterReview common.WatchlistBucket
	FinalActionAfterReview common.InvestingActionType
	ActionRationaleSummary string
	WhatChangedSummary     string
	ReviewerType           *common.ReviewerType
	ReviewMetadata         *MetadataPatch

	ExpectedCurrentLifecycleStates []common.ReviewLifecycleState
	ExpectedCurrentStatuses        []common.ReviewStatus
	Mutation                       MutationMetadata
}

// ReviewFinalizationPatch carries preconditions and audit context for the terminal finalization step.
type ReviewFinalizationPatch struct {
	FinalizedAt time.Time

	ExpectedCurrentLifecycleStates []common.ReviewLifecycleState
	ExpectedCurrentStatuses        []common.ReviewStatus
	FinalizedBy                    string
	Reason                         string
}

// ReviewSupersedePatch carries preconditions and audit context for superseding a historical final review.
type ReviewSupersedePatch struct {
	SupersededAt      time.Time
	ReplacementReview primitive.ObjectID

	ExpectedCurrentLifecycleStates []common.ReviewLifecycleState
	ExpectedCurrentStatuses        []common.ReviewStatus
	SupersededBy                   string
	Reason                         string
}

// CompanyReviewRepository stores review shells during async production and finalized immutable review history.
// Implementations should reject unsafe overwrites once a review reaches finalized or superseded lifecycle states.
type CompanyReviewRepository interface {
	CreateShell(ctx context.Context, review *review.CompanyReview) (*review.CompanyReview, error)
	GetByID(ctx context.Context, id primitive.ObjectID) (*review.CompanyReview, error)
	GetLatestByCompanyAndBook(ctx context.Context, companyID primitive.ObjectID, bookType common.BookType, options LatestCompanyReviewOptions) (*review.CompanyReview, error)
	GetPreviousFinalizedByCompanyAndBook(ctx context.Context, companyID primitive.ObjectID, bookType common.BookType, lookup PreviousFinalizedReviewLookup) (*review.CompanyReview, error)
	ListByCompany(ctx context.Context, companyID primitive.ObjectID, options CompanyReviewListOptions) (*ListResult[*review.CompanyReview], error)
	ListByWorkflowRun(ctx context.Context, workflowRunID primitive.ObjectID, options CompanyReviewListOptions) (*ListResult[*review.CompanyReview], error)
	ListPendingByLifecycleState(ctx context.Context, states []common.ReviewLifecycleState, options CompanyReviewListOptions) (*ListResult[*review.CompanyReview], error)
	List(ctx context.Context, filter CompanyReviewFilter, options CompanyReviewListOptions) (*ListResult[*review.CompanyReview], error)
	ListSummaries(ctx context.Context, filter CompanyReviewFilter, options CompanyReviewListOptions) (*ListResult[*CompanyReviewSummary], error)
	CountByWorkflowRun(ctx context.Context, workflowRunID primitive.ObjectID) (int64, error)

	// UpdateLifecycleState should fail with ErrInvalidTransition or ErrPreconditionFailed when the
	// current persisted state does not match the requested lifecycle transition.
	UpdateLifecycleState(ctx context.Context, reviewID primitive.ObjectID, patch ReviewLifecycleUpdatePatch) (*review.CompanyReview, error)
	// AttachAIResultReference should be used only while the review shell is still mutable.
	AttachAIResultReference(ctx context.Context, reviewID primitive.ObjectID, patch ReviewAIResultPatch) (*review.CompanyReview, error)
	// SaveValidatedReviewContent persists validated sections, scoring, and action outputs without finalizing.
	SaveValidatedReviewContent(ctx context.Context, reviewID primitive.ObjectID, patch ReviewValidatedContentPatch) (*review.CompanyReview, error)
	// FinalizeReview atomically transitions a validated review into immutable historical truth.
	FinalizeReview(ctx context.Context, reviewID primitive.ObjectID, patch ReviewFinalizationPatch) (*review.CompanyReview, error)
	// MarkSuperseded should fail when the target review is not already finalized and immutable.
	MarkSuperseded(ctx context.Context, reviewID primitive.ObjectID, patch ReviewSupersedePatch) (*review.CompanyReview, error)
}
