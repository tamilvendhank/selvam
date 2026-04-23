package service

import (
	"context"

	"goserver/internal/platform/domain"
	"goserver/internal/platform/ports"
)

type DefaultProjectionService struct {
	repository ports.PositionRepository
}

func NewProjectionService(repository ports.PositionRepository) *DefaultProjectionService {
	return &DefaultProjectionService{repository: repository}
}

func (service *DefaultProjectionService) ListPositions(ctx context.Context, filter ports.PositionListFilter) ([]*domain.CurrentPosition, error) {
	return service.repository.List(ctx, filter)
}

func (service *DefaultProjectionService) GetPositionByCompanyAndBook(ctx context.Context, companyID string, bookType domain.BookType) (*domain.CurrentPosition, error) {
	position, err := service.repository.GetByCompanyAndBook(ctx, companyID, bookType)
	if err != nil {
		return nil, err
	}
	if position == nil {
		return nil, ErrNotFound
	}

	return position, nil
}

func (service *DefaultProjectionService) UpsertPosition(ctx context.Context, position *domain.CurrentPosition) (*domain.CurrentPosition, error) {
	return service.repository.Upsert(ctx, position)
}
