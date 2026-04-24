package config

import (
	"encoding/json"
	"fmt"

	"goserver/internal/platform/domain"
)

type SnapshotPayload struct {
	SchemaVersion string                   `json:"schemaVersion"`
	Environment   string                   `json:"environment"`
	Timezone      string                   `json:"timezone"`
	BookType      domain.BookType          `json:"bookType"`
	Mode          string                   `json:"mode"`
	Versions      SnapshotVersions         `json:"versions"`
	Global        SnapshotGlobalConfig     `json:"global"`
	UI            UIConfig                 `json:"ui"`
	AI            SnapshotAIConfig         `json:"ai"`
	Investing     *SnapshotInvestingConfig `json:"investing,omitempty"`
	Trading       *SnapshotTradingConfig   `json:"trading,omitempty"`
}

type SnapshotVersions struct {
	ConfigSchemaVersion string `json:"configSchemaVersion"`
	ReviewSchemaVersion string `json:"reviewSchemaVersion"`
	PromptSchemaVersion string `json:"promptSchemaVersion"`
	OutputSchemaVersion string `json:"outputSchemaVersion"`
}

type SnapshotGlobalConfig struct {
	DefaultTimezone     string             `json:"defaultTimezone"`
	DefaultCurrency     string             `json:"defaultCurrency,omitempty"`
	ReviewSchemaVersion string             `json:"reviewSchemaVersion"`
	PromptSchemaVersion string             `json:"promptSchemaVersion"`
	AllowedBooks        AllowedBooksConfig `json:"allowedBooks"`
	FeatureFlags        FeatureFlags       `json:"featureFlags"`
}

type SnapshotAIConfig struct {
	Enabled              bool               `json:"enabled"`
	ProviderName         string             `json:"providerName"`
	DefaultModelName     string             `json:"defaultModelName"`
	PromptVersion        string             `json:"promptVersion"`
	SchemaVersion        string             `json:"schemaVersion"`
	EnabledBooks         AllowedBooksConfig `json:"enabledBooks"`
	Batch                AIBatchConfig      `json:"batch"`
	Worker               AIWorkerConfig     `json:"worker"`
	Snapshot             AISnapshotConfig   `json:"snapshot"`
	ResponseInstructions string             `json:"responseInstructions,omitempty"`
	BatchEndpoint        string             `json:"batchEndpoint,omitempty"`
	CompletionWindow     string             `json:"completionWindow,omitempty"`
	BaseURL              string             `json:"baseUrl,omitempty"`
}

type SnapshotInvestingConfig struct {
	SelectedMode       domain.InvestingMode            `json:"selectedMode"`
	ReviewCadence      InvestingReviewCadenceConfig    `json:"reviewCadence"`
	WatchlistBuckets   InvestingWatchlistBucketsConfig `json:"watchlistBuckets"`
	PositionSizing     InvestingPositionSizingConfig   `json:"positionSizing"`
	Allocation         InvestingAllocationConfig       `json:"allocation"`
	ValuationRules     InvestingValuationRulesConfig   `json:"valuationRules"`
	SectionWeights     InvestingSectionWeights         `json:"sectionWeights"`
	SubScoreWeights    InvestingSubScoreWeights        `json:"subScoreWeights"`
	ActionThresholds   InvestingActionThresholds       `json:"actionThresholds"`
	HardGates          InvestingHardGateConfig         `json:"hardGates"`
	Lookback           InvestingLookbackConfig         `json:"lookback"`
	TextSourcePriority []domain.EvidenceSourceType     `json:"textSourcePriority"`
	ThesisRules        ThesisRulesConfig               `json:"thesisRules"`
}

type SnapshotTradingConfig struct {
	SelectedMode   string                      `json:"selectedMode"`
	Universe       TradingUniverseConfig       `json:"universe"`
	Style          TradingStyleConfig          `json:"style"`
	RegimeFilter   TradingRegimeFilterConfig   `json:"regimeFilter"`
	Risk           TradingRiskConfig           `json:"risk"`
	CircuitBreaker TradingCircuitBreakerConfig `json:"circuitBreaker"`
}

func (config AppConfig) EffectiveTimezone() string {
	return firstNonEmptyString(config.Timezone, config.Global.DefaultTimezone, defaultTimezone)
}

func (config AppConfig) SupportsBook(bookType domain.BookType) bool {
	return config.Global.AllowedBooks.Enabled(bookType)
}

