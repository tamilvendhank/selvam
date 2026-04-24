package config

import (
	"strings"
	"time"

	"goserver/internal/platform/domain"
)

func Default() AppConfig {
	config := baseDefaultConfig()
	config.normalizeDerived()
	return config
}

func baseDefaultConfig() AppConfig {
	config := AppConfig{
		SchemaVersion: domain.SchemaVersionV1Alpha1,
		Environment:   defaultEnvironment,
		Timezone:      defaultTimezone,
		Server: ServerConfig{
			Port:              8080,
			ReadHeaderTimeout: 10 * time.Second,
		},
		Mongo: MongoConfig{
			URI:      "mongodb://127.0.0.1:27017",
			Database: "selvam_platform",
			Collections: CollectionConfig{
				Companies:               "companies",
				CompanyReviews:          "company_reviews",
				InvestmentTheses:        "investment_theses",
				WorkflowRuns:            "workflow_runs",
				WorkflowStepRuns:        "workflow_step_runs",
				ConfigSnapshots:         "config_snapshots",
				CapitalAllocationRuns:   "capital_allocation_runs",
				ManualOverrides:         "manual_overrides",
				CurrentPositions:        "current_positions",
				AIBatchJobs:             "ai_batch_jobs",
				AIBatchItems:            "ai_batch_items",
				JobReconciliationLogs:   "job_reconciliation_logs",
				ProviderBatchJobs:       "query_jobs",
				ProviderBatchIterations: "submissions_iterations",
			},
		},
		Global: GlobalConfig{
			DefaultTimezone:     defaultTimezone,
			DefaultCurrency:     defaultCurrency,
			ReviewSchemaVersion: domain.SchemaVersionV1Alpha1,
			PromptSchemaVersion: "investing-review-v1",
			AllowedBooks: AllowedBooksConfig{
				Investing: true,
				Trading:   true,
			},
			DataSources: DataSourceSettings{
				FinancialDataProvider: "placeholder-financials",
				PriceDataProvider:     "placeholder-prices",
				TextDocumentProvider:  "placeholder-textdocs",
			},
			AIProviders: AIProviderSettings{
				DefaultProvider:     "openai-batch",
				DefaultModel:        "gpt-5.4-mini",
				ReviewPromptVersion: "investing-review-v1",
				BatchEnabled:        true,
			},
			FeatureFlags: FeatureFlags{
				EnableAsyncAIReview:             true,
				EnableCurrentPositionProjection: true,
				EnableTradingWorkflow:           true,
			},
		},
		Investing: InvestingConfig{
			DefaultMode: domain.InvestingModeBalanced,
			ReviewCadence: InvestingReviewCadenceConfig{
				DefaultDays: 30,
				Research:    30,
				Watch:       30,
				BuyReady:    30,
				Hold:        30,
				ExitReview:  7,
			},
			WatchlistBuckets: InvestingWatchlistBucketsConfig{
				Research:   true,
				Watch:      true,
				BuyReady:   true,
				Hold:       true,
				ExitReview: true,
			},
			PositionSizing: InvestingPositionSizingConfig{
				DynamicSizing:          true,
				MinMeaningfulTargetPct: 3,
				MaxPositionCapPct:      10,
				TranchePolicy: TranchePolicyConfig{
					DefaultTrancheCount:      3,
					DeploymentCadence:        "monthly",
					MinimumMonthsBetweenAdds: 1,
					AllowPartialTrim:         true,
				},
			},
			Allocation: InvestingAllocationConfig{
				PortfolioTargetSplit: PortfolioSplit{
					InvestingBookPct: 70,
					TradingBookPct:   20,
					LiquidReservePct: 10,
				},
			},
			ValuationRules: InvestingValuationRulesConfig{
				HistoricalValuationLensEnabled: true,
				ExtremeOvervaluationAction:     domain.SectionActionCapCannotBuy,
				Metrics: ValuationMetricConfig{
					PE:         true,
					EVEBITDA:   true,
					PB:         true,
					PriceSales: true,
					FCFYield:   true,
				},
			},
			SectionWeights: InvestingSectionWeights{
				Investability:                      5,
				BusinessTraction:                   15,
				ProfitConversion:                   13,
				CapitalEfficiencyFinancialStrength: 16,
				StructuralSectorAttractiveness:     6,
				RunwayIndustryPositioning:          13,
				ManagementGovernance:               16,
				MarketConfirmation:                 6,
				ValuationEntryAttractiveness:       10,
			},
			SubScoreWeights: InvestingSubScoreWeights{
				Investability: InvestabilitySubScoreWeights{
					Liquidity:                          40,
					DataQualityCompleteness:            25,
					BasicInvestabilitySuitability:      20,
					ListingOperatingHistorySufficiency: 15,
				},
				BusinessTraction: BusinessTractionSubScoreWeights{
					RevenueGrowthStrength:              30,
					RevenueGrowthConsistency:           25,
					Recent12QAccelerationDeterioration: 25,
					EvidenceOfExpandingDemand:          20,
				},
				ProfitConversion: ProfitConversionSubScoreWeights{
					OperatingMarginQualityTrend:            30,
					ProfitGrowthStrength:                   25,
					CashConversionQuality:                  30,
					RecentOperatingLeverageMarginDirection: 15,
				},
				CapitalEfficiencyFinancialStrength: CapitalEfficiencyFinancialStrengthSubScoreWeights{
					ROCEROICQuality:                     35,
					BalanceSheetStrength:                30,
					WorkingCapitalEfficiency:            20,
					DilutionCapitalAllocationDiscipline: 15,
				},
				StructuralSectorAttractiveness: StructuralSectorAttractivenessSubScoreWeights{
					DemandTailwindStrength:     35,
					IndustryEconomicsQuality:   30,
					PolicyFormalizationSupport: 20,
					CyclicalityRisk:            15,
				},
				RunwayIndustryPositioning: RunwayIndustryPositioningSubScoreWeights{
					MarketOpportunitySize:          30,
					ShareGainPotential:             30,
					ExpansionOptionality:           20,
					CompetitivePositioningStrength: 20,
				},
				ManagementGovernance: ManagementGovernanceSubScoreWeights{
					CapitalAllocationQuality:            30,
					ExecutionConsistency:                30,
					ShareholderAlignmentTrustworthiness: 25,
					DisclosureQuality:                   15,
				},
				MarketConfirmation: MarketConfirmationSubScoreWeights{
					RelativeStrength:           35,
					TrendQuality:               30,
					DrawdownResilienceBehavior: 20,
					ReactionToResultsNews:      15,
				},
				ValuationEntryAttractiveness: ValuationEntryAttractivenessSubScoreWeights{
					HistoricalValuationAttractiveness: 35,
					ValuationSupportVsCurrentQuality:  30,
					OvervaluationRisk:                 20,
					EntryTimingSuitability:            15,
				},
			},
			ActionThresholds: InvestingActionThresholds{
				ScoreBands: InvestingScoreBandThresholds{
					ExceptionalMin: 8.5,
					StrongMin:      7.5,
					AcceptableMin:  6.5,
					WeakMin:        5.5,
				},
				Buy: InvestingBuyThresholds{
					WeightedTotalMin:                      7.5,
					ManagementGovernanceMin:               7.0,
					CapitalEfficiencyFinancialStrengthMin: 7.0,
					ValuationEntryMin:                     6.0,
					CoreSectionStrongMin:                  7.0,
					CoreSectionFloor:                      6.5,
					MinCoreSectionsAtOrAboveThreshold:     3,
					MaxCoreSectionsBelowFloor:             1,
				},
				ChangeEscalation: InvestingChangeEscalationConfig{
					TotalScoreDropExitReviewThreshold: 1.0,
					CoreSectionDropThreshold:          1.5,
					ManagementGovernanceDropThreshold: 1.0,
				},
				HoldMinOverall:     7.0,
				RejectBelowOverall: 6.0,
				SellBelowOverall:   5.5,
			},
			HardGates: InvestingHardGateConfig{
				GovernanceRedFlagAbsenceIsHardGate: true,
				ValuationExtremeCanBlockBuy:        true,
			},
			Lookback: InvestingLookbackConfig{
				YearsLookback:         5,
				RecentQuarterLookback: 12,
			},
			TextSourcePriority: []domain.EvidenceSourceType{
				domain.EvidenceSourceAnnualReport,
				domain.EvidenceSourceConcall,
				domain.EvidenceSourceInvestorPresentation,
				domain.EvidenceSourceExchangeFiling,
			},
			ThesisRules: ThesisRulesConfig{
				RequireWrittenThesisForBuy: true,
				SellOnThesisBreak:          true,
			},
		},
		Trading: TradingConfig{
			Universe: TradingUniverseConfig{
				IncludeStocks:        true,
				IncludeETFs:          true,
				RequireHighLiquidity: true,
			},
			Style: TradingStyleConfig{
				HoldingStyle:        TradingHoldingStylePositionTrades,
				AllowBreakoutSetups: true,
				AllowPullbackSetups: true,
				CoreEdge:            TradingCoreEdgeTrendMomentum,
				UseRelativeStrength: true,
			},
			RegimeFilter: TradingRegimeFilterConfig{
				Enabled:       true,
				UseTrend:      true,
				UseBreadth:    true,
				UseVolatility: true,
			},
			Risk: TradingRiskConfig{
				RiskPerTradePct:        1,
				MaxConcurrentPositions: 6,
				HardStopRequired:       true,
				StopMethod:             TradingStopMethodVolatility,
				ProfitManagement:       TradingProfitManagementTrailingStop,
			},
			CircuitBreaker: TradingCircuitBreakerConfig{
				KillSwitchDrawdownPct: 10,
				CooldownWeeks:         4,
			},
		},
		UI: UIConfig{
			DefaultPageSize:                 25,
			MaxPageSize:                     100,
			DefaultSortField:                "-updatedAt",
			EnableAdminRetryControls:        true,
			EnableRawAIResultInspection:     true,
			EnableValidationErrorInspection: true,
		},
	}

	config.AsyncAI = AIConfig{
		Enabled:          true,
		ProviderName:     "openai-batch",
		DefaultModelName: "gpt-5.4-mini",
		PromptVersion:    "investing-review-v1",
		SchemaVersion:    domain.SchemaVersionV1Alpha1,
		EnabledBooks: AllowedBooksConfig{
			Investing: true,
			Trading:   true,
		},
		Batch: AIBatchConfig{
			MaxBatchSize:               50,
			MaxItemsPerBatch:           500,
			SubmissionRetryLimit:       3,
			PollRetryLimit:             120,
			ItemRetryLimit:             2,
			PollInterval:               15 * time.Second,
			ReconciliationInterval:     30 * time.Second,
			ValidationFailureRetryable: false,
			ResultFetchTimeout:         2 * time.Minute,
			BatchTimeout:               24 * time.Hour,
		},
		Worker: AIWorkerConfig{
			Enabled:                          true,
			RefreshInterval:                  15 * time.Second,
			MinBatchRefreshAge:               30 * time.Second,
			FollowUpClaimTimeout:             2 * time.Minute,
			MaxBatchesPerPass:                20,
			EnableBatchSubmissionWorker:      true,
			EnablePollingWorker:              true,
			EnableReconciliationWorker:       true,
			EnableWorkflowContinuationWorker: true,
		},
		Snapshot: AISnapshotConfig{
			PromptVersion:       "investing-review-v1",
			ReviewSchemaVersion: domain.SchemaVersionV1Alpha1,
			OutputSchemaVersion: domain.SchemaVersionV1Alpha1,
		},
		ResponseInstructions: "Return structured JSON only.",
		BatchEndpoint:        "/v1/responses",
		CompletionWindow:     "24h",
		BaseURL:              "https://api.openai.com",
	}
	return config
}

