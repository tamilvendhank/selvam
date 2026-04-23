package http

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"goserver/internal/platform/domain"
	"goserver/internal/platform/ports"
	"goserver/internal/platform/testutil"
)

type companyServiceStub struct{}

func (companyServiceStub) ListCompanies(context.Context, ports.CompanyListFilter) ([]*domain.Company, error) {
	return []*domain.Company{testutil.SampleCompany()}, nil
}
func (companyServiceStub) GetCompany(context.Context, string) (*domain.Company, error) {
	return testutil.SampleCompany(), nil
}
func (companyServiceStub) ListCompanyReviews(context.Context, string, ports.CompanyReviewListFilter) ([]*domain.CompanyReview, error) {
	return []*domain.CompanyReview{testutil.SampleInvestingReview(8, true)}, nil
}
func (companyServiceStub) GetCompanyThesis(context.Context, string) (*domain.InvestmentThesis, error) {
	return testutil.SampleThesis(), nil
}
func (companyServiceStub) GetHistorySummary(context.Context, string, domain.BookType) (map[string]any, error) {
	return map[string]any{"ok": true}, nil
}

type reviewServiceStub struct{}

func (reviewServiceStub) CreateReview(context.Context, *domain.CompanyReview) (*domain.CompanyReview, error) {
	return nil, nil
}
func (reviewServiceStub) UpdateDraftReview(context.Context, *domain.CompanyReview) (*domain.CompanyReview, error) {
	return nil, nil
}
func (reviewServiceStub) FinalizeReview(context.Context, string) (*domain.CompanyReview, error) {
	return nil, nil
}
func (reviewServiceStub) ListReviews(context.Context, ports.CompanyReviewListFilter) ([]*domain.CompanyReview, error) {
	return []*domain.CompanyReview{testutil.SampleInvestingReview(8, true)}, nil
}
func (reviewServiceStub) GetReview(context.Context, string) (*domain.CompanyReview, error) {
	return testutil.SampleInvestingReview(8, true), nil
}
func (reviewServiceStub) GetReviewDiff(context.Context, string) (*domain.ReviewChangeLog, error) {
	return &domain.ReviewChangeLog{ChangeSummary: "ok"}, nil
}
func (reviewServiceStub) GetReviewEvidence(context.Context, string) ([]domain.EvidenceReference, error) {
	return []domain.EvidenceReference{}, nil
}

type workflowServiceStub struct{}

func (workflowServiceStub) ListWorkflowRuns(context.Context, ports.WorkflowRunListFilter) ([]*domain.WorkflowRun, error) {
	return []*domain.WorkflowRun{}, nil
}
func (workflowServiceStub) GetWorkflowRun(context.Context, string) (*domain.WorkflowRun, error) {
	return &domain.WorkflowRun{ID: "run-1"}, nil
}
func (workflowServiceStub) GetWorkflowSummary(context.Context, string) (map[string]any, error) {
	return map[string]any{"id": "run-1"}, nil
}

type investingWorkflowServiceStub struct{}

func (investingWorkflowServiceStub) Start(context.Context, ports.StartInvestingWorkflowRequest) (*domain.WorkflowRun, error) {
	return &domain.WorkflowRun{ID: "run-1", Status: domain.WorkflowRunStatusWaitingAsync}, nil
}
func (investingWorkflowServiceStub) DryRun(context.Context, ports.StartInvestingWorkflowRequest) (*domain.WorkflowRun, error) {
	return &domain.WorkflowRun{ID: "run-2", Status: domain.WorkflowRunStatusCompleted, DryRun: true}, nil
}

type capitalAllocationServiceStub struct{}

func (capitalAllocationServiceStub) CreateRun(context.Context, *domain.CapitalAllocationRun) (*domain.CapitalAllocationRun, error) {
	return nil, nil
}
func (capitalAllocationServiceStub) ListRuns(context.Context, ports.CapitalAllocationListFilter) ([]*domain.CapitalAllocationRun, error) {
	return []*domain.CapitalAllocationRun{}, nil
}
func (capitalAllocationServiceStub) GetRun(context.Context, string) (*domain.CapitalAllocationRun, error) {
	return &domain.CapitalAllocationRun{ID: "alloc-1"}, nil
}

