package config

import (
	"fmt"
	"math"
	"strings"
	"time"

	"goserver/internal/platform/domain"
)

func (config AppConfig) Validate() error {
	var errs ValidationErrors

	validateRequiredString(&errs, "schemaVersion", config.SchemaVersion)
	if config.SchemaVersion != "" && config.SchemaVersion != domain.SchemaVersionV1Alpha1 {
		errs.Add("schemaVersion", fmt.Sprintf("%v: %s", ErrUnsupportedSchemaVersion, config.SchemaVersion))
	}

	validateRequiredString(&errs, "environment", config.Environment)
	validateTimezone(&errs, "timezone", config.EffectiveTimezone())

	if config.Server.Port <= 0 {
		errs.Add("server.port", "must be greater than zero")
	}
	if config.Server.ReadHeaderTimeout <= 0 {
		errs.Add("server.readHeaderTimeout", "must be greater than zero")
	}

	validateRequiredString(&errs, "mongo.uri", config.Mongo.URI)
	validateRequiredString(&errs, "mongo.database", config.Mongo.Database)
	validateCollectionNames(&errs, config.Mongo.Collections)

	validateGlobalConfig(&errs, config.Global)
	validateInvestingConfig(&errs, config.Investing)
	validateTradingConfig(&errs, config.Trading)
	validateAIConfig(&errs, config.AsyncAI, config.Global)
	validateUIConfig(&errs, config.UI)

	return errs.OrNil()
}

func validateGlobalConfig(errs *ValidationErrors, config GlobalConfig) {
	validateTimezone(errs, "global.defaultTimezone", config.DefaultTimezone)
	validateRequiredString(errs, "global.reviewSchemaVersion", config.ReviewSchemaVersion)
	validateRequiredString(errs, "global.promptSchemaVersion", config.PromptSchemaVersion)
	if !config.AllowedBooks.Investing && !config.AllowedBooks.Trading {
		errs.Add("global.allowedBooks", "must enable at least one book")
	}
}

