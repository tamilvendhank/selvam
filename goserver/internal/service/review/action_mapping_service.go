package review

import (
	"context"
	"fmt"
	"time"

	domaincommon "goserver/internal/domain/common"
	domainposition "goserver/internal/domain/position"
	domainreview "goserver/internal/domain/review"
	domainthesis "goserver/internal/domain/thesis"
	platformrepo "goserver/internal/platform/repository"
	servicecommon "goserver/internal/service/common"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

const (
	defaultActionMappingMaxReviews = 100
	maxActionMappingPageSize       = 500
)

type ActionMappingConfig struct {
	DefaultMaxReviews int
	MaxPageSize       int

	ExceptionalMin float64
	StrongMin      float64
	AcceptableMin  float64
	WeakMin        float64

	BuyMinOverall               float64
	BuyMinManagementGovernance  float64
	BuyMinCapitalEfficiency     float64
	BuyMinValuation             float64
	CoreStrongThreshold         float64
	CoreFloorThreshold          float64
	CoreWeakThreshold           float64
	MinStrongCoreSectionsForBuy int
	MaxWeakCoreSectionsForBuy   int

	HoldMinOverall     float64
	RejectBelowOverall float64
	SellBelowOverall   float64

	ExitReviewTotalDrop      float64
	ExitReviewCoreDrop       float64
	ExitReviewManagementDrop float64

	RequireWrittenThesisForBuy bool
	SellOnThesisBreak          bool

	DefaultTargetPositionPct float64
	DefaultMaxPositionPct    float64
}

type ActionMappingOption func(*actionMappingService)

func WithActionMappingConfig(config ActionMappingConfig) ActionMappingOption {
	return func(service *actionMappingService) {
		if config.DefaultMaxReviews > 0 {
			service.config.DefaultMaxReviews = config.DefaultMaxReviews
		}
		if config.MaxPageSize > 0 {
			service.config.MaxPageSize = config.MaxPageSize
		}
		applyPositiveFloat(&service.config.ExceptionalMin, config.ExceptionalMin)
		applyPositiveFloat(&service.config.StrongMin, config.StrongMin)
		applyPositiveFloat(&service.config.AcceptableMin, config.AcceptableMin)
		applyPositiveFloat(&service.config.WeakMin, config.WeakMin)
		applyPositiveFloat(&service.config.BuyMinOverall, config.BuyMinOverall)
		applyPositiveFloat(&service.config.BuyMinManagementGovernance, config.BuyMinManagementGovernance)
		applyPositiveFloat(&service.config.BuyMinCapitalEfficiency, config.BuyMinCapitalEfficiency)
		applyPositiveFloat(&service.config.BuyMinValuation, config.BuyMinValuation)
		applyPositiveFloat(&service.config.CoreStrongThreshold, config.CoreStrongThreshold)
		applyPositiveFloat(&service.config.CoreFloorThreshold, config.CoreFloorThreshold)
		applyPositiveFloat(&service.config.CoreWeakThreshold, config.CoreWeakThreshold)
		applyPositiveInt(&service.config.MinStrongCoreSectionsForBuy, config.MinStrongCoreSectionsForBuy)
		applyPositiveInt(&service.config.MaxWeakCoreSectionsForBuy, config.MaxWeakCoreSectionsForBuy)
		applyPositiveFloat(&service.config.HoldMinOverall, config.HoldMinOverall)
		applyPositiveFloat(&service.config.RejectBelowOverall, config.RejectBelowOverall)
		applyPositiveFloat(&service.config.SellBelowOverall, config.SellBelowOverall)
		applyPositiveFloat(&service.config.ExitReviewTotalDrop, config.ExitReviewTotalDrop)
		applyPositiveFloat(&service.config.ExitReviewCoreDrop, config.ExitReviewCoreDrop)
		applyPositiveFloat(&service.config.ExitReviewManagementDrop, config.ExitReviewManagementDrop)
		applyPositiveFloat(&service.config.DefaultTargetPositionPct, config.DefaultTargetPositionPct)
		applyPositiveFloat(&service.config.DefaultMaxPositionPct, config.DefaultMaxPositionPct)
	}
}

func WithActionMappingPolicy(requireWrittenThesisForBuy bool, sellOnThesisBreak bool) ActionMappingOption {
	return func(service *actionMappingService) {
		service.config.RequireWrittenThesisForBuy = requireWrittenThesisForBuy
		service.config.SellOnThesisBreak = sellOnThesisBreak
	}
}

func WithActionMappingClock(clock servicecommon.ClockPort) ActionMappingOption {
	return func(service *actionMappingService) {
		if clock != nil {
			service.now = clock.Now
		}
	}
}

type actionMappingService struct {
	reviews   platformrepo.CompanyReviewRepository
	theses    platformrepo.InvestmentThesisRepository
	positions platformrepo.CurrentPositionRepository
	config    ActionMappingConfig
	now       func() time.Time
}

var _ ActionMappingService = (*actionMappingService)(nil)

func NewActionMappingService(
	reviews platformrepo.CompanyReviewRepository,
	theses platformrepo.InvestmentThesisRepository,
	positions platformrepo.CurrentPositionRepository,
	options ...ActionMappingOption,
) ActionMappingService {
	service := &actionMappingService{
		reviews:   reviews,
		theses:    theses,
		positions: positions,
		config:    defaultActionMappingConfig(),
		now:       time.Now,
	}
	for _, option := range options {
		if option != nil {
			option(service)
		}
	}
	service.config = normalizeActionMappingConfig(service.config)
	return service
}

func (service *actionMappingService) MapReviewAction(
	ctx context.Context,
	request MapReviewActionRequest,
) (*MapReviewActionResult, error) {
	if err := request.Validate(); err != nil {
		return nil, err
	}
	outcome, err := service.mapOneReviewAction(ctx, mapActionOptions{
		ReviewID:      request.ReviewID,
		WorkflowRunID: request.WorkflowRunID,
		BookType:      request.BookType,
		Mode:          request.Mode,
		DryRun:        request.DryRun,
		Force:         request.Force,
		InitiatedBy:   request.InitiatedBy,
		CorrelationID: request.CorrelationID,
	})
	if err != nil {
		return nil, err
	}
	return buildMapReviewActionResult(outcome), nil
}

func (service *actionMappingService) MapWorkflowActions(
	ctx context.Context,
	request MapWorkflowActionsRequest,
) (*MapWorkflowActionsResult, error) {
	if err := request.Validate(); err != nil {
		return nil, err
	}
	if service.reviews == nil {
		return nil, fmt.Errorf("map workflow actions %s: review repository is required", request.WorkflowRunID.Hex())
	}

	startedAt := service.now().UTC()
	reviews, hasMore, err := service.listWorkflowReviewsForAction(ctx, request)
	if err != nil {
		return nil, err
	}

	result := &MapWorkflowActionsResult{WorkflowRunID: request.WorkflowRunID}
	skipped := 0
	for _, item := range reviews {
		if item == nil {
			continue
		}
		outcome, err := service.mapOneReviewAction(ctx, mapActionOptions{
			ReviewID:              item.ID,
			WorkflowRunID:         request.WorkflowRunID,
			BookType:              request.BookType,
			Mode:                  request.Mode,
			DryRun:                request.DryRun,
			Force:                 request.Force,
			InitiatedBy:           request.InitiatedBy,
			CorrelationID:         request.CorrelationID,
			TreatIneligibleAsSkip: true,
			PreloadedReview:       item,
		})
		if err != nil {
			result.FailedReviewIDs = append(result.FailedReviewIDs, item.ID)
			result.PartialFailures = append(result.PartialFailures, actionPartialFailure(item, request.WorkflowRunID, err))
			continue
		}
		if outcome.Skipped {
			skipped++
			continue
		}
		result.MappedReviewIDs = append(result.MappedReviewIDs, outcome.ReviewID)
		result.ActionResults = append(result.ActionResults, *buildMapReviewActionResult(outcome))
	}

	result.MappedReviewIDs = uniqueObjectIDs(result.MappedReviewIDs)
	result.FailedReviewIDs = uniqueObjectIDs(result.FailedReviewIDs)
	completedAt := service.now().UTC()
	result.Summary = buildWorkflowActionSummary(
		len(reviews),
		len(result.MappedReviewIDs),
		skipped,
		len(result.PartialFailures),
		countCapitalEligible(result.ActionResults),
		countActionConstraints(result.ActionResults),
		request.DryRun,
		hasMore,
		startedAt,
		completedAt,
	)
	return result, nil
}

func (service *actionMappingService) mapOneReviewAction(
	ctx context.Context,
	options mapActionOptions,
) (actionMappingOutcome, error) {
	review, thesis, position, err := service.loadActionContext(ctx, options)
	if err != nil {
		return actionMappingOutcome{}, err
	}
	outcome := actionMappingOutcome{
		ReviewID:       review.ID,
		CompanyID:      review.CompanyID,
		WorkflowRunID:  review.WorkflowRunID,
		DryRun:         options.DryRun,
		AlreadyPresent: review.DecisionAction != nil && review.FinalActionAfterReview != "",
	}

	if err := validateActionEligibleReview(review, options); err != nil {
		if options.TreatIneligibleAsSkip && isReviewServiceSkip(err) {
			outcome.Skipped = true
			outcome.Message = trimSkipMessage(err)
			return outcome, nil
		}
		return actionMappingOutcome{}, err
	}
	if outcome.AlreadyPresent && !options.Force {
		outcome.Mapped = true
		outcome.ActionType = review.FinalActionAfterReview
		outcome.BucketAfterAction = review.FinalBucketAfterReview
		outcome.CapitalEligible = review.DecisionAction.CapitalEligible
		outcome.PriorityScore = review.DecisionAction.CapitalPriorityScore
		outcome.Message = "review already has a decision action"
		return outcome, nil
	}

	score := extractScoreContext(review, service.config)
	positionCtx := extractPositionContext(review, position, service.config)
	decision := service.determineActionDecision(review, thesis, score, positionCtx, options)
	outcome.Mapped = true
	outcome.ActionType = decision.Action.ActionType
	outcome.BucketAfterAction = decision.Action.BucketAfterAction
	outcome.CapitalEligible = decision.Action.CapitalEligible
	outcome.PriorityScore = decision.Action.CapitalPriorityScore
	outcome.Constraints = decision.ResultConstraints
	outcome.Message = decision.Message

	if !options.DryRun {
		persisted, err := service.persistActionDecisionIfMutable(ctx, review, decision.Action, options)
		if err != nil {
			return actionMappingOutcome{}, err
		}
		outcome.Persisted = persisted
	}
	return outcome, nil
}

func (service *actionMappingService) listWorkflowReviewsForAction(
	ctx context.Context,
	request MapWorkflowActionsRequest,
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
		return nil, false, fmt.Errorf("map workflow actions %s: list reviews: %w", request.WorkflowRunID.Hex(), err)
	}
	if list == nil {
		return nil, false, nil
	}
	return list.Items, list.Page.HasMore, nil
}

