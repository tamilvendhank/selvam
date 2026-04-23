package service

import (
	"context"
	"testing"

	platformconfig "goserver/internal/platform/config"
	"goserver/internal/platform/domain"
	"goserver/internal/platform/testutil"
)

func TestActionMappingBuy(t *testing.T) {
	service := NewActionMappingService(platformconfig.Default())
	review := testutil.SampleInvestingReview(8.2, false)
	thesis := testutil.SampleThesis()

	decision, err := service.MapReview(context.Background(), review, thesis, nil)
	if err != nil {
		t.Fatalf("MapReview returned error: %v", err)
	}
	if decision.ActionType != domain.ActionBuy {
		t.Fatalf("expected buy, got %s", decision.ActionType)
	}
}

func TestActionMappingRejectOnHardGateFailure(t *testing.T) {
	service := NewActionMappingService(platformconfig.Default())
	review := testutil.SampleInvestingReview(8.2, false)
	review.HardGateFailed = true

	decision, err := service.MapReview(context.Background(), review, nil, nil)
	if err != nil {
		t.Fatalf("MapReview returned error: %v", err)
	}
	if decision.ActionType != domain.ActionReject {
		t.Fatalf("expected reject, got %s", decision.ActionType)
	}
}

func TestActionMappingSellForOwnedWeakeningReview(t *testing.T) {
	service := NewActionMappingService(platformconfig.Default())
	review := testutil.SampleInvestingReview(5.2, true)

	decision, err := service.MapReview(context.Background(), review, testutil.SampleThesis(), nil)
	if err != nil {
		t.Fatalf("MapReview returned error: %v", err)
	}
	if decision.ActionType != domain.ActionSell {
		t.Fatalf("expected sell, got %s", decision.ActionType)
	}
}
