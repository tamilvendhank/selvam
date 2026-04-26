package app

import (
	"errors"
	"net/http"

	domaincommon "goserver/internal/domain/common"
	platformrepo "goserver/internal/platform/repository"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

func (api *API) listPositions(writer http.ResponseWriter, request *http.Request) {
	if api.positions == nil {
		writeError(writer, errors.New("current position repository is required"))
		return
	}
	filter, options, err := api.positionListFilter(request)
	if err != nil {
		writeError(writer, err)
		return
	}
	result, err := api.positions.List(request.Context(), filter, options)
	if err != nil {
		writeError(writer, err)
		return
	}
	writeJSON(writer, http.StatusOK, PagedResponseDTO[CurrentPositionDTO]{
		Items: mapCurrentPositions(result.Items),
		Page:  pageDTO(result.Page),
	})
}

func (api *API) listPositionsByBook(writer http.ResponseWriter, request *http.Request) {
	raw, ok := pathParam(request.URL.Path, "/api/v1/positions/", "")
	if !ok {
		http.NotFound(writer, request)
		return
	}
	bookType := domaincommon.BookType(raw)
	if !bookType.IsValid() {
		writeError(writer, badRequestf("invalid book_type %q", raw))
		return
	}
	if api.positions == nil {
		writeError(writer, errors.New("current position repository is required"))
		return
	}
	pagination, err := parsePagination(request)
	if err != nil {
		writeError(writer, err)
		return
	}
	result, err := api.positions.ListOpenByBook(request.Context(), bookType, platformrepo.CurrentPositionListOptions{
		Pagination: pagination,
		Sort:       platformrepo.CurrentPositionSortOption{By: platformrepo.CurrentPositionSortByLastUpdatedAt, Order: platformrepo.SortOrderDescending},
	})
	if err != nil {
		writeError(writer, err)
		return
	}
	writeJSON(writer, http.StatusOK, PagedResponseDTO[CurrentPositionDTO]{
		Items: mapCurrentPositions(result.Items),
		Page:  pageDTO(result.Page),
	})
}

func (api *API) listCompanyPositions(writer http.ResponseWriter, request *http.Request) {
	companyID, ok, err := pathObjectID(request, "/api/v1/companies/", "/positions")
	if err != nil {
		writeError(writer, err)
		return
	}
	if !ok {
		http.NotFound(writer, request)
		return
	}
	if api.positions == nil {
		writeError(writer, errors.New("current position repository is required"))
		return
	}
	pagination, err := parsePagination(request)
	if err != nil {
		writeError(writer, err)
		return
	}
	result, err := api.positions.List(request.Context(), platformrepo.CurrentPositionFilter{
		CompanyIDs: []primitive.ObjectID{companyID},
	}, platformrepo.CurrentPositionListOptions{
		Pagination: pagination,
		Sort:       platformrepo.CurrentPositionSortOption{By: platformrepo.CurrentPositionSortByLastUpdatedAt, Order: platformrepo.SortOrderDescending},
	})
	if err != nil {
		writeError(writer, err)
		return
	}
	writeJSON(writer, http.StatusOK, PagedResponseDTO[CurrentPositionDTO]{
		Items: mapCurrentPositions(result.Items),
		Page:  pageDTO(result.Page),
	})
}

func (api *API) positionListFilter(request *http.Request) (platformrepo.CurrentPositionFilter, platformrepo.CurrentPositionListOptions, error) {
	pagination, err := parsePagination(request)
	if err != nil {
		return platformrepo.CurrentPositionFilter{}, platformrepo.CurrentPositionListOptions{}, err
	}
	companyID, err := queryObjectID(request, "company_id")
	if err != nil {
		return platformrepo.CurrentPositionFilter{}, platformrepo.CurrentPositionListOptions{}, err
	}
	bookType, err := queryBookType(request)
	if err != nil {
		return platformrepo.CurrentPositionFilter{}, platformrepo.CurrentPositionListOptions{}, err
	}
	isOpen, err := queryBoolPtr(request, "is_open")
	if err != nil {
		return platformrepo.CurrentPositionFilter{}, platformrepo.CurrentPositionListOptions{}, err
	}
	return platformrepo.CurrentPositionFilter{
			CompanyIDs: oneObjectID(companyID),
			BookTypes:  oneBookType(bookType),
			IsOpen:     isOpen,
		}, platformrepo.CurrentPositionListOptions{
			Pagination: pagination,
			Sort:       platformrepo.CurrentPositionSortOption{By: platformrepo.CurrentPositionSortByLastUpdatedAt, Order: platformrepo.SortOrderDescending},
		}, nil
}