func (config *AppConfig) ApplyDefaults() {
	if config == nil {
		return
	}

	defaults := baseDefaultConfig()

	if strings.TrimSpace(config.SchemaVersion) == "" {
		config.SchemaVersion = defaults.SchemaVersion
	}
	if strings.TrimSpace(config.Environment) == "" {
		config.Environment = defaults.Environment
	}
	if strings.TrimSpace(config.Timezone) == "" {
		config.Timezone = firstNonEmptyString(config.Global.DefaultTimezone, defaults.Timezone)
	}

	applyServerDefaults(&config.Server, defaults.Server)
	applyMongoDefaults(&config.Mongo, defaults.Mongo)
	applyGlobalDefaults(&config.Global, defaults.Global, config.SchemaVersion, config.Timezone)
	applyInvestingDefaults(&config.Investing, defaults.Investing)
	applyTradingDefaults(&config.Trading, defaults.Trading)
	applyUIDefaults(&config.UI, defaults.UI)

	if usesCanonicalAI(config.AI, defaults.AI) {
		applyAIDefaults(&config.AI, defaults.AI, config.Global, config.SchemaVersion)
		config.AsyncAI = config.AI
	} else {
		applyAIDefaults(&config.AsyncAI, defaults.AsyncAI, config.Global, config.SchemaVersion)
		config.AI = config.AsyncAI
	}
}