func validateInvestingConfig(errs *ValidationErrors, config InvestingConfig) {
	if !domain.IsValidInvestingMode(config.DefaultMode) {
		errs.Add("investing.defaultMode", fmt.Sprintf("invalid value %q", config.DefaultMode))
	}

	validatePositiveInt(errs, "investing.reviewCadence.defaultDays", config.ReviewCadence.DefaultDays)
	validatePositiveInt(errs, "investing.reviewCadence.research", config.ReviewCadence.Research)
	validatePositiveInt(errs, "investing.reviewCadence.watch", config.ReviewCadence.Watch)
	validatePositiveInt(errs, "investing.reviewCadence.buyReady", config.ReviewCadence.BuyReady)
	validatePositiveInt(errs, "investing.reviewCadence.hold", config.ReviewCadence.Hold)
	validatePositiveInt(errs, "investing.reviewCadence.exitReview", config.ReviewCadence.ExitReview)
	if !config.WatchlistBuckets.Research || !config.WatchlistBuckets.Watch || !config.WatchlistBuckets.BuyReady || !config.WatchlistBuckets.Hold || !config.WatchlistBuckets.ExitReview {
		errs.Add("investing.watchlistBuckets", "all default watchlist buckets must remain enabled")
	}

	validatePercent(errs, "investing.positionSizing.minMeaningfulTargetPct", config.PositionSizing.MinMeaningfulTargetPct)
	validatePercent(errs, "investing.positionSizing.maxPositionCapPct", config.PositionSizing.MaxPositionCapPct)
	if config.PositionSizing.MinMeaningfulTargetPct > config.PositionSizing.MaxPositionCapPct {
		errs.Add("investing.positionSizing", "minMeaningfulTargetPct cannot exceed maxPositionCapPct")
	}
	validatePositiveInt(errs, "investing.positionSizing.tranchePolicy.defaultTrancheCount", config.PositionSizing.TranchePolicy.DefaultTrancheCount)
	validatePositiveInt(errs, "investing.positionSizing.tranchePolicy.minimumMonthsBetweenAdds", config.PositionSizing.TranchePolicy.MinimumMonthsBetweenAdds)
	if strings.TrimSpace(config.PositionSizing.TranchePolicy.DeploymentCadence) != "monthly" {
		errs.Add("investing.positionSizing.tranchePolicy.deploymentCadence", "must be \"monthly\"")
	}
	validatePortfolioSplit(errs, "investing.allocation.portfolioTargetSplit", config.Allocation.PortfolioTargetSplit)

	validateInvestingSectionWeights(errs, config.SectionWeights)
	validateInvestingSubScoreWeights(errs, config.SubScoreWeights)

	if config.ValuationRules.ExtremeOvervaluationAction != "" && !domain.IsValidSectionActionCap(config.ValuationRules.ExtremeOvervaluationAction) {
		errs.Add("investing.valuationRules.extremeOvervaluationAction", fmt.Sprintf("invalid value %q", config.ValuationRules.ExtremeOvervaluationAction))
	}
	if !config.ValuationRules.Metrics.PE &&
		!config.ValuationRules.Metrics.EVEBITDA &&
		!config.ValuationRules.Metrics.PB &&
		!config.ValuationRules.Metrics.PriceSales &&
		!config.ValuationRules.Metrics.FCFYield {
		errs.Add("investing.valuationRules.metrics", "must enable at least one valuation metric")
	}

	validateScore(errs, "investing.actionThresholds.scoreBands.exceptionalMin", config.ActionThresholds.ScoreBands.ExceptionalMin)
	validateScore(errs, "investing.actionThresholds.scoreBands.strongMin", config.ActionThresholds.ScoreBands.StrongMin)
	validateScore(errs, "investing.actionThresholds.scoreBands.acceptableMin", config.ActionThresholds.ScoreBands.AcceptableMin)
	validateScore(errs, "investing.actionThresholds.scoreBands.weakMin", config.ActionThresholds.ScoreBands.WeakMin)
	if !(config.ActionThresholds.ScoreBands.ExceptionalMin > config.ActionThresholds.ScoreBands.StrongMin &&
		config.ActionThresholds.ScoreBands.StrongMin > config.ActionThresholds.ScoreBands.AcceptableMin &&
		config.ActionThresholds.ScoreBands.AcceptableMin > config.ActionThresholds.ScoreBands.WeakMin) {
		errs.Add("investing.actionThresholds.scoreBands", "must be strictly descending")
	}

	validateScore(errs, "investing.actionThresholds.buy.weightedTotalMin", config.ActionThresholds.Buy.WeightedTotalMin)
	validateScore(errs, "investing.actionThresholds.buy.managementGovernanceMin", config.ActionThresholds.Buy.ManagementGovernanceMin)
	validateScore(errs, "investing.actionThresholds.buy.capitalEfficiencyFinancialStrengthMin", config.ActionThresholds.Buy.CapitalEfficiencyFinancialStrengthMin)
	validateScore(errs, "investing.actionThresholds.buy.valuationEntryMin", config.ActionThresholds.Buy.ValuationEntryMin)
	validateScore(errs, "investing.actionThresholds.buy.coreSectionStrongMin", config.ActionThresholds.Buy.CoreSectionStrongMin)
	validateScore(errs, "investing.actionThresholds.buy.coreSectionFloor", config.ActionThresholds.Buy.CoreSectionFloor)
	validatePositiveInt(errs, "investing.actionThresholds.buy.minCoreSectionsAtOrAboveThreshold", config.ActionThresholds.Buy.MinCoreSectionsAtOrAboveThreshold)
	validateNonNegativeInt(errs, "investing.actionThresholds.buy.maxCoreSectionsBelowFloor", config.ActionThresholds.Buy.MaxCoreSectionsBelowFloor)
	if config.ActionThresholds.Buy.CoreSectionStrongMin < config.ActionThresholds.Buy.CoreSectionFloor {
		errs.Add("investing.actionThresholds.buy", "coreSectionStrongMin cannot be lower than coreSectionFloor")
	}
	validateScore(errs, "investing.actionThresholds.holdMinOverall", config.ActionThresholds.HoldMinOverall)
	validateScore(errs, "investing.actionThresholds.rejectBelowOverall", config.ActionThresholds.RejectBelowOverall)
	validateScore(errs, "investing.actionThresholds.sellBelowOverall", config.ActionThresholds.SellBelowOverall)
	validateNonNegativeScore(errs, "investing.actionThresholds.changeEscalation.totalScoreDropExitReviewThreshold", config.ActionThresholds.ChangeEscalation.TotalScoreDropExitReviewThreshold)
	validateNonNegativeScore(errs, "investing.actionThresholds.changeEscalation.coreSectionDropThreshold", config.ActionThresholds.ChangeEscalation.CoreSectionDropThreshold)
	validateNonNegativeScore(errs, "investing.actionThresholds.changeEscalation.managementGovernanceDropThreshold", config.ActionThresholds.ChangeEscalation.ManagementGovernanceDropThreshold)

	validatePositiveInt(errs, "investing.lookback.yearsLookback", config.Lookback.YearsLookback)
	validatePositiveInt(errs, "investing.lookback.recentQuarterLookback", config.Lookback.RecentQuarterLookback)

	validateTextSourcePriority(errs, "investing.textSourcePriority", config.TextSourcePriority)
}

