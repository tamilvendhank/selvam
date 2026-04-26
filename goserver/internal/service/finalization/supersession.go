package finalization

import (
	"context"
	"errors"
	"fmt"

	domaincommon "goserver/internal/domain/common"
	domainreview "goserver/internal/domain/review"
	platformrepo "goserver/internal/platform/repository"
	servicecommon "goserver/internal/service/common"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

func (service *reviewFinalizationService) maybeSupersedePreviousReview(
	ctx context.Context,
	current *domainreview.CompanyReview,
	options finalizationRequestOptions,
) ([]primitive.ObjectID, []servicecommon.PartialFailure) {
	if current == nil || current.FinalizedAt == nil {
		return nil, nil
	}

	latest, err := service.reviews.GetLatestByCompanyAndBook(ctx, current.CompanyID, current.BookType, platformrepo.LatestCompanyReviewOptions{
		FinalizedOnly:     true,
		IncludeSuperseded: false,
	})
	if err != nil {
		if errors.Is(err, platformrepo.ErrNotFound) {
			return nil, nil
		}
		return nil, []servicecommon.PartialFailure{supersessionPartialFailure(current.ID, primitive.NilObjectID, err)}
	}
	if latest == nil || latest.ID != current.ID {
		return nil, nil
	}

	previous, err := service.reviews.GetPreviousFinalizedByCompanyAndBook(ctx, current.CompanyID, current.BookType, platformrepo.PreviousFinalizedReviewLookup{
		ExcludeReviewID:   current.ID,
		BeforeFinalizedAt: current.FinalizedAt,
		IncludeSuperseded: false,
	})
	if err != nil {
		if errors.Is(err, platformrepo.ErrNotFound) {
			return nil, nil
		}
		return nil, []servicecommon.PartialFailure{supersessionPartialFailure(current.ID, primitive.NilObjectID, err)}
	}
	if previous == nil || previous.ID == current.ID || !previous.CanSupersede() {
		return nil, nil
	}

	supersededAt := service.now().UTC()
	superseded, err := service.reviews.MarkSuperseded(ctx, previous.ID, platformrepo.ReviewSupersedePatch{
		SupersededAt:      supersededAt,
		ReplacementReview: current.ID,
		ExpectedCurrentLifecycleStates: []domaincommon.ReviewLifecycleState{
			domaincommon.ReviewLifecycleStateFinalized,
		},
		ExpectedCurrentStatuses: []domaincommon.ReviewStatus{
			domaincommon.ReviewStatusFinal,
		},
		SupersededBy: options.InitiatedBy,
		Reason:       fmt.Sprintf("superseded by finalized review %s", current.ID.Hex()),
	})
	if err != nil {
		return nil, []servicecommon.PartialFailure{supersessionPartialFailure(current.ID, previous.ID, err)}
	}
	return []primitive.ObjectID{superseded.ID}, nil
}

func supersessionPartialFailure(
	currentReviewID primitive.ObjectID,
	previousReviewID primitive.ObjectID,
	err error,
) servicecommon.PartialFailure {
	retryClass := servicecommon.RetryClassTransient
	retryable := true
	if errors.Is(err, platformrepo.ErrNotFound) {
		retryClass = servicecommon.RetryClassNone
		retryable = false
	}
	if errors.Is(err, platformrepo.ErrPreconditionFailed) || errors.Is(err, platformrepo.ErrConflict) {
		retryClass = servicecommon.RetryClassConflict
	}
	if errors.Is(err, platformrepo.ErrInvalidTransition) || errors.Is(err, platformrepo.ErrImmutableState) {
		retryClass = servicecommon.RetryClassManualReview
		retryable = false
	}
	return servicecommon.PartialFailure{
		Scope:    servicecommon.FailureScopeReview,
		ID:       previousReviewID,
		ReviewID: previousReviewID,
		Code:     "review_supersession_failed",
		Message:  err.Error(),
		Retry: servicecommon.RetryPolicyHint{
			Retryable:  retryable,
			RetryClass: retryClass,
			Reason:     fmt.Sprintf("supersession after finalizing review %s", currentReviewID.Hex()),
		},
	}
}
