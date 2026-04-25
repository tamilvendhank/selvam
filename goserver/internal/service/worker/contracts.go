package worker

import "context"

type WorkerWorkDiscoveryService interface {
	DiscoverSubmittableBatchJobs(ctx context.Context, request DiscoverSubmittableBatchJobsRequest) (*DiscoverSubmittableBatchJobsResult, error)
	DiscoverPollableBatchJobs(ctx context.Context, request DiscoverPollableBatchJobsRequest) (*DiscoverPollableBatchJobsResult, error)
	DiscoverReconciliableBatchJobs(ctx context.Context, request DiscoverReconciliableBatchJobsRequest) (*DiscoverReconciliableBatchJobsResult, error)
	DiscoverValidatableItems(ctx context.Context, request DiscoverValidatableItemsRequest) (*DiscoverValidatableItemsResult, error)
	DiscoverMaterializableReviews(ctx context.Context, request DiscoverMaterializableReviewsRequest) (*DiscoverMaterializableReviewsResult, error)
	DiscoverFinalizableReviews(ctx context.Context, request DiscoverFinalizableReviewsRequest) (*DiscoverFinalizableReviewsResult, error)
	DiscoverContinuableWorkflows(ctx context.Context, request DiscoverContinuableWorkflowsRequest) (*DiscoverContinuableWorkflowsResult, error)
}

type WorkerCoordinationService interface {
	ClaimWork(ctx context.Context, request ClaimWorkRequest) (*ClaimWorkResult, error)
	HeartbeatWork(ctx context.Context, request HeartbeatWorkRequest) (*HeartbeatWorkResult, error)
	ReleaseWork(ctx context.Context, request ReleaseWorkRequest) (*ReleaseWorkResult, error)
	CompleteWork(ctx context.Context, request CompleteWorkRequest) (*CompleteWorkResult, error)
	FailWork(ctx context.Context, request FailWorkRequest) (*FailWorkResult, error)
}
