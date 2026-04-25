package thesis

import (
	domaincommon "goserver/internal/domain/common"
	servicecommon "goserver/internal/service/common"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

type EvaluateThesisRequest struct {
	ReviewID      primitive.ObjectID    `json:"reviewId,omitempty"`
	CompanyID     primitive.ObjectID    `json:"companyId,omitempty"`
	WorkflowRunID primitive.ObjectID    `json:"workflowRunId,omitempty"`
	BookType      domaincommon.BookType `json:"bookType,omitempty"`
	DryRun        bool                  `json:"dryRun,omitempty"`
	Force         bool                  `json:"force,omitempty"`
	InitiatedBy   string                `json:"initiatedBy,omitempty"`
	CorrelationID string                `json:"correlationId,omitempty"`
}

func (request EvaluateThesisRequest) Validate() error {
	if err := servicecommon.ValidateAtLeastOneObjectID(map[string]primitive.ObjectID{
		"reviewId":  request.ReviewID,
		"companyId": request.CompanyID,
	}); err != nil {
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

type EvaluateThesisResult struct {
	ReviewID          primitive.ObjectID             `json:"reviewId,omitempty"`
	CompanyID         primitive.ObjectID             `json:"companyId,omitempty"`
	ThesisCreated     bool                           `json:"thesisCreated,omitempty"`
	ThesisUpdated     bool                           `json:"thesisUpdated,omitempty"`
	ThesisBroken      bool                           `json:"thesisBroken,omitempty"`
	ThesisUnderReview bool                           `json:"thesisUnderReview,omitempty"`
	UpdatedThesisIDs  []primitive.ObjectID           `json:"updatedThesisIds,omitempty"`
	PartialFailures   []servicecommon.PartialFailure `json:"partialFailures,omitempty"`
	Summary           servicecommon.ThesisSummary    `json:"summary,omitempty"`
}

func (result EvaluateThesisResult) HasFailures() bool {
	return len(result.PartialFailures) > 0 || result.Summary.HasFailures()
}

type EvaluateThesisForWorkflowRequest struct {
	WorkflowRunID primitive.ObjectID    `json:"workflowRunId"`
	BookType      domaincommon.BookType `json:"bookType,omitempty"`
	MaxReviews    int                   `json:"maxReviews,omitempty"`
	DryRun        bool                  `json:"dryRun,omitempty"`
	Force         bool                  `json:"force,omitempty"`
	InitiatedBy   string                `json:"initiatedBy,omitempty"`
	CorrelationID string                `json:"correlationId,omitempty"`
}

func (request EvaluateThesisForWorkflowRequest) Validate() error {
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

type EvaluateThesisForWorkflowResult struct {
	WorkflowRunID     primitive.ObjectID             `json:"workflowRunId,omitempty"`
	ThesisCreated     bool                           `json:"thesisCreated,omitempty"`
	ThesisUpdated     bool                           `json:"thesisUpdated,omitempty"`
	ThesisBroken      bool                           `json:"thesisBroken,omitempty"`
	ThesisUnderReview bool                           `json:"thesisUnderReview,omitempty"`
	UpdatedThesisIDs  []primitive.ObjectID           `json:"updatedThesisIds,omitempty"`
	FailedReviewIDs   []primitive.ObjectID           `json:"failedReviewIds,omitempty"`
	PartialFailures   []servicecommon.PartialFailure `json:"partialFailures,omitempty"`
	Summary           servicecommon.ThesisSummary    `json:"summary,omitempty"`
}

func (result EvaluateThesisForWorkflowResult) HasFailures() bool {
	return len(result.FailedReviewIDs) > 0 || len(result.PartialFailures) > 0 || result.Summary.HasFailures()
}
