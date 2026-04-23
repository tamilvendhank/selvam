package aijob

import (
	"fmt"
	"time"

	"goserver/internal/domain/common"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

type JobPollLog struct {
	ID                       primitive.ObjectID      `bson:"_id,omitempty" json:"id,omitempty"`
	AIBatchJobID             primitive.ObjectID      `bson:"aiBatchJobId" json:"aiBatchJobId"`
	PolledAt                 time.Time               `bson:"polledAt" json:"polledAt"`
	StatusBefore             common.AIBatchJobStatus `bson:"statusBefore,omitempty" json:"statusBefore,omitempty"`
	StatusAfter              common.AIBatchJobStatus `bson:"statusAfter,omitempty" json:"statusAfter,omitempty"`
	ItemsCompletedDelta      int                     `bson:"itemsCompletedDelta,omitempty" json:"itemsCompletedDelta,omitempty"`
	ItemsFailedDelta         int                     `bson:"itemsFailedDelta,omitempty" json:"itemsFailedDelta,omitempty"`
	RawProviderStatusSummary map[string]any          `bson:"rawProviderStatusSummary,omitempty" json:"rawProviderStatusSummary,omitempty"`
	ErrorSummary             string                  `bson:"errorSummary,omitempty" json:"errorSummary,omitempty"`
	CreatedAt                time.Time               `bson:"createdAt" json:"createdAt"`
	SchemaVersion            int                     `bson:"schemaVersion" json:"schemaVersion"`
}

func (log JobPollLog) Validate() error {
	if err := common.RequireObjectID("aiBatchJobId", log.AIBatchJobID); err != nil {
		return err
	}
	if err := common.RequireTime("polledAt", log.PolledAt); err != nil {
		return err
	}
	if !log.StatusBefore.IsValid() {
		return fmt.Errorf("invalid statusBefore %q", log.StatusBefore)
	}
	if !log.StatusAfter.IsValid() {
		return fmt.Errorf("invalid statusAfter %q", log.StatusAfter)
	}
	if err := common.ValidateNonNegativeInt("itemsCompletedDelta", log.ItemsCompletedDelta); err != nil {
		return err
	}
	if err := common.ValidateNonNegativeInt("itemsFailedDelta", log.ItemsFailedDelta); err != nil {
		return err
	}
	if err := common.RequireTime("createdAt", log.CreatedAt); err != nil {
		return err
	}
	if err := common.ValidateTimestampOrder("polledAt", log.PolledAt, "createdAt", log.CreatedAt); err != nil {
		return err
	}
	if err := common.ValidateSchemaVersion("schemaVersion", log.SchemaVersion); err != nil {
		return err
	}
	return nil
}
