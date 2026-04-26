package projection

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

const defaultProjectionMaxPageSize = 500

type ProjectionUpdateConfig struct {
	MaxPageSize int
}

type ProjectionUpdateOption func(*projectionUpdateService)

func WithProjectionUpdateConfig(config ProjectionUpdateConfig) ProjectionUpdateOption {
	return func(service *projectionUpdateService) {
		if config.MaxPageSize > 0 {
			service.config.MaxPageSize = config.MaxPageSize
		}
	}
}

func WithProjectionUpdateClock(clock servicecommon.ClockPort) ProjectionUpdateOption {
	return func(service *projectionUpdateService) {
		if clock != nil {
			service.now = clock.Now
		}
	}
}

type projectionUpdateService struct {
	reviews     platformrepo.CompanyReviewRepository
	positions   platformrepo.CurrentPositionRepository
	theses      platformrepo.InvestmentThesisRepository
	allocations platformrepo.CapitalAllocationRunRepository
	config      ProjectionUpdateConfig
	now         func() time.Time
}

var _ ProjectionUpdateService = (*projectionUpdateService)(nil)

func NewProjectionUpdateService(
	reviews platformrepo.CompanyReviewRepository,
	positions platformrepo.CurrentPositionRepository,
	theses platformrepo.InvestmentThesisRepository,
	allocations platformrepo.CapitalAllocationRunRepository,
	options ...ProjectionUpdateOption,
) ProjectionUpdateService {
	service := &projectionUpdateService{
		reviews:     reviews,
		positions:   positions,
		theses:      theses,
		allocations: allocations,
		config: ProjectionUpdateConfig{
			MaxPageSize: defaultProjectionMaxPageSize,
		},
		now: time.Now,
	}
	for _, option := range options {
		if option != nil {
			option(service)
		}
	}
	if service.config.MaxPageSize <= 0 || service.config.MaxPageSize > defaultProjectionMaxPageSize {
		service.config.MaxPageSize = defaultProjectionMaxPageSize
	}
	return service
}

func (service *projectionUpdateService) UpdateProjections(
	ctx context.Context,
	request UpdateProjectionsRequest,
) (*UpdateProjectionsResult, error) {
	if err := request.Validate(); err != nil {
		return nil, err
	}
	if service.reviews == nil {
		return nil, fmt.Errorf("update projections: review repository is required")
	}

	startedAt := service.now().UTC()
	targets := normalizeProjectionTargets(request.Targets)
	result := &UpdateProjectionsResult{WorkflowRunID: request.WorkflowRunID}

	reviews, skipped, partialFailures, hasMore, err := service.resolveProjectionReviews(ctx, request)
	if err != nil {
		return nil, err
	}
	result.SkippedRefs = append(result.SkippedRefs, skipped...)
	result.PartialFailures = append(result.PartialFailures, partialFailures...)

	for _, review := range reviews {
		outcome := service.updateProjectionForReview(ctx, review, request, targets)
		result.UpdatedRefs = append(result.UpdatedRefs, outcome.Updated...)
		result.SkippedRefs = append(result.SkippedRefs, outcome.Skipped...)
		result.PartialFailures = append(result.PartialFailures, outcome.Failures...)
	}

	completedAt := service.now().UTC()
	result.Summary = buildProjectionSummary(
		len(reviews),
		result.UpdatedRefs,
		result.SkippedRefs,
		result.PartialFailures,
		request.DryRun,
		startedAt,
		completedAt,
	)
	if hasMore {
		result.Summary.Message = appendProjectionSummaryMessage(result.Summary.Message, "more workflow reviews may be available beyond the configured page size")
	}
	return result, nil
}

