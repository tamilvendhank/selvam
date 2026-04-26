package finalization

import (
	"context"
	"fmt"
	"time"

	domaincommon "goserver/internal/domain/common"
	domainreview "goserver/internal/domain/review"
	platformrepo "goserver/internal/platform/repository"
	servicecommon "goserver/internal/service/common"
	workerservice "goserver/internal/service/worker"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

const (
	defaultFinalizationMaxReviews = 50
	maxFinalizationPageSize       = 500
)

type ReviewFinalizationConfig struct {
	DefaultMaxReviews       int
	MaxPageSize             int
	SupersedePriorByDefault bool
}

type ReviewFinalizationOption func(*reviewFinalizationService)

func WithReviewFinalizationConfig(config ReviewFinalizationConfig) ReviewFinalizationOption {
	return func(service *reviewFinalizationService) {
		if config.DefaultMaxReviews > 0 {
			service.config.DefaultMaxReviews = config.DefaultMaxReviews
		}
		if config.MaxPageSize > 0 {
			service.config.MaxPageSize = config.MaxPageSize
		}
		service.config.SupersedePriorByDefault = config.SupersedePriorByDefault
	}
}

func WithReviewFinalizationClock(clock servicecommon.ClockPort) ReviewFinalizationOption {
	return func(service *reviewFinalizationService) {
		if clock != nil {
			service.now = clock.Now
		}
	}
}

type reviewFinalizationService struct {
	reviews   platformrepo.CompanyReviewRepository
	discovery workerservice.WorkerWorkDiscoveryService
	config    ReviewFinalizationConfig
	now       func() time.Time
}

var _ ReviewFinalizationService = (*reviewFinalizationService)(nil)

func NewReviewFinalizationService(
	reviews platformrepo.CompanyReviewRepository,
	discovery workerservice.WorkerWorkDiscoveryService,
	options ...ReviewFinalizationOption,
) ReviewFinalizationService {
	service := &reviewFinalizationService{
		reviews:   reviews,
		discovery: discovery,
		config: ReviewFinalizationConfig{
			DefaultMaxReviews: defaultFinalizationMaxReviews,
			MaxPageSize:       maxFinalizationPageSize,
		},
		now: time.Now,
	}
	for _, option := range options {
		if option != nil {
			option(service)
		}
	}
	if service.config.DefaultMaxReviews <= 0 {
		service.config.DefaultMaxReviews = defaultFinalizationMaxReviews
	}
	if service.config.MaxPageSize <= 0 {
		service.config.MaxPageSize = maxFinalizationPageSize
	}
	return service
}

func (service *reviewFinalizationService) FinalizeReview(
	ctx context.Context,
	request FinalizeReviewRequest,
) (*FinalizeReviewResult, error) {
	if err := request.Validate(); err != nil {
		return nil, err
	}

	outcome, err := service.finalizeOneReview(ctx, request.ReviewID, finalizationRequestOptions{
		WorkflowRunID:  request.WorkflowRunID,
		Force:          request.Force,
		SupersedePrior: request.SupersedePrior || service.config.SupersedePriorByDefault,
		DryRun:         request.DryRun,
		InitiatedBy:    request.InitiatedBy,
		CorrelationID:  request.CorrelationID,
	})
	if err != nil {
		return nil, err
	}
	return buildSingleFinalizationResult(outcome), nil
}

func (service *reviewFinalizationService) FinalizeEligibleReviews(
	ctx context.Context,
	request FinalizeEligibleReviewsRequest,
) (*FinalizeEligibleReviewsResult, error) {
	if err := request.Validate(); err != nil {
		return nil, err
	}

	reviewIDs, hasMore, err := service.discoverFinalizableReviewIDs(ctx, request)
	if err != nil {
		return nil, err
	}
	if len(reviewIDs) == 0 {
		return &FinalizeEligibleReviewsResult{
			Summary: buildFinalizationSummary("finalize_eligible_reviews", 0, 0, 0, 0, 0, request.DryRun),
		}, nil
	}

	result := FinalizeEligibleReviewsResult{}
	for _, reviewID := range reviewIDs {
		outcome, err := service.finalizeOneReview(ctx, reviewID, finalizationRequestOptions{
			WorkflowRunID:         request.WorkflowRunID,
			CompanyID:             request.CompanyID,
			BookType:              request.BookType,
			Force:                 request.Force,
			SupersedePrior:        request.SupersedePrior || service.config.SupersedePriorByDefault,
			DryRun:                request.DryRun,
			InitiatedBy:           request.InitiatedBy,
			CorrelationID:         request.CorrelationID,
			TreatIneligibleAsSkip: true,
		})
		if err != nil {
			result.FailedReviewIDs = append(result.FailedReviewIDs, reviewID)
			result.PartialFailures = append(result.PartialFailures, finalizationPartialFailure(reviewID, err))
			continue
		}
		mergeFinalizationOutcome(&result, outcome)
	}

	result.FinalizedReviewIDs = uniqueObjectIDs(result.FinalizedReviewIDs)
	result.FailedReviewIDs = uniqueObjectIDs(result.FailedReviewIDs)
	result.SkippedReviewIDs = uniqueObjectIDs(result.SkippedReviewIDs)
	result.SupersededReviewIDs = uniqueObjectIDs(result.SupersededReviewIDs)
	result.Summary = buildFinalizationSummary(
		"finalize_eligible_reviews",
		len(reviewIDs),
		len(result.FinalizedReviewIDs),
		len(result.SkippedReviewIDs),
		len(result.PartialFailures),
		len(result.SupersededReviewIDs),
		request.DryRun,
	)
	if hasMore {
		result.Summary.Message = fmt.Sprintf("%s; more finalizable reviews may be available", result.Summary.Message)
	}
	return &result, nil
}

func (service *reviewFinalizationService) finalizeOneReview(
	ctx context.Context,
	reviewID primitive.ObjectID,
	options finalizationRequestOptions,
) (finalizeOneOutcome, error) {
	review, err := service.loadFinalizationContext(ctx, reviewID)
	if err != nil {
		return finalizeOneOutcome{}, err
	}

	if err := validateFinalizationEligibility(review, options); err != nil {
		if isAlreadyFinalized(review) || isSuperseded(review) || (options.TreatIneligibleAsSkip && isFinalizationSkip(err)) {
			return finalizeOneOutcome{
				ReviewID:       review.ID,
				ReviewRef:      reviewRef(review),
				Skipped:        true,
				AlreadyFinal:   isAlreadyFinalized(review),
				AlreadyInvalid: isSuperseded(review),
			}, nil
		}
		return finalizeOneOutcome{}, err
	}

	if options.DryRun {
		return finalizeOneOutcome{
			ReviewID:  review.ID,
			ReviewRef: reviewRef(review),
			DryRun:    true,
		}, nil
	}

	finalized, err := service.applyReviewFinalization(ctx, review, options)
	if err != nil {
		return finalizeOneOutcome{}, fmt.Errorf("finalize review %s: %w", review.ID.Hex(), err)
	}

	outcome := finalizeOneOutcome{
		ReviewID:  finalized.ID,
		ReviewRef: reviewRef(finalized),
		Finalized: true,
	}
	if options.SupersedePrior {
		supersededIDs, failures := service.maybeSupersedePreviousReview(ctx, finalized, options)
		outcome.SupersededReviewIDs = append(outcome.SupersededReviewIDs, supersededIDs...)
		outcome.PartialFailures = append(outcome.PartialFailures, failures...)
	}
	return outcome, nil
}

func (service *reviewFinalizationService) applyReviewFinalization(
	ctx context.Context,
	review *domainreview.CompanyReview,
	options finalizationRequestOptions,
) (*domainreview.CompanyReview, error) {
	finalizedAt := service.now().UTC()
	// Repository preconditions are the concurrency guard: discovery and service checks
	// are advisory, while the write must still prove the review is draft + ai_validated.
	return service.reviews.FinalizeReview(ctx, review.ID, platformrepo.ReviewFinalizationPatch{
		FinalizedAt: finalizedAt,
		ExpectedCurrentLifecycleStates: []domaincommon.ReviewLifecycleState{
			domaincommon.ReviewLifecycleStateAIValidated,
		},
		ExpectedCurrentStatuses: []domaincommon.ReviewStatus{
			domaincommon.ReviewStatusDraft,
		},
		FinalizedBy: options.InitiatedBy,
		Reason:      "review finalization",
	})
}
