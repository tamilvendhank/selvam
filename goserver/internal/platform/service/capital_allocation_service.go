package service

import (
	"context"
	"fmt"

	"goserver/internal/platform/domain"
	"goserver/internal/platform/ports"
)

type DefaultCapitalAllocationService struct {
	repository ports.CapitalAllocationRepository
}

func NewCapitalAllocationService(repository ports.CapitalAllocationRepository) *DefaultCapitalAllocationService {
	return &DefaultCapitalAllocationService{repository: repository}
}

func (service *DefaultCapitalAllocationService) CreateRun(ctx context.Context, run *domain.CapitalAllocationRun) (*domain.CapitalAllocationRun, error) {
	if run == nil {
		return nil, fmt.Errorf("capital allocation run is required")
	}
	return service.repository.Create(ctx, run)
}

func (service *DefaultCapitalAllocationService) ListRuns(ctx context.Context, filter ports.CapitalAllocationListFilter) ([]*domain.CapitalAllocationRun, error) {
	return service.repository.List(ctx, filter)
}

func (service *DefaultCapitalAllocationService) GetRun(ctx context.Context, id string) (*domain.CapitalAllocationRun, error) {
	run, err := service.repository.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if run == nil {
		return nil, ErrNotFound
	}

	return run, nil
}
