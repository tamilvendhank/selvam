package config

type TradingHoldingStyle string

const (
	TradingHoldingStylePositionTrades TradingHoldingStyle = "position_trades"
)

type TradingCoreEdge string

const (
	TradingCoreEdgeTrendMomentum TradingCoreEdge = "trend_momentum"
)

type TradingStopMethod string

const (
	TradingStopMethodVolatility TradingStopMethod = "volatility"
)

type TradingProfitManagement string

const (
	TradingProfitManagementTrailingStop TradingProfitManagement = "trailing_stop"
)

type TradingConfig struct {
	Universe       TradingUniverseConfig       `json:"universe" yaml:"universe"`
	Style          TradingStyleConfig          `json:"style" yaml:"style"`
	RegimeFilter   TradingRegimeFilterConfig   `json:"regimeFilter" yaml:"regimeFilter"`
	Risk           TradingRiskConfig           `json:"risk" yaml:"risk"`
	CircuitBreaker TradingCircuitBreakerConfig `json:"circuitBreaker" yaml:"circuitBreaker"`
}

func (config TradingConfig) DefaultMode() string {
	if config.Style.CoreEdge != "" {
		return string(config.Style.CoreEdge)
	}
	return string(TradingCoreEdgeTrendMomentum)
}

type TradingUniverseConfig struct {
	IncludeStocks        bool `json:"includeStocks" yaml:"includeStocks"`
	IncludeETFs          bool `json:"includeETFs" yaml:"includeETFs"`
	RequireHighLiquidity bool `json:"requireHighLiquidity" yaml:"requireHighLiquidity"`
}

type TradingStyleConfig struct {
	HoldingStyle        TradingHoldingStyle `json:"holdingStyle" yaml:"holdingStyle"`
	AllowBreakoutSetups bool                `json:"allowBreakoutSetups" yaml:"allowBreakoutSetups"`
	AllowPullbackSetups bool                `json:"allowPullbackSetups" yaml:"allowPullbackSetups"`
	CoreEdge            TradingCoreEdge     `json:"coreEdge" yaml:"coreEdge"`
	UseRelativeStrength bool                `json:"useRelativeStrength" yaml:"useRelativeStrength"`
}

type TradingRegimeFilterConfig struct {
	Enabled       bool `json:"enabled" yaml:"enabled"`
	UseTrend      bool `json:"useTrend" yaml:"useTrend"`
	UseBreadth    bool `json:"useBreadth" yaml:"useBreadth"`
	UseVolatility bool `json:"useVolatility" yaml:"useVolatility"`
}

type TradingRiskConfig struct {
	RiskPerTradePct        float64                 `json:"riskPerTradePct" yaml:"riskPerTradePct"`
	MaxConcurrentPositions int                     `json:"maxConcurrentPositions" yaml:"maxConcurrentPositions"`
	HardStopRequired       bool                    `json:"hardStopRequired" yaml:"hardStopRequired"`
	StopMethod             TradingStopMethod       `json:"stopMethod" yaml:"stopMethod"`
	ProfitManagement       TradingProfitManagement `json:"profitManagement" yaml:"profitManagement"`
}

type TradingCircuitBreakerConfig struct {
	KillSwitchDrawdownPct float64 `json:"killSwitchDrawdownPct" yaml:"killSwitchDrawdownPct"`
	CooldownWeeks         int     `json:"cooldownWeeks" yaml:"cooldownWeeks"`
}
