package investing

import (
	"fmt"
	"strings"
	"time"

	"goserver/internal/platform/domain"
	workflowasync "goserver/internal/platform/workflow/async"
	"goserver/internal/platform/workflow/common"
)

var InvestingStepDescriptors = []common.WorkflowStepDescriptor{
	{Name: common.InvestingStepScanUniverse, DisplayName: "Scan Universe", BookType: domain.BookTypeInvesting, Phase: common.StepPhasePreparation},
	{Name: common.InvestingStepApplyHardFilters, DisplayName: "Apply Hard Filters", BookType: domain.BookTypeInvesting, Phase: common.StepPhasePreparation},
	{Name: common.InvestingStepBuildReviewInputs, DisplayName: "Build Review Inputs", BookType: domain.BookTypeInvesting, Phase: common.StepPhaseInputBuild},
	{Name: common.InvestingStepCreatePendingReviewRecords, DisplayName: "Create Pending Review Records", BookType: domain.BookTypeInvesting, Phase: common.StepPhasePersistence},
	{Name: common.InvestingStepCreateBatchJob, DisplayName: "Create Batch Job", BookType: domain.BookTypeInvesting, Phase: common.StepPhaseAISubmission, AsyncBoundary: true},
	{Name: common.InvestingStepSubmitBatchJob, DisplayName: "Submit Batch Job", BookType: domain.BookTypeInvesting, Phase: common.StepPhaseAISubmission, AsyncBoundary: true},
	{Name: common.InvestingStepWaitForAsyncResults, DisplayName: "Wait For Async Results", BookType: domain.BookTypeInvesting, Phase: common.StepPhaseExternalWait, WaitsExternal: true},
	{Name: common.InvestingStepPollAndReconcileBatchResults, DisplayName: "Poll And Reconcile Batch Results", BookType: domain.BookTypeInvesting, Phase: common.StepPhaseReconciliation},
	{Name: common.InvestingStepValidateAIOutputs, DisplayName: "Validate AI Outputs", BookType: domain.BookTypeInvesting, Phase: common.StepPhaseValidation},
	{Name: common.InvestingStepMaterializeFinalReviews, DisplayName: "Materialize Final Reviews", BookType: domain.BookTypeInvesting, Phase: common.StepPhaseMaterialize},
	{Name: common.InvestingStepEvaluateThesisAndChange, DisplayName: "Evaluate Thesis And Change", BookType: domain.BookTypeInvesting, Phase: common.StepPhasePostProcessing},
	{Name: common.InvestingStepMapActions, DisplayName: "Map Actions", BookType: domain.BookTypeInvesting, Phase: common.StepPhasePostProcessing},
	{Name: common.InvestingStepAssignBuckets, DisplayName: "Assign Buckets", BookType: domain.BookTypeInvesting, Phase: common.StepPhasePostProcessing},
	{Name: common.InvestingStepBuildCapitalCandidates, DisplayName: "Build Capital Candidates", BookType: domain.BookTypeInvesting, Phase: common.StepPhasePostProcessing},
	{Name: common.InvestingStepAllocateCapital, DisplayName: "Allocate Capital", BookType: domain.BookTypeInvesting, Phase: common.StepPhasePostProcessing},
	{Name: common.InvestingStepPersistOutputs, DisplayName: "Persist Outputs", BookType: domain.BookTypeInvesting, Phase: common.StepPhasePersistence},
	{Name: common.InvestingStepPublishRunSummary, DisplayName: "Publish Run Summary", BookType: domain.BookTypeInvesting, Phase: common.StepPhaseSummary},
}

func InvestingStepSequence() []common.StepName {
	steps := make([]common.StepName, 0, len(InvestingStepDescriptors))
	for _, descriptor := range InvestingStepDescriptors {
		steps = append(steps, descriptor.Name)
	}
	return steps
}

type StartInvestingWorkflowRequest struct {
	common.WorkflowStartRequest
	ReviewDate      *time.Time           `json:"reviewDate,omitempty"`
	Mode            domain.InvestingMode `json:"mode,omitempty"`
	CompanyIDs      []string             `json:"companyIds,omitempty"`
	Limit           int                  `json:"limit,omitempty"`
	ReplayFromRunID string               `json:"replayFromRunId,omitempty"`
}

