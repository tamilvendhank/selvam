package config

import "goserver/internal/platform/domain"

type GlobalConfig struct {
	DefaultTimezone     string             `json:"defaultTimezone,omitempty" yaml:"defaultTimezone,omitempty"`
	DefaultCurrency     string             `json:"defaultCurrency,omitempty" yaml:"defaultCurrency,omitempty"`
	ReviewSchemaVersion string             `json:"reviewSchemaVersion,omitempty" yaml:"reviewSchemaVersion,omitempty"`
	PromptSchemaVersion string             `json:"promptSchemaVersion,omitempty" yaml:"promptSchemaVersion,omitempty"`
	AllowedBooks        AllowedBooksConfig `json:"allowedBooks,omitempty" yaml:"allowedBooks,omitempty"`
	DataSources         DataSourceSettings `json:"dataSources,omitempty" yaml:"dataSources,omitempty"`
	AIProviders         AIProviderSettings `json:"aiProviders,omitempty" yaml:"aiProviders,omitempty"`
	FeatureFlags        FeatureFlags       `json:"featureFlags,omitempty" yaml:"featureFlags,omitempty"`
}

type AllowedBooksConfig struct {
	Investing bool `json:"investing" yaml:"investing"`
	Trading   bool `json:"trading" yaml:"trading"`
}

func (books AllowedBooksConfig) Enabled(bookType domain.BookType) bool {
	switch bookType {
	case domain.BookTypeInvesting:
		return books.Investing
	case domain.BookTypeTrading:
		return books.Trading
	default:
		return false
	}
}

type DataSourceSettings struct {
	FinancialDataProvider string `json:"financialDataProvider,omitempty" yaml:"financialDataProvider,omitempty"`
	PriceDataProvider     string `json:"priceDataProvider,omitempty" yaml:"priceDataProvider,omitempty"`
	TextDocumentProvider  string `json:"textDocumentProvider,omitempty" yaml:"textDocumentProvider,omitempty"`
}

type AIProviderSettings struct {
	DefaultProvider     string `json:"defaultProvider,omitempty" yaml:"defaultProvider,omitempty"`
	DefaultModel        string `json:"defaultModel,omitempty" yaml:"defaultModel,omitempty"`
	ReviewPromptVersion string `json:"reviewPromptVersion,omitempty" yaml:"reviewPromptVersion,omitempty"`
	BatchEnabled        bool   `json:"batchEnabled" yaml:"batchEnabled"`
}

type FeatureFlags struct {
	EnableAsyncAIReview             bool `json:"enableAsyncAiReview" yaml:"enableAsyncAiReview"`
	EnableCurrentPositionProjection bool `json:"enableCurrentPositionProjection" yaml:"enableCurrentPositionProjection"`
	EnableTradingWorkflow           bool `json:"enableTradingWorkflow" yaml:"enableTradingWorkflow"`
}
