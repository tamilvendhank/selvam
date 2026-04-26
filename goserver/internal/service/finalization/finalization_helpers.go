package finalization

import (
	"context"
	"errors"
	"fmt"

	domaincommon "goserver/internal/domain/common"
	domainreview "goserver/internal/domain/review"
	platformrepo "goserver/internal/platform/repository"
	servicecommon "goserver/internal/service/common"
	workerservice "goserver/internal/service/worker"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

var errFinalizationSkipped = errors.New("finalization skipped")

type finalizationRequestOptions struct {
	WorkflowRunID         primitive.ObjectID
	CompanyID             primitive.ObjectID
	BookType              domaincommon.BookType
	Force                 bool
	SupersedePrior        bool
	DryRun                bool
	InitiatedBy           string
	CorrelationID         string
	TreatIneligibleAsSkip bool
}

type finalizeOneOutcome struct {
	ReviewID            primitive.ObjectID
	ReviewRef           servicecommon.ReviewRef
	Finalized           bool
	Skipped             bool
	AlreadyFinal        bool
	AlreadyInvalid      bool
	DryRun              bool
	SupersededReviewIDs []primitive.ObjectID
	PartialFailures     []servicecommon.PartialFailure
}

func (service *reviewFinalizationService) maxFinalizationReviews(requested int) int {
	if requested > 0 && requested < service.config.MaxPageSize {
		return requested
	}
	if requested > service.config.MaxPageSize {
		return service.config.MaxPageSize
	}
	if service.config.DefaultMaxReviews > service.config.MaxPageSize {
		return service.config.MaxPageSize
	}
	return service.config.DefaultMaxReviews
}

func (service *reviewFinalizationService) discoverFinalizableReviewIDs(
	ctx context.Context,
	request FinalizeEligibleReviewsRequest,
) ([]primitive.ObjectID, bool, error) {
	if !request.ReviewID.IsZero() {
		return []primitive.ObjectID{request.ReviewID}, false, nil
	}

	limit := service.maxFinalizationReviews(request.MaxReviews)
	if service.discovery != nil {
		discovered, err := service.discovery.DiscoverFinalizableReviews(ctx, workerservice.DiscoverFinalizableReviewsRequest{
			DiscoveryRequestBase: workerservice.DiscoveryRequestBase{
				WorkflowRunID: request.WorkflowRunID,
				BookType:      request.BookType,
				MaxItems:      limit,
			},
			CompanyID: request.CompanyID,
			Force:     request.Force,
		})
		if err != nil {
			return nil, false, fmt.Errorf("discover finalizable reviews: %w", err)
		}
		return reviewIDsFromRefs(discovered.Reviews, limit), discovered.HasMore, nil
	}

	if service.reviews == nil {
		return nil, false, fmt.Errorf("discover finalizable reviews: review repository is required")
	}
	filter := platformrepo.CompanyReviewFilter{
		LifecycleStates: []domaincommon.ReviewLifecycleState{
			domaincommon.ReviewLifecycleStateAIValidated,
		},
		ReviewStatuses: []domaincommon.ReviewStatus{
			domaincommon.ReviewStatusDraft,
		},
		PendingOnly: true,
	}
	if !request.WorkflowRunID.IsZero() {
		filter.WorkflowRunIDs = []primitive.ObjectID{request.WorkflowRunID}
	}
	if !request.CompanyID.IsZero() {
		filter.CompanyIDs = []primitive.ObjectID{request.CompanyID}
	}
	if request.BookType != "" {
		filter.BookTypes = []domaincommon.BookType{request.BookType}
	}

	result, err := service.reviews.List(ctx, filter, platformrepo.CompanyReviewListOptions{
		Pagination: platformrepo.PageOptions{PageSize: limit},
		Sort:       platformrepo.CompanyReviewSortOption{By: platformrepo.CompanyReviewSortByUpdatedAt, Order: platformrepo.SortOrderAscending},
	})
	if err != nil {
		return nil, false, fmt.Errorf("list finalizable reviews: %w", err)
	}
	ids := make([]primitive.ObjectID, 0, len(result.Items))
	for _, review := range result.Items {
		if review != nil {
			ids = append(ids, review.ID)
		}
	}
	return ids, result.Page.HasMore, nil
}

func (service *reviewFinalizationService) loadFinalizationContext(
	ctx context.Context,
	reviewID primitive.ObjectID,
) (*domainreview.CompanyReview, error) {
	if service.reviews == nil {
		return nil, fmt.Errorf("finalize review %s: review repository is required", reviewID.Hex())
	}
	review, err := service.reviews.GetByID(ctx, reviewID)
	if err != nil {
		return nil, fmt.Errorf("finalize review %s: load review: %w", reviewID.Hex(), err)
	}
	if review == nil {
		return nil, fmt.Errorf("finalize review %s: %w", reviewID.Hex(), platformrepo.ErrNotFound)
	}
	return review, nil
}

func reviewIDsFromRefs(refs []servicecommon.ReviewRef, limit int) []primitive.ObjectID {
	ids := make([]primitive.ObjectID, 0, len(refs))
	for _, ref := range refs {
		if ref.ID.IsZero() {
			continue
		}
		ids = append(ids, ref.ID)
		if len(ids) >= limit {
			break
		}
	}
	return ids
}

func reviewRef(review *domainreview.CompanyReview) servicecommon.ReviewRef {
	if review == nil {
		return servicecommon.ReviewRef{}
	}
	return servicecommon.ReviewRef{
		ID:             review.ID,
		CompanyID:      review.CompanyID,
		WorkflowRunID:  review.WorkflowRunID,
		BookType:       review.BookType,
		Status:         review.ReviewStatus,
		LifecycleState: review.ReviewLifecycleState,
		Symbol:         review.Symbol,
	}
}

func isFinalizationSkip(err error) bool {
	return errors.Is(err, errFinalizationSkipped)
}

func uniqueObjectIDs(ids []primitive.ObjectID) []primitive.ObjectID {
	seen := make(map[primitive.ObjectID]struct{}, len(ids))
	unique := make([]primitive.ObjectID, 0, len(ids))
	for _, id := range ids {
		if id.IsZero() {
			continue
		}
		if _, exists := seen[id]; exists {
			continue
		}
		seen[id] = struct{}{}
		unique = append(unique, id)
	}
	return unique
}
