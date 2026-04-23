package config

import "time"

type AppConfig struct {
	SchemaVersion string          `json:"schemaVersion" yaml:"schemaVersion"`
	Environment   string          `json:"environment" yaml:"environment"`
	Server        ServerConfig    `json:"server" yaml:"server"`
	Mongo         MongoConfig     `json:"mongo" yaml:"mongo"`
	Global        GlobalConfig    `json:"global" yaml:"global"`
	Investing     InvestingConfig `json:"investing" yaml:"investing"`
	Trading       TradingConfig   `json:"trading" yaml:"trading"`
	UI            UIConfig        `json:"ui" yaml:"ui"`
	AsyncAI       AsyncAIConfig   `json:"asyncAi" yaml:"asyncAi"`
}

type ServerConfig struct {
	Port              int           `json:"port" yaml:"port"`
	FrontendRootDir   string        `json:"frontendRootDir,omitempty" yaml:"frontendRootDir,omitempty"`
	ReadHeaderTimeout time.Duration `json:"readHeaderTimeout" yaml:"readHeaderTimeout"`
}

type MongoConfig struct {
	URI         string           `json:"uri" yaml:"uri"`
	Database    string           `json:"database" yaml:"database"`
	Collections CollectionConfig `json:"collections" yaml:"collections"`
}

type CollectionConfig struct {
	Companies             string `json:"companies" yaml:"companies"`
	CompanyReviews        string `json:"companyReviews" yaml:"companyReviews"`
	InvestmentTheses      string `json:"investmentTheses" yaml:"investmentTheses"`
	WorkflowRuns          string `json:"workflowRuns" yaml:"workflowRuns"`
	ConfigSnapshots       string `json:"configSnapshots" yaml:"configSnapshots"`
	CapitalAllocationRuns string `json:"capitalAllocationRuns" yaml:"capitalAllocationRuns"`
	ManualOverrides       string `json:"manualOverrides" yaml:"manualOverrides"`
	CurrentPositions      string `json:"currentPositions" yaml:"currentPositions"`
	AIBatchJobs           string `json:"aiBatchJobs" yaml:"aiBatchJobs"`
	AIBatchIterations     string `json:"aiBatchIterations" yaml:"aiBatchIterations"`
}

type GlobalConfig struct {
	SchemaVersion   string             `json:"schemaVersion" yaml:"schemaVersion"`
	DefaultTimezone string             `json:"defaultTimezone" yaml:"defaultTimezone"`
	DataSources     DataSourceSettings `json:"dataSources" yaml:"dataSources"`
	AIProviders     AIProviderSettings `json:"aiProviders" yaml:"aiProviders"`
	FeatureFlags    FeatureFlags       `json:"featureFlags" yaml:"featureFlags"`
}

type DataSourceSettings struct {
	FinancialDataProvider string `json:"financialDataProvider" yaml:"financialDataProvider"`
	PriceDataProvider     string `json:"priceDataProvider" yaml:"priceDataProvider"`
	TextDocumentProvider  string `json:"textDocumentProvider" yaml:"textDocumentProvider"`
}

type AIProviderSettings struct {
	DefaultProvider     string `json:"defaultProvider" yaml:"defaultProvider"`
	DefaultModel        string `json:"defaultModel" yaml:"defaultModel"`
	ReviewPromptVersion string `json:"reviewPromptVersion" yaml:"reviewPromptVersion"`
	BatchEnabled        bool   `json:"batchEnabled" yaml:"batchEnabled"`
}

type FeatureFlags struct {
	EnableAsyncAIReview             bool `json:"enableAsyncAiReview" yaml:"enableAsyncAiReview"`
	EnableCurrentPositionProjection bool `json:"enableCurrentPositionProjection" yaml:"enableCurrentPositionProjection"`
	EnableTradingWorkflow           bool `json:"enableTradingWorkflow" yaml:"enableTradingWorkflow"`
}

type NamedWeight struct {
	Name   string  `json:"name" yaml:"name"`
	Weight float64 `json:"weight" yaml:"weight"`
}

type SectionSubScoreWeights struct {
	SectionName string        `json:"sectionName" yaml:"sectionName"`
	SubScores   []NamedWeight `json:"subScores" yaml:"subScores"`
}

type HardGateRule struct {
	Name          string `json:"name" yaml:"name"`
	Description   string `json:"description" yaml:"description"`
	FailureAction string `json:"failureAction" yaml:"failureAction"`
}

