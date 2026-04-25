package async

import "context"

type AsyncBatchSubmissionPort interface {
	SubmitWorkflowBatch(ctx context.Context, request AsyncBatchSubmissionRequest) (*AsyncBatchSubmissionResult, error)
}

type AsyncBatchStatusPort interface {
	GetWorkflowBatchStatus(ctx context.Context, request AsyncBatchStatusRequest) (*AsyncBatchStatusResult, error)
}

type AsyncBatchReconciliationPort interface {
	ReconcileWorkflowBatch(ctx context.Context, request AsyncBatchReconciliationRequest) (*AsyncBatchReconciliationResult, error)
}

type AsyncBatchInspectionPort interface {
	CollectCompletedItems(ctx context.Context, request CollectCompletedBatchItemsRequest) (*CollectCompletedBatchItemsResult, error)
}

type WorkflowContinuationPort interface {
	DetermineContinuation(ctx context.Context, request WorkflowContinuationAssessmentRequest) (*WorkflowContinuationDecision, error)
}

type AsyncBatchWorkflowPort interface {
	AsyncBatchSubmissionPort
	AsyncBatchStatusPort
	AsyncBatchReconciliationPort
	AsyncBatchInspectionPort
	WorkflowContinuationPort
}