func (service *actionMappingService) loadActionContext(
	ctx context.Context,
	options mapActionOptions,
) (*domainreview.CompanyReview, *domainthesis.InvestmentThesis, *domainposition.CurrentPosition, error) {
	review, err := service.loadReview(ctx, options.ReviewID, options.PreloadedReview)
	if err != nil {
		return nil, nil, nil, err
	}

	var thesis *domainthesis.InvestmentThesis
	if service.theses != nil && !review.CompanyID.IsZero() {
		thesis, err = service.theses.GetActiveByCompanyID(ctx, review.CompanyID)
		if err != nil && !isRepositoryNotFound(err) {
			return nil, nil, nil, fmt.Errorf("map action review %s: load active thesis: %w", review.ID.Hex(), err)
		}
		if thesis == nil || isRepositoryNotFound(err) {
			thesis, err = service.theses.GetLatestByCompanyID(ctx, review.CompanyID)
			if err != nil && !isRepositoryNotFound(err) {
				return nil, nil, nil, fmt.Errorf("map action review %s: load latest thesis: %w", review.ID.Hex(), err)
			}
			if isRepositoryNotFound(err) {
				thesis = nil
			}
		}
	}

	var position *domainposition.CurrentPosition
	if service.positions != nil && !review.CompanyID.IsZero() {
		position, err = service.positions.GetByCompanyAndBook(ctx, review.CompanyID, domaincommon.BookTypeInvesting)
		if err != nil && !isRepositoryNotFound(err) {
			return nil, nil, nil, fmt.Errorf("map action review %s: load position: %w", review.ID.Hex(), err)
		}
		if isRepositoryNotFound(err) {
			position = nil
		}
	}

	return review, thesis, position, nil
}

