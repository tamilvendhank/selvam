package repository

import (
	"context"
	"time"

	"goserver/internal/domain/allocation"
	"goserver/internal/domain/common"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

// CapitalAllocationRunSortBy enumerates supported capital-allocation sort fields.
type CapitalAllocationRunSortBy string

const (
	CapitalAllocationRunSortByAllocationDate CapitalAllocationRunSortBy = "allocation_date"
	CapitalAllocationRunSortByCreatedAt      CapitalAllocationRunSortBy = "created_at"
)

// CapitalAllocationRunSortOption controls capital-allocation ordering.
type CapitalAllocationRunSortOption struct {
	By    CapitalAllocationRunSortBy
	Order SortOrder
}

// CapitalAllocationRunFilter captures repository query criteria for allocation runs.
type CapitalAllocationRunFilter struct {
	IDs                      []primitive.ObjectID
	WorkflowRunIDs           []primitive.ObjectID
	BookTypes                []common.BookType
	ContainsCompanyID        *primitive.ObjectID
	ContainsDecisionReviewID *primitive.ObjectID
	AllocationDate           *TimeRange
	CreatedAt                *TimeRange
}

// CapitalAllocationRunListOptions configures capital-allocation pagination and sort behavior.
type CapitalAllocationRunListOptions struct {
	Pagination PageOptions
	Sort       CapitalAllocationRunSortOption
}

// CapitalAllocationRunDraftPatch updates mutable allocation payload fields before the run is treated as frozen.
type CapitalAllocationRunDraftPatch struct {
	AvailableCashStart    *float64
	FreshMonthlyCash      *float64
	SellProceedsAvailable *float64
	CarryForwardCash      *float64
	TargetDeployableCash  *float64
	AllocatedCashTotal    *float64
	CashLeftUnallocated   *float64
	AllocationNotes       *string
	Items                 []allocation.CapitalAllocationItem
	ReplaceItems          bool

	Mutation MutationMetadata
}

// CapitalAllocationRunFinalizationPatch closes the draft mutation window for an allocation run.
type CapitalAllocationRunFinalizationPatch struct {
	FinalizedAt time.Time
	FinalizedBy string
	Reason      string
}

// CapitalAllocationRunRepository stores capital-allocation decisions derived from finalized reviews.
// The current domain model is largely append-only; implementations may use repository-managed rules to
// decide whether UpdateDraftAllocation is still allowed prior to finalization.
type CapitalAllocationRunRepository interface {
	Create(ctx context.Context, run *allocation.CapitalAllocationRun) (*allocation.CapitalAllocationRun, error)
	GetByID(ctx context.Context, id primitive.ObjectID) (*allocation.CapitalAllocationRun, error)
	List(ctx context.Context, filter CapitalAllocationRunFilter, options CapitalAllocationRunListOptions) (*ListResult[*allocation.CapitalAllocationRun], error)
	UpdateDraftAllocation(ctx context.Context, runID primitive.ObjectID, patch CapitalAllocationRunDraftPatch) (*allocation.CapitalAllocationRun, error)
	FinalizeAllocationRun(ctx context.Context, runID primitive.ObjectID, patch CapitalAllocationRunFinalizationPatch) (*allocation.CapitalAllocationRun, error)
	GetLatestByBookType(ctx context.Context, bookType common.BookType) (*allocation.CapitalAllocationRun, error)
}
