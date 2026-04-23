package review

import (
	"fmt"
	"time"

	"goserver/internal/domain/common"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

// CompanyReview is the immutable historical review snapshot for one company, one book,
// and one review cycle. After finalization the document should be treated as historical truth.
//
// ReviewStatus is the coarse business-facing state used by downstream readers.
// ReviewLifecycleState is the async workflow state used while the review is being built.
type CompanyReview struct {
	ID                        primitive.ObjectID          `bson:"_id,omitempty" json:"id,omitempty"`
	CompanyID                 primitive.ObjectID          `bson:"companyId" json:"companyId"`
	Symbol                    string                      `bson:"symbol" json:"symbol"`
	BookType                  common.BookType             `bson:"bookType" json:"bookType"`
	ReviewDate                time.Time                   `bson:"reviewDate" json:"reviewDate"`
	ReviewPeriodType          common.ReviewPeriodType     `bson:"reviewPeriodType" json:"reviewPeriodType"`
	WorkflowRunID             primitive.ObjectID          `bson:"workflowRunId,omitempty" json:"workflowRunId,omitempty"`
	ConfigSnapshotID          primitive.ObjectID          `bson:"configSnapshotId" json:"configSnapshotId"`
	ReviewStatus              common.ReviewStatus         `bson:"reviewStatus" json:"reviewStatus"`
	ReviewLifecycleState      common.ReviewLifecycleState `bson:"reviewLifecycleState" json:"reviewLifecycleState"`
	Mode                      common.InvestingMode        `bson:"mode,omitempty" json:"mode,omitempty"`
	OwnedBeforeReview         bool                        `bson:"ownedBeforeReview" json:"ownedBeforeReview"`
	CurrentBucketBeforeReview common.WatchlistBucket      `bson:"currentBucketBeforeReview,omitempty" json:"currentBucketBeforeReview,omitempty"`
	CurrentActionBeforeReview common.InvestingActionType  `bson:"currentActionBeforeReview,omitempty" json:"currentActionBeforeReview,omitempty"`
	WeightedTotalScore        float64                     `bson:"weightedTotalScore,omitempty" json:"weightedTotalScore,omitempty"`
	HardGateFailed            bool                        `bson:"hardGateFailed" json:"hardGateFailed"`
	HardGateFailureReasons    []string                    `bson:"hardGateFailureReasons,omitempty" json:"hardGateFailureReasons,omitempty"`
	ConfidenceScore           float64                     `bson:"confidenceScore,omitempty" json:"confidenceScore,omitempty"`
	FinalBucketAfterReview    common.WatchlistBucket      `bson:"finalBucketAfterReview,omitempty" json:"finalBucketAfterReview,omitempty"`
	FinalActionAfterReview    common.InvestingActionType  `bson:"finalActionAfterReview,omitempty" json:"finalActionAfterReview,omitempty"`
	ActionRationaleSummary    string                      `bson:"actionRationaleSummary,omitempty" json:"actionRationaleSummary,omitempty"`
	WhatChangedSummary        string                      `bson:"whatChangedSummary,omitempty" json:"whatChangedSummary,omitempty"`
	ReviewerType              common.ReviewerType         `bson:"reviewerType" json:"reviewerType"`
	AIModelName               string                      `bson:"aiModelName,omitempty" json:"aiModelName,omitempty"`
	AIPromptVersion           string                      `bson:"aiPromptVersion,omitempty" json:"aiPromptVersion,omitempty"`
	SchemaVersion             int                         `bson:"schemaVersion" json:"schemaVersion"`
	ReviewMetadata            map[string]any              `bson:"reviewMetadata,omitempty" json:"reviewMetadata,omitempty"`
	Sections                  []SectionScore              `bson:"sections,omitempty" json:"sections,omitempty"`
	DecisionAction            *DecisionAction             `bson:"decisionAction,omitempty" json:"decisionAction,omitempty"`
	PositionSnapshot          *PositionSnapshot           `bson:"positionSnapshot,omitempty" json:"positionSnapshot,omitempty"`
	ChangeLog                 *ReviewChangeLog            `bson:"changeLog,omitempty" json:"changeLog,omitempty"`
	RawAIResultRef            *common.PayloadReference    `bson:"rawAIResultRef,omitempty" json:"rawAIResultRef,omitempty"`
	CreatedAt                 time.Time                   `bson:"createdAt" json:"createdAt"`
	UpdatedAt                 time.Time                   `bson:"updatedAt" json:"updatedAt"`
	FinalizedAt               *time.Time                  `bson:"finalizedAt,omitempty" json:"finalizedAt,omitempty"`
}

var allowedReviewLifecycleTransitions = map[common.ReviewLifecycleState]map[common.ReviewLifecycleState]struct{}{
	common.ReviewLifecycleStatePendingInput: {
		common.ReviewLifecycleStatePendingAI: {},
	},
	common.ReviewLifecycleStatePendingAI: {
		common.ReviewLifecycleStateAICompletedUnvalidated: {},
	},
	common.ReviewLifecycleStateAICompletedUnvalidated: {
		common.ReviewLifecycleStateValidationFailed: {},
		common.ReviewLifecycleStateAIValidated:      {},
	},
	common.ReviewLifecycleStateValidationFailed: {
		common.ReviewLifecycleStatePendingAI: {},
	},
	common.ReviewLifecycleStateAIValidated: {
		common.ReviewLifecycleStateFinalized: {},
	},
	common.ReviewLifecycleStateFinalized: {
		common.ReviewLifecycleStateSuperseded: {},
	},
	common.ReviewLifecycleStateSuperseded: {},
}

func (review CompanyReview) Validate() error {
	if err := common.RequireObjectID("companyId", review.CompanyID); err != nil {
		return err
	}
	if err := common.RequireString("symbol", review.Symbol); err != nil {
		return err
	}
	if !review.BookType.IsValid() {
		return fmt.Errorf("invalid bookType %q", review.BookType)
	}
	if err := common.RequireTime("reviewDate", review.ReviewDate); err != nil {
		return err
	}
	if !review.ReviewPeriodType.IsValid() {
		return fmt.Errorf("invalid reviewPeriodType %q", review.ReviewPeriodType)
	}
	if err := common.RequireObjectID("configSnapshotId", review.ConfigSnapshotID); err != nil {
		return err
	}
	if !review.ReviewStatus.IsValid() {
		return fmt.Errorf("invalid reviewStatus %q", review.ReviewStatus)
	}
	if !review.ReviewLifecycleState.IsValid() {
		return fmt.Errorf("invalid reviewLifecycleState %q", review.ReviewLifecycleState)
	}
	if !review.ReviewerType.IsValid() {
		return fmt.Errorf("invalid reviewerType %q", review.ReviewerType)
	}
	if err := common.ValidateSchemaVersion("schemaVersion", review.SchemaVersion); err != nil {
		return err
	}
	if err := common.RequireTime("createdAt", review.CreatedAt); err != nil {
		return err
	}
	if err := common.RequireTime("updatedAt", review.UpdatedAt); err != nil {
		return err
	}
	if err := common.ValidateTimestampOrder("createdAt", review.CreatedAt, "updatedAt", review.UpdatedAt); err != nil {
		return err
	}
	if err := common.ValidateOptionalTimestampOrder("createdAt", review.CreatedAt, "finalizedAt", review.FinalizedAt); err != nil {
		return err
	}
	if err := common.ValidateStringSlice("hardGateFailureReasons", review.HardGateFailureReasons); err != nil {
		return err
	}
	if review.HardGateFailed && len(review.HardGateFailureReasons) == 0 {
		return fmt.Errorf("hardGateFailureReasons are required when hardGateFailed is true")
	}
	if !review.HardGateFailed && len(review.HardGateFailureReasons) > 0 {
		return fmt.Errorf("hardGateFailureReasons must be empty when hardGateFailed is false")
	}
	if review.BookType == common.BookTypeInvesting {
		if !review.Mode.IsValid() {
			return fmt.Errorf("invalid mode %q for investing review", review.Mode)
		}
	} else if review.Mode != "" {
		return fmt.Errorf("mode is currently only supported for investing reviews")
	}
	if review.CurrentBucketBeforeReview != "" && !review.CurrentBucketBeforeReview.IsValid() {
		return fmt.Errorf("invalid currentBucketBeforeReview %q", review.CurrentBucketBeforeReview)
	}
	if review.CurrentActionBeforeReview != "" && !review.CurrentActionBeforeReview.IsValid() {
		return fmt.Errorf("invalid currentActionBeforeReview %q", review.CurrentActionBeforeReview)
	}
	if review.FinalBucketAfterReview != "" && !review.FinalBucketAfterReview.IsValid() {
		return fmt.Errorf("invalid finalBucketAfterReview %q", review.FinalBucketAfterReview)
	}
	if review.FinalActionAfterReview != "" && !review.FinalActionAfterReview.IsValid() {
		return fmt.Errorf("invalid finalActionAfterReview %q", review.FinalActionAfterReview)
	}
	if review.BookType == common.BookTypeTrading {
		if review.CurrentBucketBeforeReview != "" || review.FinalBucketAfterReview != "" {
			return fmt.Errorf("watchlist buckets are ring-fenced to the investing book")
		}
	}
	if err := review.validateStatusConsistency(); err != nil {
		return err
	}
	if review.DecisionAction != nil {
		if err := review.DecisionAction.Validate(); err != nil {
			return err
		}
	}
	if review.PositionSnapshot != nil {
		if err := review.PositionSnapshot.Validate(); err != nil {
			return err
		}
		if review.PositionSnapshot.OwnedSinceDate != nil && review.PositionSnapshot.OwnedSinceDate.After(review.ReviewDate) {
			return fmt.Errorf("ownedSinceDate cannot be after reviewDate")
		}
	}
	if review.ChangeLog != nil {
		if err := review.ChangeLog.Validate(); err != nil {
			return err
		}
	}
	if review.RawAIResultRef != nil {
		if err := review.RawAIResultRef.Validate(); err != nil {
			return err
		}
	}

	if len(review.Sections) > 0 || review.ReviewLifecycleState == common.ReviewLifecycleStateAIValidated || review.IsFinalized() {
		if err := common.ValidateComputedScore("weightedTotalScore", review.WeightedTotalScore); err != nil {
			return err
		}
		if err := common.ValidateUnitInterval("confidenceScore", review.ConfidenceScore); err != nil {
			return err
		}
		if err := review.validateSections(); err != nil {
			return err
		}
	}

	if review.IsFinalized() {
		if err := review.validateFinalizedPayload(); err != nil {
			return err
		}
	}

	return nil
}

func (review CompanyReview) validateStatusConsistency() error {
	switch review.ReviewStatus {
	case common.ReviewStatusDraft:
		if review.ReviewLifecycleState == common.ReviewLifecycleStateFinalized || review.ReviewLifecycleState == common.ReviewLifecycleStateSuperseded {
			return fmt.Errorf("draft reviewStatus cannot be paired with lifecycle state %q", review.ReviewLifecycleState)
		}
		if review.FinalizedAt != nil {
			return fmt.Errorf("draft reviews cannot have finalizedAt")
		}
	case common.ReviewStatusFinal:
		if review.ReviewLifecycleState != common.ReviewLifecycleStateFinalized {
			return fmt.Errorf("final reviewStatus requires finalized lifecycle state")
		}
		if review.FinalizedAt == nil {
			return fmt.Errorf("final reviews require finalizedAt")
		}
	case common.ReviewStatusSuperseded:
		if review.ReviewLifecycleState != common.ReviewLifecycleStateSuperseded {
			return fmt.Errorf("superseded reviewStatus requires superseded lifecycle state")
		}
		if review.FinalizedAt == nil {
			return fmt.Errorf("superseded reviews must retain finalizedAt")
		}
	default:
		return fmt.Errorf("invalid reviewStatus %q", review.ReviewStatus)
	}
	return nil
}

func (review CompanyReview) validateSections() error {
	if len(review.Sections) == 0 {
		return fmt.Errorf("sections are required once review scoring is materialized")
	}
	seen := make(map[common.SectionName]struct{}, len(review.Sections))
	var totalWeight float64
	var weightedTotal float64
	for _, section := range review.Sections {
		if err := section.Validate(); err != nil {
			return err
		}
		if _, exists := seen[section.SectionName]; exists {
			return fmt.Errorf("duplicate section %q", section.SectionName)
		}
		seen[section.SectionName] = struct{}{}
		totalWeight += section.SectionWeight
		weightedTotal += section.SectionScoreWeighted
	}
	if !common.NearlyEqual(totalWeight, 100) {
		return fmt.Errorf("section weights must total 100")
	}
	if !common.NearlyEqual(weightedTotal, review.WeightedTotalScore) {
		return fmt.Errorf("weightedTotalScore does not match section totals")
	}
	return nil
}

func (review CompanyReview) validateFinalizedPayload() error {
	if len(review.Sections) == 0 {
		return fmt.Errorf("finalized reviews must contain section scores")
	}
	if review.DecisionAction == nil {
		return fmt.Errorf("finalized reviews must contain a decisionAction")
	}
	if review.FinalActionAfterReview == "" {
		return fmt.Errorf("finalized reviews must contain finalActionAfterReview")
	}
	if review.BookType == common.BookTypeInvesting && review.FinalBucketAfterReview == "" {
		return fmt.Errorf("finalized investing reviews must contain finalBucketAfterReview")
	}
	return nil
}

func (review CompanyReview) IsFinalized() bool {
	return review.ReviewLifecycleState == common.ReviewLifecycleStateFinalized || review.ReviewLifecycleState == common.ReviewLifecycleStateSuperseded
}

func (review CompanyReview) CanFinalize() bool {
	if review.ReviewLifecycleState != common.ReviewLifecycleStateAIValidated || review.ReviewStatus != common.ReviewStatusDraft {
		return false
	}
	if len(review.Sections) == 0 || review.DecisionAction == nil || review.FinalActionAfterReview == "" {
		return false
	}
	if review.BookType == common.BookTypeInvesting && review.FinalBucketAfterReview == "" {
		return false
	}
	return true
}

func (review CompanyReview) CanSupersede() bool {
	return review.ReviewLifecycleState == common.ReviewLifecycleStateFinalized && review.ReviewStatus == common.ReviewStatusFinal
}

func (review CompanyReview) CanTransitionLifecycleTo(next common.ReviewLifecycleState) bool {
	if review.ReviewLifecycleState == next {
		return true
	}
	nextStates, ok := allowedReviewLifecycleTransitions[review.ReviewLifecycleState]
	if !ok {
		return false
	}
	_, ok = nextStates[next]
	return ok
}

func (review *CompanyReview) TransitionLifecycleTo(next common.ReviewLifecycleState, at time.Time) error {
	if review == nil {
		return fmt.Errorf("review is required")
	}
	if !next.IsValid() {
		return fmt.Errorf("invalid next lifecycle state %q", next)
	}
	if !review.CanTransitionLifecycleTo(next) {
		return fmt.Errorf("invalid review lifecycle transition from %q to %q", review.ReviewLifecycleState, next)
	}
	if err := common.RequireTime("transitionAt", at); err != nil {
		return err
	}

	review.ReviewLifecycleState = next
	switch next {
	case common.ReviewLifecycleStateFinalized:
		review.ReviewStatus = common.ReviewStatusFinal
		finalizedAt := at.UTC()
		review.FinalizedAt = &finalizedAt
	case common.ReviewLifecycleStateSuperseded:
		review.ReviewStatus = common.ReviewStatusSuperseded
	default:
		review.ReviewStatus = common.ReviewStatusDraft
	}
	review.UpdatedAt = at.UTC()
	return nil
}

func (review *CompanyReview) Finalize(at time.Time) error {
	if review == nil {
		return fmt.Errorf("review is required")
	}
	if !review.CanFinalize() {
		return fmt.Errorf("review cannot be finalized from state %q", review.ReviewLifecycleState)
	}
	return review.TransitionLifecycleTo(common.ReviewLifecycleStateFinalized, at)
}

func (review *CompanyReview) Supersede(at time.Time) error {
	if review == nil {
		return fmt.Errorf("review is required")
	}
	if !review.CanSupersede() {
		return fmt.Errorf("review cannot be superseded from state %q", review.ReviewLifecycleState)
	}
	return review.TransitionLifecycleTo(common.ReviewLifecycleStateSuperseded, at)
}
