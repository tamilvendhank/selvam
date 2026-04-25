package trading

import (
	"fmt"
	"strings"
	"time"

	"goserver/internal/platform/domain"
	workflowasync "goserver/internal/platform/workflow/async"
	"goserver/internal/platform/workflow/common"
)

var TradingStepDescriptors = []common.WorkflowStepDescriptor{
	{Name: common.TradingStepRefreshUniverse, DisplayName: "Refresh Universe", BookType: domain.BookTypeTrading, Phase: common.StepPhasePreparation},
	{Name: common.TradingStepEvaluateRegime, DisplayName: "Evaluate Regime", BookType: domain.BookTypeTrading, Phase: common.StepPhasePreparation},
	{Name: common.TradingStepBuildTradingReviewInputs, DisplayName: "Build Trading Review Inputs", BookType: domain.BookTypeTrading, Phase: common.StepPhaseInputBuild},
	{Name: common.TradingStepCreateBatchJob, DisplayName: "Create Batch Job", BookType: domain.BookTypeTrading, Phase: common.StepPhaseAISubmission, AsyncBoundary: true},
	{Name: common.TradingStepSubmitBatchJob, DisplayName: "Submit Batch Job", BookType: domain.BookTypeTrading, Phase: common.StepPhaseAISubmission, AsyncBoundary: true},
	{Name: common.TradingStepWaitForAsyncResults, DisplayName: "Wait For Async Results", BookType: domain.BookTypeTrading, Phase: common.StepPhaseExternalWait, WaitsExternal: true},
	{Name: common.TradingStepPollAndReconcileBatchResults, DisplayName: "Poll And Reconcile Batch Results", BookType: domain.BookTypeTrading, Phase: common.StepPhaseReconciliation},
	{Name: common.TradingStepValidateAIOutputs, DisplayName: "Validate AI Outputs", BookType: domain.BookTypeTrading, Phase: common.StepPhaseValidation},
	{Name: common.TradingStepApproveTradeCandidates, DisplayName: "Approve Trade Candidates", BookType: domain.BookTypeTrading, Phase: common.StepPhasePostProcessing},
	{Name: common.TradingStepPersistTradingReview, DisplayName: "Persist Trading Review", BookType: domain.BookTypeTrading, Phase: common.StepPhasePersistence},
	{Name: common.TradingStepPublishRunSummary, DisplayName: "Publish Run Summary", BookType: domain.BookTypeTrading, Phase: common.StepPhaseSummary},
}

func TradingStepSequence() []common.StepName {
	steps := make([]common.StepName, 0, len(TradingStepDescriptors))
	for _, descriptor := range TradingStepDescriptors {
		steps = append(steps, descriptor.Name)
	}
	return steps
}

type StartTradingWorkflowRequest struct {
	common.WorkflowStartRequest
	AsOf       *time.Time `json:"asOf,omitempty"`
	Mode       string     `json:"mode,omitempty"`
	CompanyIDs []string   `json:"companyIds,omitempty"`
	Symbols    []string   `json:"symbols,omitempty"`
	Limit      int        `json:"limit,omitempty"`
}

func (request StartTradingWorkflowRequest) Validate() error {
	if err := request.WorkflowStartRequest.Validate(); err != nil {
		return err
	}
	if request.BookType != "" && request.BookType != domain.BookTypeTrading {
		return fmt.Errorf("start trading workflow request bookType must be %q", domain.BookTypeTrading)
	}
	if request.AsOf != nil && request.AsOf.IsZero() {
		return fmt.Errorf("asOf cannot be zero")
	}
	if request.Mode != "" && strings.TrimSpace(request.Mode) == "" {
		return fmt.Errorf("mode cannot be blank")
	}
	if request.Limit < 0 {
		return fmt.Errorf("limit cannot be negative")
	}
	if err := validateIDs("companyIds", request.CompanyIDs); err != nil {
		return err
	}
	if err := validateIDs("symbols", request.Symbols); err != nil {
		return err
	}
	return validateTradingStepRange(request.AllowedStepRange)
}

type ResumeTradingWorkflowRequest struct {
	common.WorkflowResumeRequest
}

func (request ResumeTradingWorkflowRequest) Validate() error {
	if err := request.WorkflowResumeRequest.Validate(); err != nil {
		return err
	}
	return validateTradingStepRange(request.AllowedStepRange)
}

type ReconcileTradingWorkflowRequest struct {
	common.WorkflowReconcileRequest
}

func (request ReconcileTradingWorkflowRequest) Validate() error {
	if err := request.WorkflowReconcileRequest.Validate(); err != nil {
		return err
	}
	return validateTradingStepRange(request.AllowedStepRange)
}

type StartTradingWorkflowResult struct {
	Run    common.WorkflowStartResult `json:"run"`
	Status TradingWorkflowStatus      `json:"status"`
}

