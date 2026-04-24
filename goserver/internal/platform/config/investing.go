package config

import "goserver/internal/platform/domain"

type InvestingConfig struct {
	DefaultMode        domain.InvestingMode            `json:"defaultMode" yaml:"defaultMode"`
	ReviewCadence      InvestingReviewCadenceConfig    `json:"reviewCadence" yaml:"reviewCadence"`
	WatchlistBuckets   InvestingWatchlistBucketsConfig `json:"watchlistBuckets" yaml:"watchlistBuckets"`
	PositionSizing     InvestingPositionSizingConfig   `json:"positionSizing" yaml:"positionSizing"`
	Allocation         InvestingAllocationConfig       `json:"allocation,omitempty" yaml:"allocation,omitempty"`
	ValuationRules     InvestingValuationRulesConfig   `json:"valuationRules" yaml:"valuationRules"`
	SectionWeights     InvestingSectionWeights         `json:"sectionWeights" yaml:"sectionWeights"`
	SubScoreWeights    InvestingSubScoreWeights        `json:"subScoreWeights" yaml:"subScoreWeights"`
	ActionThresholds   InvestingActionThresholds       `json:"actionThresholds" yaml:"actionThresholds"`
	HardGates          InvestingHardGateConfig         `json:"hardGates" yaml:"hardGates"`
	Lookback           InvestingLookbackConfig         `json:"lookback" yaml:"lookback"`
	TextSourcePriority []domain.EvidenceSourceType     `json:"textSourcePriority" yaml:"textSourcePriority"`
	ThesisRules        ThesisRulesConfig               `json:"thesisRules" yaml:"thesisRules"`
	CoreSections       []string                        `json:"-" yaml:"-"`
}

type InvestingReviewCadenceConfig struct {
	DefaultDays int `json:"defaultDays" yaml:"defaultDays"`
	Research    int `json:"research" yaml:"research"`
	Watch       int `json:"watch" yaml:"watch"`
	BuyReady    int `json:"buyReady" yaml:"buyReady"`
	Hold        int `json:"hold" yaml:"hold"`
	ExitReview  int `json:"exitReview" yaml:"exitReview"`
}

type InvestingWatchlistBucketsConfig struct {
	Research   bool `json:"research" yaml:"research"`
	Watch      bool `json:"watch" yaml:"watch"`
	BuyReady   bool `json:"buyReady" yaml:"buyReady"`
	Hold       bool `json:"hold" yaml:"hold"`
	ExitReview bool `json:"exitReview" yaml:"exitReview"`
}

type InvestingPositionSizingConfig struct {
	DynamicSizing          bool                `json:"dynamicSizing" yaml:"dynamicSizing"`
	MinMeaningfulTargetPct float64             `json:"minMeaningfulTargetPct" yaml:"minMeaningfulTargetPct"`
	MaxPositionCapPct      float64             `json:"maxPositionCapPct" yaml:"maxPositionCapPct"`
	TranchePolicy          TranchePolicyConfig `json:"tranchePolicy" yaml:"tranchePolicy"`
}

type TranchePolicyConfig struct {
	DefaultTrancheCount      int    `json:"defaultTrancheCount" yaml:"defaultTrancheCount"`
	DeploymentCadence        string `json:"deploymentCadence" yaml:"deploymentCadence"`
	MinimumMonthsBetweenAdds int    `json:"minimumMonthsBetweenAdds" yaml:"minimumMonthsBetweenAdds"`
	AllowPartialTrim         bool   `json:"allowPartialTrim" yaml:"allowPartialTrim"`
}

type InvestingAllocationConfig struct {
	PortfolioTargetSplit PortfolioSplit `json:"portfolioTargetSplit" yaml:"portfolioTargetSplit"`
}

type PortfolioSplit struct {
	InvestingBookPct float64 `json:"investingBookPct" yaml:"investingBookPct"`
	TradingBookPct   float64 `json:"tradingBookPct" yaml:"tradingBookPct"`
	LiquidReservePct float64 `json:"liquidReservePct" yaml:"liquidReservePct"`
}

type InvestingValuationRulesConfig struct {
	HistoricalValuationLensEnabled bool                    `json:"historicalValuationLensEnabled" yaml:"historicalValuationLensEnabled"`
	ExtremeOvervaluationAction     domain.SectionActionCap `json:"extremeOvervaluationAction" yaml:"extremeOvervaluationAction"`
	Metrics                        ValuationMetricConfig   `json:"metrics" yaml:"metrics"`
}

