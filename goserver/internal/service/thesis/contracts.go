package thesis

import "context"

type ThesisEvaluationService interface {
	EvaluateThesis(ctx context.Context, request EvaluateThesisRequest) (*EvaluateThesisResult, error)
	EvaluateThesisForWorkflow(ctx context.Context, request EvaluateThesisForWorkflowRequest) (*EvaluateThesisForWorkflowResult, error)
}
