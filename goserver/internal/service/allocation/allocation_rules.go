package allocation

import (
	"fmt"
	"math"

	domaincommon "goserver/internal/domain/common"
)

type allocationPlan struct {
	Decisions []allocationDecision
	Blocked   []BlockedCapitalCandidate
}

type allocationDecision struct {
	Candidate            capitalCandidate
	Amount               float64
	CapAmount            float64
	Weight               float64
	CapConversionLimited bool
	ConstraintReason     string
}

type allocationState struct {
	candidate            capitalCandidate
	weight               float64
	capAmount            float64
	capConversionLimited bool
	allocated            float64
	blockedReason        string
}

func allocateCapitalAcrossCandidates(
	candidates []capitalCandidate,
	deployableCash float64,
	config CapitalAllocationConfig,
) allocationPlan {
	plan := allocationPlan{Decisions: make([]allocationDecision, 0, len(candidates))}
	if len(candidates) == 0 {
		return plan
	}

	states := make([]allocationState, 0, len(candidates))
	for _, candidate := range candidates {
		state := allocationState{candidate: candidate}
		state.blockedReason = allocationBlockReason(candidate, deployableCash)
		if state.blockedReason == "" {
			state.weight = candidateAllocationWeight(candidate)
			state.capAmount, state.capConversionLimited = candidateAllocationCap(candidate, deployableCash)
			if state.capAmount <= scoreEpsilon {
				state.blockedReason = "target_gap_not_convertible_to_amount"
			}
			if config.MinimumAllocationAmount > 0 && state.capAmount < config.MinimumAllocationAmount {
				state.blockedReason = "below_minimum_allocation"
			}
		}
		states = append(states, state)
	}

	remaining := deployableCash
	for pass := 0; pass < config.AllocationPasses && remaining > scoreEpsilon; pass++ {
		totalWeight := 0.0
		for _, state := range states {
			if state.blockedReason == "" && state.weight > 0 && state.allocated < state.capAmount-scoreEpsilon {
				totalWeight += state.weight
			}
		}
		if totalWeight <= scoreEpsilon {
			break
		}

		passAllocated := 0.0
		for index := range states {
			state := &states[index]
			if state.blockedReason != "" || state.weight <= 0 || state.allocated >= state.capAmount-scoreEpsilon {
				continue
			}
			share := remaining * (state.weight / totalWeight)
			capRemaining := math.Max(state.capAmount-state.allocated, 0)
			amount := math.Min(share, capRemaining)
			if amount <= scoreEpsilon {
				continue
			}
			state.allocated += amount
			passAllocated += amount
		}
		if passAllocated <= scoreEpsilon {
			break
		}
		remaining = math.Max(remaining-passAllocated, 0)
	}

	for index := range states {
		state := &states[index]
		if state.allocated > 0 {
			state.allocated = roundMoney(state.allocated)
		}
		if config.MinimumAllocationAmount > 0 && state.allocated > 0 && state.allocated < config.MinimumAllocationAmount {
			state.blockedReason = "below_minimum_allocation"
			state.allocated = 0
		}
	}
	trimRoundedAllocationsToDeployable(states, deployableCash)

	for index := range states {
		state := &states[index]
		decision := allocationDecision{
			Candidate:            state.candidate,
			Amount:               state.allocated,
			CapAmount:            state.capAmount,
			Weight:               state.weight,
			CapConversionLimited: state.capConversionLimited,
			ConstraintReason:     state.blockedReason,
		}
		plan.Decisions = append(plan.Decisions, decision)
		if state.blockedReason != "" || state.allocated <= scoreEpsilon {
			reason := state.blockedReason
			if reason == "" {
				reason = "insufficient_cash_after_higher_priority_allocations"
			}
			plan.Blocked = append(plan.Blocked, BlockedCapitalCandidate{
				Candidate:        state.candidate.Ref,
				ConstraintReason: reason,
			})
		}
	}

	return plan
}

func trimRoundedAllocationsToDeployable(states []allocationState, deployableCash float64) {
	total := 0.0
	for _, state := range states {
		total += state.allocated
	}
	excess := roundMoney(total - deployableCash)
	for index := len(states) - 1; index >= 0 && excess > scoreEpsilon; index-- {
		if states[index].allocated <= 0 {
			continue
		}
		reduction := math.Min(states[index].allocated, excess)
		states[index].allocated = roundMoney(states[index].allocated - reduction)
		excess = roundMoney(excess - reduction)
	}
}

func allocationBlockReason(candidate capitalCandidate, deployableCash float64) string {
	if deployableCash <= scoreEpsilon {
		return "no_deployable_cash"
	}
	if reason := firstBlockingConstraint(candidate.Ref.ConstraintReasons); reason != "" {
		return reason
	}
	if candidate.Ref.ActionType != "" && candidate.Ref.ActionType != domaincommon.InvestingActionTypeBuy {
		return "not_buy_action"
	}
	if candidate.Ref.PriorityScore <= 0 {
		return "missing_priority_score"
	}
	if candidate.Position.GapToMaxPct <= scoreEpsilon {
		return "max_position_reached"
	}
	if candidate.Position.GapToTargetPct <= scoreEpsilon {
		return "already_at_or_above_target"
	}
	return ""
}

func candidateAllocationWeight(candidate capitalCandidate) float64 {
	priority := candidate.Ref.PriorityScore
	if priority <= 0 {
		priority = candidate.Score.WeightedTotal
	}
	gapFactor := 1.0
	if candidate.Position.TargetPct > 0 {
		gapFactor = 0.5 + math.Min(candidate.Position.GapToTargetPct/candidate.Position.TargetPct, 1)
	}
	return math.Max(priority, 0.1) * gapFactor
}

func candidateAllocationCap(candidate capitalCandidate, deployableCash float64) (float64, bool) {
	bookValue, ok := inferBookValue(candidate)
	if !ok {
		return deployableCash, true
	}
	targetGapAmount := bookValue * candidate.Position.GapToTargetPct / 100
	maxGapAmount := bookValue * candidate.Position.GapToMaxPct / 100
	return roundMoney(math.Max(math.Min(targetGapAmount, maxGapAmount), 0)), false
}

func allocationReason(decision allocationDecision) string {
	candidate := decision.Candidate
	reason := fmt.Sprintf(
		"rank %d; priority %.1f; score %.1f; valuation %.1f; target gap %.2f%%",
		candidate.Ref.PriorityRank,
		candidate.Ref.PriorityScore,
		candidate.Score.WeightedTotal,
		candidate.Score.Valuation,
		candidate.Position.GapToTargetPct,
	)
	if decision.CapConversionLimited {
		reason += "; position value unavailable, cap-to-rupee conversion limited"
	}
	return reason
}
