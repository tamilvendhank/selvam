package thesis

import (
	"context"
	"errors"
	"fmt"
	"time"

	domaincommon "goserver/internal/domain/common"
	domainreview "goserver/internal/domain/review"
	domainthesis "goserver/internal/domain/thesis"
	platformrepo "goserver/internal/platform/repository"
	servicecommon "goserver/internal/service/common"
)

const (
	defaultThesisMaxReviews = 100
	maxThesisPageSize       = 500
)

type ThesisEvaluationConfig struct {
	DefaultMaxReviews             int
	MaxPageSize                   int
	DesiredHoldingPeriod          string
	BrokenScoreThreshold          float64
	WeakCoreSectionThreshold      float64
	StrongCoreSectionThreshold    float64
	UnderReviewTotalDropThreshold float64
	UnderReviewCoreDropThreshold  float64
	UnderReviewMgmtDropThreshold  float64
	RequireWrittenThesisForBuy    bool
	SellOnThesisBreak             bool
}

type ThesisEvaluationOption func(*thesisEvaluationService)

func WithThesisEvaluationConfig(config ThesisEvaluationConfig) ThesisEvaluationOption {
	return func(service *thesisEvaluationService) {
		if config.DefaultMaxReviews > 0 {
			service.config.DefaultMaxReviews = config.DefaultMaxReviews
		}
		if config.MaxPageSize > 0 {
			service.config.MaxPageSize = config.MaxPageSize
		}
		if config.DesiredHoldingPeriod != "" {
			service.config.DesiredHoldingPeriod = config.DesiredHoldingPeriod
		}
		if config.BrokenScoreThreshold > 0 {
			service.config.BrokenScoreThreshold = config.BrokenScoreThreshold
		}
		if config.WeakCoreSectionThreshold > 0 {
			service.config.WeakCoreSectionThreshold = config.WeakCoreSectionThreshold
		}
		if config.StrongCoreSectionThreshold > 0 {
			service.config.StrongCoreSectionThreshold = config.StrongCoreSectionThreshold
		}
		if config.UnderReviewTotalDropThreshold > 0 {
			service.config.UnderReviewTotalDropThreshold = config.UnderReviewTotalDropThreshold
		}
		if config.UnderReviewCoreDropThreshold > 0 {
			service.config.UnderReviewCoreDropThreshold = config.UnderReviewCoreDropThreshold
		}
		if config.UnderReviewMgmtDropThreshold > 0 {
			service.config.UnderReviewMgmtDropThreshold = config.UnderReviewMgmtDropThreshold
		}
		if config.RequireWrittenThesisForBuy {
			service.config.RequireWrittenThesisForBuy = true
		}
		if config.SellOnThesisBreak {
			service.config.SellOnThesisBreak = true
		}
	}
}

func WithThesisEvaluationPolicy(requireWrittenThesisForBuy bool, sellOnThesisBreak bool) ThesisEvaluationOption {
	return func(service *thesisEvaluationService) {
		service.config.RequireWrittenThesisForBuy = requireWrittenThesisForBuy
		service.config.SellOnThesisBreak = sellOnThesisBreak
	}
}

func WithThesisEvaluationClock(clock servicecommon.ClockPort) ThesisEvaluationOption {
	return func(service *thesisEvaluationService) {
		if clock != nil {
			service.now = clock.Now
		}
	}
}

type thesisEvaluationService struct {
	theses  platformrepo.InvestmentThesisRepository
	reviews platformrepo.CompanyReviewRepository
	config  ThesisEvaluationConfig
	now     func() time.Time
}

var _ ThesisEvaluationService = (*thesisEvaluationService)(nil)

