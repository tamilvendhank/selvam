package common

import (
	"fmt"
	"strings"

	"goserver/internal/platform/domain"
	platformworkflow "goserver/internal/platform/workflow"
)

type StepName = platformworkflow.StepName

const (
	InvestingStepScanUniverse                 StepName = platformworkflow.InvestingStepScanUniverse
	InvestingStepApplyHardFilters             StepName = platformworkflow.InvestingStepApplyHardFilters
	InvestingStepBuildReviewInputs            StepName = platformworkflow.InvestingStepBuildReviewInputs
	InvestingStepCreatePendingReviewRecords   StepName = platformworkflow.InvestingStepCreatePendingReviewRecords
	InvestingStepCreateBatchJob               StepName = platformworkflow.InvestingStepCreateBatchJob
	InvestingStepSubmitBatchJob               StepName = platformworkflow.InvestingStepSubmitBatchJob
	InvestingStepWaitForAsyncResults          StepName = platformworkflow.InvestingStepWaitForAsyncResults
	InvestingStepPollAndReconcileBatchResults StepName = platformworkflow.InvestingStepPollAndReconcileBatchResults
	InvestingStepValidateAIOutputs            StepName = platformworkflow.InvestingStepValidateAIOutputs
	InvestingStepMaterializeFinalReviews      StepName = platformworkflow.InvestingStepMaterializeFinalReviews
	InvestingStepEvaluateThesisAndChange      StepName = platformworkflow.InvestingStepEvaluateThesisAndChange
	InvestingStepMapActions                   StepName = platformworkflow.InvestingStepMapActions
	InvestingStepAssignBuckets                StepName = platformworkflow.InvestingStepAssignBuckets
	InvestingStepBuildCapitalCandidates       StepName = platformworkflow.InvestingStepBuildCapitalCandidates
	InvestingStepAllocateCapital              StepName = platformworkflow.InvestingStepAllocateCapital
	InvestingStepPersistOutputs               StepName = platformworkflow.InvestingStepPersistOutputs
	InvestingStepPublishRunSummary            StepName = platformworkflow.InvestingStepPublishRunSummary

	TradingStepRefreshUniverse              StepName = platformworkflow.TradingStepRefreshUniverse
	TradingStepEvaluateRegime               StepName = platformworkflow.TradingStepEvaluateRegime
	TradingStepBuildTradingReviewInputs     StepName = platformworkflow.TradingStepBuildTradingReviewInputs
	TradingStepCreateBatchJob               StepName = platformworkflow.TradingStepCreateBatchJob
	TradingStepSubmitBatchJob               StepName = platformworkflow.TradingStepSubmitBatchJob
	TradingStepWaitForAsyncResults          StepName = platformworkflow.TradingStepWaitForAsyncResults
	TradingStepPollAndReconcileBatchResults StepName = platformworkflow.TradingStepPollAndReconcileResults
	TradingStepValidateAIOutputs            StepName = platformworkflow.TradingStepValidateAIOutputs
	TradingStepApproveTradeCandidates       StepName = platformworkflow.TradingStepApproveTradeCandidates
	TradingStepPersistTradingReview         StepName = platformworkflow.TradingStepPersistTradingReview
	TradingStepPublishRunSummary            StepName = platformworkflow.TradingStepPublishRunSummary
)

type StepPhase string

const (
	StepPhasePreparation    StepPhase = "preparation"
	StepPhaseInputBuild     StepPhase = "input_build"
	StepPhasePersistence    StepPhase = "persistence"
	StepPhaseAISubmission   StepPhase = "ai_submission"
	StepPhaseExternalWait   StepPhase = "external_wait"
	StepPhaseReconciliation StepPhase = "reconciliation"
	StepPhaseValidation     StepPhase = "validation"
	StepPhaseMaterialize    StepPhase = "materialize"
	StepPhasePostProcessing StepPhase = "post_processing"
	StepPhaseSummary        StepPhase = "summary"
)

func isValidStepPhase(phase StepPhase) bool {
	switch phase {
	case StepPhasePreparation,
		StepPhaseInputBuild,
		StepPhasePersistence,
		StepPhaseAISubmission,
		StepPhaseExternalWait,
		StepPhaseReconciliation,
		StepPhaseValidation,
		StepPhaseMaterialize,
		StepPhasePostProcessing,
		StepPhaseSummary:
		return true
	default:
		return false
	}
}

type WorkflowStepDescriptor struct {
	Name          StepName        `json:"name"`
	DisplayName   string          `json:"displayName,omitempty"`
	BookType      domain.BookType `json:"bookType"`
	Phase         StepPhase       `json:"phase"`
	Optional      bool            `json:"optional,omitempty"`
	AsyncBoundary bool            `json:"asyncBoundary,omitempty"`
	WaitsExternal bool            `json:"waitsExternal,omitempty"`
}

func (descriptor WorkflowStepDescriptor) Validate() error {
	if strings.TrimSpace(string(descriptor.Name)) == "" {
		return fmt.Errorf("workflow step name is required")
	}
	if !domain.IsValidBookType(descriptor.BookType) {
		return fmt.Errorf("invalid workflow step book type %q", descriptor.BookType)
	}
	if !isValidStepPhase(descriptor.Phase) {
		return fmt.Errorf("invalid workflow step phase %q", descriptor.Phase)
	}
	return nil
}

type StepRange struct {
	Start StepName `json:"start,omitempty"`
	End   StepName `json:"end,omitempty"`
}

func (stepRange StepRange) IsZero() bool {
	return strings.TrimSpace(string(stepRange.Start)) == "" && strings.TrimSpace(string(stepRange.End)) == ""
}

func (stepRange StepRange) Validate() error {
	if stepRange.Start != "" && strings.TrimSpace(string(stepRange.Start)) == "" {
		return fmt.Errorf("step range start cannot be blank")
	}
	if stepRange.End != "" && strings.TrimSpace(string(stepRange.End)) == "" {
		return fmt.Errorf("step range end cannot be blank")
	}
	return nil
}