type ValuationMetricConfig struct {
	PE         bool `json:"pe" yaml:"pe"`
	EVEBITDA   bool `json:"evEbitda" yaml:"evEbitda"`
	PB         bool `json:"pb" yaml:"pb"`
	PriceSales bool `json:"priceSales" yaml:"priceSales"`
	FCFYield   bool `json:"fcfYield" yaml:"fcfYield"`
}

type InvestingSectionWeights struct {
	Investability                      float64 `json:"investability" yaml:"investability"`
	BusinessTraction                   float64 `json:"businessTraction" yaml:"businessTraction"`
	ProfitConversion                   float64 `json:"profitConversion" yaml:"profitConversion"`
	CapitalEfficiencyFinancialStrength float64 `json:"capitalEfficiencyFinancialStrength" yaml:"capitalEfficiencyFinancialStrength"`
	StructuralSectorAttractiveness     float64 `json:"structuralSectorAttractiveness" yaml:"structuralSectorAttractiveness"`
	RunwayIndustryPositioning          float64 `json:"runwayIndustryPositioning" yaml:"runwayIndustryPositioning"`
	ManagementGovernance               float64 `json:"managementGovernance" yaml:"managementGovernance"`
	MarketConfirmation                 float64 `json:"marketConfirmation" yaml:"marketConfirmation"`
	ValuationEntryAttractiveness       float64 `json:"valuationEntryAttractiveness" yaml:"valuationEntryAttractiveness"`
}

type InvestingSubScoreWeights struct {
	Investability                      InvestabilitySubScoreWeights                      `json:"investability" yaml:"investability"`
	BusinessTraction                   BusinessTractionSubScoreWeights                   `json:"businessTraction" yaml:"businessTraction"`
	ProfitConversion                   ProfitConversionSubScoreWeights                   `json:"profitConversion" yaml:"profitConversion"`
	CapitalEfficiencyFinancialStrength CapitalEfficiencyFinancialStrengthSubScoreWeights `json:"capitalEfficiencyFinancialStrength" yaml:"capitalEfficiencyFinancialStrength"`
	StructuralSectorAttractiveness     StructuralSectorAttractivenessSubScoreWeights     `json:"structuralSectorAttractiveness" yaml:"structuralSectorAttractiveness"`
	RunwayIndustryPositioning          RunwayIndustryPositioningSubScoreWeights          `json:"runwayIndustryPositioning" yaml:"runwayIndustryPositioning"`
	ManagementGovernance               ManagementGovernanceSubScoreWeights               `json:"managementGovernance" yaml:"managementGovernance"`
	MarketConfirmation                 MarketConfirmationSubScoreWeights                 `json:"marketConfirmation" yaml:"marketConfirmation"`
	ValuationEntryAttractiveness       ValuationEntryAttractivenessSubScoreWeights       `json:"valuationEntryAttractiveness" yaml:"valuationEntryAttractiveness"`
}

type InvestabilitySubScoreWeights struct {
	Liquidity                          float64 `json:"liquidity" yaml:"liquidity"`
	DataQualityCompleteness            float64 `json:"dataQualityCompleteness" yaml:"dataQualityCompleteness"`
	BasicInvestabilitySuitability      float64 `json:"basicInvestabilitySuitability" yaml:"basicInvestabilitySuitability"`
	ListingOperatingHistorySufficiency float64 `json:"listingOperatingHistorySufficiency" yaml:"listingOperatingHistorySufficiency"`
}

type BusinessTractionSubScoreWeights struct {
	RevenueGrowthStrength              float64 `json:"revenueGrowthStrength" yaml:"revenueGrowthStrength"`
	RevenueGrowthConsistency           float64 `json:"revenueGrowthConsistency" yaml:"revenueGrowthConsistency"`
	Recent12QAccelerationDeterioration float64 `json:"recent12QAccelerationDeterioration" yaml:"recent12QAccelerationDeterioration"`
	EvidenceOfExpandingDemand          float64 `json:"evidenceOfExpandingDemand" yaml:"evidenceOfExpandingDemand"`
}