type InvestingActionThresholds struct {
	ExceptionalMin              float64 `json:"exceptionalMin" yaml:"exceptionalMin"`
	StrongMin                   float64 `json:"strongMin" yaml:"strongMin"`
	AcceptableMin               float64 `json:"acceptableMin" yaml:"acceptableMin"`
	WeakMin                     float64 `json:"weakMin" yaml:"weakMin"`
	BuyMinOverall               float64 `json:"buyMinOverall" yaml:"buyMinOverall"`
	HoldMinOverall              float64 `json:"holdMinOverall" yaml:"holdMinOverall"`
	RejectBelowOverall          float64 `json:"rejectBelowOverall" yaml:"rejectBelowOverall"`
	SellBelowOverall            float64 `json:"sellBelowOverall" yaml:"sellBelowOverall"`
	BuyMinManagement            float64 `json:"buyMinManagement" yaml:"buyMinManagement"`
	BuyMinCapitalEfficiency     float64 `json:"buyMinCapitalEfficiency" yaml:"buyMinCapitalEfficiency"`
	BuyMinValuation             float64 `json:"buyMinValuation" yaml:"buyMinValuation"`
	CoreStrongThreshold         float64 `json:"coreStrongThreshold" yaml:"coreStrongThreshold"`
	CoreWeakThreshold           float64 `json:"coreWeakThreshold" yaml:"coreWeakThreshold"`
	MaxWeakCoreSectionsForBuy   int     `json:"maxWeakCoreSectionsForBuy" yaml:"maxWeakCoreSectionsForBuy"`
	MinStrongCoreSectionsForBuy int     `json:"minStrongCoreSectionsForBuy" yaml:"minStrongCoreSectionsForBuy"`
	ExitReviewTotalDrop         float64 `json:"exitReviewTotalDrop" yaml:"exitReviewTotalDrop"`
	ExitReviewCoreDrop          float64 `json:"exitReviewCoreDrop" yaml:"exitReviewCoreDrop"`
	ExitReviewManagementDrop    float64 `json:"exitReviewManagementDrop" yaml:"exitReviewManagementDrop"`
}

type ValuationRules struct {
	UseOwnHistoryOnly          bool     `json:"useOwnHistoryOnly" yaml:"useOwnHistoryOnly"`
	ExtremeOvervaluationAction string   `json:"extremeOvervaluationAction" yaml:"extremeOvervaluationAction"`
	Metrics                    []string `json:"metrics" yaml:"metrics"`
}

type BucketCadence struct {
	Bucket          string `json:"bucket" yaml:"bucket"`
	ReviewEveryDays int    `json:"reviewEveryDays" yaml:"reviewEveryDays"`
}

type PositionSizingRules struct {
	DynamicSizing          bool    `json:"dynamicSizing" yaml:"dynamicSizing"`
	MinMeaningfulTargetPct float64 `json:"minMeaningfulTargetPct" yaml:"minMeaningfulTargetPct"`
	MaxPositionCapPct      float64 `json:"maxPositionCapPct" yaml:"maxPositionCapPct"`
}

type PortfolioSplit struct {
	InvestingBookPct float64 `json:"investingBookPct" yaml:"investingBookPct"`
	TradingBookPct   float64 `json:"tradingBookPct" yaml:"tradingBookPct"`
	LiquidReservePct float64 `json:"liquidReservePct" yaml:"liquidReservePct"`
}

type AllocationRules struct {
	PortfolioTargetSplit PortfolioSplit `json:"portfolioTargetSplit" yaml:"portfolioTargetSplit"`
	DefaultTrancheCount  int            `json:"defaultTrancheCount" yaml:"defaultTrancheCount"`
	AllowPartialTrim     bool           `json:"allowPartialTrim" yaml:"allowPartialTrim"`
}

type ThesisRules struct {
	RequireWrittenThesisForBuy bool `json:"requireWrittenThesisForBuy" yaml:"requireWrittenThesisForBuy"`
	SellOnThesisBreak          bool `json:"sellOnThesisBreak" yaml:"sellOnThesisBreak"`
}