type ResumeTradingWorkflowResult struct {
	Run    common.WorkflowResumeResult `json:"run"`
	Status TradingWorkflowStatus       `json:"status"`
}

type ReconcileTradingWorkflowResult struct {
	Run    common.WorkflowReconcileResult `json:"run"`
	Status TradingWorkflowStatus          `json:"status"`
}

type TradingWorkflowCounts struct {
	common.WorkflowCounts
	RegimeEligible      int `json:"regimeEligible,omitempty"`
	CandidatesValidated int `json:"candidatesValidated,omitempty"`
	CandidatesApproved  int `json:"candidatesApproved,omitempty"`
	TradingReviews      int `json:"tradingReviews,omitempty"`
}

type TradingWorkflowStatus struct {
	common.WorkflowStatusView
	ConfigSnapshotID string                              `json:"configSnapshotId,omitempty"`
	RequestedSymbols []string                            `json:"requestedSymbols,omitempty"`
	BatchJobs        []workflowasync.BatchStatusSnapshot `json:"batchJobs,omitempty"`
	Counts           TradingWorkflowCounts               `json:"counts,omitempty"`
	Summary          *TradingWorkflowSummary             `json:"summary,omitempty"`
}

type TradingWorkflowSummary struct {
	WorkflowRunID        string                        `json:"workflowRunId"`
	RunType              domain.WorkflowRunType        `json:"runType,omitempty"`
	Mode                 string                        `json:"mode,omitempty"`
	AsOf                 *time.Time                    `json:"asOf,omitempty"`
	ConfigSnapshotID     string                        `json:"configSnapshotId,omitempty"`
	Regime               string                        `json:"regime,omitempty"`
	Counts               TradingWorkflowCounts         `json:"counts,omitempty"`
	BatchJobIDs          []string                      `json:"batchJobIds,omitempty"`
	ApprovedCandidateIDs []string                      `json:"approvedCandidateIds,omitempty"`
	ReviewIDs            []string                      `json:"reviewIds,omitempty"`
	PartialFailure       *common.PartialFailureSummary `json:"partialFailure,omitempty"`
	Published            bool                          `json:"published,omitempty"`
	PublishedAt          *time.Time                    `json:"publishedAt,omitempty"`
}

type TradingReviewInputRef struct {
	CompanyID        string         `json:"companyId"`
	Symbol           string         `json:"symbol,omitempty"`
	ConfigSnapshotID string         `json:"configSnapshotId,omitempty"`
	Metadata         map[string]any `json:"metadata,omitempty"`
}

type ApprovedTradeCandidateRef struct {
	CandidateID string  `json:"candidateId"`
	CompanyID   string  `json:"companyId"`
	Symbol      string  `json:"symbol,omitempty"`
	Score       float64 `json:"score,omitempty"`
}

type ValidationApprovalHandoff struct {
	WorkflowRunID       string   `json:"workflowRunId"`
	Ready               bool     `json:"ready"`
	ValidCandidateIDs   []string `json:"validCandidateIds,omitempty"`
	InvalidCandidateIDs []string `json:"invalidCandidateIds,omitempty"`
	PendingCandidateIDs []string `json:"pendingCandidateIds,omitempty"`
}

type RefreshUniverseInput struct {
	AsOf       *time.Time `json:"asOf,omitempty"`
	CompanyIDs []string   `json:"companyIds,omitempty"`
	Symbols    []string   `json:"symbols,omitempty"`
	Limit      int        `json:"limit,omitempty"`
}

type RefreshUniverseOutput struct {
	CompanyIDs []string `json:"companyIds,omitempty"`
	Symbols    []string `json:"symbols,omitempty"`
	Count      int      `json:"count,omitempty"`
}

type EvaluateRegimeInput struct {
	WorkflowRunID string     `json:"workflowRunId"`
	AsOf          *time.Time `json:"asOf,omitempty"`
	UniverseCount int        `json:"universeCount,omitempty"`
}

type EvaluateRegimeOutput struct {
	Regime        string `json:"regime,omitempty"`
	Allowed       bool   `json:"allowed"`
	BlockedReason string `json:"blockedReason,omitempty"`
}

type BuildTradingReviewInputsInput struct {
	WorkflowRunID    string   `json:"workflowRunId"`
	CompanyIDs       []string `json:"companyIds,omitempty"`
	ConfigSnapshotID string   `json:"configSnapshotId"`
	Mode             string   `json:"mode,omitempty"`
}

type BuildTradingReviewInputsOutput struct {
	InputCount int                     `json:"inputCount,omitempty"`
	Inputs     []TradingReviewInputRef `json:"inputs,omitempty"`
}

type CreateBatchJobInput struct {
	WorkflowRunID    string `json:"workflowRunId"`
	ConfigSnapshotID string `json:"configSnapshotId,omitempty"`
	Mode             string `json:"mode,omitempty"`
	InputCount       int    `json:"inputCount,omitempty"`
}

