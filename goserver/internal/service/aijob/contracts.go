package aijob

import "context"

type BatchJobSubmissionService interface {
	SubmitBatchJob(ctx context.Context, request SubmitBatchJobRequest) (*SubmitBatchJobResult, error)
	SubmitPendingBatchJobs(ctx context.Context, request SubmitPendingBatchJobsRequest) (*SubmitPendingBatchJobsResult, error)
}

type BatchJobPollingService interface {
	PollBatchJob(ctx context.Context, request PollBatchJobRequest) (*PollBatchJobResult, error)
	PollPendingBatchJobs(ctx context.Context, request PollPendingBatchJobsRequest) (*PollPendingBatchJobsResult, error)
}

type BatchReconciliationService interface {
	ReconcileBatchJob(ctx context.Context, request ReconcileBatchJobRequest) (*ReconcileBatchJobResult, error)
	ReconcilePendingBatchJobs(ctx context.Context, request ReconcilePendingBatchJobsRequest) (*ReconcilePendingBatchJobsResult, error)
}

type BatchJobLifecycleService interface {
	BatchJobSubmissionService
	BatchJobPollingService
	BatchReconciliationService
}

// BatchEnginePort is the narrow provider boundary needed by lifecycle services.
// Repositories remain outside this package and should be injected through the
// existing repository interfaces when concrete services are built.
type BatchEnginePort interface {
	SubmitBatch(ctx context.Context, request BatchEngineSubmitRequest) (*BatchEngineSubmitResult, error)
	PollBatch(ctx context.Context, request BatchEnginePollRequest) (*BatchEnginePollResult, error)
	FetchBatchResults(ctx context.Context, request BatchEngineFetchResultsRequest) (*BatchEngineFetchResultsResult, error)
}