func NewThesisEvaluationService(
	theses platformrepo.InvestmentThesisRepository,
	reviews platformrepo.CompanyReviewRepository,
	options ...ThesisEvaluationOption,
) ThesisEvaluationService {
	service := &thesisEvaluationService{
		theses:  theses,
		reviews: reviews,
		config: ThesisEvaluationConfig{
			DefaultMaxReviews:             defaultThesisMaxReviews,
			MaxPageSize:                   maxThesisPageSize,
			DesiredHoldingPeriod:          "3-10 years",
			BrokenScoreThreshold:          5.5,
			WeakCoreSectionThreshold:      5.5,
			StrongCoreSectionThreshold:    7.5,
			UnderReviewTotalDropThreshold: 1.0,
			UnderReviewCoreDropThreshold:  1.5,
			UnderReviewMgmtDropThreshold:  1.0,
			RequireWrittenThesisForBuy:    true,
			SellOnThesisBreak:             true,
		},
		now: time.Now,
	}
	for _, option := range options {
		if option != nil {
			option(service)
		}
	}
	if service.config.DefaultMaxReviews <= 0 {
		service.config.DefaultMaxReviews = defaultThesisMaxReviews
	}
	if service.config.MaxPageSize <= 0 {
		service.config.MaxPageSize = maxThesisPageSize
	}
	if service.config.DesiredHoldingPeriod == "" {
		service.config.DesiredHoldingPeriod = "3-10 years"
	}
	if service.config.BrokenScoreThreshold <= 0 {
		service.config.BrokenScoreThreshold = 5.5
	}
	if service.config.WeakCoreSectionThreshold <= 0 {
		service.config.WeakCoreSectionThreshold = 5.5
	}
	if service.config.StrongCoreSectionThreshold <= 0 {
		service.config.StrongCoreSectionThreshold = 7.5
	}
	if service.config.UnderReviewTotalDropThreshold <= 0 {
		service.config.UnderReviewTotalDropThreshold = 1.0
	}
	if service.config.UnderReviewCoreDropThreshold <= 0 {
		service.config.UnderReviewCoreDropThreshold = 1.5
	}
	if service.config.UnderReviewMgmtDropThreshold <= 0 {
		service.config.UnderReviewMgmtDropThreshold = 1.0
	}
	return service
}

func (service *thesisEvaluationService) EvaluateThesis(
	ctx context.Context,
	request EvaluateThesisRequest,
) (*EvaluateThesisResult, error) {
	if err := request.Validate(); err != nil {
		return nil, err
	}

	outcome, err := service.evaluateOneThesis(ctx, thesisEvaluationOptions{
		ReviewID:      request.ReviewID,
		CompanyID:     request.CompanyID,
		WorkflowRunID: request.WorkflowRunID,
		BookType:      request.BookType,
		DryRun:        request.DryRun,
		Force:         request.Force,
		InitiatedBy:   request.InitiatedBy,
		CorrelationID: request.CorrelationID,
	})
	if err != nil {
		return nil, err
	}
	return buildSingleThesisResult(outcome), nil
}

func (service *thesisEvaluationService) EvaluateThesisForWorkflow(
	ctx context.Context,
	request EvaluateThesisForWorkflowRequest,
) (*EvaluateThesisForWorkflowResult, error) {
	if err := request.Validate(); err != nil {
		return nil, err
	}
	if service.reviews == nil {
		return nil, fmt.Errorf("evaluate thesis workflow %s: review repository is required", request.WorkflowRunID.Hex())
	}

	startedAt := service.now().UTC()
	reviews, hasMore, err := service.listWorkflowReviews(ctx, request)
	if err != nil {
		return nil, err
	}

	result := &EvaluateThesisForWorkflowResult{WorkflowRunID: request.WorkflowRunID}
	for _, review := range reviews {
		if review == nil {
			continue
		}
		outcome, err := service.evaluateOneThesis(ctx, thesisEvaluationOptions{
			ReviewID:        review.ID,
			CompanyID:       review.CompanyID,
			WorkflowRunID:   request.WorkflowRunID,
			BookType:        request.BookType,
			DryRun:          request.DryRun,
			Force:           request.Force,
			InitiatedBy:     request.InitiatedBy,
			CorrelationID:   request.CorrelationID,
			PreloadedReview: review,
		})
		if err != nil {
			result.FailedReviewIDs = append(result.FailedReviewIDs, review.ID)
			result.PartialFailures = append(result.PartialFailures, thesisPartialFailure(review, request.WorkflowRunID, err))
			continue
		}
		mergeThesisOutcome(result, outcome)
	}

	result.UpdatedThesisIDs = uniqueObjectIDs(result.UpdatedThesisIDs)
	result.FailedReviewIDs = uniqueObjectIDs(result.FailedReviewIDs)
	completedAt := service.now().UTC()
	result.Summary = buildWorkflowThesisSummary(
		len(reviews),
		result.Summary.CreatedCount,
		result.Summary.UpdatedCount,
		result.Summary.BrokenCount,
		result.Summary.UnderReviewCount,
		result.Summary.SkippedCount,
		len(result.PartialFailures),
		request.DryRun,
		hasMore,
		startedAt,
		completedAt,
	)
	result.ThesisCreated = result.Summary.CreatedCount > 0
	result.ThesisUpdated = result.Summary.UpdatedCount > 0
	result.ThesisBroken = result.Summary.BrokenCount > 0
	result.ThesisUnderReview = result.Summary.UnderReviewCount > 0
	return result, nil
}