type configServiceAPIStub struct{}

func (configServiceAPIStub) CurrentConfig(context.Context) (map[string]any, error) {
	return map[string]any{"schemaVersion": "v1alpha1"}, nil
}
func (configServiceAPIStub) CreateSnapshot(context.Context, domain.BookType, string) (*domain.ConfigSnapshot, error) {
	return nil, nil
}
func (configServiceAPIStub) ListSnapshots(context.Context, ports.ConfigSnapshotListFilter) ([]*domain.ConfigSnapshot, error) {
	return []*domain.ConfigSnapshot{}, nil
}
func (configServiceAPIStub) GetSnapshot(context.Context, string) (*domain.ConfigSnapshot, error) {
	return &domain.ConfigSnapshot{ID: "snapshot-1"}, nil
}

type overrideServiceStub struct{}

func (overrideServiceStub) CreateOverride(_ context.Context, override *domain.ManualOverride) (*domain.ManualOverride, error) {
	override.ID = "override-1"
	return override, nil
}
func (overrideServiceStub) ListOverrides(context.Context, ports.ManualOverrideListFilter) ([]*domain.ManualOverride, error) {
	return []*domain.ManualOverride{}, nil
}
func (overrideServiceStub) GetOverride(context.Context, string) (*domain.ManualOverride, error) {
	return &domain.ManualOverride{ID: "override-1"}, nil
}

type projectionServiceStub struct{}

func (projectionServiceStub) ListPositions(context.Context, ports.PositionListFilter) ([]*domain.CurrentPosition, error) {
	return []*domain.CurrentPosition{}, nil
}
func (projectionServiceStub) GetPositionByCompanyAndBook(context.Context, string, domain.BookType) (*domain.CurrentPosition, error) {
	return nil, nil
}
func (projectionServiceStub) UpsertPosition(context.Context, *domain.CurrentPosition) (*domain.CurrentPosition, error) {
	return nil, nil
}

func TestListCompaniesHandler(t *testing.T) {
	api := NewAPI(
		companyServiceStub{},
		reviewServiceStub{},
		workflowServiceStub{},
		investingWorkflowServiceStub{},
		capitalAllocationServiceStub{},
		configServiceAPIStub{},
		overrideServiceStub{},
		projectionServiceStub{},
	)

	request := httptest.NewRequest(http.MethodGet, "/api/v1/companies", nil)
	recorder := httptest.NewRecorder()
	api.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", recorder.Code)
	}
}

func TestCreateOverrideHandler(t *testing.T) {
	api := NewAPI(
		companyServiceStub{},
		reviewServiceStub{},
		workflowServiceStub{},
		investingWorkflowServiceStub{},
		capitalAllocationServiceStub{},
		configServiceAPIStub{},
		overrideServiceStub{},
		projectionServiceStub{},
	)

	body := []byte(`{
		"companyId":"507f1f77bcf86cd799439011",
		"reviewId":"507f1f77bcf86cd799439021",
		"bookType":"investing",
		"originalAction":"hold",
		"overriddenAction":"trim",
		"overrideReason":"Reduce concentration",
		"overrideBy":"pm"
	}`)
	request := httptest.NewRequest(http.MethodPost, "/api/v1/overrides", bytes.NewReader(body))
	recorder := httptest.NewRecorder()
	api.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d", recorder.Code)
	}
}

var _ ports.OverrideService = overrideServiceStub{}
var _ ports.CompanyService = companyServiceStub{}
var _ ports.ReviewService = reviewServiceStub{}
var _ ports.WorkflowService = workflowServiceStub{}
var _ ports.InvestingWorkflowService = investingWorkflowServiceStub{}
var _ ports.CapitalAllocationService = capitalAllocationServiceStub{}
var _ ports.ConfigService = configServiceAPIStub{}
var _ ports.ProjectionService = projectionServiceStub{}

func init() {
	_ = time.Now()
}
