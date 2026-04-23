package override

import (
	"fmt"
	"time"

	"goserver/internal/domain/common"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

// ManualOverride captures a human override of an investing action recommendation.
type ManualOverride struct {
	ID               primitive.ObjectID         `bson:"_id,omitempty" json:"id,omitempty"`
	CompanyID        primitive.ObjectID         `bson:"companyId" json:"companyId"`
	ReviewID         primitive.ObjectID         `bson:"reviewId" json:"reviewId"`
	BookType         common.BookType            `bson:"bookType" json:"bookType"`
	OriginalAction   common.InvestingActionType `bson:"originalAction" json:"originalAction"`
	OverriddenAction common.InvestingActionType `bson:"overriddenAction" json:"overriddenAction"`
	OverrideReason   string                     `bson:"overrideReason" json:"overrideReason"`
	OverrideBy       string                     `bson:"overrideBy" json:"overrideBy"`
	OverrideDate     time.Time                  `bson:"overrideDate" json:"overrideDate"`
	CreatedAt        time.Time                  `bson:"createdAt" json:"createdAt"`
	SchemaVersion    int                        `bson:"schemaVersion" json:"schemaVersion"`
}

func (override ManualOverride) Validate() error {
	if err := common.RequireObjectID("companyId", override.CompanyID); err != nil {
		return err
	}
	if err := common.RequireObjectID("reviewId", override.ReviewID); err != nil {
		return err
	}
	if !override.BookType.IsValid() {
		return fmt.Errorf("invalid bookType %q", override.BookType)
	}
	if !override.OriginalAction.IsValid() {
		return fmt.Errorf("invalid originalAction %q", override.OriginalAction)
	}
	if !override.OverriddenAction.IsValid() {
		return fmt.Errorf("invalid overriddenAction %q", override.OverriddenAction)
	}
	if err := common.RequireString("overrideReason", override.OverrideReason); err != nil {
		return err
	}
	if err := common.RequireString("overrideBy", override.OverrideBy); err != nil {
		return err
	}
	if err := common.RequireTime("overrideDate", override.OverrideDate); err != nil {
		return err
	}
	if err := common.RequireTime("createdAt", override.CreatedAt); err != nil {
		return err
	}
	if err := common.ValidateTimestampOrder("overrideDate", override.OverrideDate, "createdAt", override.CreatedAt); err != nil {
		return err
	}
	if err := common.ValidateSchemaVersion("schemaVersion", override.SchemaVersion); err != nil {
		return err
	}
	return nil
}
