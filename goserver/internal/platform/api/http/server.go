package http

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"

	"goserver/internal/platform/api/http/dto"
	"goserver/internal/platform/domain"
	"goserver/internal/platform/ports"
	platformservice "goserver/internal/platform/service"
)

type API struct {
	companyService           ports.CompanyService
	reviewService            ports.ReviewService
	workflowService          ports.WorkflowService
	investingWorkflowService ports.InvestingWorkflowService
	capitalAllocationService ports.CapitalAllocationService
	configService            ports.ConfigService
	overrideService          ports.OverrideService
	projectionService        ports.ProjectionService
}

func NewAPI(
	companyService ports.CompanyService,
	reviewService ports.ReviewService,
	workflowService ports.WorkflowService,
	investingWorkflowService ports.InvestingWorkflowService,
	capitalAllocationService ports.CapitalAllocationService,
	configService ports.ConfigService,
	overrideService ports.OverrideService,
	projectionService ports.ProjectionService,
) http.Handler {
	return &API{
		companyService:           companyService,
		reviewService:            reviewService,
		workflowService:          workflowService,
		investingWorkflowService: investingWorkflowService,
		capitalAllocationService: capitalAllocationService,
		configService:            configService,
		overrideService:          overrideService,
		projectionService:        projectionService,
	}
}

func (api *API) ServeHTTP(writer http.ResponseWriter, request *http.Request) {
	switch {
	case request.Method == http.MethodGet && request.URL.Path == "/api/v1/companies":
		api.listCompanies(writer, request)
	case request.Method == http.MethodGet && pathMatches(request.URL.Path, "/api/v1/companies/", ""):
		api.getCompany(writer, request)
	case request.Method == http.MethodGet && pathMatches(request.URL.Path, "/api/v1/companies/", "/reviews"):
		api.listCompanyReviews(writer, request)
	case request.Method == http.MethodGet && pathMatches(request.URL.Path, "/api/v1/companies/", "/thesis"):
		api.getCompanyThesis(writer, request)
	case request.Method == http.MethodGet && pathMatches(request.URL.Path, "/api/v1/companies/", "/history-summary"):
		api.getCompanyHistorySummary(writer, request)
	case request.Method == http.MethodGet && request.URL.Path == "/api/v1/reviews":
		api.listReviews(writer, request)
	case request.Method == http.MethodGet && pathMatches(request.URL.Path, "/api/v1/reviews/", ""):
		api.getReview(writer, request)
	case request.Method == http.MethodGet && pathMatches(request.URL.Path, "/api/v1/reviews/", "/diff"):
		api.getReviewDiff(writer, request)
	case request.Method == http.MethodGet && pathMatches(request.URL.Path, "/api/v1/reviews/", "/evidence"):
		api.getReviewEvidence(writer, request)
	case request.Method == http.MethodGet && request.URL.Path == "/api/v1/workflow-runs":
		api.listWorkflowRuns(writer, request)
	case request.Method == http.MethodGet && pathMatches(request.URL.Path, "/api/v1/workflow-runs/", ""):
		api.getWorkflowRun(writer, request)
	case request.Method == http.MethodGet && pathMatches(request.URL.Path, "/api/v1/workflow-runs/", "/summary"):
		api.getWorkflowSummary(writer, request)
	case request.Method == http.MethodPost && request.URL.Path == "/api/v1/workflow-runs/investing/start":
		api.startInvestingWorkflow(writer, request, false)
	case request.Method == http.MethodPost && request.URL.Path == "/api/v1/workflow-runs/investing/dry-run":
		api.startInvestingWorkflow(writer, request, true)
	case request.Method == http.MethodGet && request.URL.Path == "/api/v1/capital-allocations":
		api.listCapitalAllocations(writer, request)
	case request.Method == http.MethodGet && pathMatches(request.URL.Path, "/api/v1/capital-allocations/", ""):
		api.getCapitalAllocation(writer, request)
	case request.Method == http.MethodGet && request.URL.Path == "/api/v1/config/current":
		api.getCurrentConfig(writer, request)
	case request.Method == http.MethodGet && request.URL.Path == "/api/v1/config/snapshots":
		api.listConfigSnapshots(writer, request)
	case request.Method == http.MethodGet && pathMatches(request.URL.Path, "/api/v1/config/snapshots/", ""):
		api.getConfigSnapshot(writer, request)
	case request.Method == http.MethodPost && request.URL.Path == "/api/v1/overrides":
		api.createOverride(writer, request)
	case request.Method == http.MethodGet && request.URL.Path == "/api/v1/overrides":
		api.listOverrides(writer, request)
	case request.Method == http.MethodGet && pathMatches(request.URL.Path, "/api/v1/overrides/", ""):
		api.getOverride(writer, request)
	case request.Method == http.MethodGet && request.URL.Path == "/api/v1/positions":
		api.listPositions(writer, request)
	case request.Method == http.MethodGet && pathMatches(request.URL.Path, "/api/v1/positions/", ""):
		api.listPositionsByBook(writer, request)
	default:
		http.NotFound(writer, request)
	}
}