func (service *thesisEvaluationService) listWorkflowReviews(
	ctx context.Context,
	request EvaluateThesisForWorkflowRequest,
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
		return nil, false, fmt.Errorf("evaluate thesis workflow %s: list reviews: %w", request.WorkflowRunID.Hex(), err)
	}
	if list == nil {
		return nil, false, nil
	}
	return list.Items, list.Page.HasMore, nil
}

func (service *thesisEvaluationService) evaluateOneThesis(
	ctx context.Context,
	options thesisEvaluationOptions,
) (thesisOneOutcome, error) {
	review, existing, active, err := service.loadThesisContext(ctx, options)
	if err != nil {
		return thesisOneOutcome{}, err
	}

	outcome := thesisOneOutcome{
		ReviewID:  review.ID,
		CompanyID: review.CompanyID,
		DryRun:    options.DryRun,
	}
	if existing != nil {
		outcome.ThesisID = existing.ID
		outcome.StatusBefore = existing.ThesisStatus
	}

	if err := validateReviewForThesis(review, options); err != nil {
		if isThesisSkip(err) {
			outcome.Skipped = true
			outcome.Summary = thesisSkipSummary(err)
			return outcome, nil
		}
		return thesisOneOutcome{}, err
	}
	if existing != nil && existing.LastUpdatedFromReviewID == review.ID && !options.Force {
		outcome.Skipped = true
		outcome.StatusAfter = existing.ThesisStatus
		outcome.HealthScore = existing.ThesisHealthScore
		outcome.Summary = "thesis already reflects this review"
		return outcome, nil
	}

	decision := determineThesisDecision(review, existing, service.config)
	outcome.StatusAfter = decision.Status
	outcome.HealthScore = decision.HealthScore
	outcome.Broken = decision.Status == domaincommon.ThesisStatusBroken
	outcome.UnderReview = decision.Status == domaincommon.ThesisStatusUnderReview
	outcome.Summary = decision.Summary

	switch {
	case shouldCreateThesis(review, existing, service.config):
		if options.DryRun {
			outcome.Created = true
			return outcome, nil
		}
		created, err := service.createThesis(ctx, review, decision)
		if err != nil {
			return thesisOneOutcome{}, fmt.Errorf("evaluate thesis review %s company %s: create thesis: %w", review.ID.Hex(), review.CompanyID.Hex(), err)
		}
		outcome.Created = true
		outcome.ThesisID = created.ID
		return outcome, nil
	case shouldUpdateThesis(review, existing):
		if options.DryRun {
			outcome.Updated = true
			return outcome, nil
		}
		updated, err := service.saveUpdatedThesisVersion(ctx, review, existing, active, decision, options)
		if err != nil {
			return thesisOneOutcome{}, fmt.Errorf("evaluate thesis review %s company %s: update thesis: %w", review.ID.Hex(), review.CompanyID.Hex(), err)
		}
		outcome.Updated = true
		outcome.ThesisID = updated.ID
		outcome.StatusAfter = updated.ThesisStatus
		outcome.HealthScore = updated.ThesisHealthScore
		return outcome, nil
	default:
		outcome.Skipped = true
		outcome.Summary = "review does not require thesis creation or update"
		return outcome, nil
	}
}

