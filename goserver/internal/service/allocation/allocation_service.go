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

type CapitalAllocatorOption func(*capitalAllocationService)

func WithCapitalAllocatorConfig(config CapitalAllocationConfig) CapitalAllocatorOption {
	return func(service *capitalAllocationService) {
		service.config = mergeCapitalAllocationConfig(service.config, config)
	}
}

func WithCapitalAllocatorClock(clock servicecommon.ClockPort) CapitalAllocatorOption {
	return func(service *capitalAllocationService) {
		if clock != nil {
			service.now = clock.Now
		}
	}
}

type capitalAllocationService struct {
	reviews     platformrepo.CompanyReviewRepository
	positions   platformrepo.CurrentPositionRepository
	allocations platformrepo.CapitalAllocationRunRepository
	candidates  CapitalCandidateBuilderService
	config      CapitalAllocationConfig
	now         func() time.Time
}

var _ CapitalAllocationService = (*capitalAllocationService)(nil)

func NewCapitalAllocationService(
	reviews platformrepo.CompanyReviewRepository,
	positions platformrepo.CurrentPositionRepository,
	allocations platformrepo.CapitalAllocationRunRepository,
	candidates CapitalCandidateBuilderService,
	options ...CapitalAllocatorOption,
) CapitalAllocationService {
	service := &capitalAllocationService{
		reviews:     reviews,
		positions:   positions,
		allocations: allocations,
		candidates:  candidates,
		config:      defaultCapitalAllocationConfig(),
		now:         time.Now,
	}
	for _, option := range options {
		if option != nil {
			option(service)
		}
	}
	service.config = normalizeCapitalAllocationConfig(service.config)
	return service
}

func (service *capitalAllocationService) AllocateCapital(
	ctx context.Context,
	request AllocateCapitalRequest,
) (*AllocateCapitalResult, error) {
	if err := request.Validate(); err != nil {
		return nil, err
	}
	if !request.DryRun && service.allocations == nil {
		return nil, fmt.Errorf("allocate capital workflow=%s: allocation repository is required", request.WorkflowRunID.Hex())
	}

	startedAt := service.now().UTC()
	deployableCash := computeDeployableCash(request)
	candidates, preBlocked, partialFailures, err := service.resolveAllocationCandidates(ctx, request)
	if err != nil {
		return nil, err
	}
	rankCandidates(candidates)

	plan := allocateCapitalAcrossCandidates(candidates, deployableCash, service.config)
	plan.Blocked = append(preBlocked, plan.Blocked...)
	items := buildAllocationItems(plan)
	allocatedTotal := sumAllocatedItems(items)
	unallocatedCash := roundMoney(mathMax(deployableCash-allocatedTotal, 0))

	run := buildAllocationRun(request, deployableCash, allocatedTotal, unallocatedCash, items, service.now().UTC())
	if err := run.Validate(); err != nil {
		return nil, fmt.Errorf("build allocation run workflow=%s: %w", request.WorkflowRunID.Hex(), err)
	}

	var runID primitive.ObjectID
	if !request.DryRun {
		created, err := service.allocations.Create(ctx, run)
		if err != nil {
			return nil, fmt.Errorf("persist allocation run workflow=%s allocationDate=%s: %w", request.WorkflowRunID.Hex(), request.AllocationDate.Format(time.RFC3339), err)
		}
		if created != nil {
			runID = created.ID
		}
	}

	allocated := allocatedCandidatesFromPlan(plan, allocatedTotal)
	blocked := blockedCandidatesFromPlan(plan)
	completedAt := service.now().UTC()
	summary := buildCapitalAllocationSummary(
		"allocate_capital",
		len(candidates)+len(preBlocked),
		len(allocated),
		len(blocked),
		len(partialFailures),
		allocatedTotal,
		unallocatedCash,
		request.DryRun,
		startedAt,
		completedAt,
	)

	return &AllocateCapitalResult{
		WorkflowRunID:          request.WorkflowRunID,
		CapitalAllocationRunID: runID,
		AllocatedCandidates:    allocated,
		BlockedCandidates:      blocked,
		UnallocatedCash:        unallocatedCash,
		PartialFailures:        partialFailures,
		Summary:                summary,
	}, nil
}

