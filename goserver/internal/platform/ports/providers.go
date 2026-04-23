package ports

import (
	"context"
	"time"
)

type FinancialDataProvider interface {
	LoadFinancialSnapshot(ctx context.Context, symbol string, asOf time.Time) (map[string]any, error)
}

type PriceDataProvider interface {
	LoadPriceSnapshot(ctx context.Context, symbol string, asOf time.Time) (map[string]any, error)
}

type TextDocumentProvider interface {
	LoadDocumentMetadata(ctx context.Context, symbol string) ([]map[string]any, error)
}

type AIReviewBatchItem struct {
	ReferenceID     string
	Prompt          string
	TemplateRecord  map[string]any
	Model           string
	ReasoningEffort string
	Metadata        map[string]any
}

type AIReviewBatchRequest struct {
	BookType             string
	PromptVersion        string
	ModelName            string
	ResponseInstructions string
	Items                []AIReviewBatchItem
}

type AIAsyncTask struct {
	Provider            string
	TaskKind            string
	LocalObjectType     string
	LocalObjectID       string
	SubmissionID        string
	RepresentativeJobID string
	BatchID             string
	JobIDs              []string
	Status              string
	ResultAvailable     bool
	SubmittedAt         *time.Time
	LastSyncedAt        *time.Time
	Metadata            map[string]any
}

type AIReviewEngine interface {
	SubmitReviewBatch(ctx context.Context, request AIReviewBatchRequest) (*AIAsyncTask, error)
	RefreshTask(ctx context.Context, task AIAsyncTask) (*AIAsyncTask, error)
}

type TimeProvider interface {
	Now() time.Time
}

type IDGenerator interface {
	NewID() string
}
