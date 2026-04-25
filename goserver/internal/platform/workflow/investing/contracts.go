package investing

import (
	"context"

	"goserver/internal/platform/ports"
	workflowasync "goserver/internal/platform/workflow/async"
)

type InvestingWorkflowService interface {
	Start(ctx context.Context, request StartInvestingWorkflowRequest) (*StartInvestingWorkflowResult, error)
	Resume(ctx context.Context, request ResumeInvestingWorkflowRequest) (*ResumeInvestingWorkflowResult, error)
	Reconcile(ctx context.Context, request ReconcileInvestingWorkflowRequest) (*ReconcileInvestingWorkflowResult, error)
	GetStatus(ctx context.Context, workflowRunID string) (*InvestingWorkflowStatus, error)
	GetSummary(ctx context.Context, workflowRunID string) (*InvestingWorkflowSummary, error)
}

type InvestingWorkflowStorage struct {
	WorkflowRuns       ports.WorkflowRunRepository
	WorkflowSteps      ports.WorkflowStepRunRepository
	Reviews            ports.CompanyReviewRepository
	Theses             ports.ThesisRepository
	BatchJobs          ports.AIBatchJobRepository
	BatchItems         ports.AIBatchItemRepository
	ReconciliationLogs ports.JobReconciliationLogRepository
}

type InvestingWorkflowDependencies struct {
	Storage                 InvestingWorkflowStorage
	ConfigSnapshots         ports.ConfigService
	AsyncBatches            workflowasync.AsyncBatchWorkflowPort
	UniverseScanner         UniverseScanner
	HardFilterEvaluator     HardFilterEvaluator
	ReviewInputBuilder      ReviewInputBuilder
	ReviewShellCreator      ReviewShellCreator
	AIOutputValidator       AIOutputValidator
	ReviewMaterializer      ReviewMaterializer
	ThesisChangeEvaluator   ThesisChangeEvaluator
	ActionMapper            ActionMapper
	BucketAssigner          BucketAssigner
	CapitalCandidateBuilder CapitalCandidateBuilder
	CapitalAllocator        CapitalAllocator
	OutputPersister         WorkflowOutputPersister
	SummaryPublisher        WorkflowSummaryPublisher
}

type UniverseScanner interface {
	ScanUniverse(ctx context.Context, input ScanUniverseInput) (*ScanUniverseOutput, error)
}

type HardFilterEvaluator interface {
	ApplyHardFilters(ctx context.Context, input ApplyHardFiltersInput) (*ApplyHardFiltersOutput, error)
}

type ReviewInputBuilder interface {
	BuildReviewInputs(ctx context.Context, input BuildReviewInputsInput) (*BuildReviewInputsOutput, error)
}

type ReviewShellCreator interface {
	CreatePendingReviewRecords(ctx context.Context, input CreatePendingReviewRecordsInput) (*CreatePendingReviewRecordsOutput, error)
}

type AIOutputValidator interface {
	ValidateAIOutputs(ctx context.Context, input ValidateAIOutputsInput) (*ValidateAIOutputsOutput, error)
}

type ReviewMaterializer interface {
	MaterializeFinalReviews(ctx context.Context, input MaterializeFinalReviewsInput) (*MaterializeFinalReviewsOutput, error)
}

type ThesisChangeEvaluator interface {
	EvaluateThesisAndChange(ctx context.Context, input EvaluateThesisAndChangeInput) (*EvaluateThesisAndChangeOutput, error)
}

type ActionMapper interface {
	MapActions(ctx context.Context, input MapActionsInput) (*MapActionsOutput, error)
}

type BucketAssigner interface {
	AssignBuckets(ctx context.Context, input AssignBucketsInput) (*AssignBucketsOutput, error)
}

type CapitalCandidateBuilder interface {
	BuildCapitalCandidates(ctx context.Context, input BuildCapitalCandidatesInput) (*BuildCapitalCandidatesOutput, error)
}

type CapitalAllocator interface {
	AllocateCapital(ctx context.Context, input AllocateCapitalInput) (*AllocateCapitalOutput, error)
}

type WorkflowOutputPersister interface {
	PersistOutputs(ctx context.Context, input PersistOutputsInput) (*PersistOutputsOutput, error)
}

type WorkflowSummaryPublisher interface {
	PublishRunSummary(ctx context.Context, input PublishRunSummaryInput) (*PublishRunSummaryOutput, error)
}
