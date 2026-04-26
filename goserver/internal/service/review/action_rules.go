package review

import (
	"fmt"
	"strings"

	domaincommon "goserver/internal/domain/common"
	domainreview "goserver/internal/domain/review"
	domainthesis "goserver/internal/domain/thesis"
)

type actionDecision struct {
	Action            *domainreview.DecisionAction
	ResultConstraints []ActionConstraint
	Reasons           []string
	Message           string
}

type buyEligibility struct {
	Eligible      bool
	CapitalOK     bool
	Reasons       []string
	Constraints   []ActionConstraint
	BlockingCodes []string
}

func (service *actionMappingService) determineActionDecision(
	review *domainreview.CompanyReview,
	thesis *domainthesis.InvestmentThesis,
	score scoreContext,
	position positionContext,
	options mapActionOptions,
) actionDecision {
	builder := newActionReasonBuilder()
	var action domaincommon.InvestingActionType

	if hardGateAction, ok := service.evaluateHardGateAction(review, position, builder); ok {
		action = hardGateAction
	} else if thesisAction, ok := service.evaluateThesisAction(thesis, review, position, builder); ok {
		action = thesisAction
	} else if position.Owned {
		action = service.evaluateOwnedAction(review, thesis, score, position, options, builder)
	} else {
		action = service.evaluateUnownedAction(review, thesis, score, position, builder)
	}

	action = service.applyActionCaps(review, action, position, builder)
	capitalEligible := action == domaincommon.InvestingActionTypeBuy &&
		!position.AtOrAboveMax &&
		!builder.hasBlockingConstraint()
	if action == domaincommon.InvestingActionTypeBuy && position.Owned && !position.UnderTarget {
		capitalEligible = false
		builder.addConstraint("position_not_under_target", "Position is not meaningfully below target size.", true)
	}

	bucket := determineBucketFromAction(action, position, bucketContext{
		CapitalEligible:   capitalEligible,
		WeakeningDetected: weakeningDetected(review, service.config),
	})
	decision := service.buildDecisionAction(review, action, bucket, capitalEligible, builder)
	return actionDecision{
		Action:            decision,
		ResultConstraints: builder.constraints,
		Reasons:           builder.reasons,
		Message:           fmt.Sprintf("mapped review %s to %s: %s", review.ID.Hex(), action, strings.Join(builder.reasons, ", ")),
	}
}

func (service *actionMappingService) evaluateHardGateAction(
	review *domainreview.CompanyReview,
	position positionContext,
	builder *actionReasonBuilder,
) (domaincommon.InvestingActionType, bool) {
	if review == nil || !review.HardGateFailed {
		return "", false
	}
	builder.addReason("hard_gate_failed")
	builder.addConstraint("hard_gate_failed", strings.Join(nonBlankStrings(review.HardGateFailureReasons), "; "), true)
	if position.Owned {
		return domaincommon.InvestingActionTypeSell, true
	}
	return domaincommon.InvestingActionTypeReject, true
}

func (service *actionMappingService) evaluateThesisAction(
	thesis *domainthesis.InvestmentThesis,
	review *domainreview.CompanyReview,
	position positionContext,
	builder *actionReasonBuilder,
) (domaincommon.InvestingActionType, bool) {
	if isThesisBroken(thesis, review) {
		builder.addReason("thesis_broken")
		builder.addConstraint("thesis_broken", "Written thesis or change log indicates thesis break.", true)
		if position.Owned {
			return domaincommon.InvestingActionTypeSell, true
		}
		return domaincommon.InvestingActionTypeReject, true
	}
	return "", false
}

func (service *actionMappingService) evaluateUnownedAction(
	review *domainreview.CompanyReview,
	thesis *domainthesis.InvestmentThesis,
	score scoreContext,
	position positionContext,
	builder *actionReasonBuilder,
) domaincommon.InvestingActionType {
	if score.WeightedTotal < service.config.RejectBelowOverall || investabilityTooWeak(review, service.config) {
		builder.addReason("unowned_context_reject")
		builder.addReason("score_below_research_threshold")
		return domaincommon.InvestingActionTypeReject
	}

	buy := service.evaluateBuyEligibility(review, thesis, score, position)
	builder.addReasons(buy.Reasons...)
	builder.addConstraints(buy.Constraints...)
	if buy.Eligible {
		builder.addReason("score_above_buy_threshold")
		return domaincommon.InvestingActionTypeBuy
	}

	if score.WeightedTotal >= service.config.AcceptableMin {
		builder.addReason("unowned_context_watch")
		return domaincommon.InvestingActionTypeWatch
	}
	if score.WeightedTotal >= service.config.RejectBelowOverall {
		builder.addReason("research_quality_incomplete")
		return domaincommon.InvestingActionTypeWatch
	}

	builder.addReason("score_below_research_threshold")
	return domaincommon.InvestingActionTypeReject
}

