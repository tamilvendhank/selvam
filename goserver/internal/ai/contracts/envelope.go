package contracts

import (
	"time"

	"goserver/internal/domain/common"
)

type AIReviewInputEnvelope[T any] struct {
	SchemaVersion       string          `json:"schema_version"`
	PromptVersion       string          `json:"prompt_version"`
	OutputSchemaVersion string          `json:"output_schema_version"`
	ItemCorrelationID   string          `json:"item_correlation_id"`
	WorkflowRunID       string          `json:"workflow_run_id"`
	BatchJobID          string          `json:"batch_job_id,omitempty"`
	BatchItemID         string          `json:"batch_item_id,omitempty"`
	CompanyID           string          `json:"company_id"`
	Symbol              string          `json:"symbol"`
	BookType            common.BookType `json:"book_type"`
	ReviewType          ReviewType      `json:"review_type"`
	GeneratedAt         time.Time       `json:"generated_at"`
	ConfigSnapshotID    string          `json:"config_snapshot_id"`
	Payload             T               `json:"payload"`
}

type AIReviewOutputEnvelope[T any] struct {
	SchemaVersion       string          `json:"schema_version"`
	PromptVersion       string          `json:"prompt_version"`
	OutputSchemaVersion string          `json:"output_schema_version"`
	ItemCorrelationID   string          `json:"item_correlation_id"`
	WorkflowRunID       string          `json:"workflow_run_id"`
	BatchJobID          string          `json:"batch_job_id,omitempty"`
	BatchItemID         string          `json:"batch_item_id,omitempty"`
	CompanyID           string          `json:"company_id"`
	Symbol              string          `json:"symbol"`
	BookType            common.BookType `json:"book_type"`
	ReviewType          ReviewType      `json:"review_type"`
	ModelName           string          `json:"model_name,omitempty"`
	GeneratedAt         *time.Time      `json:"generated_at,omitempty"`
	Payload             T               `json:"payload"`
}

type InvestingReviewInputEnvelope = AIReviewInputEnvelope[InvestingReviewInputPayload]
type InvestingReviewOutputEnvelope = AIReviewOutputEnvelope[InvestingReviewOutputPayload]

type AIBatchItemInputContract[T any] struct {
	ItemCorrelationID   string                 `json:"item_correlation_id"`
	ItemType            common.AIBatchItemType `json:"item_type"`
	CompanyID           string                 `json:"company_id"`
	Symbol              string                 `json:"symbol"`
	WorkflowRunID       string                 `json:"workflow_run_id"`
	ReviewType          ReviewType             `json:"review_type"`
	InputSchemaVersion  string                 `json:"input_schema_version"`
	OutputSchemaVersion string                 `json:"output_schema_version"`
	PromptVersion       string                 `json:"prompt_version"`
	Payload             T                      `json:"payload"`
}

type InvestingReviewBatchItemInput = AIBatchItemInputContract[InvestingReviewInputPayload]
