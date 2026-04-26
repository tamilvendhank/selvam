package app

import (
	"errors"
	"net/http"
	"strings"

	domaincommon "goserver/internal/domain/common"
	platformrepo "goserver/internal/platform/repository"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

func (api *API) listCompanies(writer http.ResponseWriter, request *http.Request) {
	if api.companies == nil {
		writeError(writer, errors.New("company repository is required"))
		return
	}
	pagination, err := parsePagination(request)
	if err != nil {
		writeError(writer, err)
		return
	}
	inInvesting, err := queryBoolPtr(request, "is_in_investing_universe")
	if err != nil {
		writeError(writer, err)
		return
	}
	inTrading, err := queryBoolPtr(request, "is_in_trading_universe")
	if err != nil {
		writeError(writer, err)
		return
	}
	statusActive, err := queryBoolPtr(request, "status_active")
	if err != nil {
		writeError(writer, err)
		return
	}
	result, err := api.companies.List(request.Context(), platformrepo.CompanyFilter{
		Symbols:             optionalStringList(request.URL.Query().Get("symbol")),
		Exchange:            request.URL.Query().Get("exchange"),
		Sector:              request.URL.Query().Get("sector"),
		Industry:            request.URL.Query().Get("industry"),
		SubIndustry:         request.URL.Query().Get("sub_industry"),
		MarketCapBucket:     request.URL.Query().Get("market_cap_bucket"),
		InInvestingUniverse: inInvesting,
		InTradingUniverse:   inTrading,
		StatusActive:        statusActive,
	}, platformrepo.CompanyListOptions{
		Pagination: pagination,
		Sort:       platformrepo.CompanySortOption{By: platformrepo.CompanySortBySymbol, Order: platformrepo.SortOrderAscending},
	})
	if err != nil {
		writeError(writer, err)
		return
	}
	writeJSON(writer, http.StatusOK, PagedResponseDTO[CompanyListItemDTO]{
		Items: mapCompanyListItems(result.Items),
		Page:  pageDTO(result.Page),
	})
}

func (api *API) getCompany(writer http.ResponseWriter, request *http.Request) {
	id, ok, err := pathObjectID(request, "/api/v1/companies/", "")
	if err != nil {
		writeError(writer, err)
		return
	}
	if !ok {
		http.NotFound(writer, request)
		return
	}
	if api.companies == nil {
		writeError(writer, errors.New("company repository is required"))
		return
	}
	company, err := api.companies.GetByID(request.Context(), id)
	if err != nil {
		writeError(writer, err)
		return
	}
	detail := CompanyDetailDTO{
		CompanyID:             objectIDString(company.ID),
		Symbol:                company.Symbol,
		Exchange:              company.Exchange,
		CompanyName:           company.CompanyName,
		Sector:                company.Sector,
		Industry:              company.Industry,
		SubIndustry:           company.SubIndustry,
		BusinessSummary:       company.BusinessSummary,
		ListingDate:           company.ListingDate,
		MarketCapBucket:       company.MarketCapBucket,
		IsInInvestingUniverse: company.IsInInvestingUniverse,
		IsInTradingUniverse:   company.IsInTradingUniverse,
		StatusActive:          company.StatusActive,
		CreatedAt:             company.CreatedAt,
		UpdatedAt:             company.UpdatedAt,
	}
	if api.reviews != nil {
		detail.LatestInvestingReview = api.latestReviewSummary(request, id, domaincommon.BookTypeInvesting)
		detail.LatestTradingReview = api.latestReviewSummary(request, id, domaincommon.BookTypeTrading)
	}
	if api.theses != nil {
		if thesis, err := api.theses.GetActiveByCompanyID(request.Context(), id); err == nil {
			mapped := mapThesisListItem(thesis)
			detail.ActiveThesis = &mapped
		}
	}
	if api.positions != nil {
		if positions, err := api.positions.List(request.Context(), platformrepo.CurrentPositionFilter{
			CompanyIDs: []primitive.ObjectID{id},
		}, platformrepo.CurrentPositionListOptions{
			Pagination: platformrepo.PageOptions{PageSize: maxPageSize},
			Sort:       platformrepo.CurrentPositionSortOption{By: platformrepo.CurrentPositionSortByLastUpdatedAt, Order: platformrepo.SortOrderDescending},
		}); err == nil {
			detail.CurrentPositions = mapCurrentPositions(positions.Items)
		}
	}
	if api.allocations != nil {
		if allocations, err := api.allocations.List(request.Context(), platformrepo.CapitalAllocationRunFilter{
			ContainsCompanyID: &id,
		}, platformrepo.CapitalAllocationRunListOptions{
			Pagination: platformrepo.PageOptions{PageSize: 5},
			Sort:       platformrepo.CapitalAllocationRunSortOption{By: platformrepo.CapitalAllocationRunSortByAllocationDate, Order: platformrepo.SortOrderDescending},
		}); err == nil {
			detail.LatestAllocationRelevance = allocationItemsForCompany(allocations.Items, id)
		}
	}
	writeJSON(writer, http.StatusOK, map[string]any{"company": detail})
}

func (api *API) getCompanyHistorySummary(writer http.ResponseWriter, request *http.Request) {
	id, ok, err := pathObjectID(request, "/api/v1/companies/", "/history-summary")
	if err != nil {
		writeError(writer, err)
		return
	}
	if !ok {
		http.NotFound(writer, request)
		return
	}
	summary := CompanyHistorySummaryDTO{CompanyID: objectIDString(id)}
	if api.reviews != nil {
		if reviews, err := api.reviews.ListByCompany(request.Context(), id, platformrepo.CompanyReviewListOptions{
			Pagination: platformrepo.PageOptions{PageSize: 20},
			Sort:       platformrepo.CompanyReviewSortOption{By: platformrepo.CompanyReviewSortByReviewDate, Order: platformrepo.SortOrderDescending},
		}); err == nil {
			summary.ReviewCount = len(reviews.Items)
			for _, review := range reviews.Items {
				if review == nil {
					continue
				}
				if summary.LatestReviewDate == nil || review.ReviewDate.After(*summary.LatestReviewDate) {
					date := review.ReviewDate
					summary.LatestReviewDate = &date
				}
				summary.ScoreTrend = append(summary.ScoreTrend, ScorePointDTO{
					ReviewID:           objectIDString(review.ID),
					BookType:           review.BookType,
					ReviewDate:         review.ReviewDate,
					WeightedTotalScore: review.WeightedTotalScore,
				})
				summary.ActionHistory = append(summary.ActionHistory, ActionHistoryItemDTO{
					ReviewID: objectIDString(review.ID),
					Date:     review.ReviewDate,
					Action:   review.FinalActionAfterReview,
					Bucket:   review.FinalBucketAfterReview,
					Summary:  review.ActionRationaleSummary,
				})
			}
		}
	}
	if api.theses != nil {
		if theses, err := api.theses.ListByCompanyID(request.Context(), id, platformrepo.InvestmentThesisListOptions{
			Pagination: platformrepo.PageOptions{PageSize: 20},
			Sort:       platformrepo.InvestmentThesisSortOption{By: platformrepo.InvestmentThesisSortByVersion, Order: platformrepo.SortOrderDescending},
		}); err == nil {
			for _, thesis := range theses.Items {
				summary.ThesisHistory = append(summary.ThesisHistory, mapThesisHistoryItem(thesis))
			}
		}
	}
	if api.allocations != nil {
		if allocations, err := api.allocations.List(request.Context(), platformrepo.CapitalAllocationRunFilter{
			ContainsCompanyID: &id,
		}, platformrepo.CapitalAllocationRunListOptions{
			Pagination: platformrepo.PageOptions{PageSize: 20},
			Sort:       platformrepo.CapitalAllocationRunSortOption{By: platformrepo.CapitalAllocationRunSortByAllocationDate, Order: platformrepo.SortOrderDescending},
		}); err == nil {
			summary.AllocationHistory = allocationHistoryForCompany(allocations.Items, id)
		}
	}
	writeJSON(writer, http.StatusOK, map[string]any{"summary": summary})
}

func (api *API) latestReviewSummary(request *http.Request, companyID primitive.ObjectID, bookType domaincommon.BookType) *ReviewSummaryDTO {
	review, err := api.reviews.GetLatestByCompanyAndBook(request.Context(), companyID, bookType, platformrepo.LatestCompanyReviewOptions{
		FinalizedOnly:     true,
		IncludeSuperseded: false,
	})
	if err != nil {
		return nil
	}
	mapped := mapReviewSummary(review)
	return &mapped
}

func optionalStringList(raw string) []string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil
	}
	return []string{raw}
}
