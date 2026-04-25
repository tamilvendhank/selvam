package review

import "context"

type ActionMappingService interface {
	MapReviewAction(ctx context.Context, request MapReviewActionRequest) (*MapReviewActionResult, error)
	MapWorkflowActions(ctx context.Context, request MapWorkflowActionsRequest) (*MapWorkflowActionsResult, error)
}

type BucketAssignmentService interface {
	AssignBucket(ctx context.Context, request AssignBucketRequest) (*AssignBucketResult, error)
	AssignBucketsForWorkflow(ctx context.Context, request AssignBucketsForWorkflowRequest) (*AssignBucketsForWorkflowResult, error)
}
