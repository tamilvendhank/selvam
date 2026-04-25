package review

import (
	domaincommon "goserver/internal/domain/common"
	servicecommon "goserver/internal/service/common"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

type ActionMappingMode string

const (
	ActionMappingModeDefault       ActionMappingMode = ""
	ActionMappingModeInitialReview ActionMappingMode = "initial_review"
	ActionMappingModeRefresh       ActionMappingMode = "refresh"
	ActionMappingModeExitReview    ActionMappingMode = "exit_review"
)

type ActionConstraint struct {
	Code     string `json:"code,omitempty"`
	Message  string `json:"message"`
	Blocking bool   `json:"blocking,omitempty"`
}

type MapReviewActionRequest struct {
	ReviewID      primitive.ObjectID    `json:"reviewId"`
	WorkflowRunID primitive.ObjectID    `json:"workflowRunId,omitempty"`
	BookType      domaincommon.BookType `json:"bookType"`
	Mode          ActionMappingMode     `json:"mode,omitempty"`
	DryRun        bool                  `json:"dryRun,omitempty"`
	Force         bool                  `json:"force,omitempty"`
	InitiatedBy   string                `json:"initiatedBy,omitempty"`
	CorrelationID string                `json:"correlationId,omitempty"`
}

func (request MapReviewActionRequest) Validate() error {
	if err := servicecommon.ValidateRequiredObjectID("reviewId", request.ReviewID); err != nil {
		return err
	}
	if err := servicecommon.ValidateRequiredBookType(request.BookType); err != nil {
		return err
	}
	if err := servicecommon.ValidateOptionalText("mode", string(request.Mode)); err != nil {
		return err
	}
	if err := servicecommon.ValidateOptionalText("initiatedBy", request.InitiatedBy); err != nil {
		return err
	}
	return servicecommon.ValidateOptionalText("correlationId", request.CorrelationID)
}

type MapReviewActionResult struct {
	ReviewID          primitive.ObjectID                 `json:"reviewId,omitempty"`
	WorkflowRunID     primitive.ObjectID                 `json:"workflowRunId,omitempty"`
	ActionType        domaincommon.InvestingActionType   `json:"actionType,omitempty"`
	BucketAfterAction domaincommon.WatchlistBucket       `json:"bucketAfterAction,omitempty"`
	Constraints       []ActionConstraint                 `json:"constraints,omitempty"`
	CapitalEligible   bool                               `json:"capitalEligible,omitempty"`
	PriorityScore     float64                            `json:"priorityScore,omitempty"`
	PartialFailures   []servicecommon.PartialFailure     `json:"partialFailures,omitempty"`
	Summary           servicecommon.ActionMappingSummary `json:"summary,omitempty"`
}

func (result MapReviewActionResult) HasFailures() bool {
	return len(result.PartialFailures) > 0 || result.Summary.HasFailures()
}

type MapWorkflowActionsRequest struct {
	WorkflowRunID primitive.ObjectID    `json:"workflowRunId"`
	BookType      domaincommon.BookType `json:"bookType"`
	Mode          ActionMappingMode     `json:"mode,omitempty"`
	MaxReviews    int                   `json:"maxReviews,omitempty"`
	DryRun        bool                  `json:"dryRun,omitempty"`
	Force         bool                  `json:"force,omitempty"`
	InitiatedBy   string                `json:"initiatedBy,omitempty"`
	CorrelationID string                `json:"correlationId,omitempty"`
}

func (request MapWorkflowActionsRequest) Validate() error {
	if err := servicecommon.ValidateRequiredObjectID("workflowRunId", request.WorkflowRunID); err != nil {
		return err
	}
	if err := servicecommon.ValidateRequiredBookType(request.BookType); err != nil {
		return err
	}
	if err := servicecommon.ValidateOptionalMax("maxReviews", request.MaxReviews); err != nil {
		return err
	}
	if err := servicecommon.ValidateOptionalText("mode", string(request.Mode)); err != nil {
		return err
	}
	if err := servicecommon.ValidateOptionalText("initiatedBy", request.InitiatedBy); err != nil {
		return err
	}
	return servicecommon.ValidateOptionalText("correlationId", request.CorrelationID)
}

type MapWorkflowActionsResult struct {
	WorkflowRunID   primitive.ObjectID                 `json:"workflowRunId,omitempty"`
	MappedReviewIDs []primitive.ObjectID               `json:"mappedReviewIds,omitempty"`
	FailedReviewIDs []primitive.ObjectID               `json:"failedReviewIds,omitempty"`
	ActionResults   []MapReviewActionResult            `json:"actionResults,omitempty"`
	PartialFailures []servicecommon.PartialFailure     `json:"partialFailures,omitempty"`
	Summary         servicecommon.ActionMappingSummary `json:"summary,omitempty"`
}

func (result MapWorkflowActionsResult) HasFailures() bool {
	return len(result.FailedReviewIDs) > 0 || len(result.PartialFailures) > 0 || result.Summary.HasFailures()
}

type AssignBucketRequest struct {
	ReviewID        primitive.ObjectID               `json:"reviewId"`
	WorkflowRunID   primitive.ObjectID               `json:"workflowRunId,omitempty"`
	CompanyID       primitive.ObjectID               `json:"companyId,omitempty"`
	BookType        domaincommon.BookType            `json:"bookType,omitempty"`
	ActionType      domaincommon.InvestingActionType `json:"actionType,omitempty"`
	RequestedBucket domaincommon.WatchlistBucket     `json:"requestedBucket,omitempty"`
	DryRun          bool                             `json:"dryRun,omitempty"`
	Force           bool                             `json:"force,omitempty"`
	InitiatedBy     string                           `json:"initiatedBy,omitempty"`
	CorrelationID   string                           `json:"correlationId,omitempty"`
}

func (request AssignBucketRequest) Validate() error {
	if err := servicecommon.ValidateRequiredObjectID("reviewId", request.ReviewID); err != nil {
		return err
	}
	if err := servicecommon.ValidateOptionalBookType(request.BookType); err != nil {
		return err
	}
	if err := servicecommon.ValidateOptionalActionType(request.ActionType); err != nil {
		return err
	}
	if err := servicecommon.ValidateOptionalBucket(request.RequestedBucket); err != nil {
		return err
	}
	if err := servicecommon.ValidateOptionalText("initiatedBy", request.InitiatedBy); err != nil {
		return err
	}
	return servicecommon.ValidateOptionalText("correlationId", request.CorrelationID)
}

type AssignBucketResult struct {
	ReviewID        primitive.ObjectID                    `json:"reviewId,omitempty"`
	CompanyID       primitive.ObjectID                    `json:"companyId,omitempty"`
	BucketBefore    domaincommon.WatchlistBucket          `json:"bucketBefore,omitempty"`
	BucketAfter     domaincommon.WatchlistBucket          `json:"bucketAfter,omitempty"`
	BucketChanged   bool                                  `json:"bucketChanged,omitempty"`
	PartialFailures []servicecommon.PartialFailure        `json:"partialFailures,omitempty"`
	Summary         servicecommon.BucketAssignmentSummary `json:"summary,omitempty"`
}

func (result AssignBucketResult) HasFailures() bool {
	return len(result.PartialFailures) > 0 || result.Summary.HasFailures()
}

type AssignBucketsForWorkflowRequest struct {
	WorkflowRunID primitive.ObjectID    `json:"workflowRunId"`
	BookType      domaincommon.BookType `json:"bookType,omitempty"`
	MaxReviews    int                   `json:"maxReviews,omitempty"`
	DryRun        bool                  `json:"dryRun,omitempty"`
	Force         bool                  `json:"force,omitempty"`
	InitiatedBy   string                `json:"initiatedBy,omitempty"`
	CorrelationID string                `json:"correlationId,omitempty"`
}

func (request AssignBucketsForWorkflowRequest) Validate() error {
	if err := servicecommon.ValidateRequiredObjectID("workflowRunId", request.WorkflowRunID); err != nil {
		return err
	}
	if err := servicecommon.ValidateOptionalMax("maxReviews", request.MaxReviews); err != nil {
		return err
	}
	if err := servicecommon.ValidateOptionalBookType(request.BookType); err != nil {
		return err
	}
	if err := servicecommon.ValidateOptionalText("initiatedBy", request.InitiatedBy); err != nil {
		return err
	}
	return servicecommon.ValidateOptionalText("correlationId", request.CorrelationID)
}

type AssignBucketsForWorkflowResult struct {
	WorkflowRunID     primitive.ObjectID                    `json:"workflowRunId,omitempty"`
	AssignedReviewIDs []primitive.ObjectID                  `json:"assignedReviewIds,omitempty"`
	FailedReviewIDs   []primitive.ObjectID                  `json:"failedReviewIds,omitempty"`
	BucketResults     []AssignBucketResult                  `json:"bucketResults,omitempty"`
	PartialFailures   []servicecommon.PartialFailure        `json:"partialFailures,omitempty"`
	Summary           servicecommon.BucketAssignmentSummary `json:"summary,omitempty"`
}

func (result AssignBucketsForWorkflowResult) HasFailures() bool {
	return len(result.FailedReviewIDs) > 0 || len(result.PartialFailures) > 0 || result.Summary.HasFailures()
}
