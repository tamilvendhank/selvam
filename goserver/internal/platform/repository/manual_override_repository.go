package repository

import (
	"context"

	"goserver/internal/domain/common"
	overridepkg "goserver/internal/domain/override"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

// ManualOverrideSortBy enumerates supported manual-override sort fields.
type ManualOverrideSortBy string

const (
	ManualOverrideSortByOverrideDate ManualOverrideSortBy = "override_date"
	ManualOverrideSortByCreatedAt    ManualOverrideSortBy = "created_at"
)

// ManualOverrideSortOption controls manual-override ordering.
type ManualOverrideSortOption struct {
	By    ManualOverrideSortBy
	Order SortOrder
}

// ManualOverrideFilter captures repository query criteria for override history.
type ManualOverrideFilter struct {
	IDs               []primitive.ObjectID
	CompanyIDs        []primitive.ObjectID
	ReviewIDs         []primitive.ObjectID
	BookTypes         []common.BookType
	OriginalActions   []common.InvestingActionType
	OverriddenActions []common.InvestingActionType
	OverrideBy        string
	OverrideDate      *TimeRange
	CreatedAt         *TimeRange
}

// ManualOverrideListOptions configures manual-override pagination and sort behavior.
type ManualOverrideListOptions struct {
	Pagination PageOptions
	Sort       ManualOverrideSortOption
}

// ManualOverrideRepository stores human override history as an append-only audit trail.
type ManualOverrideRepository interface {
	Create(ctx context.Context, override *overridepkg.ManualOverride) (*overridepkg.ManualOverride, error)
	GetByID(ctx context.Context, id primitive.ObjectID) (*overridepkg.ManualOverride, error)
	List(ctx context.Context, filter ManualOverrideFilter, options ManualOverrideListOptions) (*ListResult[*overridepkg.ManualOverride], error)
	GetLatestByReviewID(ctx context.Context, reviewID primitive.ObjectID) (*overridepkg.ManualOverride, error)
}
