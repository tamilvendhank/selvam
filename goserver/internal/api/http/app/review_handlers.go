package app

import (
	"errors"
	"net/http"

	domaincommon "goserver/internal/domain/common"
	"goserver/internal/domain/review"
	platformrepo "goserver/internal/platform/repository"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

func (api *API) listReviews(writer http.ResponseWriter, request *http.Request) {
	if api.reviews == nil {
		writeError(writer, errors.New("review repository is required"))
		return
	}
	filter, options, err := api.reviewListFilter(request)
	if err != nil {
		writeError(writer, err)
		return
	}
	result, err := api.reviews.ListSummaries(request.Context(), filter, options)
	if err != nil {
		writeError(writer, err)
		return
	}
	writeJSON(writer, http.StatusOK, PagedResponseDTO[ReviewListItemDTO]{
		Items: mapReviewListItemsFromSummaries(result.Items),
		Page:  pageDTO(result.Page),
	})
}

func (api *API) listCompanyReviews(writer http.ResponseWriter, request *http.Request) {
	companyID, ok, err := pathObjectID(request, "/api/v1/companies/", "/reviews")
	if err != nil {
		writeError(writer, err)
		return
	}
	if !ok {
		http.NotFound(writer, request)
		return
	}
	if api.reviews == nil {
		writeError(writer, errors.New("review repository is required"))
		return
	}
	filter, options, err := api.reviewListFilter(request)
	if err != nil {
		writeError(writer, err)
		return
	}
	filter.CompanyIDs = []primitive.ObjectID{companyID}
	result, err := api.reviews.ListSummaries(request.Context(), filter, options)
	if err != nil {
		writeError(writer, err)
		return
	}
	writeJSON(writer, http.StatusOK, PagedResponseDTO[ReviewListItemDTO]{
		Items: mapReviewListItemsFromSummaries(result.Items),
		Page:  pageDTO(result.Page),
	})
}

func (api *API) getReview(writer http.ResponseWriter, request *http.Request) {
	review, ok := api.loadReviewFromPath(writer, request, "")
	if !ok {
		return
	}
	writeJSON(writer, http.StatusOK, map[string]any{"review": mapReviewDetail(review)})
}

func (api *API) getReviewScorecard(writer http.ResponseWriter, request *http.Request) {
	review, ok := api.loadReviewFromPath(writer, request, "/scorecard")
	if !ok {
		return
	}
	writeJSON(writer, http.StatusOK, map[string]any{"scorecard": mapScorecard(review)})
}

func (api *API) getReviewEvidence(writer http.ResponseWriter, request *http.Request) {
	review, ok := api.loadReviewFromPath(writer, request, "/evidence")
	if !ok {
		return
	}
	sectionFilter := domaincommon.SectionName(request.URL.Query().Get("section_name"))
	if sectionFilter != "" && !sectionFilter.IsValid() {
		writeError(writer, badRequestf("invalid section_name %q", sectionFilter))
		return
	}
	subScoreFilter := domaincommon.SubScoreName(request.URL.Query().Get("sub_score_name"))
	if subScoreFilter != "" && !subScoreFilter.IsValid() {
		writeError(writer, badRequestf("invalid sub_score_name %q", subScoreFilter))
		return
	}
	sourceType := domaincommon.EvidenceSourceType(request.URL.Query().Get("source_type"))
	if sourceType != "" && !sourceType.IsValid() {
		writeError(writer, badRequestf("invalid source_type %q", sourceType))
		return
	}
	direction := domaincommon.EvidenceDirection(request.URL.Query().Get("evidence_direction"))
	if direction != "" && !direction.IsValid() {
		writeError(writer, badRequestf("invalid evidence_direction %q", direction))
		return
	}
	var refs []EvidenceReferenceDTO
	for _, section := range review.Sections {
		if sectionFilter != "" && section.SectionName != sectionFilter {
			continue
		}
		for _, ref := range mapEvidenceRefs(section.SectionName, "", section.EvidenceRefs) {
			if sourceType != "" && ref.SourceType != sourceType {
				continue
			}
			if direction != "" && ref.EvidenceDirection != direction {
				continue
			}
			if subScoreFilter != "" {
				continue
			}
			refs = append(refs, ref)
		}
		for _, subScore := range section.SubScores {
			if subScoreFilter != "" && subScore.SubScoreName != subScoreFilter {
				continue
			}
			for _, refID := range subScore.EvidenceRefIDs {
				refs = append(refs, EvidenceReferenceDTO{
					EvidenceID:   objectIDString(refID),
					SectionName:  section.SectionName,
					SubScoreName: subScore.SubScoreName,
				})
			}
		}
	}
	writeJSON(writer, http.StatusOK, map[string]any{"evidenceRefs": refs})
}

func (api *API) getReviewDiff(writer http.ResponseWriter, request *http.Request) {
	review, ok := api.loadReviewFromPath(writer, request, "/diff")
	if !ok {
		return
	}
	if review.ChangeLog != nil {
		writeJSON(writer, http.StatusOK, map[string]any{"diff": mapReviewChangeLog(review.ChangeLog)})
		return
	}
	diff := ReviewDiffDTO{}
	if api.reviews != nil {
		previous, err := api.reviews.GetPreviousFinalizedByCompanyAndBook(request.Context(), review.CompanyID, review.BookType, platformrepo.PreviousFinalizedReviewLookup{
			ExcludeReviewID:  review.ID,
			BeforeReviewDate: &review.ReviewDate,
		})
		if err == nil && previous != nil {
			diff.PreviousReviewID = objectIDString(previous.ID)
			diff.WeightedTotalScoreChange = review.WeightedTotalScore - previous.WeightedTotalScore
			if review.FinalActionAfterReview != previous.FinalActionAfterReview {
				diff.ActionChange = string(previous.FinalActionAfterReview) + " -> " + string(review.FinalActionAfterReview)
			}
			if review.FinalBucketAfterReview != previous.FinalBucketAfterReview {
				diff.BucketChange = string(previous.FinalBucketAfterReview) + " -> " + string(review.FinalBucketAfterReview)
			}
			diff.ChangeSummary = review.WhatChangedSummary
		}
	}
	writeJSON(writer, http.StatusOK, map[string]any{"diff": diff})
}

func (api *API) loadReviewFromPath(writer http.ResponseWriter, request *http.Request, suffix string) (*review.CompanyReview, bool) {
	id, ok, err := pathObjectID(request, "/api/v1/reviews/", suffix)
	if err != nil {
		writeError(writer, err)
		return nil, false
	}
	if !ok {
		http.NotFound(writer, request)
		return nil, false
	}
	if api.reviews == nil {
		writeError(writer, errors.New("review repository is required"))
		return nil, false
	}
	review, err := api.reviews.GetByID(request.Context(), id)
	if err != nil {
		writeError(writer, err)
		return nil, false
	}
	return review, true
}

func (api *API) reviewListFilter(request *http.Request) (platformrepo.CompanyReviewFilter, platformrepo.CompanyReviewListOptions, error) {
	pagination, err := parsePagination(request)
	if err != nil {
		return platformrepo.CompanyReviewFilter{}, platformrepo.CompanyReviewListOptions{}, err
	}
	companyID, err := queryObjectID(request, "company_id")
	if err != nil {
		return platformrepo.CompanyReviewFilter{}, platformrepo.CompanyReviewListOptions{}, err
	}
	workflowRunID, err := queryObjectID(request, "workflow_run_id")
	if err != nil {
		return platformrepo.CompanyReviewFilter{}, platformrepo.CompanyReviewListOptions{}, err
	}
	bookType, err := queryBookType(request)
	if err != nil {
		return platformrepo.CompanyReviewFilter{}, platformrepo.CompanyReviewListOptions{}, err
	}
	status, err := queryReviewStatus(request)
	if err != nil {
		return platformrepo.CompanyReviewFilter{}, platformrepo.CompanyReviewListOptions{}, err
	}
	lifecycle, err := queryLifecycleState(request)
	if err != nil {
		return platformrepo.CompanyReviewFilter{}, platformrepo.CompanyReviewListOptions{}, err
	}
	action, err := queryActionType(request)
	if err != nil {
		return platformrepo.CompanyReviewFilter{}, platformrepo.CompanyReviewListOptions{}, err
	}
	bucket, err := queryBucket(request)
	if err != nil {
		return platformrepo.CompanyReviewFilter{}, platformrepo.CompanyReviewListOptions{}, err
	}
	ownedBefore, err := queryBoolPtr(request, "owned_before_review")
	if err != nil {
		return platformrepo.CompanyReviewFilter{}, platformrepo.CompanyReviewListOptions{}, err
	}
	finalizedOnly, err := queryBoolPtr(request, "finalized_only")
	if err != nil {
		return platformrepo.CompanyReviewFilter{}, platformrepo.CompanyReviewListOptions{}, err
	}
	pendingOnly, err := queryBoolPtr(request, "pending_only")
	if err != nil {
		return platformrepo.CompanyReviewFilter{}, platformrepo.CompanyReviewListOptions{}, err
	}
	reviewDate, err := parseTimeRange(request, "review_date_from", "review_date_to")
	if err != nil {
		return platformrepo.CompanyReviewFilter{}, platformrepo.CompanyReviewListOptions{}, err
	}
	filter := platformrepo.CompanyReviewFilter{
		CompanyIDs:        oneObjectID(companyID),
		Symbols:           optionalStringList(request.URL.Query().Get("symbol")),
		BookTypes:         oneBookType(bookType),
		WorkflowRunIDs:    oneObjectID(workflowRunID),
		ReviewStatuses:    oneReviewStatus(status),
		LifecycleStates:   oneLifecycleState(lifecycle),
		FinalActions:      oneActionType(action),
		FinalBuckets:      oneBucket(bucket),
		OwnedBeforeReview: ownedBefore,
		ReviewDate:        reviewDate,
		FinalizedOnly:     finalizedOnly != nil && *finalizedOnly,
		PendingOnly:       pendingOnly != nil && *pendingOnly,
	}
	options := platformrepo.CompanyReviewListOptions{
		Pagination: pagination,
		Sort:       platformrepo.CompanyReviewSortOption{By: platformrepo.CompanyReviewSortByReviewDate, Order: platformrepo.SortOrderDescending},
	}
	return filter, options, nil
}
