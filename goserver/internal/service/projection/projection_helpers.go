package projection

import (
	"context"
	"errors"
	"strings"

	domaincommon "goserver/internal/domain/common"
	domainreview "goserver/internal/domain/review"
	domainthesis "goserver/internal/domain/thesis"
	platformrepo "goserver/internal/platform/repository"
	servicecommon "goserver/internal/service/common"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

type projectionTargetSet map[ProjectionTarget]struct{}

type projectionReviewOutcome struct {
	Updated  []ProjectionUpdateRef
	Skipped  []ProjectionUpdateRef
	Failures []servicecommon.PartialFailure
}

func normalizeProjectionTargets(targets []ProjectionTarget) projectionTargetSet {
	if len(targets) == 0 {
		return projectionTargetSet{
			ProjectionTargetPosition:     {},
			ProjectionTargetCompanyState: {},
			ProjectionTargetReview:       {},
			ProjectionTargetWorkflow:     {},
		}
	}

	set := make(projectionTargetSet, len(targets))
	for _, target := range targets {
		if isSupportedProjectionTarget(target) {
			set[target] = struct{}{}
		}
	}
	return set
}

func (targets projectionTargetSet) has(target ProjectionTarget) bool {
	_, ok := targets[target]
	return ok
}

func isSupportedProjectionTarget(target ProjectionTarget) bool {
	switch target {
	case ProjectionTargetCompanyState,
		ProjectionTargetPosition,
		ProjectionTargetReview,
		ProjectionTargetWorkflow:
		return true
	default:
		return false
	}
}

func reviewEligibleForProjection(review *domainreview.CompanyReview, requestedBookType domaincommon.BookType) bool {
	if review == nil {
		return false
	}
	if requestedBookType != "" && review.BookType != requestedBookType {
		return false
	}
	return review.ReviewStatus == domaincommon.ReviewStatusFinal &&
		review.ReviewLifecycleState == domaincommon.ReviewLifecycleStateFinalized
}

func projectionRefForReview(review *domainreview.CompanyReview, target ProjectionTarget) ProjectionUpdateRef {
	if review == nil {
		return ProjectionUpdateRef{Target: target}
	}
	return ProjectionUpdateRef{
		Target:        target,
		CompanyID:     review.CompanyID,
		ReviewID:      review.ID,
		WorkflowRunID: review.WorkflowRunID,
	}
}

func skippedReviewRef(review *domainreview.CompanyReview, target ProjectionTarget, reason string) ProjectionUpdateRef {
	return markSkipped(projectionRefForReview(review, target), reason)
}

func markSkipped(ref ProjectionUpdateRef, reason string) ProjectionUpdateRef {
	ref.Updated = false
	if strings.TrimSpace(reason) == "" {
		return ref
	}
	// The result contract does not carry a skip reason, so keep the ref compact.
	return ref
}

func isRepositoryNotFound(err error) bool {
	return errors.Is(err, platformrepo.ErrNotFound)
}

func (service *projectionUpdateService) loadActiveThesis(
	ctx context.Context,
	companyID primitive.ObjectID,
) (*domainthesis.InvestmentThesis, error) {
	if service.theses == nil || companyID.IsZero() {
		return nil, nil
	}
	thesis, err := service.theses.GetActiveByCompanyID(ctx, companyID)
	if err == nil {
		return thesis, nil
	}
	if isRepositoryNotFound(err) {
		return nil, nil
	}
	return nil, err
}