func (service *actionMappingService) evaluateOwnedAction(
	review *domainreview.CompanyReview,
	thesis *domainthesis.InvestmentThesis,
	score scoreContext,
	position positionContext,
	options mapActionOptions,
	builder *actionReasonBuilder,
) domaincommon.InvestingActionType {
	if service.evaluateSellEligibility(review, score, builder) {
		return domaincommon.InvestingActionTypeSell
	}
	if service.evaluateTrimEligibility(review, thesis, score, position, options, builder) {
		return domaincommon.InvestingActionTypeTrim
	}

	buy := service.evaluateBuyEligibility(review, thesis, score, position)
	if buy.Eligible && position.UnderTarget && !position.AtOrAboveMax {
		builder.addReasons(buy.Reasons...)
		builder.addReason("eligible_for_new_capital")
		return domaincommon.InvestingActionTypeBuy
	}

	if score.WeightedTotal >= service.config.HoldMinOverall && score.CoreBelowWeak == 0 {
		builder.addReason("owned_context_hold")
		return domaincommon.InvestingActionTypeHold
	}
	if score.WeightedTotal >= service.config.WeakMin {
		builder.addReason("owned_context_trim_review")
		return domaincommon.InvestingActionTypeTrim
	}

	builder.addReason("owned_context_sell")
	return domaincommon.InvestingActionTypeSell
}

func (service *actionMappingService) evaluateBuyEligibility(
	review *domainreview.CompanyReview,
	thesis *domainthesis.InvestmentThesis,
	score scoreContext,
	position positionContext,
) buyEligibility {
	result := buyEligibility{Eligible: true, CapitalOK: true}
	if review.HardGateFailed {
		result.Eligible = false
		result.BlockingCodes = append(result.BlockingCodes, "hard_gate_failed")
		result.Constraints = append(result.Constraints, actionConstraint("hard_gate_failed", "Hard gate failure blocks buying.", true))
	}
	if score.WeightedTotal < service.config.BuyMinOverall {
		result.Eligible = false
		result.Reasons = append(result.Reasons, "score_below_buy_threshold")
	}
	if !score.HasManagement || score.ManagementGovernance < service.config.BuyMinManagementGovernance {
		result.Eligible = false
		result.Reasons = append(result.Reasons, "management_governance_below_minimum")
	}
	if !score.HasCapitalEfficiency || score.CapitalEfficiency < service.config.BuyMinCapitalEfficiency {
		result.Eligible = false
		result.Reasons = append(result.Reasons, "capital_efficiency_below_minimum")
	}
	if !score.HasValuation || score.Valuation < service.config.BuyMinValuation {
		result.Eligible = false
		result.Reasons = append(result.Reasons, "valuation_below_buy_minimum")
		if score.HasValuation {
			result.Reasons = append(result.Reasons, "valuation_stretched")
		}
	}
	if score.CoreAtOrAboveStrong < service.config.MinStrongCoreSectionsForBuy {
		result.Eligible = false
		result.Reasons = append(result.Reasons, "core_sections_not_strong_enough")
	}
	if score.CoreBelowFloor > service.config.MaxWeakCoreSectionsForBuy {
		result.Eligible = false
		result.Reasons = append(result.Reasons, "core_sections_weak")
	}
	if len(score.MissingCoreSections) > 0 {
		result.Eligible = false
		result.Reasons = append(result.Reasons, "evidence_incomplete")
		result.Constraints = append(result.Constraints, actionConstraint("missing_core_sections", "Missing core section scores: "+strings.Join(score.MissingCoreSections, ", "), true))
	}
	if position.AtOrAboveMax {
		result.Eligible = false
		result.CapitalOK = false
		result.Reasons = append(result.Reasons, "position_at_max_cap")
		result.Constraints = append(result.Constraints, actionConstraint("position_at_max_cap", "Position is at or above max cap.", true))
	}
	if service.config.RequireWrittenThesisForBuy && !hasActiveThesis(thesis) {
		result.CapitalOK = false
		result.Reasons = append(result.Reasons, "requires_written_thesis")
		result.Constraints = append(result.Constraints, actionConstraint("requires_written_thesis", "BUY requires a persisted active written thesis before capital deployment.", true))
	}
	if result.Eligible {
		result.Reasons = append(result.Reasons, "buy_thresholds_satisfied")
	}
	return result
}

func (service *actionMappingService) evaluateTrimEligibility(
	review *domainreview.CompanyReview,
	thesis *domainthesis.InvestmentThesis,
	score scoreContext,
	position positionContext,
	options mapActionOptions,
	builder *actionReasonBuilder,
) bool {
	trim := false
	if hasActionCap(review, domaincommon.SectionActionCapExitReviewOnly) {
		builder.addReason("action_cap_exit_review_only")
		trim = true
	}
	if isThesisUnderReview(thesis, review) {
		builder.addReason("thesis_under_review")
		trim = true
	}
	if requiresExitReview(review) || options.Mode == ActionMappingModeExitReview {
		builder.addReason("exit_review_required")
		trim = true
	}
	if totalScoreDropped(review, service.config.ExitReviewTotalDrop) {
		builder.addReason("score_drop_detected")
		trim = true
	}
	if anyCoreSectionDropped(review, service.config.ExitReviewCoreDrop) {
		builder.addReason("core_section_drop_detected")
		trim = true
	}
	if managementGovernanceDropped(review, service.config.ExitReviewManagementDrop) {
		builder.addReason("management_governance_drop_detected")
		trim = true
	}
	if valuationExtremeWithBusinessSoftening(review) {
		builder.addReason("valuation_extreme_with_business_softening")
		trim = true
	}
	if position.AtOrAboveMax || position.OverTarget {
		builder.addReason("position_concentration_risk")
		trim = true
	}
	if score.WeightedTotal >= 6.0 && score.WeightedTotal < service.config.HoldMinOverall && totalScoreDropped(review, 0.6) {
		builder.addReason("weighted_score_fell_to_trim_zone")
		trim = true
	}
	return trim
}