func (api *API) listCompanies(writer http.ResponseWriter, request *http.Request) {
	companies, err := api.companyService.ListCompanies(request.Context(), ports.CompanyListFilter{
		Search: request.URL.Query().Get("search"),
		Limit:  queryInt(request, "limit", 25),
		Offset: queryInt(request, "offset", 0),
	})
	if err != nil {
		api.writeError(writer, err)
		return
	}

	api.writeJSON(writer, http.StatusOK, map[string]any{"companies": dto.MapCompanySummaries(companies)})
}

func (api *API) getCompany(writer http.ResponseWriter, request *http.Request) {
	id, ok := pathParam(request.URL.Path, "/api/v1/companies/", "")
	if !ok {
		http.NotFound(writer, request)
		return
	}

	company, err := api.companyService.GetCompany(request.Context(), id)
	if err != nil {
		api.writeError(writer, err)
		return
	}

	api.writeJSON(writer, http.StatusOK, map[string]any{"company": company})
}

func (api *API) listCompanyReviews(writer http.ResponseWriter, request *http.Request) {
	id, ok := pathParam(request.URL.Path, "/api/v1/companies/", "/reviews")
	if !ok {
		http.NotFound(writer, request)
		return
	}

	bookType := domain.BookType(request.URL.Query().Get("book_type"))
	reviews, err := api.companyService.ListCompanyReviews(request.Context(), id, ports.CompanyReviewListFilter{
		BookType: bookType,
		Limit:    queryInt(request, "limit", 25),
		Offset:   queryInt(request, "offset", 0),
	})
	if err != nil {
		api.writeError(writer, err)
		return
	}

	api.writeJSON(writer, http.StatusOK, map[string]any{"reviews": dto.MapReviewSummaries(reviews)})
}

func (api *API) getCompanyThesis(writer http.ResponseWriter, request *http.Request) {
	id, ok := pathParam(request.URL.Path, "/api/v1/companies/", "/thesis")
	if !ok {
		http.NotFound(writer, request)
		return
	}

	thesis, err := api.companyService.GetCompanyThesis(request.Context(), id)
	if err != nil {
		api.writeError(writer, err)
		return
	}

	api.writeJSON(writer, http.StatusOK, map[string]any{"thesis": thesis})
}

func (api *API) getCompanyHistorySummary(writer http.ResponseWriter, request *http.Request) {
	id, ok := pathParam(request.URL.Path, "/api/v1/companies/", "/history-summary")
	if !ok {
		http.NotFound(writer, request)
		return
	}
	bookType := domain.BookType(request.URL.Query().Get("book_type"))
	if bookType == "" {
		bookType = domain.BookTypeInvesting
	}

	summary, err := api.companyService.GetHistorySummary(request.Context(), id, bookType)
	if err != nil {
		api.writeError(writer, err)
		return
	}

	api.writeJSON(writer, http.StatusOK, map[string]any{"summary": summary})
}

