package app

import (
	"errors"
	"net/http"

	"goserver/internal/domain/allocation"
	platformrepo "goserver/internal/platform/repository"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

func (api *API) listCapitalAllocations(writer http.ResponseWriter, request *http.Request) {
	if api.allocations == nil {
		writeError(writer, errors.New("capital allocation repository is required"))
		return
	}
	filter, options, err := api.allocationListFilter(request)
	if err != nil {
		writeError(writer, err)
		return
	}
	result, err := api.allocations.List(request.Context(), filter, options)
	if err != nil {
		writeError(writer, err)
		return
	}
	writeJSON(writer, http.StatusOK, PagedResponseDTO[AllocationRunListItemDTO]{
		Items: mapAllocationRunListItems(result.Items),
		Page:  pageDTO(result.Page),
	})
}

func (api *API) getCapitalAllocation(writer http.ResponseWriter, request *http.Request) {
	id, ok, err := pathObjectID(request, "/api/v1/capital-allocations/", "")
	if err != nil {
		writeError(writer, err)
		return
	}
	if !ok {
		http.NotFound(writer, request)
		return
	}
	if api.allocations == nil {
		writeError(writer, errors.New("capital allocation repository is required"))
		return
	}
	run, err := api.allocations.GetByID(request.Context(), id)
	if err != nil {
		writeError(writer, err)
		return
	}
	writeJSON(writer, http.StatusOK, map[string]any{"capitalAllocation": mapAllocationRunDetail(run)})
}

func (api *API) getWorkflowCapitalAllocation(writer http.ResponseWriter, request *http.Request) {
	workflowRunID, ok, err := pathObjectID(request, "/api/v1/workflow-runs/", "/capital-allocation")
	if err != nil {
		writeError(writer, err)
		return
	}
	if !ok {
		http.NotFound(writer, request)
		return
	}
	if api.allocations == nil {
		writeError(writer, errors.New("capital allocation repository is required"))
		return
	}
	result, err := api.allocations.List(request.Context(), platformrepo.CapitalAllocationRunFilter{
		WorkflowRunIDs: []primitive.ObjectID{workflowRunID},
	}, platformrepo.CapitalAllocationRunListOptions{
		Pagination: platformrepo.PageOptions{PageSize: 1},
		Sort:       platformrepo.CapitalAllocationRunSortOption{By: platformrepo.CapitalAllocationRunSortByAllocationDate, Order: platformrepo.SortOrderDescending},
	})
	if err != nil {
		writeError(writer, err)
		return
	}
	if len(result.Items) == 0 {
		writeError(writer, platformrepo.ErrNotFound)
		return
	}
	writeJSON(writer, http.StatusOK, map[string]any{"capitalAllocation": mapAllocationRunDetail(result.Items[0])})
}

func (api *API) allocationListFilter(request *http.Request) (platformrepo.CapitalAllocationRunFilter, platformrepo.CapitalAllocationRunListOptions, error) {
	pagination, err := parsePagination(request)
	if err != nil {
		return platformrepo.CapitalAllocationRunFilter{}, platformrepo.CapitalAllocationRunListOptions{}, err
	}
	workflowRunID, err := queryObjectID(request, "workflow_run_id")
	if err != nil {
		return platformrepo.CapitalAllocationRunFilter{}, platformrepo.CapitalAllocationRunListOptions{}, err
	}
	bookType, err := queryBookType(request)
	if err != nil {
		return platformrepo.CapitalAllocationRunFilter{}, platformrepo.CapitalAllocationRunListOptions{}, err
	}
	allocationDate, err := parseTimeRange(request, "allocation_date_from", "allocation_date_to")
	if err != nil {
		return platformrepo.CapitalAllocationRunFilter{}, platformrepo.CapitalAllocationRunListOptions{}, err
	}
	return platformrepo.CapitalAllocationRunFilter{
			WorkflowRunIDs: oneObjectID(workflowRunID),
			BookTypes:      oneBookType(bookType),
			AllocationDate: allocationDate,
		}, platformrepo.CapitalAllocationRunListOptions{
			Pagination: pagination,
			Sort:       platformrepo.CapitalAllocationRunSortOption{By: platformrepo.CapitalAllocationRunSortByAllocationDate, Order: platformrepo.SortOrderDescending},
		}, nil
}

func allocationItemsForCompany(runs []*allocation.CapitalAllocationRun, companyID primitive.ObjectID) []AllocationItemDTO {
	var items []AllocationItemDTO
	for _, run := range runs {
		if run == nil {
			continue
		}
		for _, item := range run.Items {
			if item.CompanyID == companyID {
				items = append(items, mapAllocationItems([]allocation.CapitalAllocationItem{item})...)
			}
		}
	}
	return items
}

func allocationHistoryForCompany(runs []*allocation.CapitalAllocationRun, companyID primitive.ObjectID) []AllocationHistoryItemDTO {
	var history []AllocationHistoryItemDTO
	for _, run := range runs {
		if run == nil {
			continue
		}
		for _, item := range run.Items {
			if item.CompanyID != companyID {
				continue
			}
			history = append(history, AllocationHistoryItemDTO{
				AllocationRunID: objectIDString(run.ID),
				WorkflowRunID:   objectIDString(run.WorkflowRunID),
				AllocationDate:  run.AllocationDate,
				Amount:          item.RecommendedAllocationAmount,
				Blocked:         item.BlockedByConstraint,
				Reason:          firstNonEmpty(item.ConstraintReason, item.AllocationReason),
			})
		}
	}
	return history
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if value != "" {
			return value
		}
	}
	return ""
}