func (request StartInvestingWorkflowRequest) Validate() error {
	if err := request.WorkflowStartRequest.Validate(); err != nil {
		return err
	}
	if request.BookType != "" && request.BookType != domain.BookTypeInvesting {
		return fmt.Errorf("start investing workflow request bookType must be %q", domain.BookTypeInvesting)
	}
	if request.Mode != "" && !domain.IsValidInvestingMode(request.Mode) {
		return fmt.Errorf("invalid investing mode %q", request.Mode)
	}
	if request.ReviewDate != nil && request.ReviewDate.IsZero() {
		return fmt.Errorf("reviewDate cannot be zero")
	}
	if request.Limit < 0 {
		return fmt.Errorf("limit cannot be negative")
	}
	if err := validateIDs("companyIds", request.CompanyIDs); err != nil {
		return err
	}
	if request.ReplayFromRunID != "" && strings.TrimSpace(request.ReplayFromRunID) == "" {
		return fmt.Errorf("replayFromRunId cannot be blank")
	}
	return validateInvestingStepRange(request.AllowedStepRange)
}

type ResumeInvestingWorkflowRequest struct {
	common.WorkflowResumeRequest
}

func (request ResumeInvestingWorkflowRequest) Validate() error {
	if err := request.WorkflowResumeRequest.Validate(); err != nil {
		return err
	}
	return validateInvestingStepRange(request.AllowedStepRange)
}

type ReconcileInvestingWorkflowRequest struct {
	common.WorkflowReconcileRequest
}

func (request ReconcileInvestingWorkflowRequest) Validate() error {
	if err := request.WorkflowReconcileRequest.Validate(); err != nil {
		return err
	}
	return validateInvestingStepRange(request.AllowedStepRange)
}

type StartInvestingWorkflowResult struct {
	Run    common.WorkflowStartResult `json:"run"`
	Status InvestingWorkflowStatus    `json:"status"`
}

type ResumeInvestingWorkflowResult struct {
	Run    common.WorkflowResumeResult `json:"run"`
	Status InvestingWorkflowStatus     `json:"status"`
}

type ReconcileInvestingWorkflowResult struct {
	Run    common.WorkflowReconcileResult `json:"run"`
	Status InvestingWorkflowStatus        `json:"status"`
}

type InvestingWorkflowCounts struct {
	common.WorkflowCounts
	FinalReviews      int `json:"finalReviews,omitempty"`
	ThesisChanges     int `json:"thesisChanges,omitempty"`
	ActionsMapped     int `json:"actionsMapped,omitempty"`
	BucketsAssigned   int `json:"bucketsAssigned,omitempty"`
	CapitalCandidates int `json:"capitalCandidates,omitempty"`
	AllocationRuns    int `json:"allocationRuns,omitempty"`
}

type InvestingWorkflowStatus struct {
	common.WorkflowStatusView
	ConfigSnapshotID    string                              `json:"configSnapshotId,omitempty"`
	RequestedCompanyIDs []string                            `json:"requestedCompanyIds,omitempty"`
	BatchJobs           []workflowasync.BatchStatusSnapshot `json:"batchJobs,omitempty"`
	Counts              InvestingWorkflowCounts             `json:"counts,omitempty"`
	Summary             *InvestingWorkflowSummary           `json:"summary,omitempty"`
}

type InvestingWorkflowSummary struct {
	WorkflowRunID       string                         `json:"workflowRunId"`
	RunType             domain.WorkflowRunType         `json:"runType,omitempty"`
	Mode                domain.InvestingMode           `json:"mode,omitempty"`
	ReviewDate          *time.Time                     `json:"reviewDate,omitempty"`
	ConfigSnapshotID    string                         `json:"configSnapshotId,omitempty"`
	Counts              InvestingWorkflowCounts        `json:"counts,omitempty"`
	ActionCounts        map[domain.ActionType]int      `json:"actionCounts,omitempty"`
	BucketCounts        map[domain.WatchlistBucket]int `json:"bucketCounts,omitempty"`
	BatchJobIDs         []string                       `json:"batchJobIds,omitempty"`
	ReviewIDs           []string                       `json:"reviewIds,omitempty"`
	CapitalCandidateIDs []string                       `json:"capitalCandidateIds,omitempty"`
	AllocationRunIDs    []string                       `json:"allocationRunIds,omitempty"`
	PartialFailure      *common.PartialFailureSummary  `json:"partialFailure,omitempty"`
	Published           bool                           `json:"published,omitempty"`
	PublishedAt         *time.Time                     `json:"publishedAt,omitempty"`
}