type ProfitConversionSubScoreWeights struct {
	OperatingMarginQualityTrend            float64 `json:"operatingMarginQualityTrend" yaml:"operatingMarginQualityTrend"`
	ProfitGrowthStrength                   float64 `json:"profitGrowthStrength" yaml:"profitGrowthStrength"`
	CashConversionQuality                  float64 `json:"cashConversionQuality" yaml:"cashConversionQuality"`
	RecentOperatingLeverageMarginDirection float64 `json:"recentOperatingLeverageMarginDirection" yaml:"recentOperatingLeverageMarginDirection"`
}

type CapitalEfficiencyFinancialStrengthSubScoreWeights struct {
	ROCEROICQuality                     float64 `json:"roceRoicQuality" yaml:"roceRoicQuality"`
	BalanceSheetStrength                float64 `json:"balanceSheetStrength" yaml:"balanceSheetStrength"`
	WorkingCapitalEfficiency            float64 `json:"workingCapitalEfficiency" yaml:"workingCapitalEfficiency"`
	DilutionCapitalAllocationDiscipline float64 `json:"dilutionCapitalAllocationDiscipline" yaml:"dilutionCapitalAllocationDiscipline"`
}

type StructuralSectorAttractivenessSubScoreWeights struct {
	DemandTailwindStrength     float64 `json:"demandTailwindStrength" yaml:"demandTailwindStrength"`
	IndustryEconomicsQuality   float64 `json:"industryEconomicsQuality" yaml:"industryEconomicsQuality"`
	PolicyFormalizationSupport float64 `json:"policyFormalizationSupport" yaml:"policyFormalizationSupport"`
	CyclicalityRisk            float64 `json:"cyclicalityRisk" yaml:"cyclicalityRisk"`
}

type RunwayIndustryPositioningSubScoreWeights struct {
	MarketOpportunitySize          float64 `json:"marketOpportunitySize" yaml:"marketOpportunitySize"`
	ShareGainPotential             float64 `json:"shareGainPotential" yaml:"shareGainPotential"`
	ExpansionOptionality           float64 `json:"expansionOptionality" yaml:"expansionOptionality"`
	CompetitivePositioningStrength float64 `json:"competitivePositioningStrength" yaml:"competitivePositioningStrength"`
}

type ManagementGovernanceSubScoreWeights struct {
	CapitalAllocationQuality            float64 `json:"capitalAllocationQuality" yaml:"capitalAllocationQuality"`
	ExecutionConsistency                float64 `json:"executionConsistency" yaml:"executionConsistency"`
	ShareholderAlignmentTrustworthiness float64 `json:"shareholderAlignmentTrustworthiness" yaml:"shareholderAlignmentTrustworthiness"`
	DisclosureQuality                   float64 `json:"disclosureQuality" yaml:"disclosureQuality"`
}

type MarketConfirmationSubScoreWeights struct {
	RelativeStrength           float64 `json:"relativeStrength" yaml:"relativeStrength"`
	TrendQuality               float64 `json:"trendQuality" yaml:"trendQuality"`
	DrawdownResilienceBehavior float64 `json:"drawdownResilienceBehavior" yaml:"drawdownResilienceBehavior"`
	ReactionToResultsNews      float64 `json:"reactionToResultsNews" yaml:"reactionToResultsNews"`
}

type ValuationEntryAttractivenessSubScoreWeights struct {
	HistoricalValuationAttractiveness float64 `json:"historicalValuationAttractiveness" yaml:"historicalValuationAttractiveness"`
	ValuationSupportVsCurrentQuality  float64 `json:"valuationSupportVsCurrentQuality" yaml:"valuationSupportVsCurrentQuality"`
	OvervaluationRisk                 float64 `json:"overvaluationRisk" yaml:"overvaluationRisk"`
	EntryTimingSuitability            float64 `json:"entryTimingSuitability" yaml:"entryTimingSuitability"`
}

