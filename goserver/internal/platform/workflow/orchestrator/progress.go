package orchestrator

import (
	"context"
	"fmt"
	"time"

	"goserver/internal/platform/domain"
	"goserver/internal/platform/workflow/common"
)

type ProgressBuildRequest struct {
	WorkflowRunID    string                          `json:"workflowRunId"`
	BookType         domain.BookType                 `json:"bookType"`
	Status           common.WorkflowExecutionStatus  `json:"status"`
	StartedAt        *time.Time                      `json:"startedAt,omitempty"`
	UpdatedAt        *time.Time                      `json:"updatedAt,omitempty"`
	StepSummaries    []common.StepExecutionSummary   `json:"stepSummaries,omitempty"`
	AsyncSummary     common.WorkflowAsyncSummary     `json:"asyncSummary,omitempty"`
	ContinuationHint common.WorkflowContinuationHint `json:"continuationHint"`
	ExternalWait     *common.ExternalWaitSummary     `json:"externalWait,omitempty"`
}

func (request ProgressBuildRequest) Validate() error {
	if request.WorkflowRunID == "" {
		return fmt.Errorf("workflowRunId is required")
	}
	if !domain.IsValidBookType(request.BookType) {
		return fmt.Errorf("invalid book type %q", request.BookType)
	}
	for index := range request.StepSummaries {
		if err := request.StepSummaries[index].Descriptor.Validate(); err != nil {
			return fmt.Errorf("stepSummaries[%d]: %w", index, err)
		}
	}
	return nil
}

type WorkflowProgressView struct {
	WorkflowRunID    string                          `json:"workflowRunId"`
	BookType         domain.BookType                 `json:"bookType"`
	Status           common.WorkflowExecutionStatus  `json:"status"`
	StartedAt        *time.Time                      `json:"startedAt,omitempty"`
	UpdatedAt        *time.Time                      `json:"updatedAt,omitempty"`
	Progress         common.WorkflowProgressSummary  `json:"progress"`
	ContinuationHint common.WorkflowContinuationHint `json:"continuationHint"`
	ExternalWait     *common.ExternalWaitSummary     `json:"externalWait,omitempty"`
}

type WorkflowProgressProjector interface {
	BuildWorkflowProgress(ctx context.Context, request ProgressBuildRequest) (*WorkflowProgressView, error)
}