func (api *API) listReviews(writer http.ResponseWriter, request *http.Request) {
	reviews, err := api.reviewService.ListReviews(request.Context(), ports.CompanyReviewListFilter{
		CompanyID: request.URL.Query().Get("company_id"),
		BookType:  domain.BookType(request.URL.Query().Get("book_type")),
		Limit:     queryInt(request, "limit", 25),
		Offset:    queryInt(request, "offset", 0),
	})
	if err != nil {
		api.writeError(writer, err)
		return
	}

	api.writeJSON(writer, http.StatusOK, map[string]any{"reviews": dto.MapReviewSummaries(reviews)})
}

func (api *API) getReview(writer http.ResponseWriter, request *http.Request) {
	id, ok := pathParam(request.URL.Path, "/api/v1/reviews/", "")
	if !ok {
		http.NotFound(writer, request)
		return
	}

	review, err := api.reviewService.GetReview(request.Context(), id)
	if err != nil {
		api.writeError(writer, err)
		return
	}

	api.writeJSON(writer, http.StatusOK, map[string]any{"review": review})
}

func (api *API) getReviewDiff(writer http.ResponseWriter, request *http.Request) {
	id, ok := pathParam(request.URL.Path, "/api/v1/reviews/", "/diff")
	if !ok {
		http.NotFound(writer, request)
		return
	}

	diff, err := api.reviewService.GetReviewDiff(request.Context(), id)
	if err != nil {
		api.writeError(writer, err)
		return
	}

	api.writeJSON(writer, http.StatusOK, map[string]any{"diff": diff})
}

func (api *API) getReviewEvidence(writer http.ResponseWriter, request *http.Request) {
	id, ok := pathParam(request.URL.Path, "/api/v1/reviews/", "/evidence")
	if !ok {
		http.NotFound(writer, request)
		return
	}

	evidence, err := api.reviewService.GetReviewEvidence(request.Context(), id)
	if err != nil {
		api.writeError(writer, err)
		return
	}

	api.writeJSON(writer, http.StatusOK, map[string]any{"evidence": evidence})
}

func (api *API) listWorkflowRuns(writer http.ResponseWriter, request *http.Request) {
	runs, err := api.workflowService.ListWorkflowRuns(request.Context(), ports.WorkflowRunListFilter{
		BookType: domain.BookType(request.URL.Query().Get("book_type")),
		Status:   domain.WorkflowRunStatus(request.URL.Query().Get("status")),
		Limit:    queryInt(request, "limit", 25),
		Offset:   queryInt(request, "offset", 0),
	})
	if err != nil {
		api.writeError(writer, err)
		return
	}

	api.writeJSON(writer, http.StatusOK, map[string]any{"workflowRuns": dto.MapWorkflowRunSummaries(runs)})
}

func (api *API) getWorkflowRun(writer http.ResponseWriter, request *http.Request) {
	id, ok := pathParam(request.URL.Path, "/api/v1/workflow-runs/", "")
	if !ok {
		http.NotFound(writer, request)
		return
	}

	run, err := api.workflowService.GetWorkflowRun(request.Context(), id)
	if err != nil {
		api.writeError(writer, err)
		return
	}

	api.writeJSON(writer, http.StatusOK, map[string]any{"workflowRun": run})
}

func (api *API) getWorkflowSummary(writer http.ResponseWriter, request *http.Request) {
	id, ok := pathParam(request.URL.Path, "/api/v1/workflow-runs/", "/summary")
	if !ok {
		http.NotFound(writer, request)
		return
	}

	summary, err := api.workflowService.GetWorkflowSummary(request.Context(), id)
	if err != nil {
		api.writeError(writer, err)
		return
	}

	api.writeJSON(writer, http.StatusOK, map[string]any{"summary": summary})
}

func (api *API) startInvestingWorkflow(writer http.ResponseWriter, request *http.Request, dryRun bool) {
	var payload dto.StartInvestingWorkflowRequest
	if err := decodeJSONBody(request, &payload); err != nil {
		api.writeJSON(writer, http.StatusBadRequest, map[string]any{"error": "request body must be valid JSON"})
		return
	}

	runRequest := payload.ToPort(dryRun)
	var (
		run *domain.WorkflowRun
		err error
	)
	if dryRun {
		run, err = api.investingWorkflowService.DryRun(request.Context(), runRequest)
	} else {
		run, err = api.investingWorkflowService.Start(request.Context(), runRequest)
	}
	if err != nil {
		api.writeError(writer, err)
		return
	}

	api.writeJSON(writer, http.StatusCreated, map[string]any{"workflowRun": run})
}

