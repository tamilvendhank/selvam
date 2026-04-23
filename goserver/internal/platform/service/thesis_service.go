package service

import (
	"context"
	"fmt"

	"goserver/internal/platform/domain"
	"goserver/internal/platform/ports"
)

type DefaultThesisService struct {
	repository   ports.ThesisRepository
	timeProvider ports.TimeProvider
}

func NewThesisService(repository ports.ThesisRepository, timeProvider ports.TimeProvider) *DefaultThesisService {
	return &DefaultThesisService{
		repository:   repository,
		timeProvider: resolveTimeProvider(timeProvider),
	}
}

func (service *DefaultThesisService) UpsertThesis(ctx context.Context, thesis *domain.InvestmentThesis) (*domain.InvestmentThesis, error) {
	if thesis == nil {
		return nil, fmt.Errorf("thesis is required")
	}
	if thesis.ID == "" {
		return service.repository.Create(ctx, thesis)
	}

	return service.repository.Update(ctx, thesis)
}

func (service *DefaultThesisService) BuildOrUpdateFromReview(ctx context.Context, review *domain.CompanyReview) (*domain.InvestmentThesis, error) {
	if review == nil {
		return nil, fmt.Errorf("review is required")
	}

	existing, err := service.repository.GetActiveByCompanyID(ctx, review.CompanyID)
	if err != nil {
		return nil, err
	}

	now := service.timeProvider.Now()
	if existing == nil {
		existing = &domain.InvestmentThesis{
			CompanyID:                  review.CompanyID,
			ThesisStatus:               domain.ThesisStatusActive,
			ThesisVersion:              1,
			CreatedFromReviewID:        review.ID,
			LastUpdatedFromReviewID:    review.ID,
			ThesisSummary:              firstNonEmpty(review.ActionRationaleSummary, "Initial thesis scaffold from review."),
			WhyThisBusinessCanCompound: firstNonEmpty(review.WhatChangedSummary, "Review-generated thesis placeholder."),
			DesiredHoldingPeriod:       "3-10 years",
			ConfidenceLevel:            review.ConfidenceScore,
			ThesisHealthScore:          review.WeightedTotalScore,
			CurrentPositionRole:        domain.PositionRoleStarter,
			SchemaVersion:              review.SchemaVersion,
			CreatedAt:                  now,
			UpdatedAt:                  now,
		}
		return service.repository.Create(ctx, existing)
	}

	existing.LastUpdatedFromReviewID = review.ID
	existing.ThesisVersion++
	existing.ThesisChangeSummary = review.WhatChangedSummary
	existing.ConfidenceLevel = review.ConfidenceScore
	existing.ThesisHealthScore = review.WeightedTotalScore
	existing.UpdatedAt = now
	if review.FinalActionAfterReview == domain.ActionSell {
		existing.ThesisStatus = domain.ThesisStatusBroken
	}

	return service.repository.Update(ctx, existing)
}

func (service *DefaultThesisService) GetActiveThesis(ctx context.Context, companyID string) (*domain.InvestmentThesis, error) {
	thesis, err := service.repository.GetActiveByCompanyID(ctx, companyID)
	if err != nil {
		return nil, err
	}
	if thesis == nil {
		return nil, ErrNotFound
	}

	return thesis, nil
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if value != "" {
			return value
		}
	}

	return ""
}