func (config *AppConfig) Normalize() {
	if config == nil {
		return
	}

	config.ApplyDefaults()
	config.normalizeDerived()
}

func NormalizeAndValidate(config *AppConfig) error {
	if config == nil {
		return ValidationErrors{{Field: "config", Message: "config is required"}}
	}

	config.Normalize()
	return config.Validate()
}

func (config *AppConfig) normalizeDerived() {
	if config == nil {
		return
	}

	if strings.TrimSpace(config.Timezone) == "" {
		config.Timezone = config.Global.DefaultTimezone
	}
	if strings.TrimSpace(config.Global.DefaultTimezone) == "" {
		config.Global.DefaultTimezone = config.Timezone
	}
	if strings.TrimSpace(config.Global.ReviewSchemaVersion) == "" {
		config.Global.ReviewSchemaVersion = config.SchemaVersion
	}
	if strings.TrimSpace(config.Global.PromptSchemaVersion) == "" {
		config.Global.PromptSchemaVersion = firstNonEmptyString(
			config.AsyncAI.Snapshot.PromptVersion,
			config.AsyncAI.PromptVersion,
			config.Global.AIProviders.ReviewPromptVersion,
		)
	}
	config.Investing.CoreSections = investingCoreSections()
	config.Investing.ActionThresholds.syncLegacyFields()
	config.AsyncAI.syncCompatibilityFields()
	config.AI = config.AsyncAI
}

