package allocation

import (
	"fmt"
	"math"
	"strings"
	"time"

	domainallocation "goserver/internal/domain/allocation"
	domaincommon "goserver/internal/domain/common"
)

func computeDeployableCash(request AllocateCapitalRequest) float64 {
	return roundMoney(request.AvailableCashStart +
		request.FreshMonthlyCash +
		request.SellProceedsAvailable +
		request.CarryForwardCash)
}

func buildAllocationItems(plan allocationPlan) []domainallocation.CapitalAllocationItem {
	items := make([]domainallocation.CapitalAllocationItem, 0, len(plan.Decisions))
	allocatedTotal := 0.0
	for _, decision := range plan.Decisions {
		allocatedTotal += decision.Amount
	}
	allocatedTotal = roundMoney(allocatedTotal)

	for _, decision := range plan.Decisions {
		candidate := decision.Candidate
		if candidate.Ref.CompanyID.IsZero() || candidate.Ref.ReviewID.IsZero() || !candidate.Ref.ActionType.IsValid() {
			continue
		}
		item := domainallocation.CapitalAllocationItem{
			CompanyID:                     candidate.Ref.CompanyID,
			DecisionReviewID:              candidate.Ref.ReviewID,
			ActionType:                    candidate.Ref.ActionType,
			BuyPriorityRank:               candidate.Ref.PriorityRank,
			CapitalPriorityScore:          candidate.Ref.PriorityScore,
			RecommendedAllocationAmount:   decision.Amount,
			RecommendedAllocationPctOfRun: allocationPctOfRun(decision.Amount, allocatedTotal),
			RecommendedTrancheNumber:      recommendedTrancheNumber(candidate),
			AllocationReason:              allocationReason(decision),
			BlockedByConstraint:           decision.ConstraintReason != "",
			ConstraintReason:              decision.ConstraintReason,
		}
		if decision.Amount <= scoreEpsilon {
			item.RecommendedTrancheNumber = 0
			item.RecommendedAllocationPctOfRun = 0
			item.RecommendedAllocationAmount = 0
			if item.ConstraintReason == "" {
				item.BlockedByConstraint = true
				item.ConstraintReason = "not_allocated"
			}
		}
		items = append(items, item)
	}

	return items
}

func buildAllocationRun(
	request AllocateCapitalRequest,
	deployableCash float64,
	allocatedTotal float64,
	unallocatedCash float64,
	items []domainallocation.CapitalAllocationItem,
	now time.Time,
) *domainallocation.CapitalAllocationRun {
	return &domainallocation.CapitalAllocationRun{
		WorkflowRunID:         request.WorkflowRunID,
		AllocationDate:        request.AllocationDate.UTC(),
		BookType:              domaincommon.BookTypeInvesting,
		AvailableCashStart:    request.AvailableCashStart,
		FreshMonthlyCash:      request.FreshMonthlyCash,
		SellProceedsAvailable: request.SellProceedsAvailable,
		CarryForwardCash:      request.CarryForwardCash,
		TargetDeployableCash:  deployableCash,
		AllocatedCashTotal:    allocatedTotal,
		CashLeftUnallocated:   unallocatedCash,
		AllocationNotes:       buildAllocationNotes(deployableCash, allocatedTotal, unallocatedCash, items, request.DryRun),
		Items:                 items,
		CreatedAt:             now.UTC(),
		SchemaVersion:         domaincommon.SchemaVersion1,
	}
}

func allocatedCandidatesFromPlan(plan allocationPlan, allocatedTotal float64) []AllocatedCandidateRef {
	allocated := make([]AllocatedCandidateRef, 0)
	for _, decision := range plan.Decisions {
		if decision.Amount <= scoreEpsilon {
			continue
		}
		allocated = append(allocated, AllocatedCandidateRef{
			CapitalCandidateRef:           decision.Candidate.Ref,
			RecommendedAllocationAmount:   decision.Amount,
			RecommendedAllocationPctOfRun: allocationPctOfRun(decision.Amount, allocatedTotal),
			RecommendedTrancheNumber:      recommendedTrancheNumber(decision.Candidate),
			AllocationReason:              allocationReason(decision),
		})
	}
	return allocated
}

func blockedCandidatesFromPlan(plan allocationPlan) []BlockedCapitalCandidate {
	if len(plan.Blocked) == 0 {
		return nil
	}
	seen := make(map[string]struct{}, len(plan.Blocked))
	blocked := make([]BlockedCapitalCandidate, 0, len(plan.Blocked))
	for _, candidate := range plan.Blocked {
		key := candidate.Candidate.ReviewID.Hex() + ":" + candidate.ConstraintReason
		if _, exists := seen[key]; exists {
			continue
		}
		seen[key] = struct{}{}
		blocked = append(blocked, candidate)
	}
	return blocked
}

func sumAllocatedItems(items []domainallocation.CapitalAllocationItem) float64 {
	total := 0.0
	for _, item := range items {
		total += item.RecommendedAllocationAmount
	}
	return roundMoney(total)
}

func allocationPctOfRun(amount float64, allocatedTotal float64) float64 {
	if amount <= scoreEpsilon || allocatedTotal <= scoreEpsilon {
		return 0
	}
	return roundToHundredth(amount / allocatedTotal * 100)
}

func recommendedTrancheNumber(candidate capitalCandidate) int {
	// Allocation runs are recommendation snapshots, not execution state. Without
	// allocation-history lookup, V1 treats new positions as tranche 1 and adds to
	// existing positions as tranche 2.
	if candidate.Position.Owned {
		return 2
	}
	return 1
}

func buildAllocationNotes(
	deployableCash float64,
	allocatedTotal float64,
	unallocatedCash float64,
	items []domainallocation.CapitalAllocationItem,
	dryRun bool,
) string {
	parts := []string{
		fmt.Sprintf("deployable %.2f", deployableCash),
		fmt.Sprintf("allocated %.2f", allocatedTotal),
		fmt.Sprintf("unallocated %.2f", unallocatedCash),
		fmt.Sprintf("items %d", len(items)),
	}
	if dryRun {
		parts = append(parts, "dry run")
	}
	return strings.Join(parts, "; ")
}

func roundMoney(value float64) float64 {
	return math.Round(value*100) / 100
}

func roundToTenth(value float64) float64 {
	return math.Round(value*10) / 10
}

func roundToHundredth(value float64) float64 {
	return math.Round(value*100) / 100
}

func clamp(value, minValue, maxValue float64) float64 {
	if value < minValue {
		return minValue
	}
	if value > maxValue {
		return maxValue
	}
	return value
}

func nearlyEqual(left, right float64) bool {
	return math.Abs(left-right) <= scoreEpsilon
}

func mathMax(left, right float64) float64 {
	if left > right {
		return left
	}
	return right
}
