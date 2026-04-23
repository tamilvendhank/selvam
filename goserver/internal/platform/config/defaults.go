package config

import (
	"time"

	"goserver/internal/platform/domain"
)

func Default() AppConfig {
	return AppConfig{
		SchemaVersion: domain.SchemaVersionV1Alpha1,
		Environment:   "development",
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
			SchemaVersion:   domain.SchemaVersionV1Alpha1,
			DefaultTimezone: "Asia/Kolkata",
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
			DefaultMode: string(domain.InvestingModeBalanced),
			SectionWeights: []NamedWeight{
				{Name: string(domain.SectionInvestability), Weight: 5},
				{Name: string(domain.SectionBusinessTraction), Weight: 15},
				{Name: string(domain.SectionProfitConversion), Weight: 13},
				{Name: string(domain.SectionCapitalEfficiencyFinancialStrength), Weight: 16},
				{Name: string(domain.SectionStructuralSectorAttractiveness), Weight: 6},
				{Name: string(domain.SectionRunwayIndustryPositioning), Weight: 13},
				{Name: string(domain.SectionManagementGovernance), Weight: 16},
				{Name: string(domain.SectionMarketConfirmation), Weight: 6},
				{Name: string(domain.SectionValuationEntryAttractiveness), Weight: 10},
			},
			SubScoreWeights: []SectionSubScoreWeights{
				{SectionName: string(domain.SectionInvestability), SubScores: []NamedWeight{{Name: "Liquidity", Weight: 40}, {Name: "Data quality / completeness", Weight: 25}, {Name: "Basic investability suitability", Weight: 20}, {Name: "Listing / operating history sufficiency", Weight: 15}}},
				{SectionName: string(domain.SectionBusinessTraction), SubScores: []NamedWeight{{Name: "Revenue growth strength", Weight: 30}, {Name: "Revenue growth consistency", Weight: 25}, {Name: "Recent 12-quarter acceleration / deterioration", Weight: 25}, {Name: "Evidence of expanding demand", Weight: 20}}},
				{SectionName: string(domain.SectionProfitConversion), SubScores: []NamedWeight{{Name: "Operating margin quality / trend", Weight: 30}, {Name: "Profit growth strength", Weight: 25}, {Name: "Cash conversion quality", Weight: 30}, {Name: "Recent operating leverage / margin direction", Weight: 15}}},
				{SectionName: string(domain.SectionCapitalEfficiencyFinancialStrength), SubScores: []NamedWeight{{Name: "ROCE / ROIC quality", Weight: 35}, {Name: "Balance-sheet strength", Weight: 30}, {Name: "Working-capital efficiency", Weight: 20}, {Name: "Dilution / capital-allocation discipline", Weight: 15}}},
				{SectionName: string(domain.SectionStructuralSectorAttractiveness), SubScores: []NamedWeight{{Name: "Demand tailwind strength", Weight: 35}, {Name: "Industry economics quality", Weight: 30}, {Name: "Policy / formalization support", Weight: 20}, {Name: "Cyclicality risk", Weight: 15}}},
				{SectionName: string(domain.SectionRunwayIndustryPositioning), SubScores: []NamedWeight{{Name: "Market opportunity size", Weight: 30}, {Name: "Share-gain potential", Weight: 30}, {Name: "Expansion optionality", Weight: 20}, {Name: "Competitive positioning strength", Weight: 20}}},
				{SectionName: string(domain.SectionManagementGovernance), SubScores: []NamedWeight{{Name: "Capital allocation quality", Weight: 30}, {Name: "Execution consistency", Weight: 30}, {Name: "Shareholder alignment / trustworthiness", Weight: 25}, {Name: "Disclosure quality", Weight: 15}}},
				{SectionName: string(domain.SectionMarketConfirmation), SubScores: []NamedWeight{{Name: "Relative strength", Weight: 35}, {Name: "Trend quality", Weight: 30}, {Name: "Drawdown / resilience behavior", Weight: 20}, {Name: "Reaction to results / news", Weight: 15}}},
				{SectionName: string(domain.SectionValuationEntryAttractiveness), SubScores: []NamedWeight{{Name: "Historical valuation attractiveness", Weight: 35}, {Name: "Valuation support vs current quality", Weight: 30}, {Name: "Overvaluation risk", Weight: 20}, {Name: "Entry timing suitability", Weight: 15}}},
			},
			HardGateRules: []HardGateRule{
				{Name: "governance_red_flag_absence", Description: "Governance red-flag absence is mandatory.", FailureAction: string(domain.ActionReject)},
			},
			ActionThresholds: InvestingActionThresholds{
				ExceptionalMin:              8.5,
				StrongMin:                   7.5,
				AcceptableMin:               6.5,
				WeakMin:                     5.5,
				BuyMinOverall:               7.5,
				HoldMinOverall:              7.0,
				RejectBelowOverall:          6.0,
				SellBelowOverall:            5.5,
				BuyMinManagement:            7.0,
				BuyMinCapitalEfficiency:     7.0,
				BuyMinValuation:             6.0,
				CoreStrongThreshold:         7.0,
				CoreWeakThreshold:           6.5,
				MaxWeakCoreSectionsForBuy:   1,
				MinStrongCoreSectionsForBuy: 3,
				ExitReviewTotalDrop:         1.0,
				ExitReviewCoreDrop:          1.5,
				ExitReviewManagementDrop:    1.0,
			},
			ValuationRules: ValuationRules{
				UseOwnHistoryOnly:          true,
				ExtremeOvervaluationAction: string(domain.SectionActionCapCannotBuy),
				Metrics:                    []string{"P/E", "EV/EBITDA", "P/B", "Price/Sales", "Free Cash Flow Yield"},
			},
			ReviewCadenceByBucket: []BucketCadence{
				{Bucket: string(domain.WatchlistBucketResearch), ReviewEveryDays: 30},
				{Bucket: string(domain.WatchlistBucketWatch), ReviewEveryDays: 30},
				{Bucket: string(domain.WatchlistBucketBuyReady), ReviewEveryDays: 30},
				{Bucket: string(domain.WatchlistBucketHold), ReviewEveryDays: 30},
				{Bucket: string(domain.WatchlistBucketExitReview), ReviewEveryDays: 7},
			},
			PositionSizing: PositionSizingRules{
				DynamicSizing:          true,
				MinMeaningfulTargetPct: 3,
				MaxPositionCapPct:      10,
			},
			Allocation: AllocationRules{
				PortfolioTargetSplit: PortfolioSplit{
					InvestingBookPct: 70,
					TradingBookPct:   20,
					LiquidReservePct: 10,
				},
				DefaultTrancheCount: 3,
				AllowPartialTrim:    true,
			},
			ThesisRules: ThesisRules{
				RequireWrittenThesisForBuy: true,
				SellOnThesisBreak:          true,
			},
			WatchlistBuckets: []string{
				string(domain.WatchlistBucketResearch),
				string(domain.WatchlistBucketWatch),
				string(domain.WatchlistBucketBuyReady),
				string(domain.WatchlistBucketHold),
				string(domain.WatchlistBucketExitReview),
			},
			CoreSections: []string{
				string(domain.SectionBusinessTraction),
				string(domain.SectionProfitConversion),
				string(domain.SectionCapitalEfficiencyFinancialStrength),
				string(domain.SectionRunwayIndustryPositioning),
				string(domain.SectionManagementGovernance),
			},
		},
		Trading: TradingConfig{
			Universe: UniverseRules{
				RequireLiquidityOnly: true,
				AllowedInstruments:   []string{"equity", "etf"},
			},
			RegimeFilter: RegimeFilterConfig{
				UseTrend:      true,
				UseBreadth:    true,
				UseVolatility: true,
			},
			RiskPerTradePct:        1,
			MaxConcurrentPositions: 6,
			StopStyle:              "volatility_based",
			TrailingStop: TrailingStopSettings{
				Mode:        "atr_trailing",
				ATRMultiple: 2.5,
			},
			DrawdownKillSwitch: DrawdownKillSwitchConfig{
				StopOpeningNewTradesDrawdownPct: 10,
				CooldownDays:                    28,
			},
		},
		UI: UIConfig{
			DefaultPageSize: 25,
			MaxPageSize:     100,
			DefaultSort:     "-updatedAt",
			FeatureToggles: map[string]bool{
				"companyHistorySummary": true,
				"reviewDiff":            true,
				"configInspection":      true,
			},
			ReviewDetailRenderingHints: []string{
				"sections",
				"decisionAction",
				"changeLog",
				"positionSnapshot",
			},
		},
		AsyncAI: AsyncAIConfig{
			Enabled:              true,
			Provider:             "openai-batch",
			Model:                "gpt-5.4-mini",
			PromptVersion:        "investing-review-v1",
			ResponseInstructions: "Return structured JSON only.",
			BatchEndpoint:        "/v1/responses",
			CompletionWindow:     "24h",
			BaseURL:              "https://api.openai.com",
			Worker: AsyncAIWorkerConfig{
				Enabled:              true,
				RefreshInterval:      15 * time.Second,
				MinBatchRefreshAge:   30 * time.Second,
				FollowUpClaimTimeout: 2 * time.Minute,
				MaxBatchesPerPass:    20,
			},
		},
	}
}
