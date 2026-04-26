package projection

import (
	"context"
	"fmt"
	"time"

	domaincommon "goserver/internal/domain/common"
	domainposition "goserver/internal/domain/position"
	domainreview "goserver/internal/domain/review"
	servicecommon "goserver/internal/service/common"
)

func (service *projectionUpdateService) updatePositionProjectionFromReview(
	ctx context.Context,
	review *domainreview.CompanyReview,
	request UpdateProjectionsRequest,
) (ProjectionUpdateRef, *servicecommon.PartialFailure) {
	ref := projectionRefForReview(review, ProjectionTargetPosition)
	if service.positions == nil {
		return markSkipped(ref, "current_position_repository_unavailable"), nil
	}
	if review == nil || review.PositionSnapshot == nil {
		return markSkipped(ref, "position_snapshot_missing"), nil
	}
	if request.BookType != "" && review.BookType != request.BookType {
		return markSkipped(ref, "book_type_mismatch"), nil
	}

	sourceTime := projectionSourceTime(review)
	current, currentErr := service.positions.GetByCompanyAndBook(ctx, review.CompanyID, review.BookType)
	if currentErr != nil && !isRepositoryNotFound(currentErr) {
		failure := projectionPartialFailure(request.WorkflowRunID, review.ID, review.CompanyID, review.BookType, "current_position_lookup_failed", currentErr)
		return ref, &failure
	}
	if !shouldUpdatePositionFromReview(review, current, request.Force) {
		return markSkipped(ref, "current_position_newer_than_review_snapshot"), nil
	}

	position, err := buildCurrentPositionFromReview(review, current, sourceTime)
	if err != nil {
		failure := projectionPartialFailure(request.WorkflowRunID, review.ID, review.CompanyID, review.BookType, "current_position_build_failed", err)
		return ref, &failure
	}
	if request.DryRun {
		ref.Updated = true
		return ref, nil
	}

	updated, err := service.positions.Upsert(ctx, position)
	if err != nil {
		failure := projectionPartialFailure(request.WorkflowRunID, review.ID, review.CompanyID, review.BookType, "current_position_upsert_failed", err)
		return ref, &failure
	}
	if updated != nil {
		ref.ID = updated.ID
	}
	ref.Updated = true
	return ref, nil
}

func buildCurrentPositionFromReview(
	review *domainreview.CompanyReview,
	existing *domainposition.CurrentPosition,
	sourceTime time.Time,
) (*domainposition.CurrentPosition, error) {
	if review == nil {
		return nil, fmt.Errorf("review is required")
	}
	snapshot := review.PositionSnapshot
	if snapshot == nil {
		return nil, fmt.Errorf("position snapshot is required")
	}

	position := &domainposition.CurrentPosition{
		CompanyID:                     review.CompanyID,
		BookType:                      review.BookType,
		IsOpen:                        snapshot.IsOwned,
		Quantity:                      snapshot.Quantity,
		AverageCost:                   snapshot.AverageCost,
		CurrentMarketValue:            snapshot.MarketValue,
		CurrentPositionPctOfBook:      snapshot.PositionPctOfBook,
		CurrentPositionPctOfPortfolio: snapshot.PositionPctOfTotalPortfolio,
		LastUpdatedAt:                 sourceTime.UTC(),
		SchemaVersion:                 domaincommon.SchemaVersion1,
	}
	if existing != nil {
		position.ID = existing.ID
	}
	if !position.IsOpen {
		position.Quantity = 0
		position.CurrentMarketValue = 0
		position.CurrentPositionPctOfBook = 0
		position.CurrentPositionPctOfPortfolio = 0
	}
	if err := position.Validate(); err != nil {
		return nil, err
	}
	return position, nil
}

func projectionSourceTime(review *domainreview.CompanyReview) time.Time {
	if review == nil {
		return time.Time{}
	}
	sourceTime := review.ReviewDate
	if review.FinalizedAt != nil && review.FinalizedAt.After(sourceTime) {
		sourceTime = review.FinalizedAt.UTC()
	}
	if sourceTime.IsZero() {
		sourceTime = review.UpdatedAt.UTC()
	}
	return sourceTime.UTC()
}

func shouldUpdatePositionFromReview(review *domainreview.CompanyReview, current *domainposition.CurrentPosition, force bool) bool {
	if review == nil || review.PositionSnapshot == nil {
		return false
	}
	if current == nil || force {
		return true
	}
	return !current.LastUpdatedAt.After(projectionSourceTime(review))
}
