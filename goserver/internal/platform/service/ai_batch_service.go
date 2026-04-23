package service

import (
	"context"
	"fmt"

	"goserver/internal/platform/domain"
	"goserver/internal/platform/ports"
)

type DefaultAIBatchService struct {
	jobs    ports.AIBatchJobRepository
	items   ports.AIBatchItemRepository
	reviews ports.CompanyReviewRepository
}

func NewAIBatchService(
	jobs ports.AIBatchJobRepository,
	items ports.AIBatchItemRepository,
	reviews ports.CompanyReviewRepository,
) *DefaultAIBatchService {
	return &DefaultAIBatchService{
		jobs:    jobs,
		items:   items,
		reviews: reviews,
	}
}

func (service *DefaultAIBatchService) ListJobs(ctx context.Context, filter ports.AIBatchJobListFilter) ([]*domain.AIBatchJob, error) {
	return service.jobs.List(ctx, filter)
}

func (service *DefaultAIBatchService) GetJob(ctx context.Context, id string) (*domain.AIBatchJob, error) {
	job, err := service.jobs.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if job == nil {
		return nil, ErrNotFound
	}

	return job, nil
}

func (service *DefaultAIBatchService) ListItems(ctx context.Context, filter ports.AIBatchItemListFilter) ([]*domain.AIBatchItem, error) {
	return service.items.List(ctx, filter)
}

func (service *DefaultAIBatchService) RetryJob(ctx context.Context, id string) (*domain.AIBatchJob, error) {
	job, err := service.GetJob(ctx, id)
	if err != nil {
		return nil, err
	}
	if job.MaxRetryCount > 0 && job.RetryCount >= job.MaxRetryCount {
		return nil, ErrRetryExhausted
	}

	job.RetryCount++
	job.Status = domain.BatchJobStatusCreated
	job.ErrorSummary = ""
	job.FailedAt = nil
	job.CompletedAt = nil

	items, err := service.items.List(ctx, ports.AIBatchItemListFilter{
		AIBatchJobID: job.ID,
		Limit:        500,
	})
	if err != nil {
		return nil, err
	}
	for _, item := range items {
		if item.Status == domain.BatchItemStatusCompleted || item.Status == domain.BatchItemStatusSkipped {
			continue
		}
		item.Status = domain.BatchItemStatusPending
		item.ValidationStatus = domain.ValidationStatusNotValidated
		item.ValidationErrors = nil
		item.ErrorSummary = ""
		item.ResultPayload = nil
		item.CompletedAt = nil
		if _, err := service.items.Update(ctx, item); err != nil {
			return nil, err
		}
	}

	return service.jobs.Update(ctx, job)
}

func (service *DefaultAIBatchService) RetryItem(ctx context.Context, id string) (*domain.AIBatchItem, error) {
	item, err := service.items.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if item == nil {
		return nil, ErrNotFound
	}

	item.Status = domain.BatchItemStatusPending
	item.ValidationStatus = domain.ValidationStatusNotValidated
	item.ValidationErrors = nil
	item.ErrorSummary = ""
	item.ResultPayload = nil
	item.CompletedAt = nil
	updated, err := service.items.Update(ctx, item)
	if err != nil {
		return nil, err
	}

	job, err := service.jobs.GetByID(ctx, item.AIBatchJobID)
	if err != nil {
		return nil, err
	}
	if job != nil && (job.Status == domain.BatchJobStatusFailed || job.Status == domain.BatchJobStatusCompleted || job.Status == domain.BatchJobStatusPartiallyCompleted) {
		job.Status = domain.BatchJobStatusCreated
		job.ErrorSummary = ""
		job.CompletedAt = nil
		job.FailedAt = nil
		if _, err := service.jobs.Update(ctx, job); err != nil {
			return nil, err
		}
	}

	return updated, nil
}

func (service *DefaultAIBatchService) SkipItem(ctx context.Context, id string) (*domain.AIBatchItem, error) {
	item, err := service.items.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if item == nil {
		return nil, ErrNotFound
	}

	item.Status = domain.BatchItemStatusSkipped
	item.ValidationStatus = domain.ValidationStatusValid
	item.ValidationErrors = nil
	item.ErrorSummary = "manually skipped"
	updated, err := service.items.Update(ctx, item)
	if err != nil {
		return nil, err
	}

	if item.TargetReviewID != "" {
		review, err := service.reviews.GetByID(ctx, item.TargetReviewID)
		if err != nil {
			return nil, err
		}
		if review != nil && review.IsMutable() {
			review.ReviewStatus = domain.ReviewStatusValidationFailed
			review.ValidationStatus = domain.ValidationStatusInvalid
			review.ValidationErrors = []string{"manually skipped"}
			if _, err := service.reviews.UpdateMutable(ctx, review); err != nil {
				return nil, fmt.Errorf("update skipped review: %w", err)
			}
		}
	}

	return updated, nil
}
