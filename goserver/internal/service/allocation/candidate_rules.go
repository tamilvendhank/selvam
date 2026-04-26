package allocation

import (
	"math"
	"strings"

	domaincommon "goserver/internal/domain/common"
	domainreview "goserver/internal/domain/review"
	domainthesis "goserver/internal/domain/thesis"
)

const scoreEpsilon = 0.0001

type CapitalAllocationConfig struct {
	DefaultMaxCandidates int
	MaxPageSize          int

	MinMeaningfulTargetPct float64
	MaxPositionCapPct      float64

	ExceptionalMin float64
	StrongMin      float64
	AcceptableMin  float64

	ExceptionalTargetPct float64
	StrongTargetPct      float64
	AcceptableTargetPct  float64

	BuyMinOverall              float64
	BuyMinManagementGovernance float64
	BuyMinCapitalEfficiency    float64
	BuyMinValuation            float64
	CoreStrongThreshold        float64
	CoreFloorThreshold         float64
	MinStrongCoreSections      int
	MaxCoreSectionsBelowFloor  int

	RequireActiveThesis     bool
	MinimumAllocationAmount float64
	AllocationPasses        int
}

func defaultCapitalAllocationConfig() CapitalAllocationConfig {
	return CapitalAllocationConfig{
		DefaultMaxCandidates: defaultCapitalCandidateMaxCandidates,
		MaxPageSize:          maxCapitalCandidatePageSize,

		MinMeaningfulTargetPct: 3,
		MaxPositionCapPct:      10,

		ExceptionalMin: 8.5,
		StrongMin:      7.5,
		AcceptableMin:  6.5,

		ExceptionalTargetPct: 8,
		StrongTargetPct:      5,
		AcceptableTargetPct:  3.5,

		BuyMinOverall:              7.5,
		BuyMinManagementGovernance: 7,
		BuyMinCapitalEfficiency:    7,
		BuyMinValuation:            6,
		CoreStrongThreshold:        7,
		CoreFloorThreshold:         6.5,
		MinStrongCoreSections:      3,
		MaxCoreSectionsBelowFloor:  1,

		RequireActiveThesis:     false,
		MinimumAllocationAmount: 0,
		AllocationPasses:        3,
	}
}

func mergeCapitalAllocationConfig(base CapitalAllocationConfig, override CapitalAllocationConfig) CapitalAllocationConfig {
	if override.DefaultMaxCandidates > 0 {
		base.DefaultMaxCandidates = override.DefaultMaxCandidates
	}
	if override.MaxPageSize > 0 {
		base.MaxPageSize = override.MaxPageSize
	}
	applyPositiveFloat(&base.MinMeaningfulTargetPct, override.MinMeaningfulTargetPct)
	applyPositiveFloat(&base.MaxPositionCapPct, override.MaxPositionCapPct)
	applyPositiveFloat(&base.ExceptionalMin, override.ExceptionalMin)
	applyPositiveFloat(&base.StrongMin, override.StrongMin)
	applyPositiveFloat(&base.AcceptableMin, override.AcceptableMin)
	applyPositiveFloat(&base.ExceptionalTargetPct, override.ExceptionalTargetPct)
	applyPositiveFloat(&base.StrongTargetPct, override.StrongTargetPct)
	applyPositiveFloat(&base.AcceptableTargetPct, override.AcceptableTargetPct)
	applyPositiveFloat(&base.BuyMinOverall, override.BuyMinOverall)
	applyPositiveFloat(&base.BuyMinManagementGovernance, override.BuyMinManagementGovernance)
	applyPositiveFloat(&base.BuyMinCapitalEfficiency, override.BuyMinCapitalEfficiency)
	applyPositiveFloat(&base.BuyMinValuation, override.BuyMinValuation)
	applyPositiveFloat(&base.CoreStrongThreshold, override.CoreStrongThreshold)
	applyPositiveFloat(&base.CoreFloorThreshold, override.CoreFloorThreshold)
	if override.MinStrongCoreSections > 0 {
		base.MinStrongCoreSections = override.MinStrongCoreSections
	}
	if override.MaxCoreSectionsBelowFloor > 0 {
		base.MaxCoreSectionsBelowFloor = override.MaxCoreSectionsBelowFloor
	}
	if override.RequireActiveThesis {
		base.RequireActiveThesis = true
	}
	if override.MinimumAllocationAmount > 0 {
		base.MinimumAllocationAmount = override.MinimumAllocationAmount
	}
	if override.AllocationPasses > 0 {
		base.AllocationPasses = override.AllocationPasses
	}
	return normalizeCapitalAllocationConfig(base)
}