func applyServerDefaults(config *ServerConfig, defaults ServerConfig) {
	if config.Port == 0 {
		config.Port = defaults.Port
	}
	if config.ReadHeaderTimeout == 0 {
		config.ReadHeaderTimeout = defaults.ReadHeaderTimeout
	}
}

func applyMongoDefaults(config *MongoConfig, defaults MongoConfig) {
	if strings.TrimSpace(config.URI) == "" {
		config.URI = defaults.URI
	}
	if strings.TrimSpace(config.Database) == "" {
		config.Database = defaults.Database
	}
	values := []*string{
		&config.Collections.Companies,
		&config.Collections.CompanyReviews,
		&config.Collections.InvestmentTheses,
		&config.Collections.WorkflowRuns,
		&config.Collections.WorkflowStepRuns,
		&config.Collections.ConfigSnapshots,
		&config.Collections.CapitalAllocationRuns,
		&config.Collections.ManualOverrides,
		&config.Collections.CurrentPositions,
		&config.Collections.AIBatchJobs,
		&config.Collections.AIBatchItems,
		&config.Collections.JobReconciliationLogs,
		&config.Collections.ProviderBatchJobs,
		&config.Collections.ProviderBatchIterations,
	}
	defaultValues := []string{
		defaults.Collections.Companies,
		defaults.Collections.CompanyReviews,
		defaults.Collections.InvestmentTheses,
		defaults.Collections.WorkflowRuns,
		defaults.Collections.WorkflowStepRuns,
		defaults.Collections.ConfigSnapshots,
		defaults.Collections.CapitalAllocationRuns,
		defaults.Collections.ManualOverrides,
		defaults.Collections.CurrentPositions,
		defaults.Collections.AIBatchJobs,
		defaults.Collections.AIBatchItems,
		defaults.Collections.JobReconciliationLogs,
		defaults.Collections.ProviderBatchJobs,
		defaults.Collections.ProviderBatchIterations,
	}
	for index := range values {
		if strings.TrimSpace(*values[index]) == "" {
			*values[index] = defaultValues[index]
		}
	}
}

