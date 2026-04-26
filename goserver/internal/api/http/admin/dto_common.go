package admin

import (
	"time"

	domaincommon "goserver/internal/domain/common"
	servicecommon "goserver/internal/service/common"
)

type PageDTO struct {
	Limit   int  `json:"limit"`
	Offset  int  `json:"offset"`
	HasMore bool `json:"hasMore"`
}

type PagedResponseDTO[T any] struct {
	Items []T     `json:"items"`
	Page  PageDTO `json:"page"`
}

type ErrorResponseDTO struct {
	Error   string `json:"error"`
	Code    string `json:"code,omitempty"`
	Details any    `json:"details,omitempty"`
}

type StatusCountsDTO struct {
	Total    int            `json:"total"`
	ByStatus map[string]int `json:"byStatus,omitempty"`
	Partial  bool           `json:"partial,omitempty"`
}

type ValidationCountsDTO struct {
	Total        int            `json:"total"`
	ByStatus     map[string]int `json:"byStatus,omitempty"`
	ErrorCount   int            `json:"errorCount,omitempty"`
	InvalidCount int            `json:"invalidCount,omitempty"`
	Partial      bool           `json:"partial,omitempty"`
}

type AdminActionRequestDTO struct {
	DryRun                bool                           `json:"dryRun,omitempty"`
	Force                 bool                           `json:"force,omitempty"`
	StrictMode            bool                           `json:"strictMode,omitempty"`
	Revalidate            bool                           `json:"revalidate,omitempty"`
	SupersedePrior        bool                           `json:"supersedePrior,omitempty"`
	IncludeCompletedItems bool                           `json:"includeCompletedItems,omitempty"`
	ContinueWorkflow      bool                           `json:"continueWorkflow,omitempty"`
	Reason                string                         `json:"reason,omitempty"`
	InitiatedBy           string                         `json:"initiatedBy,omitempty"`
	CorrelationID         string                         `json:"correlationId,omitempty"`
	MaxJobs               int                            `json:"maxJobs,omitempty"`
	MaxItems              int                            `json:"maxItems,omitempty"`
	MaxReviews            int                            `json:"maxReviews,omitempty"`
	MaxWorkflows          int                            `json:"maxWorkflows,omitempty"`
	AllowedStepRange      servicecommon.StepRange        `json:"allowedStepRange,omitempty"`
	PollOnlyStatuses      []domaincommon.AIBatchJobStatus `json:"pollOnlyStatuses,omitempty"`
}

type AdminActionResponseDTO struct {
	Action        string    `json:"action"`
	Status        string    `json:"status"`
	Success       bool      `json:"success"`
	DryRun        bool      `json:"dryRun,omitempty"`
	WorkflowRunID string    `json:"workflowRunId,omitempty"`
	AIBatchJobID  string    `json:"aiBatchJobId,omitempty"`
	AIBatchItemID string    `json:"aiBatchItemId,omitempty"`
	ReviewID      string    `json:"reviewId,omitempty"`
	Message       string    `json:"message,omitempty"`
	Result        any       `json:"result,omitempty"`
	Summary       any       `json:"summary,omitempty"`
	StartedAt     time.Time `json:"startedAt,omitempty"`
	CompletedAt   time.Time `json:"completedAt,omitempty"`
}

type WorkflowStartRequestDTO struct {
	RunType        domaincommon.WorkflowRunType `json:"runType,omitempty"`
	BookType       domaincommon.BookType        `json:"bookType,omitempty"`
	CompanyIDs     []string                     `json:"companyIds,omitempty"`
	Limit          int                          `json:"limit,omitempty"`
	DryRun         bool                         `json:"dryRun,omitempty"`
	Force          bool                         `json:"force,omitempty"`
	IdempotencyKey string                       `json:"idempotencyKey,omitempty"`
	Notes          string                       `json:"notes,omitempty"`
	RequestedBy    string                       `json:"requestedBy,omitempty"`
	CorrelationID  string                       `json:"correlationId,omitempty"`
	Metadata       map[string]any               `json:"metadata,omitempty"`
}

type WorkflowStartResultDTO struct {
	WorkflowRunID      string                         `json:"workflowRunId,omitempty"`
	BookType           domaincommon.BookType          `json:"bookType,omitempty"`
	RunType            domaincommon.WorkflowRunType   `json:"runType,omitempty"`
	Status             domaincommon.WorkflowRunStatus `json:"status,omitempty"`
	AsyncWaitRequired  bool                           `json:"asyncWaitRequired,omitempty"`
	CreatedBatchJobIDs []string                       `json:"createdBatchJobIds,omitempty"`
	Summary            any                            `json:"summary,omitempty"`
}