func (service *actionMappingService) loadReview(
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

func (service *actionMappingService) persistActionDecisionIfMutable(
	ctx context.Context,
	review *domainreview.CompanyReview,
	decision *domainreview.DecisionAction,
	options mapActionOptions,
) (bool, error) {
	if service.reviews == nil || review == nil || decision == nil || !isMutableValidatedReview(review) {
		return false, nil
	}
	_, err := service.reviews.SaveValidatedReviewContent(ctx, review.ID, platformrepo.ReviewValidatedContentPatch{
		Sections:               append([]domainreview.SectionScore(nil), review.Sections...),
		DecisionAction:         decision,
		PositionSnapshot:       review.PositionSnapshot,
		ChangeLog:              review.ChangeLog,
		WeightedTotalScore:     review.WeightedTotalScore,
		HardGateFailed:         review.HardGateFailed,
		HardGateFailureReasons: append([]string(nil), review.HardGateFailureReasons...),
		ConfidenceScore:        review.ConfidenceScore,
		FinalBucketAfterReview: decision.BucketAfterAction,
		FinalActionAfterReview: decision.ActionType,
		ActionRationaleSummary: decision.ActionReasonPrimary,
		WhatChangedSummary:     review.WhatChangedSummary,
		ReviewerType:           &review.ReviewerType,
		ExpectedCurrentLifecycleStates: []domaincommon.ReviewLifecycleState{
			domaincommon.ReviewLifecycleStateAIValidated,
		},
		ExpectedCurrentStatuses: []domaincommon.ReviewStatus{
			domaincommon.ReviewStatusDraft,
		},
		Mutation: mutationMetadata(service.now().UTC(), options.InitiatedBy, "deterministic action mapping"),
	})
	if err != nil {
		return false, fmt.Errorf("persist action decision for review %s: %w", review.ID.Hex(), err)
	}
	return true, nil
}

func (service *actionMappingService) maxReviews(requested int) int {
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
