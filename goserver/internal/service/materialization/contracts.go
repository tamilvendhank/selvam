package materialization

import "context"

type ReviewMaterializationService interface {
	MaterializeReview(ctx context.Context, request MaterializeReviewRequest) (*MaterializeReviewResult, error)
	MaterializePendingReviews(ctx context.Context, request MaterializePendingReviewsRequest) (*MaterializePendingReviewsResult, error)
}
