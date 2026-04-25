package finalization

import (
	domaincommon "goserver/internal/domain/common"
	servicecommon "goserver/internal/service/common"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

type FinalizeReviewRequest struct {
	ReviewID       primitive.ObjectID `json:"reviewId"`
	WorkflowRunID  primitive.ObjectID `json:"workflowRunId,omitempty"`
	Force          bool               `json:"force,omitempty"`
	SupersedePrior bool               `json:"supersedePrior,omitempty"`
	DryRun         bool               `json:"dryRun,omitempty"`
	InitiatedBy    string             `json:"initiatedBy,omitempty"`
	CorrelationID  string             `json:"correlationId,omitempty"`
}

func (request FinalizeReviewRequest) Validate() error {
	if err := servicecommon.ValidateRequiredObjectID("reviewId", request.ReviewID); err != nil {
		return err
	}
	if err := servicecommon.ValidateOptionalText("initiatedBy", request.InitiatedBy); err != nil {
		return err
	}
	return servicecommon.ValidateOptionalText("correlationId", request.CorrelationID)
}

type FinalizeReviewResult struct {
	ReviewID            primitive.ObjectID                `json:"reviewId,omitempty"`
	FinalizedReviewIDs  []primitive.ObjectID              `json:"finalizedReviewIds,omitempty"`
	FailedReviewIDs     []primitive.ObjectID              `json:"failedReviewIds,omitempty"`
	SupersededReviewIDs []primitive.ObjectID              `json:"supersededReviewIds,omitempty"`
	ReviewRefs          []servicecommon.ReviewRef         `json:"reviewRefs,omitempty"`
	PartialFailures     []servicecommon.PartialFailure    `json:"partialFailures,omitempty"`
	Summary             servicecommon.FinalizationSummary `json:"summary,omitempty"`
}

func (result FinalizeReviewResult) HasFailures() bool {
	return len(result.FailedReviewIDs) > 0 || len(result.PartialFailures) > 0 || result.Summary.HasFailures()
}

type FinalizeEligibleReviewsRequest struct {
	ReviewID       primitive.ObjectID    `json:"reviewId,omitempty"`
	WorkflowRunID  primitive.ObjectID    `json:"workflowRunId,omitempty"`
	CompanyID      primitive.ObjectID    `json:"companyId,omitempty"`
	BookType       domaincommon.BookType `json:"bookType,omitempty"`
	MaxReviews     int                   `json:"maxReviews,omitempty"`
	Force          bool                  `json:"force,omitempty"`
	SupersedePrior bool                  `json:"supersedePrior,omitempty"`
	DryRun         bool                  `json:"dryRun,omitempty"`
	InitiatedBy    string                `json:"initiatedBy,omitempty"`
	CorrelationID  string                `json:"correlationId,omitempty"`
}

func (request FinalizeEligibleReviewsRequest) Validate() error {
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

type FinalizeEligibleReviewsResult struct {
	FinalizedReviewIDs  []primitive.ObjectID              `json:"finalizedReviewIds,omitempty"`
	FailedReviewIDs     []primitive.ObjectID              `json:"failedReviewIds,omitempty"`
	SkippedReviewIDs    []primitive.ObjectID              `json:"skippedReviewIds,omitempty"`
	SupersededReviewIDs []primitive.ObjectID              `json:"supersededReviewIds,omitempty"`
	ReviewRefs          []servicecommon.ReviewRef         `json:"reviewRefs,omitempty"`
	PartialFailures     []servicecommon.PartialFailure    `json:"partialFailures,omitempty"`
	Summary             servicecommon.FinalizationSummary `json:"summary,omitempty"`
}

func (result FinalizeEligibleReviewsResult) HasFailures() bool {
	return len(result.FailedReviewIDs) > 0 || len(result.PartialFailures) > 0 || result.Summary.HasFailures()
}