func applyGlobalDefaults(config *GlobalConfig, defaults GlobalConfig, schemaVersion string, timezone string) {
	if strings.TrimSpace(config.DefaultTimezone) == "" {
		config.DefaultTimezone = firstNonEmptyString(timezone, defaults.DefaultTimezone)
	}
	if strings.TrimSpace(config.DefaultCurrency) == "" {
		config.DefaultCurrency = defaults.DefaultCurrency
	}
	if strings.TrimSpace(config.ReviewSchemaVersion) == "" {
		config.ReviewSchemaVersion = firstNonEmptyString(schemaVersion, defaults.ReviewSchemaVersion)
	}
	if strings.TrimSpace(config.PromptSchemaVersion) == "" {
		config.PromptSchemaVersion = firstNonEmptyString(config.AIProviders.ReviewPromptVersion, defaults.PromptSchemaVersion)
	}
	if strings.TrimSpace(config.DataSources.FinancialDataProvider) == "" {
		config.DataSources.FinancialDataProvider = defaults.DataSources.FinancialDataProvider
	}
	if strings.TrimSpace(config.DataSources.PriceDataProvider) == "" {
		config.DataSources.PriceDataProvider = defaults.DataSources.PriceDataProvider
	}
	if strings.TrimSpace(config.DataSources.TextDocumentProvider) == "" {
		config.DataSources.TextDocumentProvider = defaults.DataSources.TextDocumentProvider
	}
	if strings.TrimSpace(config.AIProviders.DefaultProvider) == "" {
		config.AIProviders.DefaultProvider = defaults.AIProviders.DefaultProvider
	}
	if strings.TrimSpace(config.AIProviders.DefaultModel) == "" {
		config.AIProviders.DefaultModel = defaults.AIProviders.DefaultModel
	}
	if strings.TrimSpace(config.AIProviders.ReviewPromptVersion) == "" {
		config.AIProviders.ReviewPromptVersion = defaults.AIProviders.ReviewPromptVersion
	}
	if !config.AllowedBooks.Investing && !config.AllowedBooks.Trading {
		config.AllowedBooks = defaults.AllowedBooks
	}
}

func applyInvestingDefaults(config *InvestingConfig, defaults InvestingConfig) {
	if config.DefaultMode == "" {
		config.DefaultMode = defaults.DefaultMode
	}
	if config.ReviewCadence.DefaultDays == 0 {
		config.ReviewCadence.DefaultDays = defaults.ReviewCadence.DefaultDays
	}
	if config.ReviewCadence.Research == 0 {
		config.ReviewCadence.Research = config.ReviewCadence.DefaultDays
	}
	if config.ReviewCadence.Watch == 0 {
		config.ReviewCadence.Watch = config.ReviewCadence.DefaultDays
	}
	if config.ReviewCadence.BuyReady == 0 {
		config.ReviewCadence.BuyReady = config.ReviewCadence.DefaultDays
	}
	if config.ReviewCadence.Hold == 0 {
		config.ReviewCadence.Hold = config.ReviewCadence.DefaultDays
	}
	if config.ReviewCadence.ExitReview == 0 {
		config.ReviewCadence.ExitReview = defaults.ReviewCadence.ExitReview
	}
	if !config.WatchlistBuckets.Research && !config.WatchlistBuckets.Watch && !config.WatchlistBuckets.BuyReady && !config.WatchlistBuckets.Hold && !config.WatchlistBuckets.ExitReview {
		config.WatchlistBuckets = defaults.WatchlistBuckets
	}
	if config.PositionSizing.MinMeaningfulTargetPct == 0 {
		config.PositionSizing.MinMeaningfulTargetPct = defaults.PositionSizing.MinMeaningfulTargetPct
	}
	if config.PositionSizing.MaxPositionCapPct == 0 {
		config.PositionSizing.MaxPositionCapPct = defaults.PositionSizing.MaxPositionCapPct
	}
	if config.PositionSizing.TranchePolicy.DefaultTrancheCount == 0 {
		config.PositionSizing.TranchePolicy.DefaultTrancheCount = defaults.PositionSizing.TranchePolicy.DefaultTrancheCount
	}
	if strings.TrimSpace(config.PositionSizing.TranchePolicy.DeploymentCadence) == "" {
		config.PositionSizing.TranchePolicy.DeploymentCadence = defaults.PositionSizing.TranchePolicy.DeploymentCadence
	}
	if config.PositionSizing.TranchePolicy.MinimumMonthsBetweenAdds == 0 {
		config.PositionSizing.TranchePolicy.MinimumMonthsBetweenAdds = defaults.PositionSizing.TranchePolicy.MinimumMonthsBetweenAdds
	}
	if config.Allocation.PortfolioTargetSplit.InvestingBookPct == 0 &&
		config.Allocation.PortfolioTargetSplit.TradingBookPct == 0 &&
		config.Allocation.PortfolioTargetSplit.LiquidReservePct == 0 {
		config.Allocation = defaults.Allocation
	}
	if config.SectionWeights == (InvestingSectionWeights{}) {
		config.SectionWeights = defaults.SectionWeights
	}
	if config.SubScoreWeights == (InvestingSubScoreWeights{}) {
		config.SubScoreWeights = defaults.SubScoreWeights
	}
	if config.ActionThresholds.ScoreBands == (InvestingScoreBandThresholds{}) {
		config.ActionThresholds.ScoreBands = defaults.ActionThresholds.ScoreBands
	}
	if config.ActionThresholds.Buy == (InvestingBuyThresholds{}) {
		config.ActionThresholds.Buy = defaults.ActionThresholds.Buy
	}
	if config.ActionThresholds.ChangeEscalation == (InvestingChangeEscalationConfig{}) {
		config.ActionThresholds.ChangeEscalation = defaults.ActionThresholds.ChangeEscalation
	}
	if config.ActionThresholds.HoldMinOverall == 0 {
		config.ActionThresholds.HoldMinOverall = defaults.ActionThresholds.HoldMinOverall
	}
	if config.ActionThresholds.RejectBelowOverall == 0 {
		config.ActionThresholds.RejectBelowOverall = defaults.ActionThresholds.RejectBelowOverall
	}
	if config.ActionThresholds.SellBelowOverall == 0 {
		config.ActionThresholds.SellBelowOverall = defaults.ActionThresholds.SellBelowOverall
	}
	if config.ValuationRules == (InvestingValuationRulesConfig{}) {
		config.ValuationRules = defaults.ValuationRules
	}
	if config.Lookback.YearsLookback == 0 {
		config.Lookback.YearsLookback = defaults.Lookback.YearsLookback
	}
	if config.Lookback.RecentQuarterLookback == 0 {
		config.Lookback.RecentQuarterLookback = defaults.Lookback.RecentQuarterLookback
	}
	if len(config.TextSourcePriority) == 0 {
		config.TextSourcePriority = append([]domain.EvidenceSourceType(nil), defaults.TextSourcePriority...)
	}
	config.CoreSections = investingCoreSections()
	config.ActionThresholds.syncLegacyFields()
}

