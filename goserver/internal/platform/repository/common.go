package repository

import (
	"errors"
	"time"
)

var (
	// ErrNotFound indicates that the requested aggregate or projection does not exist.
	ErrNotFound = errors.New("repository: not found")
	// ErrAlreadyExists indicates that the requested create/upsert operation would violate uniqueness.
	ErrAlreadyExists = errors.New("repository: already exists")
	// ErrConflict indicates a storage-level write conflict that is not necessarily a lifecycle violation.
	ErrConflict = errors.New("repository: conflict")
	// ErrImmutableState indicates the target document can no longer be mutated safely.
	ErrImmutableState = errors.New("repository: immutable state")
	// ErrInvalidTransition indicates a requested lifecycle/status transition is not allowed.
	ErrInvalidTransition = errors.New("repository: invalid transition")
	// ErrPreconditionFailed indicates an optimistic precondition did not match the current persisted state.
	ErrPreconditionFailed = errors.New("repository: precondition failed")
)

// SortOrder controls ascending or descending ordering for list operations.
type SortOrder string

const (
	SortOrderAscending  SortOrder = "asc"
	SortOrderDescending SortOrder = "desc"
)

// PageOptions uses offset-based pagination to remain compatible with the existing codebase,
// admin/UI patterns, and deterministic operator-driven queries.
type PageOptions struct {
	PageSize int
	Offset   int
}

// PageInfo describes the page that was actually returned.
type PageInfo struct {
	PageSize int
	Offset   int
	HasMore  bool
}

// ListResult wraps paginated repository responses without forcing a count query.
type ListResult[T any] struct {
	Items []T
	Page  PageInfo
}

// TimeRange is a repository-level helper for inclusive time filtering.
type TimeRange struct {
	From *time.Time
	To   *time.Time
}

// MutationMetadata carries optional audit context for patch-style mutations.
// Implementations may persist the actor/reason directly or map them to change metadata.
type MutationMetadata struct {
	OccurredAt time.Time
	Actor      string
	Reason     string
}

// MetadataPatch makes merge-vs-replace intent explicit for metadata blobs.
type MetadataPatch struct {
	Values  map[string]any
	Replace bool
}
