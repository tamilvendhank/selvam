package domain

import (
	"fmt"
	"strings"
	"time"
)

type CapitalAllocationItem struct {
	CompanyID                     string     `json:"companyId" bson:"companyId"`
	DecisionReviewID              string     `json:"decisionReviewId" bson:"decisionReviewId"`
	ActionType                    ActionType `json:"actionType" bson:"actionType"`
	BuyPriorityRank               int        `json:"buyPriorityRank,omitempty" bson:"buyPriorityRank,omitempty"`
	CapitalPriorityScore          float64    `json:"capitalPriorityScore,omitempty" bson:"capitalPriorityScore,omitempty"`
	RecommendedAllocationAmount   float64    `json:"recommendedAllocationAmount,omitempty" bson:"recommendedAllocationAmount,omitempty"`
	RecommendedAllocationPctOfRun float64    `json:"recommendedAllocationPctOfRun,omitempty" bson:"recommendedAllocationPctOfRun,omitempty"`
	RecommendedTrancheNumber      int        `json:"recommendedTrancheNumber,omitempty" bson:"recommendedTrancheNumber,omitempty"`
	AllocationReason              string     `json:"allocationReason,omitempty" bson:"allocationReason,omitempty"`
	BlockedByConstraint           bool       `json:"blockedByConstraint" bson:"blockedByConstraint"`
	ConstraintReason              string     `json:"constraintReason,omitempty" bson:"constraintReason,omitempty"`
}

func (item *CapitalAllocationItem) Validate() error {
	if item == nil {
		return nil
	}
	if strings.TrimSpace(item.CompanyID) == "" {
		return fmt.Errorf("capital allocation item companyId is required")
	}
	if strings.TrimSpace(item.DecisionReviewID) == "" {
		return fmt.Errorf("capital allocation item decisionReviewId is required")
	}
	if !IsValidActionType(item.ActionType) {
		return fmt.Errorf("invalid capital allocation action type %q", item.ActionType)
	}
	if err := ValidatePercentRange("recommended allocation pct of run", item.RecommendedAllocationPctOfRun); err != nil {
		return err
	}

	return nil
}

type CapitalAllocationRun struct {
	ID                    string                  `json:"id" bson:"-"`
	WorkflowRunID         string                  `json:"workflowRunId" bson:"workflowRunId"`
	AllocationDate        time.Time               `json:"allocationDate" bson:"allocationDate"`
	BookType              BookType                `json:"bookType" bson:"bookType"`
	AvailableCashStart    float64                 `json:"availableCashStart" bson:"availableCashStart"`
	FreshMonthlyCash      float64                 `json:"freshMonthlyCash" bson:"freshMonthlyCash"`
	SellProceedsAvailable float64                 `json:"sellProceedsAvailable" bson:"sellProceedsAvailable"`
	CarryForwardCash      float64                 `json:"carryForwardCash" bson:"carryForwardCash"`
	TargetDeployableCash  float64                 `json:"targetDeployableCash" bson:"targetDeployableCash"`
	AllocatedCashTotal    float64                 `json:"allocatedCashTotal" bson:"allocatedCashTotal"`
	CashLeftUnallocated   float64                 `json:"cashLeftUnallocated" bson:"cashLeftUnallocated"`
	AllocationNotes       string                  `json:"allocationNotes,omitempty" bson:"allocationNotes,omitempty"`
	Items                 []CapitalAllocationItem `json:"items,omitempty" bson:"items,omitempty"`
	SchemaVersion         string                  `json:"schemaVersion" bson:"schemaVersion"`
	CreatedAt             time.Time               `json:"createdAt" bson:"createdAt"`
}

func (run *CapitalAllocationRun) Validate() error {
	if run == nil {
		return fmt.Errorf("capital allocation run is required")
	}
	if !IsValidBookType(run.BookType) {
		return fmt.Errorf("invalid capital allocation book type %q", run.BookType)
	}
	if err := ValidateNonZeroTime("allocation date", run.AllocationDate); err != nil {
		return err
	}
	if strings.TrimSpace(run.SchemaVersion) == "" {
		return fmt.Errorf("capital allocation schema version is required")
	}
	for index := range run.Items {
		if err := run.Items[index].Validate(); err != nil {
			return err
		}
	}
	if err := ValidateNonZeroTime("allocation createdAt", run.CreatedAt); err != nil {
		return err
	}

	return nil
}
