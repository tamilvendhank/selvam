package ports

import (
	"context"
	"time"

	"goserver/internal/platform/domain"
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
	CorrelationID   string
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

type SubmitBatchItem struct {
	CorrelationID   string
	ReferenceID     string
	ItemType        domain.BatchItemType
	Prompt          string
	InputPayload    map[string]any
	TemplateRecord  map[string]any
	Model           string
	ReasoningEffort string
	Metadata        map[string]any
}

type SubmitBatchRequest struct {
	JobType              domain.BatchJobType
	BookType             domain.BookType
	WorkflowRunID        string
	IdempotencyKey       string
	PromptVersion        string
	ModelName            string
	ResponseInstructions string
	ProviderMetadata     map[string]any
	Items                []SubmitBatchItem
}

type BatchSubmissionItem struct {
	CorrelationID      string
	ProviderItemHandle string
	Status             domain.BatchItemStatus
	Metadata           map[string]any
}

type BatchSubmissionResult struct {
	ProviderName      string
	ProviderJobHandle string
	LocalJobHandle    string
	Status            domain.BatchJobStatus
	SubmittedAt       *time.Time
	Metadata          map[string]any
	Items             []BatchSubmissionItem
}

type BatchStatusItem struct {
	CorrelationID string
	Status        domain.BatchItemStatus
	ErrorSummary  string
	Metadata      map[string]any
}

type BatchStatusResult struct {
	ProviderName         string
	ProviderJobHandle    string
	Status               domain.BatchJobStatus
	SubmittedAt          *time.Time
	LastPolledAt         *time.Time
	CompletedAt          *time.Time
	ResultAvailable      bool
	ItemsCompletedCount  int
	ItemsFailedCount     int
	ItemsProcessingCount int
	Retryable            bool
	RawProviderStatus    map[string]any
	Items                []BatchStatusItem
}

type BatchResultItem struct {
	CorrelationID    string
	Status           domain.BatchItemStatus
	OutputPayload    map[string]any
	ErrorSummary     string
	Retryable        bool
	ProviderMetadata map[string]any
}

type BatchResultsResult struct {
	ProviderName      string
	ProviderJobHandle string
	Status            domain.BatchJobStatus
	CompletedAt       *time.Time
	RawPayload        map[string]any
	Items             []BatchResultItem
}

type AIBatchEngine interface {
	SubmitBatch(ctx context.Context, request SubmitBatchRequest) (*BatchSubmissionResult, error)
	GetBatchStatus(ctx context.Context, jobHandle string) (*BatchStatusResult, error)
	GetBatchResults(ctx context.Context, jobHandle string) (*BatchResultsResult, error)
}

type TimeProvider interface {
	Now() time.Time
}

type IDGenerator interface {
	NewID() string
}
