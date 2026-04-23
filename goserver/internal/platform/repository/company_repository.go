package repository

import (
	"context"
	"time"

	"goserver/internal/domain/company"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

// CompanySortBy enumerates supported company list sort fields.
type CompanySortBy string

const (
	CompanySortBySymbol      CompanySortBy = "symbol"
	CompanySortByCompanyName CompanySortBy = "company_name"
	CompanySortByMarketCap   CompanySortBy = "market_cap_bucket"
	CompanySortByCreatedAt   CompanySortBy = "created_at"
	CompanySortByUpdatedAt   CompanySortBy = "updated_at"
	CompanySortByListingDate CompanySortBy = "listing_date"
)

// CompanySortOption controls ordering for company list queries.
type CompanySortOption struct {
	By    CompanySortBy
	Order SortOrder
}

// CompanyFilter captures repository-level company query criteria.
type CompanyFilter struct {
	IDs                 []primitive.ObjectID
	Symbols             []string
	Search              string
	Exchange            string
	Sector              string
	Industry            string
	SubIndustry         string
	MarketCapBucket     string
	InInvestingUniverse *bool
	InTradingUniverse   *bool
	StatusActive        *bool
	CreatedAt           *TimeRange
	UpdatedAt           *TimeRange
}

// CompanyListOptions configures company list pagination and sort behavior.
type CompanyListOptions struct {
	Pagination PageOptions
	Sort       CompanySortOption
}

// CompanyUpdatePatch captures mutable company master-data fields without allowing identifier replacement.
type CompanyUpdatePatch struct {
	Exchange        *string
	CompanyName     *string
	Sector          *string
	Industry        *string
	SubIndustry     *string
	BusinessSummary *string
	ListingDate     *time.Time
	MarketCapBucket *string
	StatusActive    *bool

	ExpectedUpdatedAt *time.Time
	Mutation          MutationMetadata
}

// CompanyUniverseFlagPatch updates book-universe membership flags atomically.
type CompanyUniverseFlagPatch struct {
	InInvestingUniverse *bool
	InTradingUniverse   *bool

	ExpectedUpdatedAt *time.Time
	Mutation          MutationMetadata
}

// CompanyRepository stores canonical company master data shared by investing and trading flows.
type CompanyRepository interface {
	Create(ctx context.Context, company *company.Company) (*company.Company, error)
	GetByID(ctx context.Context, id primitive.ObjectID) (*company.Company, error)
	GetBySymbol(ctx context.Context, symbol string) (*company.Company, error)
	ExistsBySymbol(ctx context.Context, symbol string) (bool, error)
	List(ctx context.Context, filter CompanyFilter, options CompanyListOptions) (*ListResult[*company.Company], error)
	UpdateMetadata(ctx context.Context, companyID primitive.ObjectID, patch CompanyUpdatePatch) (*company.Company, error)
	UpdateUniverseFlags(ctx context.Context, companyID primitive.ObjectID, patch CompanyUniverseFlagPatch) (*company.Company, error)
}
