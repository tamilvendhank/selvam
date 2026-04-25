package orchestrator

import (
	"context"
	"fmt"
	"strings"

	"goserver/internal/platform/domain"
	"goserver/internal/platform/ports"
	workflowasync "goserver/internal/platform/workflow/async"
	"goserver/internal/platform/workflow/common"
	"goserver/internal/platform/workflow/investing"
	"goserver/internal/platform/workflow/trading"
)

type WorkflowStatusService interface {
	GetWorkflowStatus(ctx context.Context, workflowRunID string) (*common.WorkflowStatusView, error)
}

type WorkflowProgressService interface {
	GetWorkflowProgress(ctx context.Context, workflowRunID string) (*common.WorkflowProgressSummary, error)
}

type WorkflowContinuationService interface {
	AssessContinuation(ctx context.Context, request ContinuationAssessmentRequest) (*workflowasync.WorkflowContinuationDecision, error)
}

type WorkflowOrchestrator interface {
	WorkflowStatusService
	WorkflowProgressService
	WorkflowContinuationService
}

type WorkflowReadDependencies struct {
	WorkflowRuns       ports.WorkflowRunRepository
	WorkflowSteps      ports.WorkflowStepRunRepository
	BatchJobs          ports.AIBatchJobRepository
	BatchItems         ports.AIBatchItemRepository
	ReconciliationLogs ports.JobReconciliationLogRepository
}

type WorkflowOrchestratorDependencies struct {
	Reads        WorkflowReadDependencies
	AsyncBatches workflowasync.AsyncBatchWorkflowPort
	Investing    investing.InvestingWorkflowService
	Trading      trading.TradingWorkflowService
}

type BookWorkflowResolver interface {
	InvestingWorkflow() investing.InvestingWorkflowService
	TradingWorkflow() trading.TradingWorkflowService
}

type ContinuationAssessmentRequest struct {
	WorkflowRunID string                         `json:"workflowRunId"`
	BookType      domain.BookType                `json:"bookType"`
	CurrentStatus common.WorkflowExecutionStatus `json:"currentStatus"`
	CurrentStep   common.StepName                `json:"currentStep,omitempty"`
	Force         bool                           `json:"force,omitempty"`
	CorrelationID string                         `json:"correlationId,omitempty"`
}

func (request ContinuationAssessmentRequest) Validate() error {
	if strings.TrimSpace(request.WorkflowRunID) == "" {
		return fmt.Errorf("workflowRunId is required")
	}
	if !domain.IsValidBookType(request.BookType) {
		return fmt.Errorf("invalid book type %q", request.BookType)
	}
	if request.CorrelationID != "" && strings.TrimSpace(request.CorrelationID) == "" {
		return fmt.Errorf("correlationId cannot be blank")
	}
	return nil
}
