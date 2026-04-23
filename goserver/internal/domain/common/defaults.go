package common

import "fmt"

type ScoreBand struct {
	Name string  `json:"name" bson:"name"`
	Min  float64 `json:"min" bson:"min"`
	Max  float64 `json:"max" bson:"max"`
}

type BalancedBuyThresholds struct {
	MinWeightedTotalScore                      float64       `json:"minWeightedTotalScore" bson:"minWeightedTotalScore"`
	MinManagementGovernanceScore               float64       `json:"minManagementGovernanceScore" bson:"minManagementGovernanceScore"`
	MinCapitalEfficiencyFinancialStrengthScore float64       `json:"minCapitalEfficiencyFinancialStrengthScore" bson:"minCapitalEfficiencyFinancialStrengthScore"`
	MinCoreSectionScore                        float64       `json:"minCoreSectionScore" bson:"minCoreSectionScore"`
	MinCoreSectionsAtOrAboveThreshold          int           `json:"minCoreSectionsAtOrAboveThreshold" bson:"minCoreSectionsAtOrAboveThreshold"`
	MaxCoreSectionsBelowFloor                  int           `json:"maxCoreSectionsBelowFloor" bson:"maxCoreSectionsBelowFloor"`
	CoreSectionFloor                           float64       `json:"coreSectionFloor" bson:"coreSectionFloor"`
	MinValuationEntryAttractivenessScore       float64       `json:"minValuationEntryAttractivenessScore" bson:"minValuationEntryAttractivenessScore"`
	RequiredCoreSections                       []SectionName `json:"requiredCoreSections" bson:"requiredCoreSections"`
	RequiresNoHardGateFailure                  bool          `json:"requiresNoHardGateFailure" bson:"requiresNoHardGateFailure"`
}

type TradingRiskGuardrails struct {
	MaxRiskPerTradePct      float64 `json:"maxRiskPerTradePct" bson:"maxRiskPerTradePct"`
	MaxConcurrentPositions  int     `json:"maxConcurrentPositions" bson:"maxConcurrentPositions"`
	KillSwitchDrawdownPct   float64 `json:"killSwitchDrawdownPct" bson:"killSwitchDrawdownPct"`
	CooldownDaysAfterSwitch int     `json:"cooldownDaysAfterSwitch" bson:"cooldownDaysAfterSwitch"`
}

var DefaultSectionWeights = map[SectionName]float64{
	SectionNameInvestability:                      5,
	SectionNameBusinessTraction:                   15,
	SectionNameProfitConversion:                   13,
	SectionNameCapitalEfficiencyFinancialStrength: 16,
	SectionNameStructuralSectorAttractiveness:     6,
	SectionNameRunwayIndustryPositioning:          13,
	SectionNameManagementGovernance:               16,
	SectionNameMarketConfirmation:                 6,
	SectionNameValuationEntryAttractiveness:       10,
}

var DefaultSubScoreWeights = map[SectionName]map[SubScoreName]float64{
	SectionNameInvestability: {
		SubScoreNameLiquidity:                          40,
		SubScoreNameDataQualityCompleteness:            25,
		SubScoreNameBasicInvestabilitySuitability:      20,
		SubScoreNameListingOperatingHistorySufficiency: 15,
	},
	SectionNameBusinessTraction: {
		SubScoreNameRevenueGrowthStrength:              30,
		SubScoreNameRevenueGrowthConsistency:           25,
		SubScoreNameRecent12QAccelerationDeterioration: 25,
		SubScoreNameEvidenceOfExpandingDemand:          20,
	},
	SectionNameProfitConversion: {
		SubScoreNameOperatingMarginQualityTrend:            30,
		SubScoreNameProfitGrowthStrength:                   25,
		SubScoreNameCashConversionQuality:                  30,
		SubScoreNameRecentOperatingLeverageMarginDirection: 15,
	},
	SectionNameCapitalEfficiencyFinancialStrength: {
		SubScoreNameROCEROICQuality:                     35,
		SubScoreNameBalanceSheetStrength:                30,
		SubScoreNameWorkingCapitalEfficiency:            20,
		SubScoreNameDilutionCapitalAllocationDiscipline: 15,
	},
	SectionNameStructuralSectorAttractiveness: {
		SubScoreNameDemandTailwindStrength:     35,
		SubScoreNameIndustryEconomicsQuality:   30,
		SubScoreNamePolicyFormalizationSupport: 20,
		SubScoreNameCyclicalityRisk:            15,
	},
	SectionNameRunwayIndustryPositioning: {
		SubScoreNameMarketOpportunitySize:          30,
		SubScoreNameShareGainPotential:             30,
		SubScoreNameExpansionOptionality:           20,
		SubScoreNameCompetitivePositioningStrength: 20,
	},
	SectionNameManagementGovernance: {
		SubScoreNameCapitalAllocationQuality:            30,
		SubScoreNameExecutionConsistency:                30,
		SubScoreNameShareholderAlignmentTrustworthiness: 25,
		SubScoreNameDisclosureQuality:                   15,
	},
	SectionNameMarketConfirmation: {
		SubScoreNameRelativeStrength:           35,
		SubScoreNameTrendQuality:               30,
		SubScoreNameDrawdownResilienceBehavior: 20,
		SubScoreNameReactionToResultsNews:      15,
	},
	SectionNameValuationEntryAttractiveness: {
		SubScoreNameHistoricalValuationAttractiveness: 35,
		SubScoreNameValuationSupportVsCurrentQuality:  30,
		SubScoreNameOvervaluationRisk:                 20,
		SubScoreNameEntryTimingSuitability:            15,
	},
}