type InvestingActionThresholds struct {
	ScoreBands         InvestingScoreBandThresholds    `json:"scoreBands" yaml:"scoreBands"`
	Buy                InvestingBuyThresholds          `json:"buy" yaml:"buy"`
	ChangeEscalation   InvestingChangeEscalationConfig `json:"changeEscalation" yaml:"changeEscalation"`
	HoldMinOverall     float64                         `json:"holdMinOverall" yaml:"holdMinOverall"`
	RejectBelowOverall float64                         `json:"rejectBelowOverall" yaml:"rejectBelowOverall"`
	SellBelowOverall   float64                         `json:"sellBelowOverall" yaml:"sellBelowOverall"`

	ExceptionalMin              float64 `json:"-" yaml:"-"`
	StrongMin                   float64 `json:"-" yaml:"-"`
	AcceptableMin               float64 `json:"-" yaml:"-"`
	WeakMin                     float64 `json:"-" yaml:"-"`
	BuyMinOverall               float64 `json:"-" yaml:"-"`
	BuyMinManagement            float64 `json:"-" yaml:"-"`
	BuyMinCapitalEfficiency     float64 `json:"-" yaml:"-"`
	BuyMinValuation             float64 `json:"-" yaml:"-"`
	CoreStrongThreshold         float64 `json:"-" yaml:"-"`
	CoreWeakThreshold           float64 `json:"-" yaml:"-"`
	MaxWeakCoreSectionsForBuy   int     `json:"-" yaml:"-"`
	MinStrongCoreSectionsForBuy int     `json:"-" yaml:"-"`
	ExitReviewTotalDrop         float64 `json:"-" yaml:"-"`
	ExitReviewCoreDrop          float64 `json:"-" yaml:"-"`
	ExitReviewManagementDrop    float64 `json:"-" yaml:"-"`
}

type InvestingScoreBandThresholds struct {
	ExceptionalMin float64 `json:"exceptionalMin" yaml:"exceptionalMin"`
	StrongMin      float64 `json:"strongMin" yaml:"strongMin"`
	AcceptableMin  float64 `json:"acceptableMin" yaml:"acceptableMin"`
	WeakMin        float64 `json:"weakMin" yaml:"weakMin"`
}

type InvestingBuyThresholds struct {
	WeightedTotalMin                      float64 `json:"weightedTotalMin" yaml:"weightedTotalMin"`
	ManagementGovernanceMin               float64 `json:"managementGovernanceMin" yaml:"managementGovernanceMin"`
	CapitalEfficiencyFinancialStrengthMin float64 `json:"capitalEfficiencyFinancialStrengthMin" yaml:"capitalEfficiencyFinancialStrengthMin"`
	ValuationEntryMin                     float64 `json:"valuationEntryMin" yaml:"valuationEntryMin"`
	CoreSectionStrongMin                  float64 `json:"coreSectionStrongMin" yaml:"coreSectionStrongMin"`
	CoreSectionFloor                      float64 `json:"coreSectionFloor" yaml:"coreSectionFloor"`
	MinCoreSectionsAtOrAboveThreshold     int     `json:"minCoreSectionsAtOrAboveThreshold" yaml:"minCoreSectionsAtOrAboveThreshold"`
	MaxCoreSectionsBelowFloor             int     `json:"maxCoreSectionsBelowFloor" yaml:"maxCoreSectionsBelowFloor"`
}

type InvestingChangeEscalationConfig struct {
	TotalScoreDropExitReviewThreshold float64 `json:"totalScoreDropExitReviewThreshold" yaml:"totalScoreDropExitReviewThreshold"`
	CoreSectionDropThreshold          float64 `json:"coreSectionDropThreshold" yaml:"coreSectionDropThreshold"`
	ManagementGovernanceDropThreshold float64 `json:"managementGovernanceDropThreshold" yaml:"managementGovernanceDropThreshold"`
}

type InvestingHardGateConfig struct {
	GovernanceRedFlagAbsenceIsHardGate bool `json:"governanceRedFlagAbsenceIsHardGate" yaml:"governanceRedFlagAbsenceIsHardGate"`
	ValuationExtremeCanBlockBuy        bool `json:"valuationExtremeCanBlockBuy" yaml:"valuationExtremeCanBlockBuy"`
}

type InvestingLookbackConfig struct {
	YearsLookback         int `json:"yearsLookback" yaml:"yearsLookback"`
	RecentQuarterLookback int `json:"recentQuarterLookback" yaml:"recentQuarterLookback"`
}

type ThesisRulesConfig struct {
	RequireWrittenThesisForBuy bool `json:"requireWrittenThesisForBuy" yaml:"requireWrittenThesisForBuy"`
	SellOnThesisBreak          bool `json:"sellOnThesisBreak" yaml:"sellOnThesisBreak"`
}
