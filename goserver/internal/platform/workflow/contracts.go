package workflow

import "goserver/internal/platform/domain"

type StepName string

const (
	InvestingStepScanUniverse                 StepName = "ScanUniverse"
	InvestingStepApplyHardFilters             StepName = "ApplyHardFilters"
	InvestingStepBuildReviewInputs            StepName = "BuildReviewInputs"
	InvestingStepCreatePendingReviewRecords   StepName = "CreatePendingReviewRecords"
	InvestingStepCreateBatchJob               StepName = "CreateBatchJob"
	InvestingStepSubmitBatchJob               StepName = "SubmitBatchJob"
	InvestingStepWaitForAsyncResults          StepName = "WaitForAsyncResults"
	InvestingStepPollAndReconcileBatchResults StepName = "PollAndReconcileBatchResults"
	InvestingStepValidateAIOutputs            StepName = "ValidateAIOutputs"
	InvestingStepMaterializeFinalReviews      StepName = "MaterializeFinalReviews"
	InvestingStepEvaluateThesisAndChange      StepName = "EvaluateThesisAndChange"
	InvestingStepMapActions                   StepName = "MapActions"
	InvestingStepAssignBuckets                StepName = "AssignBuckets"
	InvestingStepBuildCapitalCandidates       StepName = "BuildCapitalCandidates"
	InvestingStepAllocateCapital              StepName = "AllocateCapital"
	InvestingStepPersistOutputs               StepName = "PersistOutputs"
	InvestingStepPublishRunSummary            StepName = "PublishRunSummary"
)

const (
	TradingStepRefreshUniverse          StepName = "RefreshUniverse"
	TradingStepEvaluateRegime           StepName = "EvaluateRegime"
	TradingStepBuildTradingReviewInputs StepName = "BuildTradingReviewInputs"
	TradingStepCreateBatchJob           StepName = "CreateBatchJob"
	TradingStepSubmitBatchJob           StepName = "SubmitBatchJob"
	TradingStepWaitForAsyncResults      StepName = "WaitForAsyncResults"
	TradingStepPollAndReconcileResults  StepName = "PollAndReconcileBatchResults"
	TradingStepValidateAIOutputs        StepName = "ValidateAIOutputs"
	TradingStepApproveTradeCandidates   StepName = "ApproveTradeCandidates"
	TradingStepPersistTradingReview     StepName = "PersistTradingReview"
	TradingStepPublishRunSummary        StepName = "PublishRunSummary"
)

type ScanUniverseInput struct {
	RequestedCompanyIDs []string        `json:"requestedCompanyIds,omitempty"`
	Limit               int             `json:"limit,omitempty"`
	BookType            domain.BookType `json:"bookType"`
}

type ScanUniverseOutput struct {
	CompanyIDs []string `json:"companyIds"`
	Count      int      `json:"count"`
}

type ApplyHardFiltersInput struct {
	CompanyIDs []string `json:"companyIds"`
}

type ApplyHardFiltersOutput struct {
	EligibleCompanyIDs []string `json:"eligibleCompanyIds"`
	RejectedCompanyIDs []string `json:"rejectedCompanyIds,omitempty"`
}

type BuildReviewInputInput struct {
	CompanyIDs       []string `json:"companyIds"`
	ConfigSnapshotID string   `json:"configSnapshotId"`
}

type BuildReviewInputOutput struct {
	ReviewInputCount int `json:"reviewInputCount"`
}

type GenerateScorecardInput struct {
	CompanyIDs       []string `json:"companyIds"`
	ConfigSnapshotID string   `json:"configSnapshotId"`
}

type GenerateScorecardOutput struct {
	AsyncOnly bool   `json:"asyncOnly"`
	Mode      string `json:"mode"`
}

type EvaluateThesisInput struct {
	ReviewCount int `json:"reviewCount"`
}

type EvaluateThesisOutput struct {
	Placeholder bool `json:"placeholder"`
}

type MapActionInput struct {
	ReviewCount int `json:"reviewCount"`
}

type MapActionOutput struct {
	Placeholder bool `json:"placeholder"`
}

type AssignBucketInput struct {
	ReviewCount int `json:"reviewCount"`
}

type AssignBucketOutput struct {
	Placeholder bool `json:"placeholder"`
}

type CapitalCandidateInput struct {
	ReviewCount int `json:"reviewCount"`
}

type CapitalCandidateOutput struct {
	CandidateCount int `json:"candidateCount"`
}

type AllocateCapitalInput struct {
	CandidateCount int `json:"candidateCount"`
}

type AllocateCapitalOutput struct {
	AllocationRunPlanned bool `json:"allocationRunPlanned"`
}

type PersistOutputsInput struct {
	ReviewCount int `json:"reviewCount"`
}

type PersistOutputsOutput struct {
	Persisted bool `json:"persisted"`
}

type PublishSummaryInput struct {
	RunID string `json:"runId"`
}

type PublishSummaryOutput struct {
	SummaryReady bool `json:"summaryReady"`
}

func InvestingStepNames() []string {
	return []string{
		string(InvestingStepScanUniverse),
		string(InvestingStepApplyHardFilters),
		string(InvestingStepBuildReviewInputs),
		string(InvestingStepCreatePendingReviewRecords),
		string(InvestingStepCreateBatchJob),
		string(InvestingStepSubmitBatchJob),
		string(InvestingStepWaitForAsyncResults),
		string(InvestingStepPollAndReconcileBatchResults),
		string(InvestingStepValidateAIOutputs),
		string(InvestingStepMaterializeFinalReviews),
		string(InvestingStepEvaluateThesisAndChange),
		string(InvestingStepMapActions),
		string(InvestingStepAssignBuckets),
		string(InvestingStepBuildCapitalCandidates),
		string(InvestingStepAllocateCapital),
		string(InvestingStepPersistOutputs),
		string(InvestingStepPublishRunSummary),
	}
}

func TradingStepNames() []string {
	return []string{
		string(TradingStepRefreshUniverse),
		string(TradingStepEvaluateRegime),
		string(TradingStepBuildTradingReviewInputs),
		string(TradingStepCreateBatchJob),
		string(TradingStepSubmitBatchJob),
		string(TradingStepWaitForAsyncResults),
		string(TradingStepPollAndReconcileResults),
		string(TradingStepValidateAIOutputs),
		string(TradingStepApproveTradeCandidates),
		string(TradingStepPersistTradingReview),
		string(TradingStepPublishRunSummary),
	}
}
