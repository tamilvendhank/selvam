package allocation

import (
	"fmt"
	"time"

	"goserver/internal/domain/common"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

type CapitalAllocationItem struct {
	CompanyID                     primitive.ObjectID         `bson:"companyId" json:"companyId"`
	DecisionReviewID              primitive.ObjectID         `bson:"decisionReviewId" json:"decisionReviewId"`
	ActionType                    common.InvestingActionType `bson:"actionType" json:"actionType"`
	BuyPriorityRank               int                        `bson:"buyPriorityRank,omitempty" json:"buyPriorityRank,omitempty"`
	CapitalPriorityScore          float64                    `bson:"capitalPriorityScore,omitempty" json:"capitalPriorityScore,omitempty"`
	RecommendedAllocationAmount   float64                    `bson:"recommendedAllocationAmount" json:"recommendedAllocationAmount"`
	RecommendedAllocationPctOfRun float64                    `bson:"recommendedAllocationPctOfRun" json:"recommendedAllocationPctOfRun"`
	RecommendedTrancheNumber      int                        `bson:"recommendedTrancheNumber,omitempty" json:"recommendedTrancheNumber,omitempty"`
	AllocationReason              string                     `bson:"allocationReason,omitempty" json:"allocationReason,omitempty"`
	BlockedByConstraint           bool                       `bson:"blockedByConstraint" json:"blockedByConstraint"`
	ConstraintReason              string                     `bson:"constraintReason,omitempty" json:"constraintReason,omitempty"`
}

func (item CapitalAllocationItem) Validate() error {
	if err := common.RequireObjectID("companyId", item.CompanyID); err != nil {
		return err
	}
	if err := common.RequireObjectID("decisionReviewId", item.DecisionReviewID); err != nil {
		return err
	}
	if !item.ActionType.IsValid() {
		return fmt.Errorf("invalid actionType %q", item.ActionType)
	}
	if err := common.ValidateNonNegativeInt("buyPriorityRank", item.BuyPriorityRank); err != nil {
		return err
	}
	if err := common.ValidateNonNegativeFloat("capitalPriorityScore", item.CapitalPriorityScore); err != nil {
		return err
	}
	if err := common.ValidateNonNegativeFloat("recommendedAllocationAmount", item.RecommendedAllocationAmount); err != nil {
		return err
	}
	if err := common.ValidatePercentage("recommendedAllocationPctOfRun", item.RecommendedAllocationPctOfRun); err != nil {
		return err
	}
	if item.RecommendedAllocationAmount > 0 {
		if err := common.ValidatePositiveInt("recommendedTrancheNumber", item.RecommendedTrancheNumber); err != nil {
			return err
		}
	}
	if item.BlockedByConstraint && item.ConstraintReason == "" {
		return fmt.Errorf("constraintReason is required when blockedByConstraint is true")
	}
	return nil
}

type CapitalAllocationRun struct {
	ID                    primitive.ObjectID      `bson:"_id,omitempty" json:"id,omitempty"`
	WorkflowRunID         primitive.ObjectID      `bson:"workflowRunId" json:"workflowRunId"`
	AllocationDate        time.Time               `bson:"allocationDate" json:"allocationDate"`
	BookType              common.BookType         `bson:"bookType" json:"bookType"`
	AvailableCashStart    float64                 `bson:"availableCashStart" json:"availableCashStart"`
	FreshMonthlyCash      float64                 `bson:"freshMonthlyCash" json:"freshMonthlyCash"`
	SellProceedsAvailable float64                 `bson:"sellProceedsAvailable" json:"sellProceedsAvailable"`
	CarryForwardCash      float64                 `bson:"carryForwardCash" json:"carryForwardCash"`
	TargetDeployableCash  float64                 `bson:"targetDeployableCash" json:"targetDeployableCash"`
	AllocatedCashTotal    float64                 `bson:"allocatedCashTotal" json:"allocatedCashTotal"`
	CashLeftUnallocated   float64                 `bson:"cashLeftUnallocated" json:"cashLeftUnallocated"`
	AllocationNotes       string                  `bson:"allocationNotes,omitempty" json:"allocationNotes,omitempty"`
	Items                 []CapitalAllocationItem `bson:"items,omitempty" json:"items,omitempty"`
	CreatedAt             time.Time               `bson:"createdAt" json:"createdAt"`
	SchemaVersion         int                     `bson:"schemaVersion" json:"schemaVersion"`
}

func (run CapitalAllocationRun) Validate() error {
	if err := common.RequireObjectID("workflowRunId", run.WorkflowRunID); err != nil {
		return err
	}
	if err := common.RequireTime("allocationDate", run.AllocationDate); err != nil {
		return err
	}
	if !run.BookType.IsValid() {
		return fmt.Errorf("invalid bookType %q", run.BookType)
	}
	if err := common.ValidateNonNegativeFloat("availableCashStart", run.AvailableCashStart); err != nil {
		return err
	}
	if err := common.ValidateNonNegativeFloat("freshMonthlyCash", run.FreshMonthlyCash); err != nil {
		return err
	}
	if err := common.ValidateNonNegativeFloat("sellProceedsAvailable", run.SellProceedsAvailable); err != nil {
		return err
	}
	if err := common.ValidateNonNegativeFloat("carryForwardCash", run.CarryForwardCash); err != nil {
		return err
	}
	if err := common.ValidateNonNegativeFloat("targetDeployableCash", run.TargetDeployableCash); err != nil {
		return err
	}
	if err := common.ValidateNonNegativeFloat("allocatedCashTotal", run.AllocatedCashTotal); err != nil {
		return err
	}
	if err := common.ValidateNonNegativeFloat("cashLeftUnallocated", run.CashLeftUnallocated); err != nil {
		return err
	}
	var allocatedTotal float64
	for _, item := range run.Items {
		if err := item.Validate(); err != nil {
			return err
		}
		allocatedTotal += item.RecommendedAllocationAmount
	}
	if len(run.Items) > 0 && !common.NearlyEqual(allocatedTotal, run.AllocatedCashTotal) {
		return fmt.Errorf("allocatedCashTotal must equal the sum of allocation items")
	}
	if err := common.RequireTime("createdAt", run.CreatedAt); err != nil {
		return err
	}
	if err := common.ValidateSchemaVersion("schemaVersion", run.SchemaVersion); err != nil {
		return err
	}
	return nil
}
