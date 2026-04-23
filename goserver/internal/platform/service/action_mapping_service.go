package service

import (
	"context"
	"fmt"

	platformconfig "goserver/internal/platform/config"
	"goserver/internal/platform/domain"
)

type DefaultActionMappingService struct {
	config platformconfig.AppConfig
}

func NewActionMappingService(config platformconfig.AppConfig) *DefaultActionMappingService {
	return &DefaultActionMappingService{config: config}
}

func (service *DefaultActionMappingService) MapReview(_ context.Context, review *domain.CompanyReview, thesis *domain.InvestmentThesis, previousReview *domain.CompanyReview) (*domain.DecisionAction, error) {
	if review == nil {
		return nil, fmt.Errorf("review is required")
	}

	thresholds := service.config.Investing.ActionThresholds
	owned := review.OwnedBeforeReview
	reasonPrimary := ""
	reasonSecondary := ""
	action := domain.ActionWatch
	bucket := domain.WatchlistBucketWatch
	constraints := make([]string, 0)

	if review.HardGateFailed {
		if owned {
			action = domain.ActionSell
			bucket = domain.WatchlistBucketExitReview
			reasonPrimary = "Hard gate failed while position is owned."
		} else {
			action = domain.ActionReject
			bucket = domain.WatchlistBucketResearch
			reasonPrimary = "Hard gate failed."
		}
		return service.buildDecision(action, bucket, reasonPrimary, reasonSecondary, review, constraints), nil
	}

	if thesis != nil && thesis.ThesisStatus == domain.ThesisStatusBroken {
		return service.buildDecision(domain.ActionSell, domain.WatchlistBucketExitReview, "Thesis is marked broken.", "", review, constraints), nil
	}

	coreScores := domain.CoreSectionScores(review)
	strongCoreCount := 0
	weakCoreCount := 0
	veryWeakCoreCount := 0
	for _, score := range coreScores {
		if score >= thresholds.CoreStrongThreshold {
			strongCoreCount++
		}
		if score < thresholds.CoreWeakThreshold {
			weakCoreCount++
		}
		if score < thresholds.SellBelowOverall {
			veryWeakCoreCount++
		}
	}

	managementScore := sectionScore(review, domain.SectionManagementGovernance)
	capitalScore := sectionScore(review, domain.SectionCapitalEfficiencyFinancialStrength)
	valuationScore := sectionScore(review, domain.SectionValuationEntryAttractiveness)

	switch {
	case owned && (review.WeightedTotalScore < thresholds.SellBelowOverall || veryWeakCoreCount >= 2):
		action = domain.ActionSell
		bucket = domain.WatchlistBucketExitReview
		reasonPrimary = "Owned position now falls below sell thresholds."
	case owned && shouldTrim(review, previousReview):
		action = domain.ActionTrim
		bucket = domain.WatchlistBucketHold
		reasonPrimary = "Owned position shows meaningful weakening or stretched valuation."
	case !owned &&
		review.WeightedTotalScore >= thresholds.BuyMinOverall &&
		managementScore >= thresholds.BuyMinManagement &&
		capitalScore >= thresholds.BuyMinCapitalEfficiency &&
		valuationScore >= thresholds.BuyMinValuation &&
		strongCoreCount >= thresholds.MinStrongCoreSectionsForBuy &&
		weakCoreCount <= thresholds.MaxWeakCoreSectionsForBuy &&
		hasWritableThesis(thesis, service.config):
		action = domain.ActionBuy
		bucket = domain.WatchlistBucketBuyReady
		reasonPrimary = "Review passes default BUY thresholds."
	case owned &&
		review.WeightedTotalScore >= thresholds.HoldMinOverall &&
		managementScore >= thresholds.CoreWeakThreshold:
		action = domain.ActionHold
		bucket = domain.WatchlistBucketHold
		reasonPrimary = "Owned position remains above HOLD thresholds."
	case !owned && review.WeightedTotalScore < thresholds.RejectBelowOverall:
		action = domain.ActionReject
		bucket = domain.WatchlistBucketResearch
		reasonPrimary = "Review remains below minimum research quality threshold."
	default:
		action = domain.ActionWatch
		bucket = domain.WatchlistBucketWatch
		reasonPrimary = "Review needs more evidence or better entry conditions."
	}

	if valuationScore < thresholds.BuyMinValuation && action == domain.ActionBuy {
		action = domain.ActionWatch
		bucket = domain.WatchlistBucketWatch
		reasonSecondary = "Valuation is not supportive enough for a BUY."
	}

	action, bucket, constraints = applyActionCaps(review, action, bucket, constraints)
	return service.buildDecision(action, bucket, reasonPrimary, reasonSecondary, review, constraints), nil
}

