package aijob

import (
	"fmt"
	"time"

	"goserver/internal/domain/common"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

type AIBatchItem struct {
	ID                  primitive.ObjectID       `bson:"_id,omitempty" json:"id,omitempty"`
	AIBatchJobID        primitive.ObjectID       `bson:"aiBatchJobId" json:"aiBatchJobId"`
	WorkflowRunID       primitive.ObjectID       `bson:"workflowRunId" json:"workflowRunId"`
	CompanyID           primitive.ObjectID       `bson:"companyId,omitempty" json:"companyId,omitempty"`
	Symbol              string                   `bson:"symbol,omitempty" json:"symbol,omitempty"`
	BookType            common.BookType          `bson:"bookType" json:"bookType"`
	ItemType            common.AIBatchItemType   `bson:"itemType" json:"itemType"`
	InputPayload        map[string]any           `bson:"inputPayload,omitempty" json:"inputPayload,omitempty"`
	InputHash           string                   `bson:"inputHash,omitempty" json:"inputHash,omitempty"`
	Status              common.AIBatchItemStatus `bson:"status" json:"status"`
	ResultPayload       map[string]any           `bson:"resultPayload,omitempty" json:"resultPayload,omitempty"`
	ValidationStatus    common.ValidationStatus  `bson:"validationStatus" json:"validationStatus"`
	ValidationErrors    []string                 `bson:"validationErrors,omitempty" json:"validationErrors,omitempty"`
	TargetReviewID      primitive.ObjectID       `bson:"targetReviewId,omitempty" json:"targetReviewId,omitempty"`
	TargetThesisID      primitive.ObjectID       `bson:"targetThesisId,omitempty" json:"targetThesisId,omitempty"`
	TargetEntityVersion int                      `bson:"targetEntityVersion,omitempty" json:"targetEntityVersion,omitempty"`
	ErrorSummary        string                   `bson:"errorSummary,omitempty" json:"errorSummary,omitempty"`
	CreatedAt           time.Time                `bson:"createdAt" json:"createdAt"`
	UpdatedAt           time.Time                `bson:"updatedAt" json:"updatedAt"`
	CompletedAt         *time.Time               `bson:"completedAt,omitempty" json:"completedAt,omitempty"`
	SchemaVersion       int                      `bson:"schemaVersion" json:"schemaVersion"`
}

var allowedAIBatchItemTransitions = map[common.AIBatchItemStatus]map[common.AIBatchItemStatus]struct{}{
	common.AIBatchItemStatusPending: {
		common.AIBatchItemStatusSubmitted: {},
		common.AIBatchItemStatusSkipped:   {},
	},
	common.AIBatchItemStatusSubmitted: {
		common.AIBatchItemStatusProcessing:    {},
		common.AIBatchItemStatusCompleted:     {},
		common.AIBatchItemStatusFailed:        {},
		common.AIBatchItemStatusInvalidOutput: {},
		common.AIBatchItemStatusSkipped:       {},
	},
	common.AIBatchItemStatusProcessing: {
		common.AIBatchItemStatusCompleted:     {},
		common.AIBatchItemStatusFailed:        {},
		common.AIBatchItemStatusInvalidOutput: {},
	},
	common.AIBatchItemStatusCompleted: {},
	common.AIBatchItemStatusFailed: {
		common.AIBatchItemStatusPending: {},
	},
	common.AIBatchItemStatusInvalidOutput: {
		common.AIBatchItemStatusPending: {},
	},
	common.AIBatchItemStatusSkipped: {},
}

func (item AIBatchItem) Validate() error {
	if err := common.RequireObjectID("aiBatchJobId", item.AIBatchJobID); err != nil {
		return err
	}
	if err := common.RequireObjectID("workflowRunId", item.WorkflowRunID); err != nil {
		return err
	}
	if !item.BookType.IsValid() {
		return fmt.Errorf("invalid bookType %q", item.BookType)
	}
	if !item.ItemType.IsValid() {
		return fmt.Errorf("invalid itemType %q", item.ItemType)
	}
	if !item.Status.IsValid() {
		return fmt.Errorf("invalid status %q", item.Status)
	}
	if !item.ValidationStatus.IsValid() {
		return fmt.Errorf("invalid validationStatus %q", item.ValidationStatus)
	}
	if err := common.ValidateStringSlice("validationErrors", item.ValidationErrors); err != nil {
		return err
	}
	if item.ValidationStatus == common.ValidationStatusInvalid && len(item.ValidationErrors) == 0 {
		return fmt.Errorf("validationErrors are required when validationStatus is invalid")
	}
	if item.Status == common.AIBatchItemStatusInvalidOutput && item.ValidationStatus != common.ValidationStatusInvalid {
		return fmt.Errorf("invalid_output items must be marked invalid")
	}
	if err := common.ValidateNonNegativeInt("targetEntityVersion", item.TargetEntityVersion); err != nil {
		return err
	}
	if err := common.RequireTime("createdAt", item.CreatedAt); err != nil {
		return err
	}
	if err := common.RequireTime("updatedAt", item.UpdatedAt); err != nil {
		return err
	}
	if err := common.ValidateTimestampOrder("createdAt", item.CreatedAt, "updatedAt", item.UpdatedAt); err != nil {
		return err
	}
	if err := common.ValidateOptionalTimestampOrder("createdAt", item.CreatedAt, "completedAt", item.CompletedAt); err != nil {
		return err
	}
	if err := common.ValidateSchemaVersion("schemaVersion", item.SchemaVersion); err != nil {
		return err
	}
	if item.IsTerminal() && item.CompletedAt == nil {
		return fmt.Errorf("terminal ai batch items require completedAt")
	}
	return nil
}

func (item AIBatchItem) IsTerminal() bool {
	switch item.Status {
	case common.AIBatchItemStatusCompleted, common.AIBatchItemStatusFailed, common.AIBatchItemStatusInvalidOutput, common.AIBatchItemStatusSkipped:
		return true
	default:
		return false
	}
}

func (item AIBatchItem) CanFinalize() bool {
	return item.Status == common.AIBatchItemStatusCompleted && item.ValidationStatus == common.ValidationStatusValid
}

func (item AIBatchItem) CanRetry() bool {
	switch item.Status {
	case common.AIBatchItemStatusFailed, common.AIBatchItemStatusInvalidOutput:
		return true
	default:
		return false
	}
}

func (item AIBatchItem) CanTransitionTo(next common.AIBatchItemStatus) bool {
	if item.Status == next {
		return true
	}
	nextStates, ok := allowedAIBatchItemTransitions[item.Status]
	if !ok {
		return false
	}
	_, ok = nextStates[next]
	return ok
}

func (item *AIBatchItem) TransitionTo(next common.AIBatchItemStatus, at time.Time) error {
	if item == nil {
		return fmt.Errorf("ai batch item is required")
	}
	if !next.IsValid() {
		return fmt.Errorf("invalid next batch item status %q", next)
	}
	if !item.CanTransitionTo(next) {
		return fmt.Errorf("invalid ai batch item transition from %q to %q", item.Status, next)
	}
	if err := common.RequireTime("transitionAt", at); err != nil {
		return err
	}
	item.Status = next
	item.UpdatedAt = at.UTC()
	if item.IsTerminal() {
		completedAt := at.UTC()
		item.CompletedAt = &completedAt
	}
	return nil
}

func (item *AIBatchItem) ResetForRetry(at time.Time) error {
	if item == nil {
		return fmt.Errorf("ai batch item is required")
	}
	if !item.CanRetry() {
		return fmt.Errorf("ai batch item cannot be retried from status %q", item.Status)
	}
	if err := common.RequireTime("retryAt", at); err != nil {
		return err
	}
	item.Status = common.AIBatchItemStatusPending
	item.ValidationStatus = common.ValidationStatusNotValidated
	item.ValidationErrors = nil
	item.ErrorSummary = ""
	item.ResultPayload = nil
	item.CompletedAt = nil
	item.UpdatedAt = at.UTC()
	return nil
}