type ReviewInputCandidate struct {
	CompanyID        string         `json:"companyId"`
	Symbol           string         `json:"symbol,omitempty"`
	ConfigSnapshotID string         `json:"configSnapshotId,omitempty"`
	Metadata         map[string]any `json:"metadata,omitempty"`
}

type PendingReviewRecordRef struct {
	ReviewID    string `json:"reviewId"`
	CompanyID   string `json:"companyId"`
	Symbol      string `json:"symbol,omitempty"`
	BatchItemID string `json:"batchItemId,omitempty"`
}

type FinalReviewRef struct {
	ReviewID  string `json:"reviewId"`
	CompanyID string `json:"companyId"`
	ThesisID  string `json:"thesisId,omitempty"`
}

type CapitalCandidateRef struct {
	CandidateID string                 `json:"candidateId"`
	CompanyID   string                 `json:"companyId"`
	ReviewID    string                 `json:"reviewId"`
	Action      domain.ActionType      `json:"action,omitempty"`
	Bucket      domain.WatchlistBucket `json:"bucket,omitempty"`
}

type ValidationMaterializationHandoff struct {
	WorkflowRunID    string   `json:"workflowRunId"`
	Ready            bool     `json:"ready"`
	ValidReviewIDs   []string `json:"validReviewIds,omitempty"`
	InvalidReviewIDs []string `json:"invalidReviewIds,omitempty"`
	PendingReviewIDs []string `json:"pendingReviewIds,omitempty"`
}

type ScanUniverseInput struct {
	ReviewDate   *time.Time `json:"reviewDate,omitempty"`
	RequestedIDs []string   `json:"requestedIds,omitempty"`
	Limit        int        `json:"limit,omitempty"`
	DryRun       bool       `json:"dryRun,omitempty"`
}

type ScanUniverseOutput struct {
	CompanyIDs []string `json:"companyIds,omitempty"`
	Count      int      `json:"count,omitempty"`
}

type ApplyHardFiltersInput struct {
	CompanyIDs       []string `json:"companyIds"`
	ConfigSnapshotID string   `json:"configSnapshotId,omitempty"`
}

type ApplyHardFiltersOutput struct {
	EligibleCompanyIDs []string `json:"eligibleCompanyIds,omitempty"`
	RejectedCompanyIDs []string `json:"rejectedCompanyIds,omitempty"`
	EligibleCount      int      `json:"eligibleCount,omitempty"`
	RejectedCount      int      `json:"rejectedCount,omitempty"`
}

type BuildReviewInputsInput struct {
	CompanyIDs       []string             `json:"companyIds"`
	ConfigSnapshotID string               `json:"configSnapshotId"`
	Mode             domain.InvestingMode `json:"mode,omitempty"`
	ReviewDate       *time.Time           `json:"reviewDate,omitempty"`
}

type BuildReviewInputsOutput struct {
	InputCount int                    `json:"inputCount,omitempty"`
	Inputs     []ReviewInputCandidate `json:"inputs,omitempty"`
}

type CreatePendingReviewRecordsInput struct {
	WorkflowRunID    string                 `json:"workflowRunId"`
	ConfigSnapshotID string                 `json:"configSnapshotId"`
	Mode             domain.InvestingMode   `json:"mode,omitempty"`
	ReviewDate       *time.Time             `json:"reviewDate,omitempty"`
	Inputs           []ReviewInputCandidate `json:"inputs,omitempty"`
}

type CreatePendingReviewRecordsOutput struct {
	CreatedCount   int                      `json:"createdCount,omitempty"`
	ReviewIDs      []string                 `json:"reviewIds,omitempty"`
	PendingReviews []PendingReviewRecordRef `json:"pendingReviews,omitempty"`
}

type CreateBatchJobInput struct {
	WorkflowRunID    string               `json:"workflowRunId"`
	ConfigSnapshotID string               `json:"configSnapshotId,omitempty"`
	Mode             domain.InvestingMode `json:"mode,omitempty"`
	ReviewCount      int                  `json:"reviewCount,omitempty"`
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
	ReviewIDs     []string `json:"reviewIds,omitempty"`
	BatchJobIDs   []string `json:"batchJobIds,omitempty"`
}

