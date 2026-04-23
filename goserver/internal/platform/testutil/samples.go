package testutil

import (
	"time"

	"goserver/internal/platform/domain"
)

func SampleCompany() *domain.Company {
	now := time.Date(2026, 4, 23, 0, 0, 0, 0, time.UTC)
	return &domain.Company{
		ID:                    "507f1f77bcf86cd799439011",
		Symbol:                "INFY",
		Exchange:              "NSE",
		CompanyName:           "Infosys Limited",
		Sector:                "Information Technology",
		Industry:              "IT Services",
		SubIndustry:           "Digital Services",
		BusinessSummary:       "Large-cap India-listed technology services exporter.",
		ListingDate:           time.Date(1993, 6, 14, 0, 0, 0, 0, time.UTC),
		MarketCapBucket:       "large_cap",
		IsInInvestingUniverse: true,
		IsInTradingUniverse:   true,
		StatusActive:          true,
		SchemaVersion:         domain.SchemaVersionV1Alpha1,
		CreatedAt:             now,
		UpdatedAt:             now,
	}
}

func SampleInvestingReview(overall float64, owned bool) *domain.CompanyReview {
	now := time.Date(2026, 4, 23, 0, 0, 0, 0, time.UTC)
	subScores := func(sectionName domain.InvestingSectionName, weights []float64) []domain.SubScore {
		names := domain.InvestingSectionSubScores[sectionName]
		items := make([]domain.SubScore, 0, len(names))
		for index, name := range names {
			items = append(items, domain.SubScore{
				SubScoreName:     name,
				SubScoreWeight:   weights[index],
				SubScoreValue:    overall,
				SubScoreSummary:  "Sample sub-score.",
				TrendDirection:   domain.TrendDirectionStable,
				EvidenceStrength: domain.EvidenceStrengthMedium,
				MetricBasis:      domain.MetricBasisHybrid,
				EvidenceRefIDs:   []string{"e1"},
			})
		}
		return items
	}

	weights := map[domain.InvestingSectionName]float64{
		domain.SectionInvestability:                      5,
		domain.SectionBusinessTraction:                   15,
		domain.SectionProfitConversion:                   13,
		domain.SectionCapitalEfficiencyFinancialStrength: 16,
		domain.SectionStructuralSectorAttractiveness:     6,
		domain.SectionRunwayIndustryPositioning:          13,
		domain.SectionManagementGovernance:               16,
		domain.SectionMarketConfirmation:                 6,
		domain.SectionValuationEntryAttractiveness:       10,
	}
	subWeights := map[domain.InvestingSectionName][]float64{
		domain.SectionInvestability:                      {40, 25, 20, 15},
		domain.SectionBusinessTraction:                   {30, 25, 25, 20},
		domain.SectionProfitConversion:                   {30, 25, 30, 15},
		domain.SectionCapitalEfficiencyFinancialStrength: {35, 30, 20, 15},
		domain.SectionStructuralSectorAttractiveness:     {35, 30, 20, 15},
		domain.SectionRunwayIndustryPositioning:          {30, 30, 20, 20},
		domain.SectionManagementGovernance:               {30, 30, 25, 15},
		domain.SectionMarketConfirmation:                 {35, 30, 20, 15},
		domain.SectionValuationEntryAttractiveness:       {35, 30, 20, 15},
	}

	sections := make([]domain.SectionScore, 0, len(domain.InvestingSectionsInOrder))
	for _, sectionName := range domain.InvestingSectionsInOrder {
		sections = append(sections, domain.SectionScore{
			SectionName:               string(sectionName),
			SectionWeight:             weights[sectionName],
			SectionScoreRaw:           overall,
			SectionScoreWeighted:      overall * weights[sectionName] / 100,
			SectionPassedMinimumCheck: true,
			SectionSummary:            "Sample section summary.",
			SectionConfidenceScore:    0.8,
			SubScores:                 subScores(sectionName, subWeights[sectionName]),
			EvidenceRefs: []domain.EvidenceReference{
				{
					ID:                "e1",
					SourceType:        domain.EvidenceSourceAnnualReport,
					SourceDate:        &now,
					SourceTitle:       "Annual Report FY26",
					EvidenceSummary:   "Sample evidence.",
					EvidenceDirection: domain.EvidenceDirectionPositive,
				},
			},
		})
	}

	return &domain.CompanyReview{
		ID:                        "507f1f77bcf86cd799439021",
		CompanyID:                 "507f1f77bcf86cd799439011",
		Symbol:                    "INFY",
		BookType:                  domain.BookTypeInvesting,
		ReviewDate:                now,
		ReviewPeriodType:          domain.ReviewPeriodMonthly,
		ConfigSnapshotID:          "507f1f77bcf86cd799439031",
		ReviewStatus:              domain.ReviewStatusDraft,
		Mode:                      domain.InvestingModeBalanced,
		OwnedBeforeReview:         owned,
		CurrentBucketBeforeReview: domain.WatchlistBucketWatch,
		CurrentActionBeforeReview: domain.ActionWatch,
		WeightedTotalScore:        overall,
		HardGateFailed:            false,
		ConfidenceScore:           0.8,
		ReviewerType:              domain.ReviewerTypeHybrid,
		AIModelName:               "gpt-5.4-mini",
		AIPromptVersion:           "investing-review-v1",
		SchemaVersion:             domain.SchemaVersionV1Alpha1,
		Sections:                  sections,
		PositionSnapshot: &domain.PositionSnapshot{
			IsOwned:                     owned,
			PositionPctOfBook:           4,
			PositionPctOfTotalPortfolio: 2.8,
			TargetPositionPct:           5,
			MaxPositionPct:              10,
		},
		CreatedAt: now,
		UpdatedAt: now,
	}
}

func SampleThesis() *domain.InvestmentThesis {
	now := time.Date(2026, 4, 23, 0, 0, 0, 0, time.UTC)
	return &domain.InvestmentThesis{
		ID:                         "507f1f77bcf86cd799439041",
		CompanyID:                  "507f1f77bcf86cd799439011",
		ThesisStatus:               domain.ThesisStatusActive,
		ThesisVersion:              1,
		CreatedFromReviewID:        "507f1f77bcf86cd799439021",
		LastUpdatedFromReviewID:    "507f1f77bcf86cd799439021",
		ThesisSummary:              "Digital services compounding through scale and capability depth.",
		WhyThisBusinessCanCompound: "Strong customer retention, large talent base, and recurring enterprise demand.",
		DesiredHoldingPeriod:       "3-10 years",
		ConfidenceLevel:            0.8,
		ThesisHealthScore:          8,
		CurrentPositionRole:        domain.PositionRoleBuilding,
		SchemaVersion:              domain.SchemaVersionV1Alpha1,
		CreatedAt:                  now,
		UpdatedAt:                  now,
	}
}