var InvestingCoreSections = []SectionName{
	SectionNameBusinessTraction,
	SectionNameProfitConversion,
	SectionNameCapitalEfficiencyFinancialStrength,
	SectionNameRunwayIndustryPositioning,
	SectionNameManagementGovernance,
}

var DefaultBalancedScoreBands = []ScoreBand{
	{Name: "exceptional", Min: 8.5, Max: 10.0},
	{Name: "strong", Min: 7.5, Max: 8.4},
	{Name: "acceptable_promising", Min: 6.5, Max: 7.4},
	{Name: "weak_caution", Min: 5.5, Max: 6.4},
	{Name: "poor", Min: 0.0, Max: 5.49},
}

var DefaultBalancedBuyThresholds = BalancedBuyThresholds{
	MinWeightedTotalScore:                      7.5,
	MinManagementGovernanceScore:               7.0,
	MinCapitalEfficiencyFinancialStrengthScore: 7.0,
	MinCoreSectionScore:                        7.0,
	MinCoreSectionsAtOrAboveThreshold:          3,
	MaxCoreSectionsBelowFloor:                  1,
	CoreSectionFloor:                           6.5,
	MinValuationEntryAttractivenessScore:       6.0,
	RequiredCoreSections:                       InvestingCoreSections,
	RequiresNoHardGateFailure:                  true,
}

var DefaultTradingRiskGuardrails = TradingRiskGuardrails{
	MaxRiskPerTradePct:      1.0,
	MaxConcurrentPositions:  6,
	KillSwitchDrawdownPct:   10.0,
	CooldownDaysAfterSwitch: 28,
}

func ValidateDefaultSectionWeights() error {
	var total float64
	for sectionName, weight := range DefaultSectionWeights {
		if !sectionName.IsValid() {
			return fmt.Errorf("invalid default section name %q", sectionName)
		}
		if err := ValidatePercentage("default section weight", weight); err != nil {
			return err
		}
		total += weight
	}
	if !NearlyEqual(total, 100) {
		return fmt.Errorf("default section weights must total 100")
	}
	return nil
}

func ValidateDefaultSubScoreWeights() error {
	for sectionName, weights := range DefaultSubScoreWeights {
		if !sectionName.IsValid() {
			return fmt.Errorf("invalid sub-score section name %q", sectionName)
		}
		var total float64
		for subScoreName, weight := range weights {
			if !subScoreName.IsValid() {
				return fmt.Errorf("invalid sub-score name %q", subScoreName)
			}
			if err := ValidatePercentage("default sub-score weight", weight); err != nil {
				return err
			}
			total += weight
		}
		if !NearlyEqual(total, 100) {
			return fmt.Errorf("default sub-score weights must total 100 for section %q", sectionName)
		}
	}
	return nil
}

func ValidateBalancedBuyThresholds() error {
	thresholds := DefaultBalancedBuyThresholds
	if err := ValidateComputedScore("min weighted total score", thresholds.MinWeightedTotalScore); err != nil {
		return err
	}
	if err := ValidateComputedScore("min management governance score", thresholds.MinManagementGovernanceScore); err != nil {
		return err
	}
	if err := ValidateComputedScore("min capital efficiency score", thresholds.MinCapitalEfficiencyFinancialStrengthScore); err != nil {
		return err
	}
	if err := ValidateComputedScore("min core section score", thresholds.MinCoreSectionScore); err != nil {
		return err
	}
	if err := ValidateComputedScore("core section floor", thresholds.CoreSectionFloor); err != nil {
		return err
	}
	if err := ValidateComputedScore("min valuation score", thresholds.MinValuationEntryAttractivenessScore); err != nil {
		return err
	}
	if err := ValidatePositiveInt("min core sections at or above threshold", thresholds.MinCoreSectionsAtOrAboveThreshold); err != nil {
		return err
	}
	if err := ValidateNonNegativeInt("max core sections below floor", thresholds.MaxCoreSectionsBelowFloor); err != nil {
		return err
	}
	for _, section := range thresholds.RequiredCoreSections {
		if !section.IsValid() {
			return fmt.Errorf("invalid required core section %q", section)
		}
	}
	return nil
}
