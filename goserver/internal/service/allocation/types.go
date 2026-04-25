package allocation

import (
	"time"

	domaincommon "goserver/internal/domain/common"
	servicecommon "goserver/internal/service/common"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

type CapitalCandidateRef struct {
	CompanyID            primitive.ObjectID               `json:"companyId"`
	ReviewID             primitive.ObjectID               `json:"reviewId,omitempty"`
	WorkflowRunID        primitive.ObjectID               `json:"workflowRunId,omitempty"`
	Symbol               string                           `json:"symbol,omitempty"`
	ActionType           domaincommon.InvestingActionType `json:"actionType,omitempty"`
	CurrentBucket        domaincommon.WatchlistBucket     `json:"currentBucket,omitempty"`
	PriorityRank         int                              `json:"priorityRank,omitempty"`
	PriorityScore        float64                          `json:"priorityScore,omitempty"`
	RecommendedTargetPct float64                          `json:"recommendedTargetPct,omitempty"`
	ConstraintReasons    []string                         `json:"constraintReasons,omitempty"`
}

type SkippedCapitalCandidate struct {
	Candidate CapitalCandidateRef `json:"candidate"`
	Reason    string              `json:"reason,omitempty"`
	Code      string              `json:"code,omitempty"`
}

type BuildCapitalCandidatesRequest struct {
	WorkflowRunID primitive.ObjectID    `json:"workflowRunId"`
	BookType      domaincommon.BookType `json:"bookType,omitempty"`
	AsOfDate      time.Time             `json:"asOfDate,omitempty"`
	MaxCandidates int                   `json:"maxCandidates,omitempty"`
	DryRun        bool                  `json:"dryRun,omitempty"`
	Force         bool                  `json:"force,omitempty"`
	InitiatedBy   string                `json:"initiatedBy,omitempty"`
	CorrelationID string                `json:"correlationId,omitempty"`
}

func (request BuildCapitalCandidatesRequest) Validate() error {
	if err := servicecommon.ValidateRequiredObjectID("workflowRunId", request.WorkflowRunID); err != nil {
		return err
	}
	if err := servicecommon.ValidateOptionalMax("maxCandidates", request.MaxCandidates); err != nil {
		return err
	}
	if err := servicecommon.ValidateOptionalBookType(request.BookType); err != nil {
		return err
	}
	if err := servicecommon.ValidateOptionalTimeYear("asOfDate", request.AsOfDate, 1900); err != nil {
		return err
	}
	if err := servicecommon.ValidateOptionalText("initiatedBy", request.InitiatedBy); err != nil {
		return err
	}
	return servicecommon.ValidateOptionalText("correlationId", request.CorrelationID)
}

type BuildCapitalCandidatesResult struct {
	WorkflowRunID        primitive.ObjectID              `json:"workflowRunId,omitempty"`
	CandidateCount       int                             `json:"candidateCount,omitempty"`
	RankedCandidateRefs  []CapitalCandidateRef           `json:"rankedCandidateRefs,omitempty"`
	SkippedCandidates    []SkippedCapitalCandidate       `json:"skippedCandidates,omitempty"`
	IneligibleCandidates []SkippedCapitalCandidate       `json:"ineligibleCandidates,omitempty"`
	PartialFailures      []servicecommon.PartialFailure  `json:"partialFailures,omitempty"`
	Summary              servicecommon.AllocationSummary `json:"summary,omitempty"`
}

func (result BuildCapitalCandidatesResult) HasEligibleCandidates() bool {
	return result.CandidateCount > 0 || len(result.RankedCandidateRefs) > 0
}

func (result BuildCapitalCandidatesResult) HasFailures() bool {
	return len(result.PartialFailures) > 0 || result.Summary.HasFailures()
}

type AllocateCapitalRequest struct {
	WorkflowRunID         primitive.ObjectID    `json:"workflowRunId"`
	AllocationDate        time.Time             `json:"allocationDate"`
	AvailableCashStart    float64               `json:"availableCashStart"`
	FreshMonthlyCash      float64               `json:"freshMonthlyCash"`
	SellProceedsAvailable float64               `json:"sellProceedsAvailable"`
	CarryForwardCash      float64               `json:"carryForwardCash"`
	CandidateRefs         []CapitalCandidateRef `json:"candidateRefs,omitempty"`
	DryRun                bool                  `json:"dryRun,omitempty"`
	Force                 bool                  `json:"force,omitempty"`
	InitiatedBy           string                `json:"initiatedBy,omitempty"`
	CorrelationID         string                `json:"correlationId,omitempty"`
}

func (request AllocateCapitalRequest) Validate() error {
	if err := servicecommon.ValidateRequiredObjectID("workflowRunId", request.WorkflowRunID); err != nil {
		return err
	}
	if err := servicecommon.ValidateRequiredTime("allocationDate", request.AllocationDate); err != nil {
		return err
	}
	if err := servicecommon.ValidatePositiveMoney("availableCashStart", request.AvailableCashStart); err != nil {
		return err
	}
	if err := servicecommon.ValidatePositiveMoney("freshMonthlyCash", request.FreshMonthlyCash); err != nil {
		return err
	}
	if err := servicecommon.ValidatePositiveMoney("sellProceedsAvailable", request.SellProceedsAvailable); err != nil {
		return err
	}
	if err := servicecommon.ValidatePositiveMoney("carryForwardCash", request.CarryForwardCash); err != nil {
		return err
	}
	if err := servicecommon.ValidateOptionalText("initiatedBy", request.InitiatedBy); err != nil {
		return err
	}
	return servicecommon.ValidateOptionalText("correlationId", request.CorrelationID)
}

type AllocatedCandidateRef struct {
	CapitalCandidateRef
	RecommendedAllocationAmount   float64 `json:"recommendedAllocationAmount,omitempty"`
	RecommendedAllocationPctOfRun float64 `json:"recommendedAllocationPctOfRun,omitempty"`
	RecommendedTrancheNumber      int     `json:"recommendedTrancheNumber,omitempty"`
	AllocationReason              string  `json:"allocationReason,omitempty"`
}

type BlockedCapitalCandidate struct {
	Candidate        CapitalCandidateRef `json:"candidate"`
	ConstraintReason string              `json:"constraintReason,omitempty"`
}

type AllocateCapitalResult struct {
	WorkflowRunID          primitive.ObjectID              `json:"workflowRunId,omitempty"`
	CapitalAllocationRunID primitive.ObjectID              `json:"capitalAllocationRunId,omitempty"`
	AllocatedCandidates    []AllocatedCandidateRef         `json:"allocatedCandidates,omitempty"`
	BlockedCandidates      []BlockedCapitalCandidate       `json:"blockedCandidates,omitempty"`
	UnallocatedCash        float64                         `json:"unallocatedCash,omitempty"`
	PartialFailures        []servicecommon.PartialFailure  `json:"partialFailures,omitempty"`
	Summary                servicecommon.AllocationSummary `json:"summary,omitempty"`
}

func (result AllocateCapitalResult) HasFailures() bool {
	return len(result.PartialFailures) > 0 || result.Summary.HasFailures()
}
