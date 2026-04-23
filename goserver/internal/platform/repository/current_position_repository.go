package repository

import (
	"context"
	"time"

	"goserver/internal/domain/common"
	"goserver/internal/domain/position"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

// CurrentPositionSortBy enumerates supported current-position sort fields.
type CurrentPositionSortBy string

const (
	CurrentPositionSortByCurrentMarketValue       CurrentPositionSortBy = "current_market_value"
	CurrentPositionSortByCurrentPositionPctOfBook CurrentPositionSortBy = "current_position_pct_of_book"
	CurrentPositionSortByLastUpdatedAt            CurrentPositionSortBy = "last_updated_at"
)

// CurrentPositionSortOption controls current-position ordering.
type CurrentPositionSortOption struct {
	By    CurrentPositionSortBy
	Order SortOrder
}

// CurrentPositionFilter captures repository query criteria for the current-state position projection.
type CurrentPositionFilter struct {
	IDs           []primitive.ObjectID
	CompanyIDs    []primitive.ObjectID
	BookTypes     []common.BookType
	IsOpen        *bool
	LastUpdatedAt *TimeRange
}

// CurrentPositionListOptions configures current-position pagination and sort behavior.
type CurrentPositionListOptions struct {
	Pagination PageOptions
	Sort       CurrentPositionSortOption
}

// CurrentPositionPatch updates mutable fields on the current-position projection.
type CurrentPositionPatch struct {
	IsOpen                        *bool
	Quantity                      *float64
	AverageCost                   *float64
	CurrentMarketValue            *float64
	CurrentPositionPctOfBook      *float64
	CurrentPositionPctOfPortfolio *float64
	LastUpdatedAt                 time.Time

	ExpectedLastUpdatedAt *time.Time
	Mutation              MutationMetadata
}

// CurrentPositionClosePatch closes an open position projection without deleting history elsewhere.
type CurrentPositionClosePatch struct {
	ClosedAt time.Time
	Reason   string
	ClosedBy string

	ExpectedLastUpdatedAt *time.Time
}

// CurrentPositionRepository stores current-state position projections only; it is not the historical source of truth.
type CurrentPositionRepository interface {
	Upsert(ctx context.Context, position *position.CurrentPosition) (*position.CurrentPosition, error)
	GetByID(ctx context.Context, id primitive.ObjectID) (*position.CurrentPosition, error)
	GetByCompanyAndBook(ctx context.Context, companyID primitive.ObjectID, bookType common.BookType) (*position.CurrentPosition, error)
	List(ctx context.Context, filter CurrentPositionFilter, options CurrentPositionListOptions) (*ListResult[*position.CurrentPosition], error)
	ListOpenByBook(ctx context.Context, bookType common.BookType, options CurrentPositionListOptions) (*ListResult[*position.CurrentPosition], error)
	UpdateSnapshot(ctx context.Context, positionID primitive.ObjectID, patch CurrentPositionPatch) (*position.CurrentPosition, error)
	ClosePosition(ctx context.Context, positionID primitive.ObjectID, patch CurrentPositionClosePatch) (*position.CurrentPosition, error)
}
