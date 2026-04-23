package ports

import (
	"context"

	"goserver/internal/platform/domain"
)

type CompanyListFilter struct {
	Search     string
	BookType   domain.BookType
	ActiveOnly *bool
	Limit      int
	Offset     int
}

type CompanyReviewListFilter struct {
	CompanyID    string
	Symbol       string
	BookType     domain.BookType
	ReviewStatus domain.ReviewStatus
	Limit        int
	Offset       int
}

type WorkflowRunListFilter struct {
	BookType domain.BookType
	Status   domain.WorkflowRunStatus
	Limit    int
	Offset   int
}

type CapitalAllocationListFilter struct {
	BookType domain.BookType
	Limit    int
	Offset   int
}

type ConfigSnapshotListFilter struct {
	BookType domain.BookType
	Limit    int
	Offset   int
}

type ManualOverrideListFilter struct {
	CompanyID string
	ReviewID  string
	BookType  domain.BookType
	Limit     int
	Offset    int
}

type PositionListFilter struct {
	BookType domain.BookType
	Limit    int
	Offset   int
}

type CompanyRepository interface {
	Create(ctx context.Context, company *domain.Company) (*domain.Company, error)
	Update(ctx context.Context, company *domain.Company) (*domain.Company, error)
	GetByID(ctx context.Context, id string) (*domain.Company, error)
	GetBySymbol(ctx context.Context, symbol string) (*domain.Company, error)
	List(ctx context.Context, filter CompanyListFilter) ([]*domain.Company, error)
}

type CompanyReviewRepository interface {
	Create(ctx context.Context, review *domain.CompanyReview) (*domain.CompanyReview, error)
	UpdateDraft(ctx context.Context, review *domain.CompanyReview) (*domain.CompanyReview, error)
	Finalize(ctx context.Context, reviewID string) (*domain.CompanyReview, error)
	MarkSuperseded(ctx context.Context, reviewID string) (*domain.CompanyReview, error)
	GetByID(ctx context.Context, id string) (*domain.CompanyReview, error)
	GetLatestByCompany(ctx context.Context, companyID string, bookType domain.BookType) (*domain.CompanyReview, error)
	List(ctx context.Context, filter CompanyReviewListFilter) ([]*domain.CompanyReview, error)
}

type ThesisRepository interface {
	Create(ctx context.Context, thesis *domain.InvestmentThesis) (*domain.InvestmentThesis, error)
	Update(ctx context.Context, thesis *domain.InvestmentThesis) (*domain.InvestmentThesis, error)
	GetByID(ctx context.Context, id string) (*domain.InvestmentThesis, error)
	GetActiveByCompanyID(ctx context.Context, companyID string) (*domain.InvestmentThesis, error)
	ListByCompanyID(ctx context.Context, companyID string) ([]*domain.InvestmentThesis, error)
}

type WorkflowRunRepository interface {
	Create(ctx context.Context, run *domain.WorkflowRun) (*domain.WorkflowRun, error)
	Update(ctx context.Context, run *domain.WorkflowRun) (*domain.WorkflowRun, error)
	GetByID(ctx context.Context, id string) (*domain.WorkflowRun, error)
	GetByIdempotencyKey(ctx context.Context, key string) (*domain.WorkflowRun, error)
	List(ctx context.Context, filter WorkflowRunListFilter) ([]*domain.WorkflowRun, error)
}

type ConfigSnapshotRepository interface {
	Create(ctx context.Context, snapshot *domain.ConfigSnapshot) (*domain.ConfigSnapshot, error)
	GetByID(ctx context.Context, id string) (*domain.ConfigSnapshot, error)
	List(ctx context.Context, filter ConfigSnapshotListFilter) ([]*domain.ConfigSnapshot, error)
}

type CapitalAllocationRepository interface {
	Create(ctx context.Context, run *domain.CapitalAllocationRun) (*domain.CapitalAllocationRun, error)
	GetByID(ctx context.Context, id string) (*domain.CapitalAllocationRun, error)
	List(ctx context.Context, filter CapitalAllocationListFilter) ([]*domain.CapitalAllocationRun, error)
}

type ManualOverrideRepository interface {
	Create(ctx context.Context, override *domain.ManualOverride) (*domain.ManualOverride, error)
	GetByID(ctx context.Context, id string) (*domain.ManualOverride, error)
	List(ctx context.Context, filter ManualOverrideListFilter) ([]*domain.ManualOverride, error)
}

type PositionRepository interface {
	Upsert(ctx context.Context, position *domain.CurrentPosition) (*domain.CurrentPosition, error)
	GetByID(ctx context.Context, id string) (*domain.CurrentPosition, error)
	GetByCompanyAndBook(ctx context.Context, companyID string, bookType domain.BookType) (*domain.CurrentPosition, error)
	List(ctx context.Context, filter PositionListFilter) ([]*domain.CurrentPosition, error)
}