func validateTradingConfig(errs *ValidationErrors, config TradingConfig) {
	if !config.Universe.IncludeStocks && !config.Universe.IncludeETFs {
		errs.Add("trading.universe", "must include at least one instrument type")
	}
	if config.Style.HoldingStyle != TradingHoldingStylePositionTrades {
		errs.Add("trading.style.holdingStyle", fmt.Sprintf("invalid value %q", config.Style.HoldingStyle))
	}
	if config.Style.CoreEdge != TradingCoreEdgeTrendMomentum {
		errs.Add("trading.style.coreEdge", fmt.Sprintf("invalid value %q", config.Style.CoreEdge))
	}
	validatePercent(errs, "trading.risk.riskPerTradePct", config.Risk.RiskPerTradePct)
	validatePositiveInt(errs, "trading.risk.maxConcurrentPositions", config.Risk.MaxConcurrentPositions)
	if config.Risk.StopMethod != TradingStopMethodVolatility {
		errs.Add("trading.risk.stopMethod", fmt.Sprintf("invalid value %q", config.Risk.StopMethod))
	}
	if config.Risk.ProfitManagement != TradingProfitManagementTrailingStop {
		errs.Add("trading.risk.profitManagement", fmt.Sprintf("invalid value %q", config.Risk.ProfitManagement))
	}
	validatePercent(errs, "trading.circuitBreaker.killSwitchDrawdownPct", config.CircuitBreaker.KillSwitchDrawdownPct)
	validatePositiveInt(errs, "trading.circuitBreaker.cooldownWeeks", config.CircuitBreaker.CooldownWeeks)
}