func (service *thesisEvaluationService) createThesis(
	ctx context.Context,
	review *domainreview.CompanyReview,
	decision thesisDecision,
) (*domainthesis.InvestmentThesis, error) {
	if service.theses == nil {
		return nil, fmt.Errorf("thesis repository is required")
	}
	now := service.now().UTC()
	candidate := buildNewThesisFromReview(review, decision, service.config, now)
	return service.theses.Create(ctx, candidate)
}

func (service *thesisEvaluationService) saveUpdatedThesisVersion(
	ctx context.Context,
	review *domainreview.CompanyReview,
	existing *domainthesis.InvestmentThesis,
	active *domainthesis.InvestmentThesis,
	decision thesisDecision,
	options thesisEvaluationOptions,
) (*domainthesis.InvestmentThesis, error) {
	if service.theses == nil {
		return nil, fmt.Errorf("thesis repository is required")
	}
	now := service.now().UTC()
	candidate := buildUpdatedThesisFromReview(existing, review, decision, now)
	expectedPreviousVersion := existing.ThesisVersion

	needsActiveSwap := active != nil &&
		!active.ID.IsZero() &&
		active.ID == existing.ID &&
		decision.Status == domaincommon.ThesisStatusActive
	if needsActiveSwap {
		candidate.ThesisStatus = domaincommon.ThesisStatusUnderReview
	}

	saved, err := service.theses.SaveNewVersion(ctx, candidate, platformrepo.ThesisVersionCreateOptions{
		ExpectedPreviousVersion: &expectedPreviousVersion,
	})
	if err != nil {
		return nil, err
	}

	if active != nil && !active.ID.IsZero() {
		archived, err := service.archivePreviousActiveThesis(ctx, active, review, options)
		if err != nil {
			return nil, err
		}
		if saved.ID == archived.ID {
			return saved, nil
		}
	}

	if needsActiveSwap {
		changeSummary := candidate.ThesisChangeSummary
		activated, err := service.theses.UpdateStatus(ctx, saved.ID, platformrepo.ThesisStatusPatch{
			NextStatus:              domaincommon.ThesisStatusActive,
			LastUpdatedFromReviewID: &review.ID,
			ThesisChangeSummary:     &changeSummary,
			ExpectedCurrentStatuses: []domaincommon.ThesisStatus{domaincommon.ThesisStatusUnderReview},
			Mutation: mutationMetadata(
				service.now().UTC(),
				options.InitiatedBy,
				"activate new thesis version after archiving prior active version",
			),
		})
		if err != nil {
			return nil, err
		}
		return activated, nil
	}

	return saved, nil
}

func (service *thesisEvaluationService) archivePreviousActiveThesis(
	ctx context.Context,
	active *domainthesis.InvestmentThesis,
	review *domainreview.CompanyReview,
	options thesisEvaluationOptions,
) (*domainthesis.InvestmentThesis, error) {
	if active == nil || active.ID.IsZero() || active.ThesisStatus != domaincommon.ThesisStatusActive {
		return active, nil
	}
	changeSummary := fmt.Sprintf("Archived active version after thesis version update from review %s.", review.ID.Hex())
	archived, err := service.theses.UpdateStatus(ctx, active.ID, platformrepo.ThesisStatusPatch{
		NextStatus:              domaincommon.ThesisStatusArchived,
		LastUpdatedFromReviewID: &review.ID,
		ThesisChangeSummary:     &changeSummary,
		ExpectedCurrentStatuses: []domaincommon.ThesisStatus{domaincommon.ThesisStatusActive},
		Mutation: mutationMetadata(
			service.now().UTC(),
			options.InitiatedBy,
			"archive prior active thesis version",
		),
	})
	if err != nil {
		return nil, fmt.Errorf("archive previous active thesis %s: %w", active.ID.Hex(), err)
	}
	return archived, nil
}

func (service *thesisEvaluationService) maxReviews(requested int) int {
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

func mutationMetadata(at time.Time, actor string, reason string) platformrepo.MutationMetadata {
	return platformrepo.MutationMetadata{
		OccurredAt: at.UTC(),
		Actor:      actor,
		Reason:     reason,
	}
}

func isRepositoryNotFound(err error) bool {
	return errors.Is(err, platformrepo.ErrNotFound)
}
