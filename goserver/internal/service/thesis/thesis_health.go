package thesis

import (
	"fmt"
	"math"

	domaincommon "goserver/internal/domain/common"
	domainreview "goserver/internal/domain/review"
	domainthesis "goserver/internal/domain/thesis"
)

func determineThesisDecision(
	review *domainreview.CompanyReview,
	existing *domainthesis.InvestmentThesis,
	config ThesisEvaluationConfig,
) thesisDecision {
	status := determineThesisStatus(review, config)
	decision := thesisDecision{
		Status:                   status,
		HealthScore:              computeThesisHealthScore(review, status, config),
		PositionRole:             determinePositionRole(review, existing, status),
		NewSupportingEvidence:    collectSupportingEvidence(review),
		NewContradictingEvidence: collectContradictingEvidence(review),
		BreakSignals:             collectThesisBreakSignals(review, config),
	}
	decision.ThesisChangeSummary = buildChangeSummary(review, decision)
	decision.Summary = buildDecisionSummary(review, decision)
	return decision
}

func determineThesisStatus(review *domainreview.CompanyReview, config ThesisEvaluationConfig) domaincommon.ThesisStatus {
	if shouldMarkBroken(review, config) {
		return domaincommon.ThesisStatusBroken
	}
	if shouldMarkUnderReview(review, config) {
		return domaincommon.ThesisStatusUnderReview
	}
	return domaincommon.ThesisStatusActive
}

func shouldMarkBroken(review *domainreview.CompanyReview, config ThesisEvaluationConfig) bool {
	if review == nil {
		return false
	}
	if hardGateIsThesisBreaking(review) {
		return true
	}
	if review.WeightedTotalScore > 0 && review.WeightedTotalScore < config.BrokenScoreThreshold {
		return true
	}
	if coreWeakSectionCount(review, config.WeakCoreSectionThreshold) >= 2 {
		return true
	}
	if sectionScore(review, domaincommon.SectionNameManagementGovernance) > 0 &&
		sectionScore(review, domaincommon.SectionNameManagementGovernance) < config.WeakCoreSectionThreshold &&
		hasNegativeEvidenceForSection(review, domaincommon.SectionNameManagementGovernance) {
		return true
	}
	if sectionScore(review, domaincommon.SectionNameCapitalEfficiencyFinancialStrength) > 0 &&
		sectionScore(review, domaincommon.SectionNameCapitalEfficiencyFinancialStrength) < config.WeakCoreSectionThreshold &&
		hasWorseningRiskForSection(review, domaincommon.SectionNameCapitalEfficiencyFinancialStrength) {
		return true
	}
	if changeLogIndicatesThesisBreak(review) {
		return true
	}
	if finalAction(review) == domaincommon.InvestingActionTypeSell {
		return config.SellOnThesisBreak && sellAppearsThesisDriven(review, config)
	}
	return false
}

func shouldMarkUnderReview(review *domainreview.CompanyReview, config ThesisEvaluationConfig) bool {
	if review == nil {
		return false
	}
	if finalAction(review) == domaincommon.InvestingActionTypeTrim {
		return true
	}
	if finalBucket(review) == domaincommon.WatchlistBucketExitReview {
		return true
	}
	if review.ChangeLog != nil && review.ChangeLog.RequiresExitReview {
		return true
	}
	if totalScoreDropped(review, config.UnderReviewTotalDropThreshold) {
		return true
	}
	if coreSectionDropped(review, config.UnderReviewCoreDropThreshold) {
		return true
	}
	if managementDropped(review, config.UnderReviewMgmtDropThreshold) {
		return true
	}
	if hasMajorNegativeChanges(review) {
		return true
	}
	if valuationExtremeWithBusinessSoftening(review) {
		return true
	}
	if finalAction(review) == domaincommon.InvestingActionTypeSell {
		return true
	}
	return false
}

func sellAppearsThesisDriven(review *domainreview.CompanyReview, config ThesisEvaluationConfig) bool {
	if review == nil {
		return false
	}
	if changeLogIndicatesThesisBreak(review) || hardGateIsThesisBreaking(review) {
		return true
	}
	if review.WeightedTotalScore > 0 && review.WeightedTotalScore < config.BrokenScoreThreshold {
		return true
	}
	if coreWeakSectionCount(review, config.WeakCoreSectionThreshold) >= 2 {
		return true
	}
	if hasMajorNegativeChanges(review) {
		return true
	}
	if review.DecisionAction != nil {
		text := review.DecisionAction.ActionReasonPrimary + " " + review.DecisionAction.ActionReasonSecondary + " " + review.DecisionAction.Notes
		if containsThesisBreakLanguage(text) {
			return true
		}
	}
	return false
}

