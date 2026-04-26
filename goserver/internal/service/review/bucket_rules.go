package review

import (
	"fmt"

	domaincommon "goserver/internal/domain/common"
	domainposition "goserver/internal/domain/position"
	domainreview "goserver/internal/domain/review"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

type assignBucketOptions struct {
	ReviewID              primitive.ObjectID
	WorkflowRunID         primitive.ObjectID
	CompanyID             primitive.ObjectID
	BookType              domaincommon.BookType
	ActionType            domaincommon.InvestingActionType
	RequestedBucket       domaincommon.WatchlistBucket
	DryRun                bool
	Force                 bool
	InitiatedBy           string
	CorrelationID         string
	TreatIneligibleAsSkip bool
	PreloadedReview       *domainreview.CompanyReview
}

type bucketAssignmentOutcome struct {
	ReviewID      primitive.ObjectID
	CompanyID     primitive.ObjectID
	WorkflowRunID primitive.ObjectID
	ActionType    domaincommon.InvestingActionType
	BucketBefore  domaincommon.WatchlistBucket
	BucketAfter   domaincommon.WatchlistBucket
	BucketChanged bool
	Assigned      bool
	Persisted     bool
	Skipped       bool
	DryRun        bool
	Message       string
}

type bucketContext struct {
	CapitalEligible   bool
	WeakeningDetected bool
}

func validateBucketEligibleReview(review *domainreview.CompanyReview, options assignBucketOptions) error {
	if review == nil {
		return fmt.Errorf("review is required")
	}
	if review.ID.IsZero() {
		return fmt.Errorf("review id is required")
	}
	if review.CompanyID.IsZero() {
		return fmt.Errorf("review %s companyId is required", review.ID.Hex())
	}
	if !options.WorkflowRunID.IsZero() && !review.WorkflowRunID.IsZero() && review.WorkflowRunID != options.WorkflowRunID {
		return fmt.Errorf("%w: workflowRunId filter does not match", errReviewServiceSkipped)
	}
	if options.BookType != "" && review.BookType != options.BookType {
		return fmt.Errorf("%w: bookType filter does not match", errReviewServiceSkipped)
	}
	if review.BookType != domaincommon.BookTypeInvesting {
		return fmt.Errorf("%w: bucket assignment only applies to investing reviews", errReviewServiceSkipped)
	}
	if review.ReviewLifecycleState == domaincommon.ReviewLifecycleStateSuperseded ||
		review.ReviewStatus == domaincommon.ReviewStatusSuperseded {
		return fmt.Errorf("%w: superseded review cannot be bucket assigned", errReviewServiceSkipped)
	}
	if !isReviewMaterializedForAction(review) {
		return fmt.Errorf("%w: review is not finalized or AI-validated", errReviewServiceSkipped)
	}
	return nil
}

func determineBucketFromAction(
	action domaincommon.InvestingActionType,
	position positionContext,
	context bucketContext,
) domaincommon.WatchlistBucket {
	if !position.Owned {
		switch action {
		case domaincommon.InvestingActionTypeBuy:
			return domaincommon.WatchlistBucketBuyReady
		case domaincommon.InvestingActionTypeWatch:
			return domaincommon.WatchlistBucketWatch
		case domaincommon.InvestingActionTypeReject:
			return domaincommon.WatchlistBucketResearch
		default:
			return domaincommon.WatchlistBucketResearch
		}
	}

	switch action {
	case domaincommon.InvestingActionTypeBuy:
		if context.CapitalEligible && position.UnderTarget && !position.AtOrAboveMax {
			return domaincommon.WatchlistBucketBuyReady
		}
		return domaincommon.WatchlistBucketHold
	case domaincommon.InvestingActionTypeHold:
		return domaincommon.WatchlistBucketHold
	case domaincommon.InvestingActionTypeTrim, domaincommon.InvestingActionTypeSell:
		return domaincommon.WatchlistBucketExitReview
	case domaincommon.InvestingActionTypeWatch:
		if context.WeakeningDetected {
			return domaincommon.WatchlistBucketExitReview
		}
		return domaincommon.WatchlistBucketHold
	case domaincommon.InvestingActionTypeReject:
		return domaincommon.WatchlistBucketExitReview
	default:
		return domaincommon.WatchlistBucketWatch
	}
}

func validateActionBucketCompatibility(
	action domaincommon.InvestingActionType,
	bucket domaincommon.WatchlistBucket,
	position positionContext,
) error {
	if bucket == "" {
		return nil
	}
	switch {
	case !position.Owned && action == domaincommon.InvestingActionTypeBuy && bucket != domaincommon.WatchlistBucketBuyReady:
		return invalidReviewServiceRequestf("unowned BUY should map to buy_ready, got %q", bucket)
	case !position.Owned && action == domaincommon.InvestingActionTypeWatch && bucket != domaincommon.WatchlistBucketWatch:
		return invalidReviewServiceRequestf("unowned WATCH should map to watch, got %q", bucket)
	case position.Owned && (action == domaincommon.InvestingActionTypeTrim || action == domaincommon.InvestingActionTypeSell) && bucket != domaincommon.WatchlistBucketExitReview:
		return invalidReviewServiceRequestf("owned %s should map to exit_review, got %q", action, bucket)
	case position.Owned && action == domaincommon.InvestingActionTypeHold && bucket != domaincommon.WatchlistBucketHold:
		return invalidReviewServiceRequestf("owned HOLD should map to hold, got %q", bucket)
	default:
		return nil
	}
}

func normalizeActionBucketCompatibility(
	action domaincommon.InvestingActionType,
	bucket domaincommon.WatchlistBucket,
	position positionContext,
) (domaincommon.WatchlistBucket, string) {
	if !position.Owned {
		switch action {
		case domaincommon.InvestingActionTypeHold:
			return domaincommon.WatchlistBucketWatch, "normalized unowned HOLD to watch"
		case domaincommon.InvestingActionTypeTrim, domaincommon.InvestingActionTypeSell:
			return domaincommon.WatchlistBucketResearch, "normalized unowned exit action to research"
		}
		return bucket, ""
	}

	if action == domaincommon.InvestingActionTypeReject {
		return domaincommon.WatchlistBucketExitReview, "normalized owned REJECT to exit_review"
	}
	return bucket, ""
}

func actionFromReview(review *domainreview.CompanyReview) domaincommon.InvestingActionType {
	if review == nil {
		return ""
	}
	if review.FinalActionAfterReview != "" {
		return review.FinalActionAfterReview
	}
	if review.DecisionAction != nil {
		return review.DecisionAction.ActionType
	}
	return ""
}

func extractBucketPositionContext(
	review *domainreview.CompanyReview,
	position *domainposition.CurrentPosition,
	config BucketAssignmentConfig,
) positionContext {
	return extractPositionContext(review, position, ActionMappingConfig{
		DefaultTargetPositionPct: config.DefaultTargetPositionPct,
		DefaultMaxPositionPct:    config.DefaultMaxPositionPct,
	})
}

func bucketWeakeningDetected(review *domainreview.CompanyReview) bool {
	return requiresExitReview(review) ||
		totalScoreDropped(review, 1.0) ||
		anyCoreSectionDropped(review, 1.5) ||
		managementGovernanceDropped(review, 1.0) ||
		hasMajorNegativeChanges(review)
}