func normalizeCapitalAllocationConfig(config CapitalAllocationConfig) CapitalAllocationConfig {
	defaults := defaultCapitalAllocationConfig()
	if config.DefaultMaxCandidates <= 0 {
		config.DefaultMaxCandidates = defaults.DefaultMaxCandidates
	}
	if config.MaxPageSize <= 0 {
		config.MaxPageSize = defaults.MaxPageSize
	}
	if config.MaxPageSize > maxCapitalCandidatePageSize {
		config.MaxPageSize = maxCapitalCandidatePageSize
	}
	if config.MinMeaningfulTargetPct <= 0 {
		config.MinMeaningfulTargetPct = defaults.MinMeaningfulTargetPct
	}
	if config.MaxPositionCapPct <= 0 {
		config.MaxPositionCapPct = defaults.MaxPositionCapPct
	}
	if config.ExceptionalMin <= 0 {
		config.ExceptionalMin = defaults.ExceptionalMin
	}
	if config.StrongMin <= 0 {
		config.StrongMin = defaults.StrongMin
	}
	if config.AcceptableMin <= 0 {
		config.AcceptableMin = defaults.AcceptableMin
	}
	if config.ExceptionalTargetPct <= 0 {
		config.ExceptionalTargetPct = defaults.ExceptionalTargetPct
	}
	if config.StrongTargetPct <= 0 {
		config.StrongTargetPct = defaults.StrongTargetPct
	}
	if config.AcceptableTargetPct <= 0 {
		config.AcceptableTargetPct = defaults.AcceptableTargetPct
	}
	if config.BuyMinOverall <= 0 {
		config.BuyMinOverall = defaults.BuyMinOverall
	}
	if config.BuyMinManagementGovernance <= 0 {
		config.BuyMinManagementGovernance = defaults.BuyMinManagementGovernance
	}
	if config.BuyMinCapitalEfficiency <= 0 {
		config.BuyMinCapitalEfficiency = defaults.BuyMinCapitalEfficiency
	}
	if config.BuyMinValuation <= 0 {
		config.BuyMinValuation = defaults.BuyMinValuation
	}
	if config.CoreStrongThreshold <= 0 {
		config.CoreStrongThreshold = defaults.CoreStrongThreshold
	}
	if config.CoreFloorThreshold <= 0 {
		config.CoreFloorThreshold = defaults.CoreFloorThreshold
	}
	if config.MinStrongCoreSections <= 0 {
		config.MinStrongCoreSections = defaults.MinStrongCoreSections
	}
	if config.MaxCoreSectionsBelowFloor < 0 {
		config.MaxCoreSectionsBelowFloor = defaults.MaxCoreSectionsBelowFloor
	}
	if config.AllocationPasses <= 0 {
		config.AllocationPasses = defaults.AllocationPasses
	}
	if config.MaxPositionCapPct < config.MinMeaningfulTargetPct {
		config.MaxPositionCapPct = config.MinMeaningfulTargetPct
	}
	config.ExceptionalTargetPct = clamp(config.ExceptionalTargetPct, config.MinMeaningfulTargetPct, config.MaxPositionCapPct)
	config.StrongTargetPct = clamp(config.StrongTargetPct, config.MinMeaningfulTargetPct, config.MaxPositionCapPct)
	config.AcceptableTargetPct = clamp(config.AcceptableTargetPct, config.MinMeaningfulTargetPct, config.MaxPositionCapPct)
	return config
}

