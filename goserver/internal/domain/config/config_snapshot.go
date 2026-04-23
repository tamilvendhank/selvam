package config

import (
	"fmt"
	"time"

	"goserver/internal/domain/common"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

type ConfigSnapshot struct {
	ID            primitive.ObjectID   `bson:"_id,omitempty" json:"id,omitempty"`
	BookType      common.BookType      `bson:"bookType" json:"bookType"`
	Mode          common.InvestingMode `bson:"mode,omitempty" json:"mode,omitempty"`
	SchemaVersion int                  `bson:"schemaVersion" json:"schemaVersion"`
	RawConfig     map[string]any       `bson:"rawConfig" json:"rawConfig"`
	CreatedAt     time.Time            `bson:"createdAt" json:"createdAt"`
}

func (snapshot ConfigSnapshot) Validate() error {
	if !snapshot.BookType.IsValid() {
		return fmt.Errorf("invalid bookType %q", snapshot.BookType)
	}
	if snapshot.BookType == common.BookTypeInvesting {
		if !snapshot.Mode.IsValid() {
			return fmt.Errorf("invalid investing mode %q", snapshot.Mode)
		}
	} else if snapshot.Mode != "" {
		return fmt.Errorf("mode is currently only supported for investing config snapshots")
	}
	if len(snapshot.RawConfig) == 0 {
		return fmt.Errorf("rawConfig is required")
	}
	if err := common.ValidateSchemaVersion("schemaVersion", snapshot.SchemaVersion); err != nil {
		return err
	}
	if err := common.RequireTime("createdAt", snapshot.CreatedAt); err != nil {
		return err
	}
	return nil
}
