package app

import (
	"time"

	domaincommon "goserver/internal/domain/common"
)

type AllocationRunListItemDTO struct {
	AllocationRunID       string                `json:"allocationRunId"`
	WorkflowRunID         string                `json:"workflowRunId,omitempty"`
	AllocationDate        time.Time             `json:"allocationDate"`
	BookType              domaincommon.BookType `json:"bookType,omitempty"`
	AvailableCashStart    float64               `json:"availableCashStart"`
	FreshMonthlyCash      float64               `json:"freshMonthlyCash"`
	SellProceedsAvailable float64               `json:"sellProceedsAvailable"`
	CarryForwardCash      float64               `json:"carryForwardCash"`
	TargetDeployableCash  float64               `json:"targetDeployableCash"`
	AllocatedCashTotal    float64               `json:"allocatedCashTotal"`
	CashLeftUnallocated   float64               `json:"cashLeftUnallocated"`
	ItemCount             int                   `json:"itemCount"`
	CreatedAt             time.Time             `json:"createdAt"`
}

type AllocationRunDetailDTO struct {
	AllocationRunListItemDTO
	AllocationNotes   string              `json:"allocationNotes,omitempty"`
	Items             []AllocationItemDTO `json:"items,omitempty"`
	BlockedCandidates []AllocationItemDTO `json:"blockedCandidates,omitempty"`
}

type AllocationItemDTO struct {
	CompanyID                     string                           `json:"companyId,omitempty"`
	Symbol                        string                           `json:"symbol,omitempty"`
	DecisionReviewID              string                           `json:"decisionReviewId,omitempty"`
	ActionType                    domaincommon.InvestingActionType `json:"actionType,omitempty"`
	BuyPriorityRank               int                              `json:"buyPriorityRank,omitempty"`
	CapitalPriorityScore          float64                          `json:"capitalPriorityScore,omitempty"`
	RecommendedAllocationAmount   float64                          `json:"recommendedAllocationAmount"`
	RecommendedAllocationPctOfRun float64                          `json:"recommendedAllocationPctOfRun"`
	RecommendedTrancheNumber      int                              `json:"recommendedTrancheNumber,omitempty"`
	AllocationReason              string                           `json:"allocationReason,omitempty"`
	BlockedByConstraint           bool                             `json:"blockedByConstraint"`
	ConstraintReason              string                           `json:"constraintReason,omitempty"`
}
