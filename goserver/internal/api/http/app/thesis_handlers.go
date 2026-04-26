package app

import (
	"errors"
	"net/http"

	platformrepo "goserver/internal/platform/repository"
)

func (api *API) listTheses(writer http.ResponseWriter, request *http.Request) {
	if api.theses == nil {
		writeError(writer, errors.New("thesis repository is required"))
		return
	}
	filter, options, err := api.thesisListFilter(request)
	if err != nil {
		writeError(writer, err)
		return
	}
	result, err := api.theses.List(request.Context(), filter, options)
	if err != nil {
		writeError(writer, err)
		return
	}
	writeJSON(writer, http.StatusOK, PagedResponseDTO[ThesisListItemDTO]{
		Items: mapThesisListItems(result.Items),
		Page:  pageDTO(result.Page),
	})
}

func (api *API) getThesis(writer http.ResponseWriter, request *http.Request) {
	id, ok, err := pathObjectID(request, "/api/v1/theses/", "")
	if err != nil {
		writeError(writer, err)
		return
	}
	if !ok {
		http.NotFound(writer, request)
		return
	}
	if api.theses == nil {
		writeError(writer, errors.New("thesis repository is required"))
		return
	}
	thesis, err := api.theses.GetByID(request.Context(), id)
	if err != nil {
		writeError(writer, err)
		return
	}
	writeJSON(writer, http.StatusOK, map[string]any{"thesis": mapThesisDetail(thesis)})
}

func (api *API) getCompanyActiveThesis(writer http.ResponseWriter, request *http.Request) {
	companyID, ok, err := pathObjectID(request, "/api/v1/companies/", "/thesis")
	if err != nil {
		writeError(writer, err)
		return
	}
	if !ok {
		http.NotFound(writer, request)
		return
	}
	if api.theses == nil {
		writeError(writer, errors.New("thesis repository is required"))
		return
	}
	thesis, err := api.theses.GetActiveByCompanyID(request.Context(), companyID)
	if err != nil {
		writeError(writer, err)
		return
	}
	writeJSON(writer, http.StatusOK, map[string]any{"thesis": mapThesisDetail(thesis)})
}

func (api *API) listCompanyThesisHistory(writer http.ResponseWriter, request *http.Request) {
	companyID, ok, err := pathObjectID(request, "/api/v1/companies/", "/thesis/history")
	if err != nil {
		writeError(writer, err)
		return
	}
	if !ok {
		http.NotFound(writer, request)
		return
	}
	if api.theses == nil {
		writeError(writer, errors.New("thesis repository is required"))
		return
	}
	pagination, err := parsePagination(request)
	if err != nil {
		writeError(writer, err)
		return
	}
	result, err := api.theses.ListByCompanyID(request.Context(), companyID, platformrepo.InvestmentThesisListOptions{
		Pagination: pagination,
		Sort:       platformrepo.InvestmentThesisSortOption{By: platformrepo.InvestmentThesisSortByVersion, Order: platformrepo.SortOrderDescending},
	})
	if err != nil {
		writeError(writer, err)
		return
	}
	items := make([]ThesisHistoryItemDTO, 0, len(result.Items))
	for _, thesis := range result.Items {
		items = append(items, mapThesisHistoryItem(thesis))
	}
	writeJSON(writer, http.StatusOK, PagedResponseDTO[ThesisHistoryItemDTO]{
		Items: items,
		Page:  pageDTO(result.Page),
	})
}

func (api *API) thesisListFilter(request *http.Request) (platformrepo.InvestmentThesisFilter, platformrepo.InvestmentThesisListOptions, error) {
	pagination, err := parsePagination(request)
	if err != nil {
		return platformrepo.InvestmentThesisFilter{}, platformrepo.InvestmentThesisListOptions{}, err
	}
	companyID, err := queryObjectID(request, "company_id")
	if err != nil {
		return platformrepo.InvestmentThesisFilter{}, platformrepo.InvestmentThesisListOptions{}, err
	}
	status, err := queryThesisStatus(request)
	if err != nil {
		return platformrepo.InvestmentThesisFilter{}, platformrepo.InvestmentThesisListOptions{}, err
	}
	updatedAt, err := parseTimeRange(request, "updated_from", "updated_to")
	if err != nil {
		return platformrepo.InvestmentThesisFilter{}, platformrepo.InvestmentThesisListOptions{}, err
	}
	return platformrepo.InvestmentThesisFilter{
			CompanyIDs:     oneObjectID(companyID),
			ThesisStatuses: oneThesisStatus(status),
			UpdatedAt:      updatedAt,
		}, platformrepo.InvestmentThesisListOptions{
			Pagination: pagination,
			Sort:       platformrepo.InvestmentThesisSortOption{By: platformrepo.InvestmentThesisSortByUpdatedAt, Order: platformrepo.SortOrderDescending},
		}, nil
}
