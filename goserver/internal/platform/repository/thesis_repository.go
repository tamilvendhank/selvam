package repository

import (
	"context"

	"goserver/internal/domain/common"
	"goserver/internal/domain/thesis"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

// InvestmentThesisSortBy enumerates supported thesis list sort fields.
type InvestmentThesisSortBy string

const (
	InvestmentThesisSortByVersion   InvestmentThesisSortBy = "thesis_version"
	InvestmentThesisSortByCreatedAt InvestmentThesisSortBy = "created_at"
	InvestmentThesisSortByUpdatedAt InvestmentThesisSortBy = "updated_at"
)

// InvestmentThesisSortOption controls thesis ordering.
type InvestmentThesisSortOption struct {
	By    InvestmentThesisSortBy
	Order SortOrder
}

// InvestmentThesisFilter captures repository query criteria for thesis history and active theses.
type InvestmentThesisFilter struct {
	IDs                      []primitive.ObjectID
	CompanyIDs               []primitive.ObjectID
	ThesisStatuses           []common.ThesisStatus
	CreatedFromReviewIDs     []primitive.ObjectID
	LastUpdatedFromReviewIDs []primitive.ObjectID
	CurrentPositionRoles     []common.PositionRole
	MinVersion               *int
	MaxVersion               *int
	CreatedAt                *TimeRange
	UpdatedAt                *TimeRange
	ActiveOnly               bool
}

// InvestmentThesisListOptions configures thesis list pagination and sort behavior.
type InvestmentThesisListOptions struct {
	Pagination PageOptions
	Sort       InvestmentThesisSortOption
}

// ThesisStatusPatch performs a guarded status transition on an existing thesis version.
type ThesisStatusPatch struct {
	NextStatus              common.ThesisStatus
	LastUpdatedFromReviewID *primitive.ObjectID
	ThesisChangeSummary     *string

	ExpectedCurrentStatuses []common.ThesisStatus
	Mutation                MutationMetadata
}

// ThesisVersionCreateOptions captures append-only versioning expectations.
type ThesisVersionCreateOptions struct {
	ExpectedPreviousVersion *int
}

// InvestmentThesisRepository stores durable thesis history. New versions should be appended rather than
// replacing historical thesis documents in place.
type InvestmentThesisRepository interface {
	Create(ctx context.Context, thesis *thesis.InvestmentThesis) (*thesis.InvestmentThesis, error)
	SaveNewVersion(ctx context.Context, thesis *thesis.InvestmentThesis, options ThesisVersionCreateOptions) (*thesis.InvestmentThesis, error)
	GetByID(ctx context.Context, id primitive.ObjectID) (*thesis.InvestmentThesis, error)
	GetActiveByCompanyID(ctx context.Context, companyID primitive.ObjectID) (*thesis.InvestmentThesis, error)
	GetLatestByCompanyID(ctx context.Context, companyID primitive.ObjectID) (*thesis.InvestmentThesis, error)
	ListByCompanyID(ctx context.Context, companyID primitive.ObjectID, options InvestmentThesisListOptions) (*ListResult[*thesis.InvestmentThesis], error)
	List(ctx context.Context, filter InvestmentThesisFilter, options InvestmentThesisListOptions) (*ListResult[*thesis.InvestmentThesis], error)
	UpdateStatus(ctx context.Context, thesisID primitive.ObjectID, patch ThesisStatusPatch) (*thesis.InvestmentThesis, error)
}