type CreateBatchJobOutput struct {
	Batch workflowasync.BatchReference `json:"batch"`
}

type SubmitBatchJobInput struct {
	WorkflowRunID string                                   `json:"workflowRunId"`
	BatchJobID    string                                   `json:"batchJobId"`
	Items         []workflowasync.AsyncBatchSubmissionItem `json:"items,omitempty"`
}

type SubmitBatchJobOutput struct {
	Submission *workflowasync.AsyncBatchSubmissionResult `json:"submission,omitempty"`
}

type WaitForAsyncResultsInput struct {
	WorkflowRunID string   `json:"workflowRunId"`
	BatchJobIDs   []string `json:"batchJobIds,omitempty"`
}

type WaitForAsyncResultsOutput struct {
	Wait         common.ExternalWaitSummary                `json:"wait,omitempty"`
	Dependencies []workflowasync.PendingExternalDependency `json:"dependencies,omitempty"`
}

type PollAndReconcileBatchResultsInput struct {
	WorkflowRunID string   `json:"workflowRunId"`
	BatchJobIDs   []string `json:"batchJobIds,omitempty"`
	Force         bool     `json:"force,omitempty"`
}

type PollAndReconcileBatchResultsOutput struct {
	Result *workflowasync.AsyncBatchReconciliationResult `json:"result,omitempty"`
}

type ValidateAIOutputsInput struct {
	WorkflowRunID string   `json:"workflowRunId"`
	CandidateIDs  []string `json:"candidateIds,omitempty"`
	BatchJobIDs   []string `json:"batchJobIds,omitempty"`
}

type ValidateAIOutputsOutput struct {
	ValidCandidateIDs   []string                  `json:"validCandidateIds,omitempty"`
	InvalidCandidateIDs []string                  `json:"invalidCandidateIds,omitempty"`
	PendingCandidateIDs []string                  `json:"pendingCandidateIds,omitempty"`
	ValidCount          int                       `json:"validCount,omitempty"`
	InvalidCount        int                       `json:"invalidCount,omitempty"`
	Handoff             ValidationApprovalHandoff `json:"handoff"`
}

type ApproveTradeCandidatesInput struct {
	WorkflowRunID string                    `json:"workflowRunId"`
	Handoff       ValidationApprovalHandoff `json:"handoff"`
	Regime        string                    `json:"regime,omitempty"`
}

type ApproveTradeCandidatesOutput struct {
	ApprovedCount int                         `json:"approvedCount,omitempty"`
	RejectedCount int                         `json:"rejectedCount,omitempty"`
	Approved      []ApprovedTradeCandidateRef `json:"approved,omitempty"`
}

type PersistTradingReviewInput struct {
	WorkflowRunID string                      `json:"workflowRunId"`
	Candidates    []ApprovedTradeCandidateRef `json:"candidates,omitempty"`
}

type PersistTradingReviewOutput struct {
	PersistedCount int      `json:"persistedCount,omitempty"`
	ReviewIDs      []string `json:"reviewIds,omitempty"`
}

type PublishRunSummaryInput struct {
	WorkflowRunID string `json:"workflowRunId"`
}

type PublishRunSummaryOutput struct {
	Published bool   `json:"published"`
	SummaryID string `json:"summaryId,omitempty"`
}

func validateTradingStepRange(stepRange *common.StepRange) error {
	if stepRange == nil || stepRange.IsZero() {
		return nil
	}
	if err := stepRange.Validate(); err != nil {
		return err
	}

	indexByName := make(map[common.StepName]int, len(TradingStepDescriptors))
	for index, descriptor := range TradingStepDescriptors {
		indexByName[descriptor.Name] = index
	}

	startIndex := -1
	endIndex := -1

	if stepRange.Start != "" {
		value, ok := indexByName[stepRange.Start]
		if !ok {
			return fmt.Errorf("unknown trading step %q", stepRange.Start)
		}
		startIndex = value
	}
	if stepRange.End != "" {
		value, ok := indexByName[stepRange.End]
		if !ok {
			return fmt.Errorf("unknown trading step %q", stepRange.End)
		}
		endIndex = value
	}
	if startIndex >= 0 && endIndex >= 0 && startIndex > endIndex {
		return fmt.Errorf("trading step range start must be before or equal to end")
	}
	return nil
}

func validateIDs(fieldName string, values []string) error {
	seen := make(map[string]struct{}, len(values))
	for _, value := range values {
		trimmed := strings.TrimSpace(value)
		if trimmed == "" {
			return fmt.Errorf("%s cannot contain blank values", fieldName)
		}
		if _, exists := seen[trimmed]; exists {
			return fmt.Errorf("%s cannot contain duplicates", fieldName)
		}
		seen[trimmed] = struct{}{}
	}
	return nil
}