type ValidateAIOutputsOutput struct {
	ValidReviewIDs   []string                         `json:"validReviewIds,omitempty"`
	InvalidReviewIDs []string                         `json:"invalidReviewIds,omitempty"`
	PendingReviewIDs []string                         `json:"pendingReviewIds,omitempty"`
	ValidCount       int                              `json:"validCount,omitempty"`
	InvalidCount     int                              `json:"invalidCount,omitempty"`
	Handoff          ValidationMaterializationHandoff `json:"handoff"`
}

type MaterializeFinalReviewsInput struct {
	WorkflowRunID string                           `json:"workflowRunId"`
	Handoff       ValidationMaterializationHandoff `json:"handoff"`
}

type MaterializeFinalReviewsOutput struct {
	MaterializedCount int              `json:"materializedCount,omitempty"`
	FinalReviews      []FinalReviewRef `json:"finalReviews,omitempty"`
}

type EvaluateThesisAndChangeInput struct {
	WorkflowRunID string   `json:"workflowRunId"`
	ReviewIDs     []string `json:"reviewIds,omitempty"`
}

type EvaluateThesisAndChangeOutput struct {
	EvaluatedCount     int      `json:"evaluatedCount,omitempty"`
	ChangedThesisCount int      `json:"changedThesisCount,omitempty"`
	ChangedCompanyIDs  []string `json:"changedCompanyIds,omitempty"`
	ThesisIDs          []string `json:"thesisIds,omitempty"`
}

type MapActionsInput struct {
	WorkflowRunID string   `json:"workflowRunId"`
	ReviewIDs     []string `json:"reviewIds,omitempty"`
	ThesisIDs     []string `json:"thesisIds,omitempty"`
}

type MapActionsOutput struct {
	MappedCount  int                       `json:"mappedCount,omitempty"`
	ActionCounts map[domain.ActionType]int `json:"actionCounts,omitempty"`
}

type AssignBucketsInput struct {
	WorkflowRunID string   `json:"workflowRunId"`
	ReviewIDs     []string `json:"reviewIds,omitempty"`
}

type AssignBucketsOutput struct {
	AssignedCount int                            `json:"assignedCount,omitempty"`
	BucketCounts  map[domain.WatchlistBucket]int `json:"bucketCounts,omitempty"`
}

type BuildCapitalCandidatesInput struct {
	WorkflowRunID string   `json:"workflowRunId"`
	ReviewIDs     []string `json:"reviewIds,omitempty"`
}

type BuildCapitalCandidatesOutput struct {
	CandidateCount int                   `json:"candidateCount,omitempty"`
	Candidates     []CapitalCandidateRef `json:"candidates,omitempty"`
}

type AllocateCapitalInput struct {
	WorkflowRunID string                `json:"workflowRunId"`
	Candidates    []CapitalCandidateRef `json:"candidates,omitempty"`
	DryRun        bool                  `json:"dryRun,omitempty"`
}

type AllocateCapitalOutput struct {
	AllocationRunID string   `json:"allocationRunId,omitempty"`
	AllocatedCount  int      `json:"allocatedCount,omitempty"`
	CandidateIDs    []string `json:"candidateIds,omitempty"`
}

type PersistOutputsInput struct {
	WorkflowRunID    string   `json:"workflowRunId"`
	ReviewIDs        []string `json:"reviewIds,omitempty"`
	AllocationRunIDs []string `json:"allocationRunIds,omitempty"`
}

type PersistOutputsOutput struct {
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

func validateInvestingStepRange(stepRange *common.StepRange) error {
	if stepRange == nil || stepRange.IsZero() {
		return nil
	}
	if err := stepRange.Validate(); err != nil {
		return err
	}

	indexByName := make(map[common.StepName]int, len(InvestingStepDescriptors))
	for index, descriptor := range InvestingStepDescriptors {
		indexByName[descriptor.Name] = index
	}

	startIndex := -1
	endIndex := -1

	if stepRange.Start != "" {
		value, ok := indexByName[stepRange.Start]
		if !ok {
			return fmt.Errorf("unknown investing step %q", stepRange.Start)
		}
		startIndex = value
	}
	if stepRange.End != "" {
		value, ok := indexByName[stepRange.End]
		if !ok {
			return fmt.Errorf("unknown investing step %q", stepRange.End)
		}
		endIndex = value
	}
	if startIndex >= 0 && endIndex >= 0 && startIndex > endIndex {
		return fmt.Errorf("investing step range start must be before or equal to end")
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