type InvestingConfig struct {
	DefaultMode           string                    `json:"defaultMode" yaml:"defaultMode"`
	SectionWeights        []NamedWeight             `json:"sectionWeights" yaml:"sectionWeights"`
	SubScoreWeights       []SectionSubScoreWeights  `json:"subScoreWeights" yaml:"subScoreWeights"`
	HardGateRules         []HardGateRule            `json:"hardGateRules" yaml:"hardGateRules"`
	ActionThresholds      InvestingActionThresholds `json:"actionThresholds" yaml:"actionThresholds"`
	ValuationRules        ValuationRules            `json:"valuationRules" yaml:"valuationRules"`
	ReviewCadenceByBucket []BucketCadence           `json:"reviewCadenceByBucket" yaml:"reviewCadenceByBucket"`
	PositionSizing        PositionSizingRules       `json:"positionSizing" yaml:"positionSizing"`
	Allocation            AllocationRules           `json:"allocation" yaml:"allocation"`
	ThesisRules           ThesisRules               `json:"thesisRules" yaml:"thesisRules"`
	WatchlistBuckets      []string                  `json:"watchlistBuckets" yaml:"watchlistBuckets"`
	CoreSections          []string                  `json:"coreSections" yaml:"coreSections"`
}

type UniverseRules struct {
	RequireLiquidityOnly bool     `json:"requireLiquidityOnly" yaml:"requireLiquidityOnly"`
	AllowedInstruments   []string `json:"allowedInstruments" yaml:"allowedInstruments"`
}

type RegimeFilterConfig struct {
	UseTrend      bool `json:"useTrend" yaml:"useTrend"`
	UseBreadth    bool `json:"useBreadth" yaml:"useBreadth"`
	UseVolatility bool `json:"useVolatility" yaml:"useVolatility"`
}

type TrailingStopSettings struct {
	Mode        string  `json:"mode" yaml:"mode"`
	ATRMultiple float64 `json:"atrMultiple" yaml:"atrMultiple"`
}

type DrawdownKillSwitchConfig struct {
	StopOpeningNewTradesDrawdownPct float64 `json:"stopOpeningNewTradesDrawdownPct" yaml:"stopOpeningNewTradesDrawdownPct"`
	CooldownDays                    int     `json:"cooldownDays" yaml:"cooldownDays"`
}

type TradingConfig struct {
	Universe               UniverseRules            `json:"universe" yaml:"universe"`
	RegimeFilter           RegimeFilterConfig       `json:"regimeFilter" yaml:"regimeFilter"`
	RiskPerTradePct        float64                  `json:"riskPerTradePct" yaml:"riskPerTradePct"`
	MaxConcurrentPositions int                      `json:"maxConcurrentPositions" yaml:"maxConcurrentPositions"`
	StopStyle              string                   `json:"stopStyle" yaml:"stopStyle"`
	TrailingStop           TrailingStopSettings     `json:"trailingStop" yaml:"trailingStop"`
	DrawdownKillSwitch     DrawdownKillSwitchConfig `json:"drawdownKillSwitch" yaml:"drawdownKillSwitch"`
}

type UIConfig struct {
	DefaultPageSize            int             `json:"defaultPageSize" yaml:"defaultPageSize"`
	MaxPageSize                int             `json:"maxPageSize" yaml:"maxPageSize"`
	DefaultSort                string          `json:"defaultSort" yaml:"defaultSort"`
	FeatureToggles             map[string]bool `json:"featureToggles" yaml:"featureToggles"`
	ReviewDetailRenderingHints []string        `json:"reviewDetailRenderingHints" yaml:"reviewDetailRenderingHints"`
}

type AsyncAIConfig struct {
	Enabled              bool                `json:"enabled" yaml:"enabled"`
	Provider             string              `json:"provider" yaml:"provider"`
	Model                string              `json:"model" yaml:"model"`
	PromptVersion        string              `json:"promptVersion" yaml:"promptVersion"`
	ResponseInstructions string              `json:"responseInstructions" yaml:"responseInstructions"`
	BatchEndpoint        string              `json:"batchEndpoint" yaml:"batchEndpoint"`
	CompletionWindow     string              `json:"completionWindow" yaml:"completionWindow"`
	BaseURL              string              `json:"baseUrl" yaml:"baseUrl"`
	APIKey               string              `json:"-" yaml:"apiKey"`
	Worker               AsyncAIWorkerConfig `json:"worker" yaml:"worker"`
}

type AsyncAIWorkerConfig struct {
	Enabled              bool          `json:"enabled" yaml:"enabled"`
	RefreshInterval      time.Duration `json:"refreshInterval" yaml:"refreshInterval"`
	MinBatchRefreshAge   time.Duration `json:"minBatchRefreshAge" yaml:"minBatchRefreshAge"`
	FollowUpClaimTimeout time.Duration `json:"followUpClaimTimeout" yaml:"followUpClaimTimeout"`
	MaxBatchesPerPass    int           `json:"maxBatchesPerPass" yaml:"maxBatchesPerPass"`
}
