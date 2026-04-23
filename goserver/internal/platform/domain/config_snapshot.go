package domain

import (
	"fmt"
	"strings"
	"time"
)

type ConfigSnapshot struct {
	ID            string         `json:"id" bson:"-"`
	BookType      BookType       `json:"bookType" bson:"bookType"`
	Mode          string         `json:"mode" bson:"mode"`
	SchemaVersion string         `json:"schemaVersion" bson:"schemaVersion"`
	ConfigJSON    map[string]any `json:"configJson" bson:"configJson"`
	CreatedAt     time.Time      `json:"createdAt" bson:"createdAt"`
}

func (snapshot *ConfigSnapshot) Validate() error {
	if snapshot == nil {
		return fmt.Errorf("config snapshot is required")
	}
	if !IsValidBookType(snapshot.BookType) {
		return fmt.Errorf("invalid config snapshot book type %q", snapshot.BookType)
	}
	if strings.TrimSpace(snapshot.Mode) == "" {
		return fmt.Errorf("config snapshot mode is required")
	}
	if strings.TrimSpace(snapshot.SchemaVersion) == "" {
		return fmt.Errorf("config snapshot schema version is required")
	}
	if snapshot.ConfigJSON == nil {
		return fmt.Errorf("config snapshot configJson is required")
	}
	if err := ValidateNonZeroTime("config snapshot createdAt", snapshot.CreatedAt); err != nil {
		return err
	}

	return nil
}
