package service

import (
	"context"
	"testing"

	platformconfig "goserver/internal/platform/config"
	"goserver/internal/platform/domain"
	"goserver/internal/platform/ports"
)

type configSnapshotRepoStub struct {
	created []*domain.ConfigSnapshot
}

func (stub *configSnapshotRepoStub) Create(_ context.Context, snapshot *domain.ConfigSnapshot) (*domain.ConfigSnapshot, error) {
	snapshot.ID = "snapshot-1"
	stub.created = append(stub.created, snapshot)
	return snapshot, nil
}

func (stub *configSnapshotRepoStub) GetByID(_ context.Context, id string) (*domain.ConfigSnapshot, error) {
	for _, snapshot := range stub.created {
		if snapshot.ID == id {
			return snapshot, nil
		}
	}
	return nil, nil
}

func (stub *configSnapshotRepoStub) List(_ context.Context, _ ports.ConfigSnapshotListFilter) ([]*domain.ConfigSnapshot, error) {
	return stub.created, nil
}

func TestConfigServiceCreateSnapshotSanitizesSecrets(t *testing.T) {
	config := platformconfig.Default()
	config.AsyncAI.APIKey = "secret"
	repository := &configSnapshotRepoStub{}
	service := NewConfigService(config, repository, nil)

	snapshot, err := service.CreateSnapshot(context.Background(), domain.BookTypeInvesting, "balanced")
	if err != nil {
		t.Fatalf("CreateSnapshot returned error: %v", err)
	}
	if snapshot.ConfigJSON["ai"] == nil {
		t.Fatalf("expected ai to be present in snapshot payload")
	}

	currentConfig, err := service.CurrentConfig(context.Background())
	if err != nil {
		t.Fatalf("CurrentConfig returned error: %v", err)
	}
	ai := currentConfig["ai"].(map[string]any)
	if _, exists := ai["apiKey"]; exists {
		t.Fatalf("expected api key to be omitted from sanitized config")
	}
}
