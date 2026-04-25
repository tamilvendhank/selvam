package continuation

import "context"

type WorkflowContinuationDecisionService interface {
	EvaluateWorkflowContinuation(ctx context.Context, request EvaluateWorkflowContinuationRequest) (*EvaluateWorkflowContinuationResult, error)
	EvaluateManyWorkflowContinuations(ctx context.Context, request EvaluateManyWorkflowContinuationsRequest) (*EvaluateManyWorkflowContinuationsResult, error)
}

type WorkflowContinuationService interface {
	ContinueWorkflow(ctx context.Context, request ContinueWorkflowRequest) (*ContinueWorkflowResult, error)
	ContinueEligibleWorkflows(ctx context.Context, request ContinueEligibleWorkflowsRequest) (*ContinueEligibleWorkflowsResult, error)
}