func (config AppConfig) ToSnapshotPayload(bookType domain.BookType, mode string) (*SnapshotPayload, error) {
	if !domain.IsValidBookType(bookType) {
		return nil, fmt.Errorf("invalid book type %q", bookType)
	}
	if !config.SupportsBook(bookType) {
		return nil, fmt.Errorf("book %q is disabled by config", bookType)
	}

	payload := &SnapshotPayload{
		SchemaVersion: config.SchemaVersion,
		Environment:   config.Environment,
		Timezone:      config.EffectiveTimezone(),
		BookType:      bookType,
		Mode:          mode,
		Versions: SnapshotVersions{
			ConfigSchemaVersion: config.SchemaVersion,
			ReviewSchemaVersion: config.Global.ReviewSchemaVersion,
			PromptSchemaVersion: config.Global.PromptSchemaVersion,
			OutputSchemaVersion: config.AsyncAI.Snapshot.OutputSchemaVersion,
		},
		Global: SnapshotGlobalConfig{
			DefaultTimezone:     config.Global.DefaultTimezone,
			DefaultCurrency:     config.Global.DefaultCurrency,
			ReviewSchemaVersion: config.Global.ReviewSchemaVersion,
			PromptSchemaVersion: config.Global.PromptSchemaVersion,
			AllowedBooks:        config.Global.AllowedBooks,
			FeatureFlags:        config.Global.FeatureFlags,
		},
		UI: config.UI,
		AI: SnapshotAIConfig{
			Enabled:              config.AsyncAI.Enabled,
			ProviderName:         config.AsyncAI.ProviderName,
			DefaultModelName:     config.AsyncAI.DefaultModelName,
			PromptVersion:        config.AsyncAI.PromptVersion,
			SchemaVersion:        config.AsyncAI.SchemaVersion,
			EnabledBooks:         config.AsyncAI.EnabledBooks,
			Batch:                config.AsyncAI.Batch,
			Worker:               config.AsyncAI.Worker,
			Snapshot:             config.AsyncAI.Snapshot,
			ResponseInstructions: config.AsyncAI.ResponseInstructions,
			BatchEndpoint:        config.AsyncAI.BatchEndpoint,
			CompletionWindow:     config.AsyncAI.CompletionWindow,
			BaseURL:              config.AsyncAI.BaseURL,
		},
	}

	switch bookType {
	case domain.BookTypeInvesting:
		selectedMode := domain.InvestingMode(mode)
		if selectedMode == "" {
			selectedMode = config.Investing.DefaultMode
		}
		if !domain.IsValidInvestingMode(selectedMode) {
			return nil, fmt.Errorf("invalid investing mode %q", selectedMode)
		}
		payload.Mode = string(selectedMode)
		payload.Investing = &SnapshotInvestingConfig{
			SelectedMode:       selectedMode,
			ReviewCadence:      config.Investing.ReviewCadence,
			WatchlistBuckets:   config.Investing.WatchlistBuckets,
			PositionSizing:     config.Investing.PositionSizing,
			Allocation:         config.Investing.Allocation,
			ValuationRules:     config.Investing.ValuationRules,
			SectionWeights:     config.Investing.SectionWeights,
			SubScoreWeights:    config.Investing.SubScoreWeights,
			ActionThresholds:   config.Investing.ActionThresholds,
			HardGates:          config.Investing.HardGates,
			Lookback:           config.Investing.Lookback,
			TextSourcePriority: append([]domain.EvidenceSourceType(nil), config.Investing.TextSourcePriority...),
			ThesisRules:        config.Investing.ThesisRules,
		}
	case domain.BookTypeTrading:
		selectedMode := mode
		if selectedMode == "" {
			selectedMode = config.Trading.DefaultMode()
		}
		payload.Mode = selectedMode
		payload.Trading = &SnapshotTradingConfig{
			SelectedMode:   selectedMode,
			Universe:       config.Trading.Universe,
			Style:          config.Trading.Style,
			RegimeFilter:   config.Trading.RegimeFilter,
			Risk:           config.Trading.Risk,
			CircuitBreaker: config.Trading.CircuitBreaker,
		}
	}

	return payload, nil
}

func (config AppConfig) ToSnapshotMap(bookType domain.BookType, mode string) (map[string]any, error) {
	payload, err := config.ToSnapshotPayload(bookType, mode)
	if err != nil {
		return nil, err
	}

	data, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}

	var out map[string]any
	if err := json.Unmarshal(data, &out); err != nil {
		return nil, err
	}
	return out, nil
}
