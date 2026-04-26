package contracts

import "goserver/internal/domain/common"

func ExpectedInvestingSectionNames() []common.SectionName {
	return []common.SectionName{
		common.SectionNameInvestability,
		common.SectionNameBusinessTraction,
		common.SectionNameProfitConversion,
		common.SectionNameCapitalEfficiencyFinancialStrength,
		common.SectionNameStructuralSectorAttractiveness,
		common.SectionNameRunwayIndustryPositioning,
		common.SectionNameManagementGovernance,
		common.SectionNameMarketConfirmation,
		common.SectionNameValuationEntryAttractiveness,
	}
}

func ExpectedSubScoreNamesBySection() map[common.SectionName][]common.SubScoreName {
	return map[common.SectionName][]common.SubScoreName{
		common.SectionNameInvestability: {
			common.SubScoreNameLiquidity,
			common.SubScoreNameDataQualityCompleteness,
			common.SubScoreNameBasicInvestabilitySuitability,
			common.SubScoreNameListingOperatingHistorySufficiency,
		},
		common.SectionNameBusinessTraction: {
			common.SubScoreNameRevenueGrowthStrength,
			common.SubScoreNameRevenueGrowthConsistency,
			common.SubScoreNameRecent12QAccelerationDeterioration,
			common.SubScoreNameEvidenceOfExpandingDemand,
		},
		common.SectionNameProfitConversion: {
			common.SubScoreNameOperatingMarginQualityTrend,
			common.SubScoreNameProfitGrowthStrength,
			common.SubScoreNameCashConversionQuality,
			common.SubScoreNameRecentOperatingLeverageMarginDirection,
		},
		common.SectionNameCapitalEfficiencyFinancialStrength: {
			common.SubScoreNameROCEROICQuality,
			common.SubScoreNameBalanceSheetStrength,
			common.SubScoreNameWorkingCapitalEfficiency,
			common.SubScoreNameDilutionCapitalAllocationDiscipline,
		},
		common.SectionNameStructuralSectorAttractiveness: {
			common.SubScoreNameDemandTailwindStrength,
			common.SubScoreNameIndustryEconomicsQuality,
			common.SubScoreNamePolicyFormalizationSupport,
			common.SubScoreNameCyclicalityRisk,
		},
		common.SectionNameRunwayIndustryPositioning: {
			common.SubScoreNameMarketOpportunitySize,
			common.SubScoreNameShareGainPotential,
			common.SubScoreNameExpansionOptionality,
			common.SubScoreNameCompetitivePositioningStrength,
		},
		common.SectionNameManagementGovernance: {
			common.SubScoreNameCapitalAllocationQuality,
			common.SubScoreNameExecutionConsistency,
			common.SubScoreNameShareholderAlignmentTrustworthiness,
			common.SubScoreNameDisclosureQuality,
		},
		common.SectionNameMarketConfirmation: {
			common.SubScoreNameRelativeStrength,
			common.SubScoreNameTrendQuality,
			common.SubScoreNameDrawdownResilienceBehavior,
			common.SubScoreNameReactionToResultsNews,
		},
		common.SectionNameValuationEntryAttractiveness: {
			common.SubScoreNameHistoricalValuationAttractiveness,
			common.SubScoreNameValuationSupportVsCurrentQuality,
			common.SubScoreNameOvervaluationRisk,
			common.SubScoreNameEntryTimingSuitability,
		},
	}
}

func DefaultSectionWeights() map[common.SectionName]float64 {
	return map[common.SectionName]float64{
		common.SectionNameInvestability:                      10,
		common.SectionNameBusinessTraction:                   15,
		common.SectionNameProfitConversion:                   12,
		common.SectionNameCapitalEfficiencyFinancialStrength: 13,
		common.SectionNameStructuralSectorAttractiveness:     10,
		common.SectionNameRunwayIndustryPositioning:          12,
		common.SectionNameManagementGovernance:               10,
		common.SectionNameMarketConfirmation:                 8,
		common.SectionNameValuationEntryAttractiveness:       10,
	}
}

func DefaultSubScoreWeights() map[common.SectionName]map[common.SubScoreName]float64 {
	expected := ExpectedSubScoreNamesBySection()
	weights := make(map[common.SectionName]map[common.SubScoreName]float64, len(expected))
	for sectionName, subScoreNames := range expected {
		weights[sectionName] = make(map[common.SubScoreName]float64, len(subScoreNames))
		for _, subScoreName := range subScoreNames {
			weights[sectionName][subScoreName] = 25
		}
	}
	return weights
}
