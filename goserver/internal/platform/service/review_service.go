package service

import (
	"context"
	"errors"

	"goserver/internal/platform/domain"
	"goserver/internal/platform/ports"
	mongorepo "goserver/internal/platform/repository/mongo"
)

type DefaultReviewService struct {
	repository             ports.CompanyReviewRepository
	thesisRepository       ports.ThesisRepository
	actionMappingService   ports.ActionMappingService
	changeDetectionService ports.ChangeDetectionService
}

func NewReviewService(
	repository ports.CompanyReviewRepository,
	thesisRepository ports.ThesisRepository,
	actionMappingService ports.ActionMappingService,
	changeDetectionService ports.ChangeDetectionService,
) *DefaultReviewService {
	return &DefaultReviewService{
		repository:             repository,
		thesisRepository:       thesisRepository,
		actionMappingService:   actionMappingService,
		changeDetectionService: changeDetectionService,
	}
}

func (service *DefaultReviewService) CreateReview(ctx context.Context, review *domain.CompanyReview) (*domain.CompanyReview, error) {
	if err := ensureReviewExists(review); err != nil {
		return nil, err
	}

	if err := service.prepareReview(ctx, review); err != nil {
		return nil, err
	}

	return service.repository.Create(ctx, review)
}

func (service *DefaultReviewService) UpdateDraftReview(ctx context.Context, review *domain.CompanyReview) (*domain.CompanyReview, error) {
	if err := ensureReviewExists(review); err != nil {
		return nil, err
	}
	if err := service.prepareReview(ctx, review); err != nil {
		return nil, err
	}

	updated, err := service.repository.UpdateDraft(ctx, review)
	if errors.Is(err, mongorepo.ErrImmutableReview) {
		return nil, ErrImmutableReview
	}

	return updated, err
}

func (service *DefaultReviewService) FinalizeReview(ctx context.Context, reviewID string) (*domain.CompanyReview, error) {
	review, err := service.repository.Finalize(ctx, reviewID)
	if errors.Is(err, mongorepo.ErrImmutableReview) {
		return nil, ErrImmutableReview
	}
	if err != nil {
		return nil, err
	}
	if review == nil {
		return nil, ErrNotFound
	}

	return review, nil
}

func (service *DefaultReviewService) ListReviews(ctx context.Context, filter ports.CompanyReviewListFilter) ([]*domain.CompanyReview, error) {
	return service.repository.List(ctx, filter)
}

func (service *DefaultReviewService) GetReview(ctx context.Context, id string) (*domain.CompanyReview, error) {
	review, err := service.repository.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if review == nil {
		return nil, ErrNotFound
	}

	return review, nil
}

func (service *DefaultReviewService) GetReviewDiff(ctx context.Context, id string) (*domain.ReviewChangeLog, error) {
	review, err := service.GetReview(ctx, id)
	if err != nil {
		return nil, err
	}

	if review.ChangeLog == nil {
		return &domain.ReviewChangeLog{}, nil
	}

	return review.ChangeLog, nil
}

func (service *DefaultReviewService) GetReviewEvidence(ctx context.Context, id string) ([]domain.EvidenceReference, error) {
	review, err := service.GetReview(ctx, id)
	if err != nil {
		return nil, err
	}

	return review.FlattenEvidence(), nil
}

func (service *DefaultReviewService) prepareReview(ctx context.Context, review *domain.CompanyReview) error {
	previousReview, err := service.repository.GetLatestByCompany(ctx, review.CompanyID, review.BookType)
	if err != nil {
		return err
	}

	thesis, err := service.thesisRepository.GetActiveByCompanyID(ctx, review.CompanyID)
	if err != nil {
		return err
	}

	if review.ChangeLog == nil {
		changeLog, err := service.changeDetectionService.CompareReviews(ctx, review, previousReview, thesis)
		if err != nil {
			return err
		}
		review.ChangeLog = changeLog
	}

	if review.DecisionAction == nil {
		decision, err := service.actionMappingService.MapReview(ctx, review, thesis, previousReview)
		if err != nil {
			return err
		}
		review.DecisionAction = decision
	}
	if review.FinalActionAfterReview == "" && review.DecisionAction != nil {
		review.FinalActionAfterReview = review.DecisionAction.ActionType
	}
	if review.FinalBucketAfterReview == "" && review.DecisionAction != nil {
		review.FinalBucketAfterReview = review.DecisionAction.BucketAfterAction
	}
	if review.ActionRationaleSummary == "" && review.DecisionAction != nil {
		review.ActionRationaleSummary = review.DecisionAction.ActionReasonPrimary
	}
	if review.WhatChangedSummary == "" && review.ChangeLog != nil {
		review.WhatChangedSummary = review.ChangeLog.ChangeSummary
	}

	return review.Validate()
}
