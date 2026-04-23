package service

import (
	"context"
	"testing"
	"time"

	"goserver/internal/platform/domain"
	"goserver/internal/platform/ports"
	"goserver/internal/platform/testutil"
)

type manualOverrideRepoStub struct {
	items []*domain.ManualOverride
}

func (stub *manualOverrideRepoStub) Create(_ context.Context, override *domain.ManualOverride) (*domain.ManualOverride, error) {
	override.ID = "override-1"
	stub.items = append(stub.items, override)
	return override, nil
}

func (stub *manualOverrideRepoStub) GetByID(_ context.Context, id string) (*domain.ManualOverride, error) {
	for _, item := range stub.items {
		if item.ID == id {
			return item, nil
		}
	}
	return nil, nil
}

func (stub *manualOverrideRepoStub) List(_ context.Context, _ ports.ManualOverrideListFilter) ([]*domain.ManualOverride, error) {
	return stub.items, nil
}

type reviewRepoStub struct {
	review *domain.CompanyReview
}

func (stub *reviewRepoStub) Create(context.Context, *domain.CompanyReview) (*domain.CompanyReview, error) {
	return nil, nil
}
func (stub *reviewRepoStub) UpdateDraft(context.Context, *domain.CompanyReview) (*domain.CompanyReview, error) {
	return nil, nil
}
func (stub *reviewRepoStub) Finalize(context.Context, string) (*domain.CompanyReview, error) {
	return nil, nil
}
func (stub *reviewRepoStub) MarkSuperseded(context.Context, string) (*domain.CompanyReview, error) {
	return nil, nil
}
func (stub *reviewRepoStub) GetByID(_ context.Context, _ string) (*domain.CompanyReview, error) {
	return stub.review, nil
}
func (stub *reviewRepoStub) GetLatestByCompany(context.Context, string, domain.BookType) (*domain.CompanyReview, error) {
	return nil, nil
}
func (stub *reviewRepoStub) List(context.Context, ports.CompanyReviewListFilter) ([]*domain.CompanyReview, error) {
	return nil, nil
}

func TestOverrideServiceRejectsMismatchedOriginalAction(t *testing.T) {
	review := testutil.SampleInvestingReview(8.1, true)
	review.FinalActionAfterReview = domain.ActionHold
	service := NewOverrideService(&manualOverrideRepoStub{}, &reviewRepoStub{review: review})

	_, err := service.CreateOverride(context.Background(), &domain.ManualOverride{
		CompanyID:        review.CompanyID,
		ReviewID:         review.ID,
		BookType:         review.BookType,
		OriginalAction:   domain.ActionBuy,
		OverriddenAction: domain.ActionTrim,
		OverrideReason:   "Reduce concentration",
		OverrideBy:       "pm",
		OverrideDate:     time.Now().UTC(),
		SchemaVersion:    domain.SchemaVersionV1Alpha1,
		CreatedAt:        time.Now().UTC(),
	})
	if err == nil {
		t.Fatalf("expected mismatch error")
	}
}

func TestOverrideServicePersistsValidOverride(t *testing.T) {
	review := testutil.SampleInvestingReview(8.1, true)
	review.FinalActionAfterReview = domain.ActionHold
	repository := &manualOverrideRepoStub{}
	service := NewOverrideService(repository, &reviewRepoStub{review: review})

	override, err := service.CreateOverride(context.Background(), &domain.ManualOverride{
		CompanyID:        review.CompanyID,
		ReviewID:         review.ID,
		BookType:         review.BookType,
		OriginalAction:   domain.ActionHold,
		OverriddenAction: domain.ActionTrim,
		OverrideReason:   "Reduce concentration",
		OverrideBy:       "pm",
		OverrideDate:     time.Now().UTC(),
		SchemaVersion:    domain.SchemaVersionV1Alpha1,
		CreatedAt:        time.Now().UTC(),
	})
	if err != nil {
		t.Fatalf("CreateOverride returned error: %v", err)
	}
	if override.ID == "" {
		t.Fatalf("expected override id to be assigned")
	}
}
