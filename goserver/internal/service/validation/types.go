package validation

import (
	domaincommon "goserver/internal/domain/common"
	servicecommon "goserver/internal/service/common"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

type ValidateBatchItemOutputRequest struct {
	BatchItemID   primitive.ObjectID `json:"batchItemId"`
	WorkflowRunID primitive.ObjectID `json:"workflowRunId,omitempty"`
	StrictMode    bool               `json:"strictMode,omitempty"`
	Revalidate    bool               `json:"revalidate,omitempty"`
	InitiatedBy   string             `json:"initiatedBy,omitempty"`
	CorrelationID string             `json:"correlationId,omitempty"`
}

func (request ValidateBatchItemOutputRequest) Validate() error {
	if err := servicecommon.ValidateRequiredObjectID("batchItemId", request.BatchItemID); err != nil {
		return err
	}
	if err := servicecommon.ValidateOptionalText("initiatedBy", request.InitiatedBy); err != nil {
		return err
	}
	return servicecommon.ValidateOptionalText("correlationId", request.CorrelationID)
}

type ValidateBatchItemOutputResult struct {
	BatchItemID      primitive.ObjectID              `json:"batchItemId,omitempty"`
	ReviewID         primitive.ObjectID              `json:"reviewId,omitempty"`
	ValidationStatus domaincommon.ValidationStatus   `json:"validationStatus,omitempty"`
	ValidItemIDs     []primitive.ObjectID            `json:"validItemIds,omitempty"`
	InvalidItemIDs   []primitive.ObjectID            `json:"invalidItemIds,omitempty"`
	ValidationIssues []servicecommon.ValidationIssue `json:"validationIssues,omitempty"`
	FieldErrors      []servicecommon.FieldError      `json:"fieldErrors,omitempty"`
	PartialFailures  []servicecommon.PartialFailure  `json:"partialFailures,omitempty"`
	Summary          servicecommon.ValidationSummary `json:"summary,omitempty"`
}

func (result ValidateBatchItemOutputResult) HasFailures() bool {
	return len(result.InvalidItemIDs) > 0 || len(result.PartialFailures) > 0 || result.Summary.HasFailures()
}

type ValidatePendingAIOutputsRequest struct {
	BatchItemID   primitive.ObjectID           `json:"batchItemId,omitempty"`
	WorkflowRunID primitive.ObjectID           `json:"workflowRunId,omitempty"`
	BookType      domaincommon.BookType        `json:"bookType,omitempty"`
	ItemType      domaincommon.AIBatchItemType `json:"itemType,omitempty"`
	MaxItems      int                          `json:"maxItems,omitempty"`
	StrictMode    bool                         `json:"strictMode,omitempty"`
	Revalidate    bool                         `json:"revalidate,omitempty"`
	InitiatedBy   string                       `json:"initiatedBy,omitempty"`
	CorrelationID string                       `json:"correlationId,omitempty"`
}

func (request ValidatePendingAIOutputsRequest) Validate() error {
	if err := servicecommon.ValidateOptionalMax("maxItems", request.MaxItems); err != nil {
		return err
	}
	if err := servicecommon.ValidateOptionalBookType(request.BookType); err != nil {
		return err
	}
	if err := servicecommon.ValidateOptionalItemType(request.ItemType); err != nil {
		return err
	}
	if err := servicecommon.ValidateOptionalText("initiatedBy", request.InitiatedBy); err != nil {
		return err
	}
	return servicecommon.ValidateOptionalText("correlationId", request.CorrelationID)
}

type ValidatePendingAIOutputsResult struct {
	ValidItemIDs     []primitive.ObjectID                              `json:"validItemIds,omitempty"`
	InvalidItemIDs   []primitive.ObjectID                              `json:"invalidItemIds,omitempty"`
	SkippedItemIDs   []primitive.ObjectID                              `json:"skippedItemIds,omitempty"`
	ValidationIssues []servicecommon.ValidationIssue                   `json:"validationIssues,omitempty"`
	FieldErrors      map[primitive.ObjectID][]servicecommon.FieldError `json:"fieldErrors,omitempty"`
	PartialFailures  []servicecommon.PartialFailure                    `json:"partialFailures,omitempty"`
	Summary          servicecommon.ValidationSummary                   `json:"summary,omitempty"`
}

func (result ValidatePendingAIOutputsResult) HasFailures() bool {
	return len(result.InvalidItemIDs) > 0 || len(result.PartialFailures) > 0 || result.Summary.HasFailures()
}
