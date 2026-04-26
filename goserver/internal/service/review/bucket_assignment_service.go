package review

import (
	"context"
	"fmt"
	"time"

	domaincommon "goserver/internal/domain/common"
	domainposition "goserver/internal/domain/position"
	domainreview "goserver/internal/domain/review"
	platformrepo "goserver/internal/platform/repository"
	servicecommon "goserver/internal/service/common"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

const (
	defaultBucketAssignmentMaxReviews = 100
	maxBucketAssignmentPageSize       = 500
)

type BucketAssignmentConfig struct {
	DefaultMaxReviews        int
	MaxPageSize              int
	DefaultTargetPositionPct float64
	DefaultMaxPositionPct    float64
}

type BucketAssignmentOption func(*bucketAssignmentService)

func WithBucketAssignmentConfig(config BucketAssignmentConfig) BucketAssignmentOption {
	return func(service *bucketAssignmentService) {
		if config.DefaultMaxReviews > 0 {
			service.config.DefaultMaxReviews = config.DefaultMaxReviews
		}
		if config.MaxPageSize > 0 {
			service.config.MaxPageSize = config.MaxPageSize
		}
		applyPositiveFloat(&service.config.DefaultTargetPositionPct, config.DefaultTargetPositionPct)
		applyPositiveFloat(&service.config.DefaultMaxPositionPct, config.DefaultMaxPositionPct)
	}
}

func WithBucketAssignmentClock(clock servicecommon.ClockPort) BucketAssignmentOption {
	return func(service *bucketAssignmentService) {
		if clock != nil {
			service.now = clock.Now
		}
	}
}

type bucketAssignmentService struct {
	reviews   platformrepo.CompanyReviewRepository
	positions platformrepo.CurrentPositionRepository
	config    BucketAssignmentConfig
	now       func() time.Time
}

var _ BucketAssignmentService = (*bucketAssignmentService)(nil)

func NewBucketAssignmentService(
	reviews platformrepo.CompanyReviewRepository,
	positions platformrepo.CurrentPositionRepository,
	options ...BucketAssignmentOption,
) BucketAssignmentService {
	service := &bucketAssignmentService{
		reviews:   reviews,
		positions: positions,
		config: BucketAssignmentConfig{
			DefaultMaxReviews:        defaultBucketAssignmentMaxReviews,
			MaxPageSize:              maxBucketAssignmentPageSize,
			DefaultTargetPositionPct: 5.0,
			DefaultMaxPositionPct:    10.0,
		},
		now: time.Now,
	}
	for _, option := range options {
		if option != nil {
			option(service)
		}
	}
	if service.config.DefaultMaxReviews <= 0 {
		service.config.DefaultMaxReviews = defaultBucketAssignmentMaxReviews
	}
	if service.config.MaxPageSize <= 0 {
		service.config.MaxPageSize = maxBucketAssignmentPageSize
	}
	if service.config.DefaultTargetPositionPct <= 0 {
		service.config.DefaultTargetPositionPct = 5.0
	}
	if service.config.DefaultMaxPositionPct <= 0 {
		service.config.DefaultMaxPositionPct = 10.0
	}
	return service
}

func (service *bucketAssignmentService) AssignBucket(
	ctx context.Context,
	request AssignBucketRequest,
) (*AssignBucketResult, error) {
	if err := request.Validate(); err != nil {
		return nil, err
	}
	outcome, err := service.assignOneBucket(ctx, assignBucketOptions{
		ReviewID:        request.ReviewID,
		WorkflowRunID:   request.WorkflowRunID,
		CompanyID:       request.CompanyID,
		BookType:        request.BookType,
		ActionType:      request.ActionType,
		RequestedBucket: request.RequestedBucket,
		DryRun:          request.DryRun,
		Force:           request.Force,
		InitiatedBy:     request.InitiatedBy,
		CorrelationID:   request.CorrelationID,
	})
	if err != nil {
		return nil, err
	}
	return buildAssignBucketResult(outcome), nil
}

func (service *bucketAssignmentService) AssignBucketsForWorkflow(
	ctx context.Context,
	request AssignBucketsForWorkflowRequest,
) (*AssignBucketsForWorkflowResult, error) {
	if err := request.Validate(); err != nil {
		return nil, err
	}
	if service.reviews == nil {
		return nil, fmt.Errorf("assign workflow buckets %s: review repository is required", request.WorkflowRunID.Hex())
	}

	startedAt := service.now().UTC()
	reviews, hasMore, err := service.listWorkflowReviewsForBucket(ctx, request)
	if err != nil {
		return nil, err
	}

	result := &AssignBucketsForWorkflowResult{WorkflowRunID: request.WorkflowRunID}
	skipped := 0
	for _, item := range reviews {
		if item == nil {
			continue
		}
		outcome, err := service.assignOneBucket(ctx, assignBucketOptions{
			ReviewID:              item.ID,
			WorkflowRunID:         request.WorkflowRunID,
			BookType:              request.BookType,
			DryRun:                request.DryRun,
			Force:                 request.Force,
			InitiatedBy:           request.InitiatedBy,
			CorrelationID:         request.CorrelationID,
			TreatIneligibleAsSkip: true,
			PreloadedReview:       item,
		})
		if err != nil {
			result.FailedReviewIDs = append(result.FailedReviewIDs, item.ID)
			result.PartialFailures = append(result.PartialFailures, bucketPartialFailure(item, request.WorkflowRunID, err))
			continue
		}
		if outcome.Skipped {
			skipped++
			continue
		}
		result.AssignedReviewIDs = append(result.AssignedReviewIDs, outcome.ReviewID)
		result.BucketResults = append(result.BucketResults, *buildAssignBucketResult(outcome))
	}

	result.AssignedReviewIDs = uniqueObjectIDs(result.AssignedReviewIDs)
	result.FailedReviewIDs = uniqueObjectIDs(result.FailedReviewIDs)
	completedAt := service.now().UTC()
	result.Summary = buildWorkflowBucketSummary(
		len(reviews),
		len(result.AssignedReviewIDs),
		countChangedBuckets(result.BucketResults),
		skipped,
		len(result.PartialFailures),
		request.DryRun,
		hasMore,
		startedAt,
		completedAt,
	)
	return result, nil
}

func (service *bucketAssignmentService) assignOneBucket(
	ctx context.Context,
	options assignBucketOptions,
) (bucketAssignmentOutcome, error) {
	review, position, err := service.loadBucketContext(ctx, options)
	if err != nil {
		return bucketAssignmentOutcome{}, err
	}
	outcome := bucketAssignmentOutcome{
		ReviewID:      review.ID,
		CompanyID:     review.CompanyID,
		WorkflowRunID: review.WorkflowRunID,
		BucketBefore:  review.CurrentBucketBeforeReview,
		DryRun:        options.DryRun,
	}

	if err := validateBucketEligibleReview(review, options); err != nil {
		if options.TreatIneligibleAsSkip && isReviewServiceSkip(err) {
			outcome.Skipped = true
			outcome.Message = trimSkipMessage(err)
			return outcome, nil
		}
		return bucketAssignmentOutcome{}, err
	}

	action := options.ActionType
	if action == "" {
		action = actionFromReview(review)
	}
	if action == "" {
		return bucketAssignmentOutcome{}, fmt.Errorf("assign bucket review %s: action type is required", review.ID.Hex())
	}

	positionCtx := extractBucketPositionContext(review, position, service.config)
	bucketCtx := bucketContext{
		CapitalEligible:   review.DecisionAction != nil && review.DecisionAction.CapitalEligible,
		WeakeningDetected: bucketWeakeningDetected(review),
	}
	assigned := determineBucketFromAction(action, positionCtx, bucketCtx)
	message := fmt.Sprintf("assigned %s bucket for %s action", assigned, action)
	if options.RequestedBucket != "" {
		if err := validateActionBucketCompatibility(action, options.RequestedBucket, positionCtx); err != nil {
			return bucketAssignmentOutcome{}, err
		}
		assigned = options.RequestedBucket
		message = fmt.Sprintf("assigned requested %s bucket for %s action", assigned, action)
	}
	assigned, normalized := normalizeActionBucketCompatibility(action, assigned, positionCtx)
	if normalized != "" {
		message += "; " + normalized
	}

	outcome.ActionType = action
	outcome.BucketAfter = assigned
	outcome.BucketChanged = outcome.BucketBefore != assigned
	outcome.Assigned = true
	outcome.Message = message

	if !options.DryRun {
		persisted, err := service.persistBucketAssignmentIfMutable(ctx, review, action, assigned, options)
		if err != nil {
			return bucketAssignmentOutcome{}, err
		}
		outcome.Persisted = persisted
	}
	return outcome, nil
}

func (service *bucketAssignmentService) listWorkflowReviewsForBucket(
	ctx context.Context,
	request AssignBucketsForWorkflowRequest,
) ([]*domainreview.CompanyReview, bool, error) {
	limit := service.maxReviews(request.MaxReviews)
	list, err := service.reviews.ListByWorkflowRun(ctx, request.WorkflowRunID, platformrepo.CompanyReviewListOptions{
		Pagination: platformrepo.PageOptions{PageSize: limit},
		Sort: platformrepo.CompanyReviewSortOption{
			By:    platformrepo.CompanyReviewSortByFinalizedAt,
			Order: platformrepo.SortOrderAscending,
		},
	})
	if err != nil {
		return nil, false, fmt.Errorf("assign workflow buckets %s: list reviews: %w", request.WorkflowRunID.Hex(), err)
	}
	if list == nil {
		return nil, false, nil
	}
	return list.Items, list.Page.HasMore, nil
}

func (service *bucketAssignmentService) loadBucketContext(
	ctx context.Context,
	options assignBucketOptions,
) (*domainreview.CompanyReview, *domainposition.CurrentPosition, error) {
	review, err := service.loadReview(ctx, options.ReviewID, options.PreloadedReview)
	if err != nil {
		return nil, nil, err
	}
	if !options.CompanyID.IsZero() && review.CompanyID != options.CompanyID {
		return nil, nil, fmt.Errorf("%w: companyId filter does not match", errReviewServiceSkipped)
	}

	var position *domainposition.CurrentPosition
	if service.positions != nil && !review.CompanyID.IsZero() {
		position, err = service.positions.GetByCompanyAndBook(ctx, review.CompanyID, domaincommon.BookTypeInvesting)
		if err != nil && !isRepositoryNotFound(err) {
			return nil, nil, fmt.Errorf("assign bucket review %s: load position: %w", review.ID.Hex(), err)
		}
		if isRepositoryNotFound(err) {
			position = nil
		}
	}
	return review, position, nil
}

func (service *bucketAssignmentService) loadReview(
	ctx context.Context,
	reviewID primitive.ObjectID,
	preloaded *domainreview.CompanyReview,
) (*domainreview.CompanyReview, error) {
	if preloaded != nil {
		return preloaded, nil
	}
	if service.reviews == nil {
		return nil, fmt.Errorf("review repository is required")
	}
	review, err := service.reviews.GetByID(ctx, reviewID)
	if err != nil {
		return nil, fmt.Errorf("load review %s: %w", reviewID.Hex(), err)
	}
	if review == nil {
		return nil, fmt.Errorf("load review %s: %w", reviewID.Hex(), platformrepo.ErrNotFound)
	}
	return review, nil
}

func (service *bucketAssignmentService) persistBucketAssignmentIfMutable(
	ctx context.Context,
	review *domainreview.CompanyReview,
	action domaincommon.InvestingActionType,
	bucket domaincommon.WatchlistBucket,
	options assignBucketOptions,
) (bool, error) {
	if service.reviews == nil || review == nil || !isMutableValidatedReview(review) {
		return false, nil
	}
	decision := review.DecisionAction
	if decision == nil {
		decision = &domainreview.DecisionAction{
			ActionType:          action,
			ActionReasonPrimary: "bucket_assignment_action_context",
			CapitalEligible:     false,
		}
	} else {
		copied := *decision
		decision = &copied
	}
	decision.ActionType = action
	decision.BucketAfterAction = bucket

	_, err := service.reviews.SaveValidatedReviewContent(ctx, review.ID, platformrepo.ReviewValidatedContentPatch{
		Sections:               append([]domainreview.SectionScore(nil), review.Sections...),
		DecisionAction:         decision,
		PositionSnapshot:       review.PositionSnapshot,
		ChangeLog:              review.ChangeLog,
		WeightedTotalScore:     review.WeightedTotalScore,
		HardGateFailed:         review.HardGateFailed,
		HardGateFailureReasons: append([]string(nil), review.HardGateFailureReasons...),
		ConfidenceScore:        review.ConfidenceScore,
		FinalBucketAfterReview: bucket,
		FinalActionAfterReview: action,
		ActionRationaleSummary: review.ActionRationaleSummary,
		WhatChangedSummary:     review.WhatChangedSummary,
		ReviewerType:           &review.ReviewerType,
		ExpectedCurrentLifecycleStates: []domaincommon.ReviewLifecycleState{
			domaincommon.ReviewLifecycleStateAIValidated,
		},
		ExpectedCurrentStatuses: []domaincommon.ReviewStatus{
			domaincommon.ReviewStatusDraft,
		},
		Mutation: mutationMetadata(service.now().UTC(), options.InitiatedBy, "deterministic bucket assignment"),
	})
	if err != nil {
		return false, fmt.Errorf("persist bucket assignment for review %s: %w", review.ID.Hex(), err)
	}
	return true, nil
}

func (service *bucketAssignmentService) maxReviews(requested int) int {
	if requested > 0 && requested < service.config.MaxPageSize {
		return requested
	}
	if requested > service.config.MaxPageSize {
		return service.config.MaxPageSize
	}
	if service.config.DefaultMaxReviews > service.config.MaxPageSize {
		return service.config.MaxPageSize
	}
	return service.config.DefaultMaxReviews
}