func applyTradingDefaults(config *TradingConfig, defaults TradingConfig) {
	if !config.Universe.IncludeStocks && !config.Universe.IncludeETFs {
		config.Universe = defaults.Universe
	}
	if config.Style.HoldingStyle == "" {
		config.Style.HoldingStyle = defaults.Style.HoldingStyle
	}
	if config.Style.CoreEdge == "" {
		config.Style.CoreEdge = defaults.Style.CoreEdge
	}
	if !config.Style.AllowBreakoutSetups && !config.Style.AllowPullbackSetups && !config.Style.UseRelativeStrength {
		config.Style.AllowBreakoutSetups = defaults.Style.AllowBreakoutSetups
		config.Style.AllowPullbackSetups = defaults.Style.AllowPullbackSetups
		config.Style.UseRelativeStrength = defaults.Style.UseRelativeStrength
	}
	if !config.RegimeFilter.Enabled && !config.RegimeFilter.UseTrend && !config.RegimeFilter.UseBreadth && !config.RegimeFilter.UseVolatility {
		config.RegimeFilter = defaults.RegimeFilter
	}
	if config.Risk.RiskPerTradePct == 0 {
		config.Risk = defaults.Risk
	}
	if config.CircuitBreaker.KillSwitchDrawdownPct == 0 {
		config.CircuitBreaker = defaults.CircuitBreaker
	}
}

func applyUIDefaults(config *UIConfig, defaults UIConfig) {
	if config.DefaultPageSize == 0 {
		config.DefaultPageSize = defaults.DefaultPageSize
	}
	if config.MaxPageSize == 0 {
		config.MaxPageSize = defaults.MaxPageSize
	}
	if strings.TrimSpace(config.DefaultSortField) == "" {
		config.DefaultSortField = defaults.DefaultSortField
	}
}