func evaluateCandidateEligibility(
	review *domainreview.CompanyReview,
	score candidateScoreContext,
	position candidatePositionContext,
	thesis *domainthesis.InvestmentThesis,
	thesisLoaded bool,
	config CapitalAllocationConfig,
) candidateEligibility {
	eligibility := candidateEligibility{Eligible: true}
	if review == nil {
		return candidateEligibility{Code: "nil_review", Reasons: []string{"review is required"}}
	}
	if review.ID.IsZero() {
		return candidateEligibility{Code: "missing_review_id", Reasons: []string{"review id is required"}}
	}
	if review.CompanyID.IsZero() {
		return candidateEligibility{Code: "missing_company_id", Reasons: []string{"company id is required"}}
	}
	if review.BookType != domaincommon.BookTypeInvesting {
		return candidateEligibility{Code: "not_investing_book", Reasons: []string{"review is not for the investing book"}}
	}
	if !isFinalCapitalReview(review) {
		return candidateEligibility{Code: "review_not_finalized", Reasons: []string{"review is not finalized"}}
	}
	if review.HardGateFailed {
		eligibility.block("hard_gate_failed", "hard gate failure blocks fresh capital")
	}

	decision := review.DecisionAction
	if decision == nil {
		eligibility.block("missing_decision_action", "decision action is required")
	} else {
		if !decision.CapitalEligible {
			eligibility.block("capital_not_eligible", "decision action is not capital eligible")
		}
		if reason := firstBlockingConstraint(decision.ActionConstraints); reason != "" {
			eligibility.block(reason, "action constraint blocks capital deployment")
		}
	}

	if candidateAction(review) != domaincommon.InvestingActionTypeBuy {
		eligibility.block("not_buy_action", "fresh capital requires a BUY action")
	}
	if bucket := candidateBucket(review); bucket != "" && bucket != domaincommon.WatchlistBucketBuyReady && (decision == nil || !decision.CapitalEligible) {
		eligibility.block("not_buy_ready", "candidate is not in buy_ready bucket")
	}
	if !score.ValidWeightedTotal {
		eligibility.block("missing_weighted_score", "weighted total score is required")
	} else if score.WeightedTotal < config.BuyMinOverall {
		eligibility.block("score_below_buy_threshold", "weighted total score is below buy threshold")
	}
	if !score.HasManagementGovernance || score.ManagementGovernance < config.BuyMinManagementGovernance {
		eligibility.block("management_governance_below_minimum", "management/governance score is below buy minimum")
	}
	if !score.HasCapitalEfficiency || score.CapitalEfficiency < config.BuyMinCapitalEfficiency {
		eligibility.block("capital_efficiency_below_minimum", "capital efficiency score is below buy minimum")
	}
	if !score.HasValuation || score.Valuation < config.BuyMinValuation {
		eligibility.block("valuation_below_buy_minimum", "valuation/entry score is below buy minimum")
	}
	if len(score.MissingCoreSections) > 0 {
		eligibility.block("missing_core_sections", "required core section scores are missing: "+strings.Join(score.MissingCoreSections, ", "))
	}
	if score.CoreAtOrAboveStrong < config.MinStrongCoreSections {
		eligibility.block("core_sections_not_strong_enough", "not enough core sections meet the strong threshold")
	}
	if score.CoreBelowFloor > config.MaxCoreSectionsBelowFloor {
		eligibility.block("core_sections_weak", "too many core sections are below the floor")
	}
	if hasBlockingSectionActionCap(review) {
		eligibility.block("section_action_cap_blocks_buy", "section action cap blocks BUY allocation")
	}
	if thesis != nil && thesis.ThesisStatus == domaincommon.ThesisStatusBroken {
		eligibility.block("thesis_broken", "latest thesis is broken")
	}
	if requiresThesis(review, config) && !hasActiveThesis(thesis, thesisLoaded) {
		eligibility.block("thesis_required", "active thesis is required before capital allocation")
	}
	if position.GapToMaxPct <= scoreEpsilon {
		eligibility.block("max_position_reached", "current position is at or above max cap")
	}

	return eligibility
}

func computeCandidatePriorityScore(
	review *domainreview.CompanyReview,
	score candidateScoreContext,
	position candidatePositionContext,
	config CapitalAllocationConfig,
) float64 {
	if review == nil || candidateAction(review) != domaincommon.InvestingActionTypeBuy {
		return 0
	}

	// V1 is intentionally simple and replaceable: start with mapped action priority
	// when present, otherwise the total score, then make small deterministic
	// adjustments for entry attractiveness, core quality, position gap, and risk flags.
	priority := score.WeightedTotal
	if review.DecisionAction != nil && review.DecisionAction.CapitalPriorityScore > 0 {
		priority = review.DecisionAction.CapitalPriorityScore
	}
	if score.HasValuation && score.Valuation >= 7 {
		priority += 0.3
	}
	if score.HasValuation && score.Valuation < config.BuyMinValuation {
		priority -= 0.5
	}
	priority += math.Min(float64(score.CoreAtOrAboveStrong)*0.1, 0.4)
	if position.TargetPct > 0 && position.GapToTargetPct > 0 {
		priority += math.Min(position.GapToTargetPct/position.TargetPct, 1) * 0.4
	}
	if score.HasMarketConfirmation && score.MarketConfirmation < config.CoreFloorThreshold {
		priority -= 0.2
	}
	if review.ChangeLog != nil {
		if review.ChangeLog.RequiresExitReview {
			priority -= 0.6
		}
		if review.ChangeLog.WeightedTotalScoreChange < -1 {
			priority -= 0.4
		}
	}
	if review.DecisionAction != nil && firstBlockingConstraint(review.DecisionAction.ActionConstraints) != "" {
		priority -= 1
	}

	return roundToTenth(clamp(priority, 0, 10))
}

