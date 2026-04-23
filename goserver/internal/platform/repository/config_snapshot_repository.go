package repository

import (
	"context"

	"goserver/internal/domain/common"
	"goserver/internal/domain/config"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

// ConfigSnapshotSortBy enumerates supported config-snapshot sort fields.
type ConfigSnapshotSortBy string

const (
	ConfigSnapshotSortByCreatedAt ConfigSnapshotSortBy = "created_at"
)

// ConfigSnapshotSortOption controls config-snapshot ordering.
type ConfigSnapshotSortOption struct {
	By    ConfigSnapshotSortBy
	Order SortOrder
}

// ConfigSnapshotFilter captures repository query criteria for immutable config snapshots.
type ConfigSnapshotFilter struct {
	IDs       []primitive.ObjectID
	BookTypes []common.BookType
	Modes     []common.InvestingMode
	CreatedAt *TimeRange
}

// ConfigSnapshotListOptions configures config-snapshot pagination and sort behavior.
type ConfigSnapshotListOptions struct {
	Pagination PageOptions
	Sort       ConfigSnapshotSortOption
}

// LatestConfigSnapshotLookup constrains latest-snapshot lookups.
type LatestConfigSnapshotLookup struct {
	Mode *common.InvestingMode
}

// ConfigSnapshotRepository stores immutable historical configuration snapshots.
type ConfigSnapshotRepository interface {
	Create(ctx context.Context, snapshot *config.ConfigSnapshot) (*config.ConfigSnapshot, error)
	GetByID(ctx context.Context, id primitive.ObjectID) (*config.ConfigSnapshot, error)
	List(ctx context.Context, filter ConfigSnapshotFilter, options ConfigSnapshotListOptions) (*ListResult[*config.ConfigSnapshot], error)
	GetLatestByBookType(ctx context.Context, bookType common.BookType, lookup LatestConfigSnapshotLookup) (*config.ConfigSnapshot, error)
}
