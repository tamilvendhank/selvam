package ports

import (
	"context"

	"goserver/internal/platform/domain"
)

type CompanyService interface {
	ListCompanies(ctx context.Context, filter CompanyListFilter) ([]*domain.Company, error)
	GetCompany(ctx context.Context, id string) (*domain.Company, error)
	ListCompanyReviews(ctx context.Context, companyID string, filter CompanyReviewListFilter) ([]*domain.CompanyReview, error)
	GetCompanyThesis(ctx context.Context, companyID string) (*domain.InvestmentThesis, error)
	GetHistorySummary(ctx context.Context, companyID string, bookType domain.BookType) (map[string]any, error)
}

type ReviewService interface {
	CreateReview(ctx context.Context, review *domain.CompanyReview) (*domain.CompanyReview, error)
	UpdateDraftReview(ctx context.Context, review *domain.CompanyReview) (*domain.CompanyReview, error)
	FinalizeReview(ctx context.Context, reviewID string) (*domain.CompanyReview, error)
	ListReviews(ctx context.Context, filter CompanyReviewListFilter) ([]*domain.CompanyReview, error)
	GetReview(ctx context.Context, id string) (*domain.CompanyReview, error)
	GetReviewDiff(ctx context.Context, id string) (*domain.ReviewChangeLog, error)
	GetReviewEvidence(ctx context.Context, id string) ([]domain.EvidenceReference, error)
}

type ScorecardService interface {
	ValidateReview(ctx context.Context, review *domain.CompanyReview) error
	BuildAsyncReviewItem(ctx context.Context, company *domain.Company, snapshotID string, mode domain.InvestingMode) (AIReviewBatchItem, error)
}

type ThesisService interface {
	UpsertThesis(ctx context.Context, thesis *domain.InvestmentThesis) (*domain.InvestmentThesis, error)
	BuildOrUpdateFromReview(ctx context.Context, review *domain.CompanyReview) (*domain.InvestmentThesis, error)
	GetActiveThesis(ctx context.Context, companyID string) (*domain.InvestmentThesis, error)
}

type ActionMappingService interface {
	MapReview(ctx context.Context, review *domain.CompanyReview, thesis *domain.InvestmentThesis, previousReview *domain.CompanyReview) (*domain.DecisionAction, error)
}

type ChangeDetectionService interface {
	CompareReviews(ctx context.Context, current, previous *domain.CompanyReview, thesis *domain.InvestmentThesis) (*domain.ReviewChangeLog, error)
}

type WorkflowService interface {
	ListWorkflowRuns(ctx context.Context, filter WorkflowRunListFilter) ([]*domain.WorkflowRun, error)
	GetWorkflowRun(ctx context.Context, id string) (*domain.WorkflowRun, error)
	GetWorkflowSummary(ctx context.Context, id string) (map[string]any, error)
	GetWorkflowStatus(ctx context.Context, id string) (map[string]any, error)
	ListWorkflowSteps(ctx context.Context, workflowRunID string) ([]*domain.WorkflowStepRun, error)
	ResumeWorkflow(ctx context.Context, id string) (*domain.WorkflowRun, error)
	ReconcileWorkflow(ctx context.Context, id string) (*domain.WorkflowRun, error)
}

type InvestingWorkflowService interface {
	Start(ctx context.Context, request StartInvestingWorkflowRequest) (*domain.WorkflowRun, error)
	DryRun(ctx context.Context, request StartInvestingWorkflowRequest) (*domain.WorkflowRun, error)
	Resume(ctx context.Context, workflowRunID string) (*domain.WorkflowRun, error)
	Reconcile(ctx context.Context, workflowRunID string) (*domain.WorkflowRun, error)
}

type TradingWorkflowService interface {
	Start(ctx context.Context, request StartTradingWorkflowRequest) (*domain.WorkflowRun, error)
	Resume(ctx context.Context, workflowRunID string) (*domain.WorkflowRun, error)
	Reconcile(ctx context.Context, workflowRunID string) (*domain.WorkflowRun, error)
}

type CapitalAllocationService interface {
	CreateRun(ctx context.Context, run *domain.CapitalAllocationRun) (*domain.CapitalAllocationRun, error)
	ListRuns(ctx context.Context, filter CapitalAllocationListFilter) ([]*domain.CapitalAllocationRun, error)
	GetRun(ctx context.Context, id string) (*domain.CapitalAllocationRun, error)
}

type ConfigService interface {
	CurrentConfig(ctx context.Context) (map[string]any, error)
	CreateSnapshot(ctx context.Context, bookType domain.BookType, mode string) (*domain.ConfigSnapshot, error)
	ListSnapshots(ctx context.Context, filter ConfigSnapshotListFilter) ([]*domain.ConfigSnapshot, error)
	GetSnapshot(ctx context.Context, id string) (*domain.ConfigSnapshot, error)
}

type OverrideService interface {
	CreateOverride(ctx context.Context, override *domain.ManualOverride) (*domain.ManualOverride, error)
	ListOverrides(ctx context.Context, filter ManualOverrideListFilter) ([]*domain.ManualOverride, error)
	GetOverride(ctx context.Context, id string) (*domain.ManualOverride, error)
}

type ProjectionService interface {
	ListPositions(ctx context.Context, filter PositionListFilter) ([]*domain.CurrentPosition, error)
	GetPositionByCompanyAndBook(ctx context.Context, companyID string, bookType domain.BookType) (*domain.CurrentPosition, error)
	UpsertPosition(ctx context.Context, position *domain.CurrentPosition) (*domain.CurrentPosition, error)
}

type AIBatchService interface {
	ListJobs(ctx context.Context, filter AIBatchJobListFilter) ([]*domain.AIBatchJob, error)
	GetJob(ctx context.Context, id string) (*domain.AIBatchJob, error)
	ListItems(ctx context.Context, filter AIBatchItemListFilter) ([]*domain.AIBatchItem, error)
	RetryJob(ctx context.Context, id string) (*domain.AIBatchJob, error)
	RetryItem(ctx context.Context, id string) (*domain.AIBatchItem, error)
	SkipItem(ctx context.Context, id string) (*domain.AIBatchItem, error)
}
