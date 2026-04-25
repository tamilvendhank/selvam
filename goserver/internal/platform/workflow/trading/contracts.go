package trading

import (
	"context"

	"goserver/internal/platform/ports"
	workflowasync "goserver/internal/platform/workflow/async"
)

type TradingWorkflowService interface {
	Start(ctx context.Context, request StartTradingWorkflowRequest) (*StartTradingWorkflowResult, error)
	Resume(ctx context.Context, request ResumeTradingWorkflowRequest) (*ResumeTradingWorkflowResult, error)
	Reconcile(ctx context.Context, request ReconcileTradingWorkflowRequest) (*ReconcileTradingWorkflowResult, error)
	GetStatus(ctx context.Context, workflowRunID string) (*TradingWorkflowStatus, error)
	GetSummary(ctx context.Context, workflowRunID string) (*TradingWorkflowSummary, error)
}

type TradingWorkflowStorage struct {
	WorkflowRuns       ports.WorkflowRunRepository
	WorkflowSteps      ports.WorkflowStepRunRepository
	Reviews            ports.CompanyReviewRepository
	BatchJobs          ports.AIBatchJobRepository
	BatchItems         ports.AIBatchItemRepository
	ReconciliationLogs ports.JobReconciliationLogRepository
}

type TradingWorkflowDependencies struct {
	Storage            TradingWorkflowStorage
	ConfigSnapshots    ports.ConfigService
	AsyncBatches       workflowasync.AsyncBatchWorkflowPort
	UniverseRefresher  TradingUniverseRefresher
	RegimeEvaluator    RegimeEvaluator
	ReviewInputBuilder TradingReviewInputBuilder
	AIOutputValidator  AIOutputValidator
	CandidateApprover  TradeCandidateApprover
	ReviewPersister    TradingReviewPersister
	SummaryPublisher   WorkflowSummaryPublisher
}

type TradingUniverseRefresher interface {
	RefreshUniverse(ctx context.Context, input RefreshUniverseInput) (*RefreshUniverseOutput, error)
}

type RegimeEvaluator interface {
	EvaluateRegime(ctx context.Context, input EvaluateRegimeInput) (*EvaluateRegimeOutput, error)
}

type TradingReviewInputBuilder interface {
	BuildTradingReviewInputs(ctx context.Context, input BuildTradingReviewInputsInput) (*BuildTradingReviewInputsOutput, error)
}

type AIOutputValidator interface {
	ValidateAIOutputs(ctx context.Context, input ValidateAIOutputsInput) (*ValidateAIOutputsOutput, error)
}

type TradeCandidateApprover interface {
	ApproveTradeCandidates(ctx context.Context, input ApproveTradeCandidatesInput) (*ApproveTradeCandidatesOutput, error)
}

type TradingReviewPersister interface {
	PersistTradingReview(ctx context.Context, input PersistTradingReviewInput) (*PersistTradingReviewOutput, error)
}

type WorkflowSummaryPublisher interface {
	PublishRunSummary(ctx context.Context, input PublishRunSummaryInput) (*PublishRunSummaryOutput, error)
}