func validateAIConfig(errs *ValidationErrors, config AIConfig, global GlobalConfig) {
	validateRequiredString(errs, "ai.providerName", config.ProviderName)
	validateRequiredString(errs, "ai.defaultModelName", config.DefaultModelName)
	validateRequiredString(errs, "ai.promptVersion", config.PromptVersion)
	validateRequiredString(errs, "ai.schemaVersion", config.SchemaVersion)
	if !config.EnabledBooks.Investing && !config.EnabledBooks.Trading {
		errs.Add("ai.enabledBooks", "must enable at least one book")
	}
	validatePositiveInt(errs, "ai.batch.maxBatchSize", config.Batch.MaxBatchSize)
	validatePositiveInt(errs, "ai.batch.maxItemsPerBatch", config.Batch.MaxItemsPerBatch)
	validateNonNegativeInt(errs, "ai.batch.submissionRetryLimit", config.Batch.SubmissionRetryLimit)
	validateNonNegativeInt(errs, "ai.batch.pollRetryLimit", config.Batch.PollRetryLimit)
	validateNonNegativeInt(errs, "ai.batch.itemRetryLimit", config.Batch.ItemRetryLimit)
	validatePositiveDuration(errs, "ai.batch.pollInterval", config.Batch.PollInterval)
	validatePositiveDuration(errs, "ai.batch.reconciliationInterval", config.Batch.ReconciliationInterval)
	validatePositiveDuration(errs, "ai.batch.resultFetchTimeout", config.Batch.ResultFetchTimeout)
	validatePositiveDuration(errs, "ai.batch.batchTimeout", config.Batch.BatchTimeout)
	validatePositiveDuration(errs, "ai.worker.refreshInterval", config.Worker.RefreshInterval)
	validatePositiveDuration(errs, "ai.worker.minBatchRefreshAge", config.Worker.MinBatchRefreshAge)
	validatePositiveDuration(errs, "ai.worker.followUpClaimTimeout", config.Worker.FollowUpClaimTimeout)
	validatePositiveInt(errs, "ai.worker.maxBatchesPerPass", config.Worker.MaxBatchesPerPass)
	validateRequiredString(errs, "ai.snapshot.promptVersion", config.Snapshot.PromptVersion)
	validateRequiredString(errs, "ai.snapshot.reviewSchemaVersion", firstNonEmptyString(config.Snapshot.ReviewSchemaVersion, global.ReviewSchemaVersion))
	validateRequiredString(errs, "ai.snapshot.outputSchemaVersion", config.Snapshot.OutputSchemaVersion)
}

func validateUIConfig(errs *ValidationErrors, config UIConfig) {
	validatePositiveInt(errs, "ui.defaultPageSize", config.DefaultPageSize)
	validatePositiveInt(errs, "ui.maxPageSize", config.MaxPageSize)
	if config.MaxPageSize < config.DefaultPageSize {
		errs.Add("ui.maxPageSize", "must be greater than or equal to defaultPageSize")
	}
}

func validateCollectionNames(errs *ValidationErrors, collections CollectionConfig) {
	values := map[string]string{
		"mongo.collections.companies":               collections.Companies,
		"mongo.collections.companyReviews":          collections.CompanyReviews,
		"mongo.collections.investmentTheses":        collections.InvestmentTheses,
		"mongo.collections.workflowRuns":            collections.WorkflowRuns,
		"mongo.collections.workflowStepRuns":        collections.WorkflowStepRuns,
		"mongo.collections.configSnapshots":         collections.ConfigSnapshots,
		"mongo.collections.capitalAllocationRuns":   collections.CapitalAllocationRuns,
		"mongo.collections.manualOverrides":         collections.ManualOverrides,
		"mongo.collections.currentPositions":        collections.CurrentPositions,
		"mongo.collections.aiBatchJobs":             collections.AIBatchJobs,
		"mongo.collections.aiBatchItems":            collections.AIBatchItems,
		"mongo.collections.jobReconciliationLogs":   collections.JobReconciliationLogs,
		"mongo.collections.providerBatchJobs":       collections.ProviderBatchJobs,
		"mongo.collections.providerBatchIterations": collections.ProviderBatchIterations,
	}
	for field, value := range values {
		validateRequiredString(errs, field, value)
	}
}

func validatePortfolioSplit(errs *ValidationErrors, field string, split PortfolioSplit) {
	validatePercent(errs, field+".investingBookPct", split.InvestingBookPct)
	validatePercent(errs, field+".tradingBookPct", split.TradingBookPct)
	validatePercent(errs, field+".liquidReservePct", split.LiquidReservePct)
	if !nearlyEqual(split.InvestingBookPct+split.TradingBookPct+split.LiquidReservePct, 100) {
		errs.Add(field, "must total 100")
	}
}