func computeTargetPositionPct(review *domainreview.CompanyReview, score candidateScoreContext, config CapitalAllocationConfig) float64 {
	target := 0.0
	if review != nil && review.DecisionAction != nil && sanePercentage(review.DecisionAction.RecommendedPositionTargetPct) {
		target = review.DecisionAction.RecommendedPositionTargetPct
	}
	if target <= 0 && review != nil && review.PositionSnapshot != nil && sanePercentage(review.PositionSnapshot.TargetPositionPct) {
		target = review.PositionSnapshot.TargetPositionPct
	}
	if target <= 0 {
		switch {
		case score.WeightedTotal >= config.ExceptionalMin:
			target = config.ExceptionalTargetPct
		case score.WeightedTotal >= config.StrongMin:
			target = config.StrongTargetPct
		case score.WeightedTotal >= config.AcceptableMin:
			target = config.AcceptableTargetPct
		default:
			target = config.MinMeaningfulTargetPct
		}
	}
	return clamp(target, config.MinMeaningfulTargetPct, config.MaxPositionCapPct)
}

func computeMaxPositionPct(review *domainreview.CompanyReview, targetPct float64, config CapitalAllocationConfig) float64 {
	maxPct := config.MaxPositionCapPct
	if review != nil && review.DecisionAction != nil && sanePercentage(review.DecisionAction.RecommendedPositionCapPct) {
		maxPct = math.Min(review.DecisionAction.RecommendedPositionCapPct, config.MaxPositionCapPct)
	} else if review != nil && review.PositionSnapshot != nil && sanePercentage(review.PositionSnapshot.MaxPositionPct) {
		maxPct = math.Min(review.PositionSnapshot.MaxPositionPct, config.MaxPositionCapPct)
	}
	if maxPct < targetPct {
		maxPct = targetPct
	}
	return clamp(maxPct, config.MinMeaningfulTargetPct, config.MaxPositionCapPct)
}

func computePositionGap(currentPct, targetPct, maxPct float64) candidatePositionContext {
	return candidatePositionContext{
		CurrentPct:     clamp(currentPct, 0, 100),
		TargetPct:      clamp(targetPct, 0, 100),
		MaxPct:         clamp(maxPct, 0, 100),
		GapToTargetPct: math.Max(targetPct-currentPct, 0),
		GapToMaxPct:    math.Max(maxPct-currentPct, 0),
		Owned:          currentPct > scoreEpsilon,
	}
}

func firstBlockingConstraint(constraints []string) string {
	for _, constraint := range constraints {
		if capitalConstraintBlocksAllocation(constraint) {
			return normalizeConstraintCode(constraint)
		}
	}
	return ""
}

func capitalConstraintBlocksAllocation(constraint string) bool {
	normalized := normalizeConstraintCode(constraint)
	if normalized == "" {
		return false
	}
	blockingExact := map[string]struct{}{
		"hard_gate_failed":              {},
		"missing_core_sections":         {},
		"position_at_max_cap":           {},
		"max_position_reached":          {},
		"position_not_under_target":     {},
		"requires_written_thesis":       {},
		"thesis_required":               {},
		"thesis_broken":                 {},
		"valuation_blocked":             {},
		"insufficient_confidence":       {},
		"section_action_cap_blocks_buy": {},
	}
	if _, ok := blockingExact[normalized]; ok {
		return true
	}
	return strings.Contains(normalized, "blocked") ||
		strings.Contains(normalized, "cannot_buy") ||
		strings.Contains(normalized, "max_position") ||
		strings.Contains(normalized, "hard_gate")
}

func requiresThesis(review *domainreview.CompanyReview, config CapitalAllocationConfig) bool {
	if config.RequireActiveThesis {
		return true
	}
	if review == nil || review.DecisionAction == nil {
		return false
	}
	for _, constraint := range review.DecisionAction.ActionConstraints {
		normalized := normalizeConstraintCode(constraint)
		if normalized == "requires_written_thesis" || normalized == "thesis_required" {
			return true
		}
	}
	return false
}

func hasActiveThesis(thesis *domainthesis.InvestmentThesis, loaded bool) bool {
	return loaded && thesis != nil && thesis.ThesisStatus == domaincommon.ThesisStatusActive
}

func hasBlockingSectionActionCap(review *domainreview.CompanyReview) bool {
	for _, section := range review.Sections {
		switch section.SectionActionCap {
		case domaincommon.SectionActionCapCannotBuy,
			domaincommon.SectionActionCapWatchOnly,
			domaincommon.SectionActionCapExitReviewOnly:
			return true
		}
	}
	return false
}

func isFinalCapitalReview(review *domainreview.CompanyReview) bool {
	return review != nil &&
		review.ReviewStatus == domaincommon.ReviewStatusFinal &&
		review.ReviewLifecycleState == domaincommon.ReviewLifecycleStateFinalized
}

func isCandidateIneligibleCode(code string) bool {
	switch code {
	case "not_investing_book", "review_not_finalized", "nil_review":
		return false
	default:
		return true
	}
}

func (eligibility *candidateEligibility) block(code, reason string) {
	eligibility.Eligible = false
	if eligibility.Code == "" {
		eligibility.Code = code
	}
	if reason != "" {
		eligibility.Reasons = append(eligibility.Reasons, reason)
	}
}