func (service *projectionUpdateService) resolveProjectionReviews(
	ctx context.Context,
	request UpdateProjectionsRequest,
) ([]*domainreview.CompanyReview, []ProjectionUpdateRef, []servicecommon.PartialFailure, bool, error) {
	resolved := make([]*domainreview.CompanyReview, 0)
	skipped := make([]ProjectionUpdateRef, 0)
	failures := make([]servicecommon.PartialFailure, 0)
	seen := make(map[primitive.ObjectID]struct{})
	hasMore := false

	if !request.WorkflowRunID.IsZero() {
		reviews, workflowHasMore, err := service.listWorkflowReviews(ctx, request.WorkflowRunID)
		if err != nil {
			return nil, nil, nil, false, fmt.Errorf("resolve projection reviews workflow=%s: %w", request.WorkflowRunID.Hex(), err)
		}
		hasMore = hasMore || workflowHasMore
		for _, review := range reviews {
			if review == nil {
				continue
			}
			if !reviewEligibleForProjection(review, request.BookType) {
				skipped = append(skipped, skippedReviewRef(review, ProjectionTargetReview, "review_not_current_final"))
				continue
			}
			addResolvedReview(&resolved, seen, review)
		}
	}

	for _, reviewID := range request.ReviewIDs {
		review, err := service.reviews.GetByID(ctx, reviewID)
		if err != nil {
			failures = append(failures, projectionPartialFailure(request.WorkflowRunID, reviewID, primitive.ObjectID{}, request.BookType, "review_lookup_failed", err))
			continue
		}
		if !reviewEligibleForProjection(review, request.BookType) {
			skipped = append(skipped, skippedReviewRef(review, ProjectionTargetReview, "review_not_current_final"))
			continue
		}
		addResolvedReview(&resolved, seen, review)
	}

	for _, companyID := range request.CompanyIDs {
		for _, bookType := range projectionBookTypes(request.BookType) {
			review, err := service.reviews.GetLatestByCompanyAndBook(ctx, companyID, bookType, platformrepo.LatestCompanyReviewOptions{
				FinalizedOnly:     true,
				IncludeSuperseded: false,
			})
			if err != nil {
				if isRepositoryNotFound(err) {
					skipped = append(skipped, ProjectionUpdateRef{
						Target:    ProjectionTargetReview,
						CompanyID: companyID,
					})
					continue
				}
				failures = append(failures, projectionPartialFailure(request.WorkflowRunID, primitive.ObjectID{}, companyID, bookType, "latest_review_lookup_failed", err))
				continue
			}
			if !reviewEligibleForProjection(review, bookType) {
				skipped = append(skipped, skippedReviewRef(review, ProjectionTargetReview, "review_not_current_final"))
				continue
			}
			addResolvedReview(&resolved, seen, review)
		}
	}

	return resolved, skipped, failures, hasMore, nil
}

func (service *projectionUpdateService) listWorkflowReviews(
	ctx context.Context,
	workflowRunID primitive.ObjectID,
) ([]*domainreview.CompanyReview, bool, error) {
	result, err := service.reviews.ListByWorkflowRun(ctx, workflowRunID, platformrepo.CompanyReviewListOptions{
		Pagination: platformrepo.PageOptions{PageSize: service.config.MaxPageSize},
		Sort: platformrepo.CompanyReviewSortOption{
			By:    platformrepo.CompanyReviewSortByFinalizedAt,
			Order: platformrepo.SortOrderDescending,
		},
	})
	if err != nil {
		return nil, false, err
	}
	if result == nil {
		return nil, false, nil
	}
	return result.Items, result.Page.HasMore, nil
}

func (service *projectionUpdateService) updateProjectionForReview(
	ctx context.Context,
	review *domainreview.CompanyReview,
	request UpdateProjectionsRequest,
	targets projectionTargetSet,
) projectionReviewOutcome {
	outcome := projectionReviewOutcome{}
	if review == nil {
		return outcome
	}

	if targets.has(ProjectionTargetPosition) {
		ref, failure := service.updatePositionProjectionFromReview(ctx, review, request)
		if failure != nil {
			outcome.Failures = append(outcome.Failures, *failure)
		} else if ref.Updated {
			outcome.Updated = append(outcome.Updated, ref)
		} else {
			outcome.Skipped = append(outcome.Skipped, ref)
		}
	}

	if targets.has(ProjectionTargetCompanyState) {
		ref := projectionRefForReview(review, ProjectionTargetCompanyState)
		outcome.Skipped = append(outcome.Skipped, markSkipped(ref, "company_state_projection_unsupported"))
	}
	if targets.has(ProjectionTargetReview) {
		ref := projectionRefForReview(review, ProjectionTargetReview)
		outcome.Skipped = append(outcome.Skipped, markSkipped(ref, "review_history_is_authoritative"))
	}
	if targets.has(ProjectionTargetWorkflow) {
		ref := projectionRefForReview(review, ProjectionTargetWorkflow)
		outcome.Skipped = append(outcome.Skipped, markSkipped(ref, "workflow_projection_unsupported"))
	}

	return outcome
}

func addResolvedReview(
	resolved *[]*domainreview.CompanyReview,
	seen map[primitive.ObjectID]struct{},
	review *domainreview.CompanyReview,
) {
	if review == nil || review.ID.IsZero() {
		return
	}
	if _, exists := seen[review.ID]; exists {
		return
	}
	seen[review.ID] = struct{}{}
	*resolved = append(*resolved, review)
}

func projectionBookTypes(bookType domaincommon.BookType) []domaincommon.BookType {
	if bookType != "" {
		return []domaincommon.BookType{bookType}
	}
	return []domaincommon.BookType{domaincommon.BookTypeInvesting, domaincommon.BookTypeTrading}
}
