package allocation

import (
	"context"
	"fmt"
	"time"

	domaincommon "goserver/internal/domain/common"
	domainreview "goserver/internal/domain/review"
	platformrepo "goserver/internal/platform/repository"
	servicecommon "goserver/internal/service/common"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

const (
	defaultCapitalCandidateMaxCandidates = 100
	maxCapitalCandidatePageSize          = 500
)

type CapitalCandidateBuilderOption func(*capitalCandidateBuilderService)

func WithCapitalCandidateBuilderConfig(config CapitalAllocationConfig) CapitalCandidateBuilderOption {
	return func(service *capitalCandidateBuilderService) {
		service.config = mergeCapitalAllocationConfig(service.config, config)
	}
}

func WithCapitalCandidateBuilderClock(clock servicecommon.ClockPort) CapitalCandidateBuilderOption {
	return func(service *capitalCandidateBuilderService) {
		if clock != nil {
			service.now = clock.Now
		}
	}
}

type capitalCandidateBuilderService struct {
	reviews   platformrepo.CompanyReviewRepository
	positions platformrepo.CurrentPositionRepository
	theses    platformrepo.InvestmentThesisRepository
	config    CapitalAllocationConfig
	now       func() time.Time
}

var _ CapitalCandidateBuilderService = (*capitalCandidateBuilderService)(nil)

func NewCapitalCandidateBuilderService(
	reviews platformrepo.CompanyReviewRepository,
	positions platformrepo.CurrentPositionRepository,
	theses platformrepo.InvestmentThesisRepository,
	options ...CapitalCandidateBuilderOption,
) CapitalCandidateBuilderService {
	service := &capitalCandidateBuilderService{
		reviews:   reviews,
		positions: positions,
		theses:    theses,
		config:    defaultCapitalAllocationConfig(),
		now:       time.Now,
	}
	for _, option := range options {
		if option != nil {
			option(service)
		}
	}
	service.config = normalizeCapitalAllocationConfig(service.config)
	return service
}

func (service *capitalCandidateBuilderService) BuildCapitalCandidates(
	ctx context.Context,
	request BuildCapitalCandidatesRequest,
) (*BuildCapitalCandidatesResult, error) {
	if err := request.Validate(); err != nil {
		return nil, err
	}
	if request.BookType != "" && request.BookType != domaincommon.BookTypeInvesting {
		return nil, fmt.Errorf("%w: capital candidates are only supported for investing book workflows", servicecommon.ErrInvalidServiceRequest)
	}
	if service.reviews == nil {
		return nil, fmt.Errorf("build capital candidates %s: review repository is required", request.WorkflowRunID.Hex())
	}

	startedAt := service.now().UTC()
	candidates, skipped, ineligible, partialFailures, hasMore, err := service.buildCandidatesForWorkflow(ctx, request)
	if err != nil {
		return nil, err
	}

	rankCandidates(candidates)
	maxCandidates := service.maxCandidates(request.MaxCandidates)
	if len(candidates) > maxCandidates {
		for _, candidate := range candidates[maxCandidates:] {
			skipped = append(skipped, SkippedCapitalCandidate{
				Candidate: candidate.Ref,
				Code:      "candidate_limit_reached",
				Reason:    fmt.Sprintf("candidate omitted because maxCandidates=%d was reached", maxCandidates),
			})
		}
		candidates = candidates[:maxCandidates]
	}

	refs := make([]CapitalCandidateRef, 0, len(candidates))
	for index := range candidates {
		candidates[index].Ref.PriorityRank = index + 1
		candidates[index].Ref.ConstraintReasons = nil
		refs = append(refs, candidates[index].Ref)
	}

	completedAt := service.now().UTC()
	summary := buildCandidateSummary(
		"build_capital_candidates",
		len(skipped)+len(ineligible)+len(candidates),
		len(candidates),
		len(skipped)+len(ineligible),
		len(partialFailures),
		request.DryRun,
		startedAt,
		completedAt,
	)
	if hasMore {
		summary.Message = appendSummaryMessage(summary.Message, "more workflow reviews may be available beyond the configured page size")
	}

	return &BuildCapitalCandidatesResult{
		WorkflowRunID:        request.WorkflowRunID,
		CandidateCount:       len(refs),
		RankedCandidateRefs:  refs,
		SkippedCandidates:    skipped,
		IneligibleCandidates: ineligible,
		PartialFailures:      partialFailures,
		Summary:              summary,
	}, nil
}

func (service *capitalCandidateBuilderService) buildCandidatesForWorkflow(
	ctx context.Context,
	request BuildCapitalCandidatesRequest,
) ([]capitalCandidate, []SkippedCapitalCandidate, []SkippedCapitalCandidate, []servicecommon.PartialFailure, bool, error) {
	options := platformrepo.CompanyReviewListOptions{
		Pagination: platformrepo.PageOptions{PageSize: service.config.MaxPageSize},
		Sort: platformrepo.CompanyReviewSortOption{
			By:    platformrepo.CompanyReviewSortByReviewDate,
			Order: platformrepo.SortOrderDescending,
		},
	}
	result, err := service.reviews.ListByWorkflowRun(ctx, request.WorkflowRunID, options)
	if err != nil {
		return nil, nil, nil, nil, false, fmt.Errorf("list workflow reviews for capital candidates workflow=%s: %w", request.WorkflowRunID.Hex(), err)
	}
	if result == nil || len(result.Items) == 0 {
		return nil, nil, nil, nil, false, nil
	}

	asOf := request.AsOfDate
	if asOf.IsZero() {
		asOf = service.now().UTC()
	}

	candidates := make([]capitalCandidate, 0, len(result.Items))
	skipped := make([]SkippedCapitalCandidate, 0)
	ineligible := make([]SkippedCapitalCandidate, 0)
	partialFailures := make([]servicecommon.PartialFailure, 0)

	for _, item := range result.Items {
		candidate, skip, failures := service.evaluateReviewAsCandidate(ctx, item, request.WorkflowRunID, asOf)
		partialFailures = append(partialFailures, failures...)
		if skip != nil {
			if isCandidateIneligibleCode(skip.Code) {
				ineligible = append(ineligible, *skip)
			} else {
				skipped = append(skipped, *skip)
			}
			continue
		}
		candidates = append(candidates, candidate)
	}

	return candidates, skipped, ineligible, partialFailures, result.Page.HasMore, nil
}

func (service *capitalCandidateBuilderService) evaluateReviewAsCandidate(
	ctx context.Context,
	review *domainreview.CompanyReview,
	workflowRunID primitive.ObjectID,
	asOf time.Time,
) (capitalCandidate, *SkippedCapitalCandidate, []servicecommon.PartialFailure) {
	if review == nil {
		return capitalCandidate{}, &SkippedCapitalCandidate{
			Code:   "nil_review",
			Reason: "workflow review entry was nil",
		}, nil
	}

	ref := baseCandidateRef(review)
	score := extractCandidateScoreContext(review, service.config)
	position, positionErr := loadPositionContext(ctx, service.positions, review, service.config)
	thesis, thesisLoaded, thesisErr := loadThesisContext(ctx, service.theses, review.CompanyID)

	failures := make([]servicecommon.PartialFailure, 0, 2)
	if positionErr != nil {
		failures = append(failures, candidatePartialFailure(workflowRunID, review.ID, review.CompanyID, "position_lookup_failed", positionErr))
	}
	if thesisErr != nil {
		failures = append(failures, candidatePartialFailure(workflowRunID, review.ID, review.CompanyID, "thesis_lookup_failed", thesisErr))
	}

	eligibility := evaluateCandidateEligibility(review, score, position, thesis, thesisLoaded, service.config)
	ref.ActionType = candidateAction(review)
	ref.CurrentBucket = candidateBucket(review)
	ref.PriorityScore = computeCandidatePriorityScore(review, score, position, service.config)
	ref.RecommendedTargetPct = position.TargetPct
	ref.ConstraintReasons = eligibility.Reasons

	if !eligibility.Eligible {
		return capitalCandidate{}, &SkippedCapitalCandidate{
			Candidate: ref,
			Code:      eligibility.Code,
			Reason:    joinReasons(eligibility.Reasons),
		}, failures
	}

	return capitalCandidate{
		Ref:           ref,
		Review:        review,
		Score:         score,
		Position:      position,
		ReviewDate:    review.ReviewDate,
		AsOfDate:      asOf,
		RankingReason: buildCandidateReason(review, score, position, ref.PriorityScore),
	}, nil, failures
}

func (service *capitalCandidateBuilderService) maxCandidates(requested int) int {
	if requested > 0 {
		if requested > service.config.MaxPageSize {
			return service.config.MaxPageSize
		}
		return requested
	}
	return service.config.DefaultMaxCandidates
}
