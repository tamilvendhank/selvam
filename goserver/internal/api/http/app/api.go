package app

import (
	"net/http"

	platformrepo "goserver/internal/platform/repository"
)

type Dependencies struct {
	Companies   platformrepo.CompanyRepository
	Reviews     platformrepo.CompanyReviewRepository
	Theses      platformrepo.InvestmentThesisRepository
	Allocations platformrepo.CapitalAllocationRunRepository
	Positions   platformrepo.CurrentPositionRepository
	Workflows   platformrepo.WorkflowRunRepository
}

type API struct {
	companies   platformrepo.CompanyRepository
	reviews     platformrepo.CompanyReviewRepository
	theses      platformrepo.InvestmentThesisRepository
	allocations platformrepo.CapitalAllocationRunRepository
	positions   platformrepo.CurrentPositionRepository
	workflows   platformrepo.WorkflowRunRepository
}

func NewAPI(dependencies Dependencies) *API {
	return &API{
		companies:   dependencies.Companies,
		reviews:     dependencies.Reviews,
		theses:      dependencies.Theses,
		allocations: dependencies.Allocations,
		positions:   dependencies.Positions,
		workflows:   dependencies.Workflows,
	}
}

func NewHandler(dependencies Dependencies) http.Handler {
	return NewAPI(dependencies)
}

func (api *API) ServeHTTP(writer http.ResponseWriter, request *http.Request) {
	switch {
	case request.Method == http.MethodGet && request.URL.Path == "/api/v1/companies":
		api.listCompanies(writer, request)
	case request.Method == http.MethodGet && pathMatches(request.URL.Path, "/api/v1/companies/", "/reviews"):
		api.listCompanyReviews(writer, request)
	case request.Method == http.MethodGet && pathMatches(request.URL.Path, "/api/v1/companies/", "/thesis/history"):
		api.listCompanyThesisHistory(writer, request)
	case request.Method == http.MethodGet && pathMatches(request.URL.Path, "/api/v1/companies/", "/thesis"):
		api.getCompanyActiveThesis(writer, request)
	case request.Method == http.MethodGet && pathMatches(request.URL.Path, "/api/v1/companies/", "/history-summary"):
		api.getCompanyHistorySummary(writer, request)
	case request.Method == http.MethodGet && pathMatches(request.URL.Path, "/api/v1/companies/", "/positions"):
		api.listCompanyPositions(writer, request)
	case request.Method == http.MethodGet && pathMatches(request.URL.Path, "/api/v1/companies/", ""):
		api.getCompany(writer, request)
	case request.Method == http.MethodGet && request.URL.Path == "/api/v1/reviews":
		api.listReviews(writer, request)
	case request.Method == http.MethodGet && pathMatches(request.URL.Path, "/api/v1/reviews/", "/scorecard"):
		api.getReviewScorecard(writer, request)
	case request.Method == http.MethodGet && pathMatches(request.URL.Path, "/api/v1/reviews/", "/evidence"):
		api.getReviewEvidence(writer, request)
	case request.Method == http.MethodGet && pathMatches(request.URL.Path, "/api/v1/reviews/", "/diff"):
		api.getReviewDiff(writer, request)
	case request.Method == http.MethodGet && pathMatches(request.URL.Path, "/api/v1/reviews/", ""):
		api.getReview(writer, request)
	case request.Method == http.MethodGet && request.URL.Path == "/api/v1/theses":
		api.listTheses(writer, request)
	case request.Method == http.MethodGet && pathMatches(request.URL.Path, "/api/v1/theses/", ""):
		api.getThesis(writer, request)
	case request.Method == http.MethodGet && request.URL.Path == "/api/v1/capital-allocations":
		api.listCapitalAllocations(writer, request)
	case request.Method == http.MethodGet && pathMatches(request.URL.Path, "/api/v1/capital-allocations/", ""):
		api.getCapitalAllocation(writer, request)
	case request.Method == http.MethodGet && pathMatches(request.URL.Path, "/api/v1/workflow-runs/", "/capital-allocation"):
		api.getWorkflowCapitalAllocation(writer, request)
	case request.Method == http.MethodGet && request.URL.Path == "/api/v1/positions":
		api.listPositions(writer, request)
	case request.Method == http.MethodGet && pathMatches(request.URL.Path, "/api/v1/positions/", ""):
		api.listPositionsByBook(writer, request)
	default:
		http.NotFound(writer, request)
	}
}
