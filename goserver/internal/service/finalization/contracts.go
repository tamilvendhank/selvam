package finalization

import "context"

type ReviewFinalizationService interface {
	FinalizeReview(ctx context.Context, request FinalizeReviewRequest) (*FinalizeReviewResult, error)
	FinalizeEligibleReviews(ctx context.Context, request FinalizeEligibleReviewsRequest) (*FinalizeEligibleReviewsResult, error)
}
