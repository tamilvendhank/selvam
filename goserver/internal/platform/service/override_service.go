package service

import (
	"context"
	"fmt"

	"goserver/internal/platform/domain"
	"goserver/internal/platform/ports"
)

type DefaultOverrideService struct {
	repository       ports.ManualOverrideRepository
	reviewRepository ports.CompanyReviewRepository
}

func NewOverrideService(repository ports.ManualOverrideRepository, reviewRepository ports.CompanyReviewRepository) *DefaultOverrideService {
	return &DefaultOverrideService{
		repository:       repository,
		reviewRepository: reviewRepository,
	}
}

func (service *DefaultOverrideService) CreateOverride(ctx context.Context, override *domain.ManualOverride) (*domain.ManualOverride, error) {
	if override == nil {
		return nil, fmt.Errorf("manual override is required")
	}

	review, err := service.reviewRepository.GetByID(ctx, override.ReviewID)
	if err != nil {
		return nil, err
	}
	if review == nil {
		return nil, fmt.Errorf("review %s not found", override.ReviewID)
	}
	if review.FinalActionAfterReview != override.OriginalAction {
		return nil, fmt.Errorf("manual override original action %q does not match review action %q", override.OriginalAction, review.FinalActionAfterReview)
	}

	return service.repository.Create(ctx, override)
}

func (service *DefaultOverrideService) ListOverrides(ctx context.Context, filter ports.ManualOverrideListFilter) ([]*domain.ManualOverride, error) {
	return service.repository.List(ctx, filter)
}

func (service *DefaultOverrideService) GetOverride(ctx context.Context, id string) (*domain.ManualOverride, error) {
	override, err := service.repository.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if override == nil {
		return nil, ErrNotFound
	}

	return override, nil
}