func validateInvestingSectionWeights(errs *ValidationErrors, weights InvestingSectionWeights) {
	pairs := []struct {
		field string
		value float64
	}{
		{"investing.sectionWeights.investability", weights.Investability},
		{"investing.sectionWeights.businessTraction", weights.BusinessTraction},
		{"investing.sectionWeights.profitConversion", weights.ProfitConversion},
		{"investing.sectionWeights.capitalEfficiencyFinancialStrength", weights.CapitalEfficiencyFinancialStrength},
		{"investing.sectionWeights.structuralSectorAttractiveness", weights.StructuralSectorAttractiveness},
		{"investing.sectionWeights.runwayIndustryPositioning", weights.RunwayIndustryPositioning},
		{"investing.sectionWeights.managementGovernance", weights.ManagementGovernance},
		{"investing.sectionWeights.marketConfirmation", weights.MarketConfirmation},
		{"investing.sectionWeights.valuationEntryAttractiveness", weights.ValuationEntryAttractiveness},
	}
	var total float64
	for _, pair := range pairs {
		validatePercent(errs, pair.field, pair.value)
		total += pair.value
	}
	if !nearlyEqual(total, 100) {
		errs.Add("investing.sectionWeights", "must total 100")
	}
}

func validateInvestingSubScoreWeights(errs *ValidationErrors, weights InvestingSubScoreWeights) {
	validateWeightGroup(errs, "investing.subScoreWeights.investability", []float64{
		weights.Investability.Liquidity,
		weights.Investability.DataQualityCompleteness,
		weights.Investability.BasicInvestabilitySuitability,
		weights.Investability.ListingOperatingHistorySufficiency,
	})
	validateWeightGroup(errs, "investing.subScoreWeights.businessTraction", []float64{
		weights.BusinessTraction.RevenueGrowthStrength,
		weights.BusinessTraction.RevenueGrowthConsistency,
		weights.BusinessTraction.Recent12QAccelerationDeterioration,
		weights.BusinessTraction.EvidenceOfExpandingDemand,
	})
	validateWeightGroup(errs, "investing.subScoreWeights.profitConversion", []float64{
		weights.ProfitConversion.OperatingMarginQualityTrend,
		weights.ProfitConversion.ProfitGrowthStrength,
		weights.ProfitConversion.CashConversionQuality,
		weights.ProfitConversion.RecentOperatingLeverageMarginDirection,
	})
	validateWeightGroup(errs, "investing.subScoreWeights.capitalEfficiencyFinancialStrength", []float64{
		weights.CapitalEfficiencyFinancialStrength.ROCEROICQuality,
		weights.CapitalEfficiencyFinancialStrength.BalanceSheetStrength,
		weights.CapitalEfficiencyFinancialStrength.WorkingCapitalEfficiency,
		weights.CapitalEfficiencyFinancialStrength.DilutionCapitalAllocationDiscipline,
	})
	validateWeightGroup(errs, "investing.subScoreWeights.structuralSectorAttractiveness", []float64{
		weights.StructuralSectorAttractiveness.DemandTailwindStrength,
		weights.StructuralSectorAttractiveness.IndustryEconomicsQuality,
		weights.StructuralSectorAttractiveness.PolicyFormalizationSupport,
		weights.StructuralSectorAttractiveness.CyclicalityRisk,
	})
	validateWeightGroup(errs, "investing.subScoreWeights.runwayIndustryPositioning", []float64{
		weights.RunwayIndustryPositioning.MarketOpportunitySize,
		weights.RunwayIndustryPositioning.ShareGainPotential,
		weights.RunwayIndustryPositioning.ExpansionOptionality,
		weights.RunwayIndustryPositioning.CompetitivePositioningStrength,
	})
	validateWeightGroup(errs, "investing.subScoreWeights.managementGovernance", []float64{
		weights.ManagementGovernance.CapitalAllocationQuality,
		weights.ManagementGovernance.ExecutionConsistency,
		weights.ManagementGovernance.ShareholderAlignmentTrustworthiness,
		weights.ManagementGovernance.DisclosureQuality,
	})
	validateWeightGroup(errs, "investing.subScoreWeights.marketConfirmation", []float64{
		weights.MarketConfirmation.RelativeStrength,
		weights.MarketConfirmation.TrendQuality,
		weights.MarketConfirmation.DrawdownResilienceBehavior,
		weights.MarketConfirmation.ReactionToResultsNews,
	})
	validateWeightGroup(errs, "investing.subScoreWeights.valuationEntryAttractiveness", []float64{
		weights.ValuationEntryAttractiveness.HistoricalValuationAttractiveness,
		weights.ValuationEntryAttractiveness.ValuationSupportVsCurrentQuality,
		weights.ValuationEntryAttractiveness.OvervaluationRisk,
		weights.ValuationEntryAttractiveness.EntryTimingSuitability,
	})
}