func applyAIDefaults(config *AIConfig, defaults AIConfig, global GlobalConfig, schemaVersion string) {
	if strings.TrimSpace(config.ProviderName) == "" {
		config.ProviderName = firstNonEmptyString(config.Provider, global.AIProviders.DefaultProvider, defaults.ProviderName)
	}
	if strings.TrimSpace(config.DefaultModelName) == "" {
		config.DefaultModelName = firstNonEmptyString(config.Model, global.AIProviders.DefaultModel, defaults.DefaultModelName)
	}
	if strings.TrimSpace(config.PromptVersion) == "" {
		config.PromptVersion = firstNonEmptyString(global.AIProviders.ReviewPromptVersion, defaults.PromptVersion)
	}
	if strings.TrimSpace(config.SchemaVersion) == "" {
		config.SchemaVersion = firstNonEmptyString(schemaVersion, defaults.SchemaVersion)
	}
	if !config.EnabledBooks.Investing && !config.EnabledBooks.Trading {
		config.EnabledBooks = defaults.EnabledBooks
	}
	if config.Batch == (AIBatchConfig{}) {
		config.Batch = defaults.Batch
	}
	if config.Worker == (AIWorkerConfig{}) {
		config.Worker = defaults.Worker
	}
	if config.Snapshot == (AISnapshotConfig{}) {
		config.Snapshot = defaults.Snapshot
	}
	if strings.TrimSpace(config.ResponseInstructions) == "" {
		config.ResponseInstructions = defaults.ResponseInstructions
	}
	if strings.TrimSpace(config.BatchEndpoint) == "" {
		config.BatchEndpoint = defaults.BatchEndpoint
	}
	if strings.TrimSpace(config.CompletionWindow) == "" {
		config.CompletionWindow = defaults.CompletionWindow
	}
	if strings.TrimSpace(config.BaseURL) == "" {
		config.BaseURL = defaults.BaseURL
	}
	if strings.TrimSpace(config.Snapshot.PromptVersion) == "" {
		config.Snapshot.PromptVersion = firstNonEmptyString(config.PromptVersion, defaults.Snapshot.PromptVersion)
	}
	if strings.TrimSpace(config.Snapshot.ReviewSchemaVersion) == "" {
		config.Snapshot.ReviewSchemaVersion = firstNonEmptyString(global.ReviewSchemaVersion, defaults.Snapshot.ReviewSchemaVersion)
	}
	if strings.TrimSpace(config.Snapshot.OutputSchemaVersion) == "" {
		config.Snapshot.OutputSchemaVersion = firstNonEmptyString(config.SchemaVersion, defaults.Snapshot.OutputSchemaVersion)
	}
	config.syncCompatibilityFields()
}

func investingCoreSections() []string {
	return []string{
		string(domain.SectionBusinessTraction),
		string(domain.SectionProfitConversion),
		string(domain.SectionCapitalEfficiencyFinancialStrength),
		string(domain.SectionRunwayIndustryPositioning),
		string(domain.SectionManagementGovernance),
	}
}

func firstNonEmptyString(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}

func usesCanonicalAI(ai AIConfig, defaults AIConfig) bool {
	return ai != defaults
}

func (config *InvestingActionThresholds) syncLegacyFields() {
	if config == nil {
		return
	}
	config.ExceptionalMin = config.ScoreBands.ExceptionalMin
	config.StrongMin = config.ScoreBands.StrongMin
	config.AcceptableMin = config.ScoreBands.AcceptableMin
	config.WeakMin = config.ScoreBands.WeakMin
	config.BuyMinOverall = config.Buy.WeightedTotalMin
	config.BuyMinManagement = config.Buy.ManagementGovernanceMin
	config.BuyMinCapitalEfficiency = config.Buy.CapitalEfficiencyFinancialStrengthMin
	config.BuyMinValuation = config.Buy.ValuationEntryMin
	config.CoreStrongThreshold = config.Buy.CoreSectionStrongMin
	config.CoreWeakThreshold = config.Buy.CoreSectionFloor
	config.MaxWeakCoreSectionsForBuy = config.Buy.MaxCoreSectionsBelowFloor
	config.MinStrongCoreSectionsForBuy = config.Buy.MinCoreSectionsAtOrAboveThreshold
	config.ExitReviewTotalDrop = config.ChangeEscalation.TotalScoreDropExitReviewThreshold
	config.ExitReviewCoreDrop = config.ChangeEscalation.CoreSectionDropThreshold
	config.ExitReviewManagementDrop = config.ChangeEscalation.ManagementGovernanceDropThreshold
}

func (config *AIConfig) syncCompatibilityFields() {
	if config == nil {
		return
	}
	config.Provider = firstNonEmptyString(config.ProviderName, config.Provider)
	config.Model = firstNonEmptyString(config.DefaultModelName, config.Model)
	config.PromptVersion = firstNonEmptyString(config.PromptVersion, config.Snapshot.PromptVersion)
}