func (service *capitalAllocationService) resolveAllocationCandidates(
	ctx context.Context,
	request AllocateCapitalRequest,
) ([]capitalCandidate, []BlockedCapitalCandidate, []servicecommon.PartialFailure, error) {
	refs := request.CandidateRefs
	partialFailures := make([]servicecommon.PartialFailure, 0)

	if len(refs) == 0 {
		if service.candidates == nil {
			return nil, nil, nil, nil
		}
		result, err := service.candidates.BuildCapitalCandidates(ctx, BuildCapitalCandidatesRequest{
			WorkflowRunID: request.WorkflowRunID,
			BookType:      domaincommon.BookTypeInvesting,
			AsOfDate:      request.AllocationDate,
			MaxCandidates: service.config.DefaultMaxCandidates,
			DryRun:        true,
			Force:         request.Force,
			InitiatedBy:   request.InitiatedBy,
			CorrelationID: request.CorrelationID,
		})
		if err != nil {
			return nil, nil, nil, err
		}
		if result != nil {
			refs = result.RankedCandidateRefs
			partialFailures = append(partialFailures, result.PartialFailures...)
		}
	}

	candidates := make([]capitalCandidate, 0, len(refs))
	blocked := make([]BlockedCapitalCandidate, 0)
	seenReviews := make(map[primitive.ObjectID]struct{}, len(refs))
	for _, ref := range refs {
		if ref.ReviewID.IsZero() {
			blocked = append(blocked, BlockedCapitalCandidate{Candidate: ref, ConstraintReason: "review_id_required"})
			continue
		}
		if _, exists := seenReviews[ref.ReviewID]; exists {
			blocked = append(blocked, BlockedCapitalCandidate{Candidate: ref, ConstraintReason: "duplicate_candidate"})
			continue
		}
		seenReviews[ref.ReviewID] = struct{}{}

		candidate, reason, failure := service.hydrateAllocationCandidate(ctx, ref, request.WorkflowRunID)
		if failure != nil {
			partialFailures = append(partialFailures, *failure)
		}
		if reason != "" {
			blocked = append(blocked, BlockedCapitalCandidate{Candidate: ref, ConstraintReason: reason})
			continue
		}
		candidates = append(candidates, candidate)
	}
	return candidates, blocked, partialFailures, nil
}

func (service *capitalAllocationService) hydrateAllocationCandidate(
	ctx context.Context,
	ref CapitalCandidateRef,
	workflowRunID primitive.ObjectID,
) (capitalCandidate, string, *servicecommon.PartialFailure) {
	if reason := firstBlockingConstraint(ref.ConstraintReasons); reason != "" {
		return capitalCandidate{}, reason, nil
	}
	if service.reviews == nil {
		return capitalCandidate{}, "review_repository_required", nil
	}

	review, err := service.reviews.GetByID(ctx, ref.ReviewID)
	if err != nil {
		failure := candidatePartialFailure(workflowRunID, ref.ReviewID, ref.CompanyID, "review_lookup_failed", err)
		return capitalCandidate{}, "review_lookup_failed", &failure
	}
	candidate, reason, failure := service.candidateFromReviewForAllocation(ctx, review, ref, workflowRunID)
	if failure != nil {
		return capitalCandidate{}, reason, failure
	}
	if candidate.Ref.PriorityRank == 0 {
		candidate.Ref.PriorityRank = ref.PriorityRank
	}
	return candidate, reason, nil
}

func (service *capitalAllocationService) candidateFromReviewForAllocation(
	ctx context.Context,
	review *domainreview.CompanyReview,
	ref CapitalCandidateRef,
	workflowRunID primitive.ObjectID,
) (capitalCandidate, string, *servicecommon.PartialFailure) {
	if review == nil {
		return capitalCandidate{}, "review_lookup_failed", nil
	}
	score := extractCandidateScoreContext(review, service.config)
	position, positionErr := loadPositionContext(ctx, service.positions, review, service.config)
	if positionErr != nil {
		failure := candidatePartialFailure(workflowRunID, review.ID, review.CompanyID, "position_lookup_failed", positionErr)
		return capitalCandidate{}, "position_lookup_failed", &failure
	}

	candidateRef := baseCandidateRef(review)
	candidateRef.PriorityRank = ref.PriorityRank
	candidateRef.PriorityScore = ref.PriorityScore
	if candidateRef.PriorityScore <= 0 {
		candidateRef.PriorityScore = computeCandidatePriorityScore(review, score, position, service.config)
	}
	candidateRef.RecommendedTargetPct = position.TargetPct

	eligibilityConfig := service.config
	eligibilityConfig.RequireActiveThesis = false
	eligibility := evaluateCandidateEligibility(review, score, position, nil, false, eligibilityConfig)
	if !eligibility.Eligible {
		return capitalCandidate{}, eligibility.Code, nil
	}

	return capitalCandidate{
		Ref:           candidateRef,
		Review:        review,
		Score:         score,
		Position:      position,
		ReviewDate:    review.ReviewDate,
		AsOfDate:      service.now().UTC(),
		RankingReason: buildCandidateReason(review, score, position, candidateRef.PriorityScore),
	}, "", nil
}