func computeThesisHealthScore(
	review *domainreview.CompanyReview,
	status domaincommon.ThesisStatus,
	config ThesisEvaluationConfig,
) float64 {
	if review == nil {
		return 1
	}
	score := review.WeightedTotalScore
	if score <= 0 {
		score = averageSectionScore(review)
	}

	if hardGateIsThesisBreaking(review) {
		score -= 2.0
	} else if review.HardGateFailed {
		score -= 0.5
	}
	if review.ChangeLog != nil && review.ChangeLog.RequiresExitReview {
		score -= 0.5
	}

	weakCore := coreWeakSectionCount(review, config.WeakCoreSectionThreshold)
	if weakCore >= 2 {
		score -= 0.75
	}
	if weakCore >= 3 {
		score -= 0.5
	}
	if sectionScore(review, domaincommon.SectionNameManagementGovernance) > 0 &&
		sectionScore(review, domaincommon.SectionNameManagementGovernance) < config.WeakCoreSectionThreshold {
		score -= 0.75
	}
	if sectionScore(review, domaincommon.SectionNameCapitalEfficiencyFinancialStrength) > 0 &&
		sectionScore(review, domaincommon.SectionNameCapitalEfficiencyFinancialStrength) < config.WeakCoreSectionThreshold {
		score -= 0.5
	}
	if review.ChangeLog != nil {
		for _, change := range review.ChangeLog.SectionScoreChanges {
			if change <= -config.UnderReviewCoreDropThreshold {
				score -= 0.25
			}
		}
		negativeCount := len(nonBlankStrings(review.ChangeLog.MajorNegativeChanges))
		if negativeCount > 0 {
			score -= math.Min(float64(negativeCount)*0.25, 1.0)
		}
		positiveCount := len(nonBlankStrings(review.ChangeLog.MajorPositiveChanges))
		if positiveCount > 0 {
			score += math.Min(float64(positiveCount)*0.1, 0.3)
		}
	}

	switch finalAction(review) {
	case domaincommon.InvestingActionTypeSell:
		score -= 0.7
	case domaincommon.InvestingActionTypeTrim:
		score -= 0.4
	case domaincommon.InvestingActionTypeBuy:
		score += 0.1
	}

	strongCore := coreStrongSectionCount(review, config.StrongCoreSectionThreshold)
	score += math.Min(float64(strongCore)*0.1, 0.4)
	if review.ConfidenceScore >= 0.8 {
		score += 0.2
	} else if review.ConfidenceScore > 0 && review.ConfidenceScore < 0.45 {
		score -= 0.3
	}

	score = clampScore(score)
	if status == domaincommon.ThesisStatusBroken && score > 5.4 {
		score = 5.4
	}
	if status == domaincommon.ThesisStatusUnderReview && score > 7.0 {
		score = 7.0
	}
	return roundToTenth(score)
}

func determinePositionRole(
	review *domainreview.CompanyReview,
	existing *domainthesis.InvestmentThesis,
	status domaincommon.ThesisStatus,
) domaincommon.PositionRole {
	action := finalAction(review)
	if status == domaincommon.ThesisStatusBroken || action == domaincommon.InvestingActionTypeSell {
		return domaincommon.PositionRoleExitCandidate
	}
	if action == domaincommon.InvestingActionTypeTrim {
		return domaincommon.PositionRoleTrimCandidate
	}
	if action == domaincommon.InvestingActionTypeBuy {
		if review.PositionSnapshot == nil || !review.PositionSnapshot.IsOwned || review.PositionSnapshot.PositionPctOfBook <= 0 {
			return domaincommon.PositionRoleStarter
		}
		if review.PositionSnapshot.TargetPositionPct > 0 &&
			review.PositionSnapshot.PositionPctOfBook < review.PositionSnapshot.TargetPositionPct*0.8 {
			return domaincommon.PositionRoleBuilding
		}
		if review.PositionSnapshot.UnderweightVsTargetPct > 0 {
			return domaincommon.PositionRoleBuilding
		}
		return domaincommon.PositionRoleCore
	}
	if action == domaincommon.InvestingActionTypeHold {
		if review.PositionSnapshot != nil && review.PositionSnapshot.IsOwned {
			return domaincommon.PositionRoleCore
		}
		if review.OwnedBeforeReview {
			return domaincommon.PositionRoleCore
		}
	}
	if existing != nil && existing.CurrentPositionRole != "" {
		return existing.CurrentPositionRole
	}
	return domaincommon.PositionRoleStarter
}

func averageSectionScore(review *domainreview.CompanyReview) float64 {
	if review == nil || len(review.Sections) == 0 {
		return 1
	}
	var total float64
	var count int
	for _, section := range review.Sections {
		if section.SectionScoreRaw <= 0 {
			continue
		}
		total += section.SectionScoreRaw
		count++
	}
	if count == 0 {
		return 1
	}
	return total / float64(count)
}

func buildDecisionSummary(review *domainreview.CompanyReview, decision thesisDecision) string {
	switch decision.Status {
	case domaincommon.ThesisStatusBroken:
		return fmt.Sprintf("thesis broken for review %s; health %.1f", review.ID.Hex(), decision.HealthScore)
	case domaincommon.ThesisStatusUnderReview:
		return fmt.Sprintf("thesis under review for review %s; health %.1f", review.ID.Hex(), decision.HealthScore)
	default:
		return fmt.Sprintf("thesis active for review %s; health %.1f", review.ID.Hex(), decision.HealthScore)
	}
}
