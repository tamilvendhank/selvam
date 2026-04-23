package service

import (
	"context"
	"fmt"

	platformconfig "goserver/internal/platform/config"
	"goserver/internal/platform/domain"
	"goserver/internal/platform/ports"
)

type DefaultScorecardService struct {
	config platformconfig.AppConfig
}

func NewScorecardService(config platformconfig.AppConfig) *DefaultScorecardService {
	return &DefaultScorecardService{config: config}
}

func (service *DefaultScorecardService) ValidateReview(_ context.Context, review *domain.CompanyReview) error {
	return review.Validate()
}

func (service *DefaultScorecardService) BuildAsyncReviewItem(_ context.Context, company *domain.Company, snapshotID string, mode domain.InvestingMode) (ports.AIReviewBatchItem, error) {
	if company == nil {
		return ports.AIReviewBatchItem{}, fmt.Errorf("company is required")
	}

	prompt := fmt.Sprintf(
		"Prepare a structured investing review input payload for %s (%s on %s). Use quantitative evidence first and textual evidence second. Respect config snapshot %s and mode %s. Return structured JSON only. This is an async foundation prompt placeholder, not the final scoring prompt.",
		company.CompanyName,
		company.Symbol,
		company.Exchange,
		snapshotID,
		mode,
	)

	return ports.AIReviewBatchItem{
		ReferenceID:     company.ID,
		Prompt:          prompt,
		Model:           service.config.AsyncAI.Model,
		ReasoningEffort: "",
		Metadata: map[string]any{
			"symbol":           company.Symbol,
			"exchange":         company.Exchange,
			"bookType":         domain.BookTypeInvesting,
			"promptVersion":    service.config.AsyncAI.PromptVersion,
			"configSnapshotId": snapshotID,
			"mode":             mode,
		},
	}, nil
}