func (service *DefaultActionMappingService) buildDecision(
	action domain.ActionType,
	bucket domain.WatchlistBucket,
	reasonPrimary string,
	reasonSecondary string,
	review *domain.CompanyReview,
	constraints []string,
) *domain.DecisionAction {
	decision := &domain.DecisionAction{
		ActionType:                   action,
		BucketAfterAction:            bucket,
		ActionReasonPrimary:          reasonPrimary,
		ActionReasonSecondary:        reasonSecondary,
		ActionConstraints:            constraints,
		CapitalEligible:              action == domain.ActionBuy,
		CapitalPriorityScore:         capitalPriorityScore(action, review),
		RecommendedPositionTargetPct: service.config.Investing.PositionSizing.MinMeaningfulTargetPct,
		RecommendedPositionCapPct:    service.config.Investing.PositionSizing.MaxPositionCapPct,
		RecommendedTrancheStyle:      trancheStyleForAction(action),
		ActionPriorityRank:           priorityRankForAction(action),
	}

	if action == domain.ActionTrim {
		decision.CapitalEligible = false
	}
	if action == domain.ActionSell || action == domain.ActionReject {
		decision.RecommendedPositionTargetPct = 0
	}

	return decision
}

func sectionScore(review *domain.CompanyReview, sectionName domain.InvestingSectionName) float64 {
	section := domain.FindSection(review, sectionName)
	if section == nil {
		return 0
	}

	return section.SectionScoreRaw
}

func hasWritableThesis(thesis *domain.InvestmentThesis, config platformconfig.AppConfig) bool {
	if !config.Investing.ThesisRules.RequireWrittenThesisForBuy {
		return true
	}
	if thesis == nil {
		return false
	}

	return thesis.ThesisSummary != "" && thesis.ThesisStatus != domain.ThesisStatusBroken
}

func shouldTrim(review *domain.CompanyReview, previousReview *domain.CompanyReview) bool {
	if review == nil || previousReview == nil {
		return false
	}

	if review.WeightedTotalScore < previousReview.WeightedTotalScore-0.75 {
		return true
	}
	valuationNow := sectionScore(review, domain.SectionValuationEntryAttractiveness)
	valuationPrev := sectionScore(previousReview, domain.SectionValuationEntryAttractiveness)
	return valuationPrev >= 6 && valuationNow <= 5.5
}

func applyActionCaps(review *domain.CompanyReview, action domain.ActionType, bucket domain.WatchlistBucket, constraints []string) (domain.ActionType, domain.WatchlistBucket, []string) {
	for _, section := range review.Sections {
		switch section.SectionActionCap {
		case domain.SectionActionCapCannotBuy:
			if action == domain.ActionBuy {
				action = domain.ActionWatch
				bucket = domain.WatchlistBucketWatch
				constraints = append(constraints, fmt.Sprintf("%s imposed cannot_buy", section.SectionName))
			}
		case domain.SectionActionCapWatchOnly:
			if action == domain.ActionBuy {
				action = domain.ActionWatch
				bucket = domain.WatchlistBucketWatch
				constraints = append(constraints, fmt.Sprintf("%s imposed watch_only", section.SectionName))
			}
		case domain.SectionActionCapExitReviewOnly:
			bucket = domain.WatchlistBucketExitReview
			if action == domain.ActionBuy {
				action = domain.ActionWatch
			}
			constraints = append(constraints, fmt.Sprintf("%s imposed exit_review_only", section.SectionName))
		}
	}

	return action, bucket, constraints
}

func priorityRankForAction(action domain.ActionType) int {
	switch action {
	case domain.ActionBuy:
		return 1
	case domain.ActionHold:
		return 2
	case domain.ActionWatch:
		return 3
	case domain.ActionTrim:
		return 4
	case domain.ActionSell:
		return 5
	default:
		return 6
	}
}

func capitalPriorityScore(action domain.ActionType, review *domain.CompanyReview) float64 {
	if action != domain.ActionBuy || review == nil {
		return 0
	}

	return domain.NormalizeScore(review.WeightedTotalScore)
}

func trancheStyleForAction(action domain.ActionType) string {
	switch action {
	case domain.ActionBuy:
		return "start"
	case domain.ActionHold:
		return "pause"
	case domain.ActionTrim:
		return "reduce"
	case domain.ActionSell:
		return "exit"
	default:
		return "pause"
	}
}
