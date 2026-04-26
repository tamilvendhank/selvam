package thesis

import (
	"fmt"
	"sort"
	"strings"
	"time"

	domaincommon "goserver/internal/domain/common"
	domainreview "goserver/internal/domain/review"
	domainthesis "goserver/internal/domain/thesis"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

const (
	maxThesisListItems        = 8
	maxThesisEvidenceItems    = 10
	maxThesisBreakSignalItems = 8
)

func buildNewThesisFromReview(
	review *domainreview.CompanyReview,
	decision thesisDecision,
	config ThesisEvaluationConfig,
	now time.Time,
) *domainthesis.InvestmentThesis {
	supporting := decision.NewSupportingEvidence
	contradicting := decision.NewContradictingEvidence
	breakSignals := decision.BreakSignals

	return &domainthesis.InvestmentThesis{
		CompanyID:                  review.CompanyID,
		ThesisStatus:               decision.Status,
		ThesisVersion:              1,
		CreatedFromReviewID:        review.ID,
		LastUpdatedFromReviewID:    review.ID,
		ThesisSummary:              buildThesisSummary(review),
		WhyThisBusinessCanCompound: buildCompoundReason(review),
		KeyGrowthDrivers:           collectKeyGrowthDrivers(review),
		KeyMoatOrAdvantageFactors:  collectMoatFactors(review),
		WhyNow:                     buildWhyNow(review),
		KeyRisks:                   collectKeyRisks(review),
		DisconfirmingSignals:       collectContradictingEvidence(review),
		WhatWouldBreakTheThesis:    breakSignals,
		DesiredHoldingPeriod:       config.DesiredHoldingPeriod,
		ConfidenceLevel:            clampUnit(review.ConfidenceScore),
		ThesisHealthScore:          decision.HealthScore,
		ThesisChangeSummary:        decision.ThesisChangeSummary,
		NewSupportingEvidence:      supporting,
		NewContradictingEvidence:   contradicting,
		CurrentPositionRole:        decision.PositionRole,
		CreatedAt:                  now,
		UpdatedAt:                  now,
		SchemaVersion:              schemaVersion(review),
	}
}

func buildUpdatedThesisFromReview(
	existing *domainthesis.InvestmentThesis,
	review *domainreview.CompanyReview,
	decision thesisDecision,
	now time.Time,
) *domainthesis.InvestmentThesis {
	candidate := *existing
	candidate.ID = primitive.ObjectID{}
	candidate.ThesisStatus = decision.Status
	candidate.ThesisVersion = existing.ThesisVersion + 1
	candidate.LastUpdatedFromReviewID = review.ID
	candidate.ThesisHealthScore = decision.HealthScore
	candidate.ConfidenceLevel = clampUnit(review.ConfidenceScore)
	candidate.ThesisChangeSummary = decision.ThesisChangeSummary
	candidate.NewSupportingEvidence = decision.NewSupportingEvidence
	candidate.NewContradictingEvidence = decision.NewContradictingEvidence
	candidate.CurrentPositionRole = decision.PositionRole
	candidate.UpdatedAt = now
	if candidate.CreatedAt.IsZero() {
		candidate.CreatedAt = now
	}
	if candidate.CreatedFromReviewID.IsZero() {
		candidate.CreatedFromReviewID = existing.CreatedFromReviewID
	}
	if candidate.ThesisSummary == "" {
		candidate.ThesisSummary = buildThesisSummary(review)
	}
	if candidate.WhyThisBusinessCanCompound == "" {
		candidate.WhyThisBusinessCanCompound = buildCompoundReason(review)
	}
	candidate.KeyGrowthDrivers = mergeStringSlices(candidate.KeyGrowthDrivers, collectKeyGrowthDrivers(review), maxThesisListItems)
	candidate.KeyMoatOrAdvantageFactors = mergeStringSlices(candidate.KeyMoatOrAdvantageFactors, collectMoatFactors(review), maxThesisListItems)
	candidate.KeyRisks = mergeStringSlices(candidate.KeyRisks, collectKeyRisks(review), maxThesisListItems)
	candidate.DisconfirmingSignals = mergeStringSlices(candidate.DisconfirmingSignals, collectContradictingEvidence(review), maxThesisEvidenceItems)
	candidate.WhatWouldBreakTheThesis = mergeStringSlices(candidate.WhatWouldBreakTheThesis, decision.BreakSignals, maxThesisBreakSignalItems)
	if candidate.WhyNow == "" {
		candidate.WhyNow = buildWhyNow(review)
	}
	if candidate.DesiredHoldingPeriod == "" {
		candidate.DesiredHoldingPeriod = "3-10 years"
	}
	if candidate.SchemaVersion <= 0 {
		candidate.SchemaVersion = schemaVersion(review)
	}
	return &candidate
}

func buildThesisSummary(review *domainreview.CompanyReview) string {
	return firstNonEmpty(
		review.ActionRationaleSummary,
		review.WhatChangedSummary,
		changeSummary(review),
		fmt.Sprintf("Investment thesis derived from %s review.", review.Symbol),
	)
}

func buildCompoundReason(review *domainreview.CompanyReview) string {
	items := make([]string, 0, 5)
	for _, sectionName := range []domaincommon.SectionName{
		domaincommon.SectionNameBusinessTraction,
		domaincommon.SectionNameProfitConversion,
		domaincommon.SectionNameCapitalEfficiencyFinancialStrength,
		domaincommon.SectionNameRunwayIndustryPositioning,
		domaincommon.SectionNameManagementGovernance,
	} {
		section := sectionByName(review, sectionName)
		if section == nil {
			continue
		}
		if section.SectionScoreRaw >= 7 {
			items = append(items, firstNonEmpty(section.SectionSummary, firstString(section.SectionStrengths)))
		}
	}
	items = nonBlankStrings(items)
	if len(items) > 0 {
		return strings.Join(limitStrings(items, 4), " ")
	}
	return firstNonEmpty(
		review.ActionRationaleSummary,
		review.WhatChangedSummary,
		"Review scorecard supports a compounding business case.",
	)
}

func buildWhyNow(review *domainreview.CompanyReview) string {
	items := make([]string, 0, 3)
	if valuation := sectionByName(review, domaincommon.SectionNameValuationEntryAttractiveness); valuation != nil {
		items = append(items, valuation.SectionSummary)
	}
	if market := sectionByName(review, domaincommon.SectionNameMarketConfirmation); market != nil {
		items = append(items, market.SectionSummary)
	}
	items = append(items, review.ActionRationaleSummary)
	return strings.Join(limitStrings(nonBlankStrings(items), 3), " ")
}

func buildChangeSummary(review *domainreview.CompanyReview, decision thesisDecision) string {
	parts := make([]string, 0, 4)
	parts = append(parts, changeSummary(review))
	if decision.Status == domaincommon.ThesisStatusBroken {
		parts = append(parts, "Thesis marked broken based on review evidence.")
	} else if decision.Status == domaincommon.ThesisStatusUnderReview {
		parts = append(parts, "Thesis moved under review pending further confirmation.")
	} else {
		parts = append(parts, "Thesis remains active.")
	}
	if len(decision.BreakSignals) > 0 {
		parts = append(parts, "Key watch item: "+decision.BreakSignals[0])
	}
	return strings.Join(limitStrings(nonBlankStrings(parts), 4), " ")
}

func collectSupportingEvidence(review *domainreview.CompanyReview) []string {
	items := make([]string, 0, maxThesisEvidenceItems)
	items = append(items, review.ActionRationaleSummary, review.WhatChangedSummary)
	if review.ChangeLog != nil {
		items = append(items, review.ChangeLog.MajorPositiveChanges...)
	}
	for _, section := range review.Sections {
		if section.SectionScoreRaw >= 7 {
			items = append(items, prefixSectionItems(section.SectionName, section.SectionStrengths)...)
			for _, evidence := range section.EvidenceRefs {
				if evidence.EvidenceDirection == domaincommon.EvidenceDirectionPositive {
					items = append(items, evidenceText(section.SectionName, evidence))
				}
			}
		}
	}
	return limitStrings(uniqueStrings(items), maxThesisEvidenceItems)
}

func collectContradictingEvidence(review *domainreview.CompanyReview) []string {
	items := make([]string, 0, maxThesisEvidenceItems)
	items = append(items, review.HardGateFailureReasons...)
	if review.ChangeLog != nil {
		items = append(items, review.ChangeLog.MajorNegativeChanges...)
	}
	for _, section := range review.Sections {
		if section.SectionScoreRaw < 6.5 || len(section.SectionWeaknesses) > 0 || len(section.SectionRisks) > 0 {
			items = append(items, prefixSectionItems(section.SectionName, section.SectionWeaknesses)...)
			items = append(items, prefixSectionItems(section.SectionName, section.SectionRisks)...)
		}
		for _, evidence := range section.EvidenceRefs {
			if evidence.EvidenceDirection == domaincommon.EvidenceDirectionNegative {
				items = append(items, evidenceText(section.SectionName, evidence))
			}
		}
	}
	return limitStrings(uniqueStrings(items), maxThesisEvidenceItems)
}

func collectKeyGrowthDrivers(review *domainreview.CompanyReview) []string {
	sections := []domaincommon.SectionName{
		domaincommon.SectionNameBusinessTraction,
		domaincommon.SectionNameRunwayIndustryPositioning,
		domaincommon.SectionNameProfitConversion,
		domaincommon.SectionNameStructuralSectorAttractiveness,
	}
	return collectSectionStrengths(review, sections, maxThesisListItems)
}

func collectMoatFactors(review *domainreview.CompanyReview) []string {
	sections := []domaincommon.SectionName{
		domaincommon.SectionNameRunwayIndustryPositioning,
		domaincommon.SectionNameManagementGovernance,
		domaincommon.SectionNameCapitalEfficiencyFinancialStrength,
		domaincommon.SectionNameStructuralSectorAttractiveness,
	}
	return collectSectionStrengths(review, sections, maxThesisListItems)
}

func collectKeyRisks(review *domainreview.CompanyReview) []string {
	items := make([]string, 0, maxThesisListItems)
	items = append(items, review.HardGateFailureReasons...)
	for _, section := range sortedSectionsByWeakness(review.Sections) {
		items = append(items, prefixSectionItems(section.SectionName, section.SectionRisks)...)
		items = append(items, prefixSectionItems(section.SectionName, section.SectionWeaknesses)...)
	}
	if review.ChangeLog != nil {
		items = append(items, review.ChangeLog.MajorNegativeChanges...)
	}
	return limitStrings(uniqueStrings(items), maxThesisListItems)
}

func collectThesisBreakSignals(review *domainreview.CompanyReview, config ThesisEvaluationConfig) []string {
	items := make([]string, 0, maxThesisBreakSignalItems)
	if hardGateIsThesisBreaking(review) {
		items = append(items, review.HardGateFailureReasons...)
		if len(review.HardGateFailureReasons) == 0 {
			items = append(items, "Thesis-breaking hard gate failure.")
		}
	}
	if review.WeightedTotalScore < config.BrokenScoreThreshold {
		items = append(items, fmt.Sprintf("Weighted score below %.1f.", config.BrokenScoreThreshold))
	}
	for _, sectionName := range domaincommon.InvestingCoreSections {
		section := sectionByName(review, sectionName)
		if section == nil || section.SectionScoreRaw >= config.WeakCoreSectionThreshold {
			continue
		}
		items = append(items, fmt.Sprintf("%s score below %.1f.", humanizeSectionName(sectionName), config.WeakCoreSectionThreshold))
	}
	if review.ChangeLog != nil {
		if review.ChangeLog.RequiresExitReview {
			items = append(items, "Review change log requires exit review.")
		}
		items = append(items, review.ChangeLog.MajorNegativeChanges...)
		if containsThesisBreakLanguage(review.ChangeLog.ChangeSummary) {
			items = append(items, review.ChangeLog.ChangeSummary)
		}
	}
	return limitStrings(uniqueStrings(items), maxThesisBreakSignalItems)
}

func collectSectionStrengths(review *domainreview.CompanyReview, sections []domaincommon.SectionName, limit int) []string {
	items := make([]string, 0, limit)
	for _, name := range sections {
		section := sectionByName(review, name)
		if section == nil {
			continue
		}
		if section.SectionScoreRaw >= 7 {
			items = append(items, prefixSectionItems(section.SectionName, section.SectionStrengths)...)
			if len(section.SectionStrengths) == 0 {
				items = append(items, prefixSectionItems(section.SectionName, []string{section.SectionSummary})...)
			}
		}
	}
	return limitStrings(uniqueStrings(items), limit)
}

func sortedSectionsByWeakness(sections []domainreview.SectionScore) []domainreview.SectionScore {
	copied := append([]domainreview.SectionScore(nil), sections...)
	sort.SliceStable(copied, func(i, j int) bool {
		return copied[i].SectionScoreRaw < copied[j].SectionScoreRaw
	})
	return copied
}

func prefixSectionItems(sectionName domaincommon.SectionName, values []string) []string {
	values = nonBlankStrings(values)
	items := make([]string, 0, len(values))
	prefix := humanizeSectionName(sectionName)
	for _, value := range values {
		items = append(items, prefix+": "+value)
	}
	return items
}

func evidenceText(sectionName domaincommon.SectionName, evidence domainreview.EvidenceReference) string {
	value := firstNonEmpty(evidence.EvidenceSummary, evidence.ExcerptOrMetricValue, evidence.ExcerptOrMetricName, evidence.SourceTitle)
	if value == "" {
		return ""
	}
	return humanizeSectionName(sectionName) + ": " + value
}

func changeSummary(review *domainreview.CompanyReview) string {
	if review == nil {
		return ""
	}
	if review.ChangeLog != nil {
		return firstNonEmpty(review.ChangeLog.ChangeSummary, review.WhatChangedSummary)
	}
	return review.WhatChangedSummary
}

func schemaVersion(review *domainreview.CompanyReview) int {
	if review != nil && review.SchemaVersion > 0 {
		return review.SchemaVersion
	}
	return domaincommon.SchemaVersion1
}