func validateWeightGroup(errs *ValidationErrors, field string, values []float64) {
	var total float64
	for _, value := range values {
		validatePercent(errs, field, value)
		total += value
	}
	if !nearlyEqual(total, 100) {
		errs.Add(field, "must total 100")
	}
}

func validateTextSourcePriority(errs *ValidationErrors, field string, values []domain.EvidenceSourceType) {
	if len(values) == 0 {
		errs.Add(field, "must not be empty")
		return
	}

	required := map[domain.EvidenceSourceType]struct{}{
		domain.EvidenceSourceAnnualReport:         {},
		domain.EvidenceSourceConcall:              {},
		domain.EvidenceSourceInvestorPresentation: {},
		domain.EvidenceSourceExchangeFiling:       {},
	}
	seen := make(map[domain.EvidenceSourceType]struct{}, len(values))
	for index, value := range values {
		if !domain.IsValidEvidenceSourceType(value) {
			errs.Add(fmt.Sprintf("%s[%d]", field, index), fmt.Sprintf("invalid value %q", value))
			continue
		}
		if _, exists := seen[value]; exists {
			errs.Add(fmt.Sprintf("%s[%d]", field, index), fmt.Sprintf("duplicate value %q", value))
		}
		seen[value] = struct{}{}
		delete(required, value)
	}
	if len(required) != 0 {
		errs.Add(field, "must include annual_report, concall, investor_presentation, and exchange_filing")
	}
}

func validateRequiredString(errs *ValidationErrors, field, value string) {
	if strings.TrimSpace(value) == "" {
		errs.Add(field, "is required")
	}
}

func validateTimezone(errs *ValidationErrors, field, value string) {
	if strings.TrimSpace(value) == "" {
		errs.Add(field, "is required")
		return
	}
	if _, err := time.LoadLocation(value); err != nil {
		errs.Add(field, fmt.Sprintf("invalid timezone %q", value))
	}
}

func validatePercent(errs *ValidationErrors, field string, value float64) {
	if value < 0 || value > 100 {
		errs.Add(field, "must be between 0 and 100")
	}
}

func validateScore(errs *ValidationErrors, field string, value float64) {
	if value <= 0 || value > 10 {
		errs.Add(field, "must be between 0 and 10")
	}
}

func validateNonNegativeScore(errs *ValidationErrors, field string, value float64) {
	if value < 0 || value > 10 {
		errs.Add(field, "must be between 0 and 10")
	}
}

func validatePositiveInt(errs *ValidationErrors, field string, value int) {
	if value <= 0 {
		errs.Add(field, "must be greater than zero")
	}
}

func validateNonNegativeInt(errs *ValidationErrors, field string, value int) {
	if value < 0 {
		errs.Add(field, "must be zero or greater")
	}
}

func validatePositiveDuration(errs *ValidationErrors, field string, value time.Duration) {
	if value <= 0 {
		errs.Add(field, "must be greater than zero")
	}
}

func nearlyEqual(left, right float64) bool {
	return math.Abs(left-right) <= 0.0001
}
