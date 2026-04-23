package provider

import (
	"context"
	"time"
)

type NoopFinancialDataProvider struct{}

func (NoopFinancialDataProvider) LoadFinancialSnapshot(_ context.Context, symbol string, asOf time.Time) (map[string]any, error) {
	return map[string]any{
		"symbol": symbol,
		"asOf":   asOf,
		"status": "placeholder",
	}, nil
}

type NoopPriceDataProvider struct{}

func (NoopPriceDataProvider) LoadPriceSnapshot(_ context.Context, symbol string, asOf time.Time) (map[string]any, error) {
	return map[string]any{
		"symbol": symbol,
		"asOf":   asOf,
		"status": "placeholder",
	}, nil
}

type NoopTextDocumentProvider struct{}

func (NoopTextDocumentProvider) LoadDocumentMetadata(_ context.Context, symbol string) ([]map[string]any, error) {
	return []map[string]any{
		{
			"symbol": symbol,
			"status": "placeholder",
		},
	}, nil
}
