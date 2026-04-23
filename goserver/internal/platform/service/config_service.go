package service

import (
	"context"
	"fmt"

	platformconfig "goserver/internal/platform/config"
	"goserver/internal/platform/domain"
	"goserver/internal/platform/ports"
)

type DefaultConfigService struct {
	config       platformconfig.AppConfig
	repository   ports.ConfigSnapshotRepository
	timeProvider ports.TimeProvider
}

func NewConfigService(config platformconfig.AppConfig, repository ports.ConfigSnapshotRepository, timeProvider ports.TimeProvider) *DefaultConfigService {
	return &DefaultConfigService{
		config:       config,
		repository:   repository,
		timeProvider: resolveTimeProvider(timeProvider),
	}
}

func (service *DefaultConfigService) CurrentConfig(_ context.Context) (map[string]any, error) {
	return service.config.MarshalSanitizedJSON()
}

func (service *DefaultConfigService) CreateSnapshot(ctx context.Context, bookType domain.BookType, mode string) (*domain.ConfigSnapshot, error) {
	if !domain.IsValidBookType(bookType) {
		return nil, fmt.Errorf("invalid book type %q", bookType)
	}
	if service.repository == nil {
		return nil, fmt.Errorf("config snapshot repository is not configured")
	}

	payload, err := buildSnapshotPayload(service.config, bookType, mode)
	if err != nil {
		return nil, err
	}

	snapshot := &domain.ConfigSnapshot{
		BookType:      bookType,
		Mode:          mode,
		SchemaVersion: service.config.SchemaVersion,
		ConfigJSON:    payload,
		CreatedAt:     service.timeProvider.Now(),
	}

	return service.repository.Create(ctx, snapshot)
}

func (service *DefaultConfigService) ListSnapshots(ctx context.Context, filter ports.ConfigSnapshotListFilter) ([]*domain.ConfigSnapshot, error) {
	return service.repository.List(ctx, filter)
}

func (service *DefaultConfigService) GetSnapshot(ctx context.Context, id string) (*domain.ConfigSnapshot, error) {
	snapshot, err := service.repository.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if snapshot == nil {
		return nil, ErrNotFound
	}

	return snapshot, nil
}

func buildSnapshotPayload(config platformconfig.AppConfig, bookType domain.BookType, mode string) (map[string]any, error) {
	currentConfig, err := config.MarshalSanitizedJSON()
	if err != nil {
		return nil, err
	}

	payload := map[string]any{
		"schemaVersion": config.SchemaVersion,
		"environment":   config.Environment,
		"global":        currentConfig["global"],
		"ui":            currentConfig["ui"],
		"asyncAi":       currentConfig["asyncAi"],
		"bookType":      bookType,
		"mode":          mode,
	}
	if bookType == domain.BookTypeInvesting {
		payload["investing"] = currentConfig["investing"]
	}
	if bookType == domain.BookTypeTrading {
		payload["trading"] = currentConfig["trading"]
	}

	return payload, nil
}