func (api *API) listCapitalAllocations(writer http.ResponseWriter, request *http.Request) {
	runs, err := api.capitalAllocationService.ListRuns(request.Context(), ports.CapitalAllocationListFilter{
		BookType: domain.BookType(request.URL.Query().Get("book_type")),
		Limit:    queryInt(request, "limit", 25),
		Offset:   queryInt(request, "offset", 0),
	})
	if err != nil {
		api.writeError(writer, err)
		return
	}

	api.writeJSON(writer, http.StatusOK, map[string]any{"capitalAllocations": dto.MapCapitalAllocationSummaries(runs)})
}

func (api *API) getCapitalAllocation(writer http.ResponseWriter, request *http.Request) {
	id, ok := pathParam(request.URL.Path, "/api/v1/capital-allocations/", "")
	if !ok {
		http.NotFound(writer, request)
		return
	}

	run, err := api.capitalAllocationService.GetRun(request.Context(), id)
	if err != nil {
		api.writeError(writer, err)
		return
	}

	api.writeJSON(writer, http.StatusOK, map[string]any{"capitalAllocation": run})
}

func (api *API) getCurrentConfig(writer http.ResponseWriter, request *http.Request) {
	config, err := api.configService.CurrentConfig(request.Context())
	if err != nil {
		api.writeError(writer, err)
		return
	}

	api.writeJSON(writer, http.StatusOK, map[string]any{"config": config})
}

func (api *API) listConfigSnapshots(writer http.ResponseWriter, request *http.Request) {
	snapshots, err := api.configService.ListSnapshots(request.Context(), ports.ConfigSnapshotListFilter{
		BookType: domain.BookType(request.URL.Query().Get("book_type")),
		Limit:    queryInt(request, "limit", 25),
		Offset:   queryInt(request, "offset", 0),
	})
	if err != nil {
		api.writeError(writer, err)
		return
	}

	api.writeJSON(writer, http.StatusOK, map[string]any{"configSnapshots": snapshots})
}

func (api *API) getConfigSnapshot(writer http.ResponseWriter, request *http.Request) {
	id, ok := pathParam(request.URL.Path, "/api/v1/config/snapshots/", "")
	if !ok {
		http.NotFound(writer, request)
		return
	}

	snapshot, err := api.configService.GetSnapshot(request.Context(), id)
	if err != nil {
		api.writeError(writer, err)
		return
	}

	api.writeJSON(writer, http.StatusOK, map[string]any{"configSnapshot": snapshot})
}

func (api *API) createOverride(writer http.ResponseWriter, request *http.Request) {
	var payload dto.CreateManualOverrideRequest
	if err := decodeJSONBody(request, &payload); err != nil {
		api.writeJSON(writer, http.StatusBadRequest, map[string]any{"error": "request body must be valid JSON"})
		return
	}

	now := time.Now().UTC()
	override := &domain.ManualOverride{
		CompanyID:        payload.CompanyID,
		ReviewID:         payload.ReviewID,
		BookType:         payload.BookType,
		OriginalAction:   payload.OriginalAction,
		OverriddenAction: payload.OverriddenAction,
		OverrideReason:   payload.OverrideReason,
		OverrideBy:       payload.OverrideBy,
		OverrideDate:     now,
		SchemaVersion:    domain.SchemaVersionV1Alpha1,
		CreatedAt:        now,
	}

	created, err := api.overrideService.CreateOverride(request.Context(), override)
	if err != nil {
		api.writeError(writer, err)
		return
	}

	api.writeJSON(writer, http.StatusCreated, map[string]any{"override": created})
}

