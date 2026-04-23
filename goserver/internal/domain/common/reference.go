package common

import "fmt"

// PayloadReference points to large request/response payloads stored outside the aggregate document.
type PayloadReference struct {
	StorageProvider string         `bson:"storageProvider,omitempty" json:"storageProvider,omitempty"`
	URI             string         `bson:"uri,omitempty" json:"uri,omitempty"`
	Bucket          string         `bson:"bucket,omitempty" json:"bucket,omitempty"`
	Path            string         `bson:"path,omitempty" json:"path,omitempty"`
	Checksum        string         `bson:"checksum,omitempty" json:"checksum,omitempty"`
	Metadata        map[string]any `bson:"metadata,omitempty" json:"metadata,omitempty"`
}

func (reference *PayloadReference) Validate() error {
	if reference == nil {
		return nil
	}
	if reference.URI == "" && reference.Path == "" && reference.Bucket == "" {
		return fmt.Errorf("payload reference must include uri, path, or bucket")
	}
	return nil
}
