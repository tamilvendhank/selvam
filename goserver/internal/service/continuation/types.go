package continuation

import (
	domaincommon "goserver/internal/domain/common"
	servicecommon "goserver/internal/service/common"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

type ContinuationReason string

const (
	ContinuationReasonAsyncResolved       ContinuationReason = "async_dependencies_resolved"
	ContinuationReasonManualOverride      ContinuationReason = "manual_override"
	ContinuationReasonForced              ContinuationReason = "forced"
	ContinuationReasonStillBlocked        ContinuationReason = "still_blocked"
	ContinuationReasonWorkflowTerminal    ContinuationReason = "workflow_terminal"
	ContinuationReasonPreconditionsFailed ContinuationReason = "preconditions_failed"
)

type EvaluateWorkflowContinuationRequest struct {
	WorkflowRunID primitive.ObjectID    `json:"workflowRunId"`
	BookType      domaincommon.BookType `json:"bookType,omitempty"`
	Force         bool                  `json:"force,omitempty"`
	InitiatedBy   string                `json:"initiatedBy,omitempty"`
	CorrelationID string                `json:"correlationId,omitempty"`
}

func (request EvaluateWorkflowContinuationRequest) Validate() error {
	if err := servicecommon.ValidateRequiredObjectID("workflowRunId", request.WorkflowRunID); err != nil {
		return err
	}
	if err := servicecommon.ValidateOptionalBookType(request.BookType); err != nil {
		return err
	}
	if err := servicecommon.ValidateOptionalText("initiatedBy", request.InitiatedBy); err != nil {
		return err
	}
	return servicecommon.ValidateOptionalText("correlationId", request.CorrelationID)
}

type EvaluateWorkflowContinuationResult struct {
	WorkflowRunID            primitive.ObjectID                `json:"workflowRunId"`
	BookType                 domaincommon.BookType             `json:"bookType,omitempty"`
	ReadyToContinue          bool                              `json:"readyToContinue"`
	WaitingOnExternalJobs    bool                              `json:"waitingOnExternalJobs,omitempty"`
	WaitingOnValidation      bool                              `json:"waitingOnValidation,omitempty"`
	WaitingOnMaterialization bool                              `json:"waitingOnMaterialization,omitempty"`
	WaitingOnFinalization    bool                              `json:"waitingOnFinalization,omitempty"`
	NextSuggestedStep        domaincommon.WorkflowStepName     `json:"nextSuggestedStep,omitempty"`
	ContinuationReason       ContinuationReason                `json:"continuationReason,omitempty"`
	Blockers                 []servicecommon.BlockingCondition `json:"blockers,omitempty"`
	Summary                  servicecommon.ContinuationSummary `json:"summary,omitempty"`
}

func (result EvaluateWorkflowContinuationResult) ReadyToContinueNow() bool {
	return result.ReadyToContinue && len(result.Blockers) == 0
}

type EvaluateManyWorkflowContinuationsRequest struct {
	WorkflowRunIDs []primitive.ObjectID  `json:"workflowRunIds,omitempty"`
	BookType       domaincommon.BookType `json:"bookType,omitempty"`
	MaxWorkflows   int                   `json:"maxWorkflows,omitempty"`
	Force          bool                  `json:"force,omitempty"`
	InitiatedBy    string                `json:"initiatedBy,omitempty"`
	CorrelationID  string                `json:"correlationId,omitempty"`
}

func (request EvaluateManyWorkflowContinuationsRequest) Validate() error {
	if err := servicecommon.ValidateOptionalMax("maxWorkflows", request.MaxWorkflows); err != nil {
		return err
	}
	if err := servicecommon.ValidateOptionalBookType(request.BookType); err != nil {
		return err
	}
	if err := servicecommon.ValidateOptionalText("initiatedBy", request.InitiatedBy); err != nil {
		return err
	}
	return servicecommon.ValidateOptionalText("correlationId", request.CorrelationID)
}

type EvaluateManyWorkflowContinuationsResult struct {
	ReadyWorkflowRunIDs   []primitive.ObjectID                 `json:"readyWorkflowRunIds,omitempty"`
	BlockedWorkflowRunIDs []primitive.ObjectID                 `json:"blockedWorkflowRunIds,omitempty"`
	Decisions             []EvaluateWorkflowContinuationResult `json:"decisions,omitempty"`
	PartialFailures       []servicecommon.PartialFailure       `json:"partialFailures,omitempty"`
	Summary               servicecommon.ContinuationSummary    `json:"summary,omitempty"`
}

func (result EvaluateManyWorkflowContinuationsResult) HasReadyWork() bool {
	return len(result.ReadyWorkflowRunIDs) > 0
}

type ContinueWorkflowRequest struct {
	WorkflowRunID    primitive.ObjectID      `json:"workflowRunId"`
	BookType         domaincommon.BookType   `json:"bookType,omitempty"`
	DryRun           bool                    `json:"dryRun,omitempty"`
	Force            bool                    `json:"force,omitempty"`
	AllowedStepRange servicecommon.StepRange `json:"allowedStepRange,omitempty"`
	InitiatedBy      string                  `json:"initiatedBy,omitempty"`
	CorrelationID    string                  `json:"correlationId,omitempty"`
}

func (request ContinueWorkflowRequest) Validate() error {
	if err := servicecommon.ValidateRequiredObjectID("workflowRunId", request.WorkflowRunID); err != nil {
		return err
	}
	if err := servicecommon.ValidateOptionalBookType(request.BookType); err != nil {
		return err
	}
	if err := request.AllowedStepRange.Validate(); err != nil {
		return err
	}
	if err := servicecommon.ValidateOptionalText("initiatedBy", request.InitiatedBy); err != nil {
		return err
	}
	return servicecommon.ValidateOptionalText("correlationId", request.CorrelationID)
}

type ContinueWorkflowResult struct {
	WorkflowRunID              primitive.ObjectID                `json:"workflowRunId,omitempty"`
	ContinuedWorkflowRunIDs    []primitive.ObjectID              `json:"continuedWorkflowRunIds,omitempty"`
	CompletedWorkflowRunIDs    []primitive.ObjectID              `json:"completedWorkflowRunIds,omitempty"`
	StillBlockedWorkflowRunIDs []primitive.ObjectID              `json:"stillBlockedWorkflowRunIds,omitempty"`
	FailedWorkflowRunIDs       []primitive.ObjectID              `json:"failedWorkflowRunIds,omitempty"`
	ExecutedSteps              []domaincommon.WorkflowStepName   `json:"executedSteps,omitempty"`
	NextSuggestedStep          domaincommon.WorkflowStepName     `json:"nextSuggestedStep,omitempty"`
	PartialFailures            []servicecommon.PartialFailure    `json:"partialFailures,omitempty"`
	Summary                    servicecommon.ContinuationSummary `json:"summary,omitempty"`
}

func (result ContinueWorkflowResult) HasFailures() bool {
	return len(result.FailedWorkflowRunIDs) > 0 || len(result.PartialFailures) > 0 || result.Summary.HasFailures()
}

type ContinueEligibleWorkflowsRequest struct {
	WorkflowRunID    primitive.ObjectID      `json:"workflowRunId,omitempty"`
	BookType         domaincommon.BookType   `json:"bookType,omitempty"`
	MaxWorkflows     int                     `json:"maxWorkflows,omitempty"`
	DryRun           bool                    `json:"dryRun,omitempty"`
	Force            bool                    `json:"force,omitempty"`
	AllowedStepRange servicecommon.StepRange `json:"allowedStepRange,omitempty"`
	InitiatedBy      string                  `json:"initiatedBy,omitempty"`
	CorrelationID    string                  `json:"correlationId,omitempty"`
}

func (request ContinueEligibleWorkflowsRequest) Validate() error {
	if err := servicecommon.ValidateOptionalMax("maxWorkflows", request.MaxWorkflows); err != nil {
		return err
	}
	if err := servicecommon.ValidateOptionalBookType(request.BookType); err != nil {
		return err
	}
	if err := request.AllowedStepRange.Validate(); err != nil {
		return err
	}
	if err := servicecommon.ValidateOptionalText("initiatedBy", request.InitiatedBy); err != nil {
		return err
	}
	return servicecommon.ValidateOptionalText("correlationId", request.CorrelationID)
}

type ContinueEligibleWorkflowsResult struct {
	ContinuedWorkflowRunIDs    []primitive.ObjectID                 `json:"continuedWorkflowRunIds,omitempty"`
	CompletedWorkflowRunIDs    []primitive.ObjectID                 `json:"completedWorkflowRunIds,omitempty"`
	StillBlockedWorkflowRunIDs []primitive.ObjectID                 `json:"stillBlockedWorkflowRunIds,omitempty"`
	FailedWorkflowRunIDs       []primitive.ObjectID                 `json:"failedWorkflowRunIds,omitempty"`
	Decisions                  []EvaluateWorkflowContinuationResult `json:"decisions,omitempty"`
	PartialFailures            []servicecommon.PartialFailure       `json:"partialFailures,omitempty"`
	Summary                    servicecommon.ContinuationSummary    `json:"summary,omitempty"`
}

func (result ContinueEligibleWorkflowsResult) HasFailures() bool {
	return len(result.FailedWorkflowRunIDs) > 0 || len(result.PartialFailures) > 0 || result.Summary.HasFailures()
}