func (api *API) listOverrides(writer http.ResponseWriter, request *http.Request) {
	overrides, err := api.overrideService.ListOverrides(request.Context(), ports.ManualOverrideListFilter{
		CompanyID: request.URL.Query().Get("company_id"),
		ReviewID:  request.URL.Query().Get("review_id"),
		BookType:  domain.BookType(request.URL.Query().Get("book_type")),
		Limit:     queryInt(request, "limit", 25),
		Offset:    queryInt(request, "offset", 0),
	})
	if err != nil {
		api.writeError(writer, err)
		return
	}

	api.writeJSON(writer, http.StatusOK, map[string]any{"overrides": dto.MapManualOverrideSummaries(overrides)})
}

func (api *API) getOverride(writer http.ResponseWriter, request *http.Request) {
	id, ok := pathParam(request.URL.Path, "/api/v1/overrides/", "")
	if !ok {
		http.NotFound(writer, request)
		return
	}

	override, err := api.overrideService.GetOverride(request.Context(), id)
	if err != nil {
		api.writeError(writer, err)
		return
	}

	api.writeJSON(writer, http.StatusOK, map[string]any{"override": override})
}

func (api *API) listPositions(writer http.ResponseWriter, request *http.Request) {
	positions, err := api.projectionService.ListPositions(request.Context(), ports.PositionListFilter{
		Limit:  queryInt(request, "limit", 25),
		Offset: queryInt(request, "offset", 0),
	})
	if err != nil {
		api.writeError(writer, err)
		return
	}

	api.writeJSON(writer, http.StatusOK, map[string]any{"positions": dto.MapPositionSummaries(positions)})
}

func (api *API) listPositionsByBook(writer http.ResponseWriter, request *http.Request) {
	bookType, ok := pathParam(request.URL.Path, "/api/v1/positions/", "")
	if !ok {
		http.NotFound(writer, request)
		return
	}

	positions, err := api.projectionService.ListPositions(request.Context(), ports.PositionListFilter{
		BookType: domain.BookType(bookType),
		Limit:    queryInt(request, "limit", 25),
		Offset:   queryInt(request, "offset", 0),
	})
	if err != nil {
		api.writeError(writer, err)
		return
	}

	api.writeJSON(writer, http.StatusOK, map[string]any{"positions": dto.MapPositionSummaries(positions)})
}

func (api *API) writeError(writer http.ResponseWriter, err error) {
	status := http.StatusInternalServerError
	switch {
	case errors.Is(err, platformservice.ErrNotFound):
		status = http.StatusNotFound
	case errors.Is(err, platformservice.ErrImmutableReview):
		status = http.StatusConflict
	case errors.Is(err, context.DeadlineExceeded):
		status = http.StatusGatewayTimeout
	}

	api.writeJSON(writer, status, map[string]any{"error": err.Error()})
}

func (api *API) writeJSON(writer http.ResponseWriter, status int, payload any) {
	writer.Header().Set("Content-Type", "application/json")
	writer.WriteHeader(status)
	_ = json.NewEncoder(writer).Encode(payload)
}

func decodeJSONBody(request *http.Request, out any) error {
	defer request.Body.Close()
	return json.NewDecoder(request.Body).Decode(out)
}

func pathMatches(path, prefix, suffix string) bool {
	if !strings.HasPrefix(path, prefix) {
		return false
	}
	if suffix != "" && !strings.HasSuffix(path, suffix) {
		return false
	}

	trimmed := strings.TrimPrefix(path, prefix)
	trimmed = strings.TrimSuffix(trimmed, suffix)
	trimmed = strings.Trim(trimmed, "/")
	return trimmed != "" && !strings.Contains(trimmed, "/")
}

func pathParam(path, prefix, suffix string) (string, bool) {
	if !pathMatches(path, prefix, suffix) {
		return "", false
	}

	trimmed := strings.TrimPrefix(path, prefix)
	trimmed = strings.TrimSuffix(trimmed, suffix)
	return strings.Trim(trimmed, "/"), true
}

func queryInt(request *http.Request, name string, fallback int) int {
	raw := strings.TrimSpace(request.URL.Query().Get(name))
	if raw == "" {
		return fallback
	}
	value, err := strconv.Atoi(raw)
	if err != nil || value < 0 {
		return fallback
	}

	return value
}
