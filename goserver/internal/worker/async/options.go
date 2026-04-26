package async

import (
	"fmt"

	domaincommon "goserver/internal/domain/common"
	servicecommon "goserver/internal/service/common"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

type AsyncWorkerOptions struct {
	WorkflowRunID    primitive.ObjectID
	BatchJobID       primitive.ObjectID
	BatchItemID      primitive.ObjectID
	ReviewID         primitive.ObjectID
	CompanyID        primitive.ObjectID
	BookType         domaincommon.BookType
	JobType          domaincommon.AIBatchJobType
	ItemType         domaincommon.AIBatchItemType
	PollOnlyStatuses []domaincommon.AIBatchJobStatus

	MaxJobsPerRun      int
	MaxItemsPerRun     int
	MaxReviewsPerRun   int
	MaxWorkflowsPerRun int

	DryRun                bool
	Force                 bool
	StrictMode            bool
	Revalidate            bool
	SupersedePrior        bool
	IncludeCompletedItems bool
	AllowedStepRange      servicecommon.StepRange

	InitiatedBy   string
	CorrelationID string
	Metadata      map[string]any
}

func (options AsyncWorkerOptions) Validate() error {
	if options.MaxJobsPerRun < 0 {
		return fmt.Errorf("maxJobsPerRun must be zero or greater")
	}
	if options.MaxItemsPerRun < 0 {
		return fmt.Errorf("maxItemsPerRun must be zero or greater")
	}
	if options.MaxReviewsPerRun < 0 {
		return fmt.Errorf("maxReviewsPerRun must be zero or greater")
	}
	if options.MaxWorkflowsPerRun < 0 {
		return fmt.Errorf("maxWorkflowsPerRun must be zero or greater")
	}
	if err := servicecommon.ValidateOptionalBookType(options.BookType); err != nil {
		return err
	}
	if err := servicecommon.ValidateOptionalJobType(options.JobType); err != nil {
		return err
	}
	if err := servicecommon.ValidateOptionalItemType(options.ItemType); err != nil {
		return err
	}
	if err := servicecommon.ValidateOptionalBatchJobStatuses(options.PollOnlyStatuses); err != nil {
		return err
	}
	if err := options.AllowedStepRange.Validate(); err != nil {
		return err
	}
	if err := servicecommon.ValidateOptionalText("initiatedBy", options.InitiatedBy); err != nil {
		return err
	}
	return servicecommon.ValidateOptionalText("correlationId", options.CorrelationID)
}

func (options AsyncWorkerOptions) metadata() map[string]any {
	metadata := make(map[string]any, len(options.Metadata)+8)
	for key, value := range options.Metadata {
		metadata[key] = value
	}
	if !options.WorkflowRunID.IsZero() {
		metadata["workflow_run_id"] = options.WorkflowRunID.Hex()
	}
	if options.BookType != "" {
		metadata["book_type"] = options.BookType
	}
	if options.JobType != "" {
		metadata["job_type"] = options.JobType
	}
	if options.ItemType != "" {
		metadata["item_type"] = options.ItemType
	}
	if options.DryRun {
		metadata["dry_run"] = true
	}
	if options.Force {
		metadata["force"] = true
	}
	if options.CorrelationID != "" {
		metadata["correlation_id"] = options.CorrelationID
	}
	return metadata
}
