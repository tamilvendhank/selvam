package materialization

import (
	domaincommon "goserver/internal/domain/common"
	servicecommon "goserver/internal/service/common"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

type MaterializeReviewRequest struct {
	ReviewID      primitive.ObjectID `json:"reviewId,omitempty"`
	BatchItemID   primitive.ObjectID `json:"batchItemId,omitempty"`
	WorkflowRunID primitive.ObjectID `json:"workflowRunId,omitempty"`
	Force         bool               `json:"force,omitempty"`
	DryRun        bool               `json:"dryRun,omitempty"`
	InitiatedBy   string             `json:"initiatedBy,omitempty"`
	CorrelationID string             `json:"correlationId,omitempty"`
}

func (request MaterializeReviewRequest) Validate() error {
	if err := servicecommon.ValidateAtLeastOneObjectID(map[string]primitive.ObjectID{
		"reviewId":    request.ReviewID,
		"batchItemId": request.BatchItemID,
	}); err != nil {
		return err
	}
	if err := servicecommon.ValidateOptionalText("initiatedBy", request.InitiatedBy); err != nil {
		return err
	}
	return servicecommon.ValidateOptionalText("correlationId", request.CorrelationID)
}

type MaterializeReviewResult struct {
	ReviewID              primitive.ObjectID                   `json:"reviewId,omitempty"`
	BatchItemID           primitive.ObjectID                   `json:"batchItemId,omitempty"`
	MaterializedReviewIDs []primitive.ObjectID                 `json:"materializedReviewIds,omitempty"`
	FailedReviewIDs       []primitive.ObjectID                 `json:"failedReviewIds,omitempty"`
	SkippedReviewIDs      []primitive.ObjectID                 `json:"skippedReviewIds,omitempty"`
	ReviewRefs            []servicecommon.ReviewRef            `json:"reviewRefs,omitempty"`
	PartialFailures       []servicecommon.PartialFailure       `json:"partialFailures,omitempty"`
	Summary               servicecommon.MaterializationSummary `json:"summary,omitempty"`
}

func (result MaterializeReviewResult) HasFailures() bool {
	return len(result.FailedReviewIDs) > 0 || len(result.PartialFailures) > 0 || result.Summary.HasFailures()
}

type MaterializePendingReviewsRequest struct {
	ReviewID      primitive.ObjectID    `json:"reviewId,omitempty"`
	BatchItemID   primitive.ObjectID    `json:"batchItemId,omitempty"`
	WorkflowRunID primitive.ObjectID    `json:"workflowRunId,omitempty"`
	BookType      domaincommon.BookType `json:"bookType,omitempty"`
	MaxItems      int                   `json:"maxItems,omitempty"`
	Force         bool                  `json:"force,omitempty"`
	DryRun        bool                  `json:"dryRun,omitempty"`
	InitiatedBy   string                `json:"initiatedBy,omitempty"`
	CorrelationID string                `json:"correlationId,omitempty"`
}

func (request MaterializePendingReviewsRequest) Validate() error {
	if err := servicecommon.ValidateOptionalMax("maxItems", request.MaxItems); err != nil {
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

type MaterializePendingReviewsResult struct {
	MaterializedReviewIDs []primitive.ObjectID                 `json:"materializedReviewIds,omitempty"`
	FailedReviewIDs       []primitive.ObjectID                 `json:"failedReviewIds,omitempty"`
	SkippedReviewIDs      []primitive.ObjectID                 `json:"skippedReviewIds,omitempty"`
	ReviewRefs            []servicecommon.ReviewRef            `json:"reviewRefs,omitempty"`
	PartialFailures       []servicecommon.PartialFailure       `json:"partialFailures,omitempty"`
	Summary               servicecommon.MaterializationSummary `json:"summary,omitempty"`
}

func (result MaterializePendingReviewsResult) HasFailures() bool {
	return len(result.FailedReviewIDs) > 0 || len(result.PartialFailures) > 0 || result.Summary.HasFailures()
}