func (service *actionMappingService) evaluateSellEligibility(
	review *domainreview.CompanyReview,
	score scoreContext,
	builder *actionReasonBuilder,
) bool {
	if review.WeightedTotalScore > 0 && review.WeightedTotalScore < service.config.SellBelowOverall {
		builder.addReason("weighted_score_below_sell_threshold")
		return true
	}
	if score.CoreBelowWeak >= 2 {
		builder.addReason("core_sections_weak")
		return true
	}
	if score.HasManagement &&
		score.ManagementGovernance < service.config.CoreWeakThreshold &&
		hasNegativeEvidenceForSection(review, domaincommon.SectionNameManagementGovernance) {
		builder.addReason("governance_trust_materially_fails")
		return true
	}
	if score.HasCapitalEfficiency &&
		score.CapitalEfficiency < service.config.CoreWeakThreshold &&
		hasWorseningRiskForSection(review, domaincommon.SectionNameCapitalEfficiencyFinancialStrength) {
		builder.addReason("capital_efficiency_deterioration")
		return true
	}
	if changeLogIndicatesThesisBreak(review) {
		builder.addReason("thesis_break_change_log")
		return true
	}
	if structuralBusinessDeteriorationVisible(review) {
		builder.addReason("structural_business_deterioration")
		return true
	}
	return false
}

func (service *actionMappingService) applyActionCaps(
	review *domainreview.CompanyReview,
	action domaincommon.InvestingActionType,
	position positionContext,
	builder *actionReasonBuilder,
) domaincommon.InvestingActionType {
	for _, section := range review.Sections {
		switch section.SectionActionCap {
		case domaincommon.SectionActionCapCannotBuy:
			if action == domaincommon.InvestingActionTypeBuy {
				builder.addReason("action_cap_cannot_buy")
				builder.addConstraint("action_cap_cannot_buy", humanizeSectionName(section.SectionName)+" imposed cannot_buy.", true)
				if position.Owned {
					action = domaincommon.InvestingActionTypeHold
				} else {
					action = domaincommon.InvestingActionTypeWatch
				}
			}
		case domaincommon.SectionActionCapWatchOnly:
			if action == domaincommon.InvestingActionTypeBuy {
				builder.addReason("action_cap_watch_only")
				builder.addConstraint("action_cap_watch_only", humanizeSectionName(section.SectionName)+" imposed watch_only.", true)
				if position.Owned {
					action = domaincommon.InvestingActionTypeHold
				} else {
					action = domaincommon.InvestingActionTypeWatch
				}
			}
		case domaincommon.SectionActionCapExitReviewOnly:
			builder.addReason("action_cap_exit_review_only")
			builder.addConstraint("action_cap_exit_review_only", humanizeSectionName(section.SectionName)+" imposed exit_review_only.", true)
			if position.Owned {
				action = domaincommon.InvestingActionTypeTrim
			} else if action == domaincommon.InvestingActionTypeBuy {
				action = domaincommon.InvestingActionTypeWatch
			}
		}
	}
	return action
}

func (service *actionMappingService) buildDecisionAction(
	review *domainreview.CompanyReview,
	action domaincommon.InvestingActionType,
	bucket domaincommon.WatchlistBucket,
	capitalEligible bool,
	builder *actionReasonBuilder,
) *domainreview.DecisionAction {
	reasons := builder.reasons
	if len(reasons) == 0 {
		reasons = []string{"deterministic_action_mapping"}
	}
	targetPct, maxPct := recommendedPositionBounds(review, service.config)
	if action == domaincommon.InvestingActionTypeSell || action == domaincommon.InvestingActionTypeReject {
		targetPct = 0
	}
	return &domainreview.DecisionAction{
		ActionType:                   action,
		BucketAfterAction:            bucket,
		ActionPriorityRank:           priorityRankForAction(action),
		ActionReasonPrimary:          reasons[0],
		ActionReasonSecondary:        strings.Join(limitStrings(reasons[1:], 5), ", "),
		ActionConstraints:            constraintCodes(builder.constraints),
		CapitalEligible:              capitalEligible,
		CapitalPriorityScore:         computeCapitalPriorityScore(review, action, capitalEligible, builder, service.config),
		RecommendedPositionTargetPct: targetPct,
		RecommendedPositionCapPct:    maxPct,
		RecommendedTrancheStyle:      trancheStyleForAction(action),
		Notes:                        "deterministic action mapping",
	}
}
