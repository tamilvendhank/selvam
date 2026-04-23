package aijob

import (
	"fmt"
	"time"

	"goserver/internal/domain/common"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

type AIBatchJob struct {
	ID                   primitive.ObjectID       `bson:"_id,omitempty" json:"id,omitempty"`
	JobType              common.AIBatchJobType    `bson:"jobType" json:"jobType"`
	WorkflowRunID        primitive.ObjectID       `bson:"workflowRunId" json:"workflowRunId"`
	BookType             common.BookType          `bson:"bookType" json:"bookType"`
	ProviderName         string                   `bson:"providerName" json:"providerName"`
	ProviderJobHandle    string                   `bson:"providerJobHandle,omitempty" json:"providerJobHandle,omitempty"`
	LocalJobHandle       string                   `bson:"localJobHandle,omitempty" json:"localJobHandle,omitempty"`
	Status               common.AIBatchJobStatus  `bson:"status" json:"status"`
	SubmissionPayloadRef *common.PayloadReference `bson:"submissionPayloadRef,omitempty" json:"submissionPayloadRef,omitempty"`
	ResultPayloadRef     *common.PayloadReference `bson:"resultPayloadRef,omitempty" json:"resultPayloadRef,omitempty"`
	SubmittedAt          *time.Time               `bson:"submittedAt,omitempty" json:"submittedAt,omitempty"`
	LastPolledAt         *time.Time               `bson:"lastPolledAt,omitempty" json:"lastPolledAt,omitempty"`
	CompletedAt          *time.Time               `bson:"completedAt,omitempty" json:"completedAt,omitempty"`
	FailedAt             *time.Time               `bson:"failedAt,omitempty" json:"failedAt,omitempty"`
	ErrorSummary         string                   `bson:"errorSummary,omitempty" json:"errorSummary,omitempty"`
	RetryCount           int                      `bson:"retryCount" json:"retryCount"`
	MaxRetryCount        int                      `bson:"maxRetryCount" json:"maxRetryCount"`
	IdempotencyKey       string                   `bson:"idempotencyKey,omitempty" json:"idempotencyKey,omitempty"`
	SchemaVersion        int                      `bson:"schemaVersion" json:"schemaVersion"`
	CreatedAt            time.Time                `bson:"createdAt" json:"createdAt"`
	UpdatedAt            time.Time                `bson:"updatedAt" json:"updatedAt"`
}

var allowedAIBatchJobTransitions = map[common.AIBatchJobStatus]map[common.AIBatchJobStatus]struct{}{
	common.AIBatchJobStatusCreated: {
		common.AIBatchJobStatusSubmitted: {},
		common.AIBatchJobStatusFailed:    {},
		common.AIBatchJobStatusCancelled: {},
	},
	common.AIBatchJobStatusSubmitted: {
		common.AIBatchJobStatusRunning:            {},
		common.AIBatchJobStatusPartiallyCompleted: {},
		common.AIBatchJobStatusCompleted:          {},
		common.AIBatchJobStatusFailed:             {},
		common.AIBatchJobStatusCancelled:          {},
		common.AIBatchJobStatusTimedOut:           {},
	},
	common.AIBatchJobStatusRunning: {
		common.AIBatchJobStatusPartiallyCompleted: {},
		common.AIBatchJobStatusCompleted:          {},
		common.AIBatchJobStatusFailed:             {},
		common.AIBatchJobStatusCancelled:          {},
		common.AIBatchJobStatusTimedOut:           {},
	},
	common.AIBatchJobStatusPartiallyCompleted: {
		common.AIBatchJobStatusRunning:   {},
		common.AIBatchJobStatusCompleted: {},
		common.AIBatchJobStatusFailed:    {},
		common.AIBatchJobStatusCancelled: {},
		common.AIBatchJobStatusTimedOut:  {},
	},
	common.AIBatchJobStatusCompleted: {},
	common.AIBatchJobStatusFailed: {
		common.AIBatchJobStatusCreated: {},
	},
	common.AIBatchJobStatusCancelled: {},
	common.AIBatchJobStatusTimedOut: {
		common.AIBatchJobStatusCreated: {},
	},
}

func (job AIBatchJob) Validate() error {
	if !job.JobType.IsValid() {
		return fmt.Errorf("invalid jobType %q", job.JobType)
	}
	if err := common.RequireObjectID("workflowRunId", job.WorkflowRunID); err != nil {
		return err
	}
	if !job.BookType.IsValid() {
		return fmt.Errorf("invalid bookType %q", job.BookType)
	}
	if err := common.RequireString("providerName", job.ProviderName); err != nil {
		return err
	}
	if !job.Status.IsValid() {
		return fmt.Errorf("invalid status %q", job.Status)
	}
	if job.SubmissionPayloadRef != nil {
		if err := job.SubmissionPayloadRef.Validate(); err != nil {
			return err
		}
	}
	if job.ResultPayloadRef != nil {
		if err := job.ResultPayloadRef.Validate(); err != nil {
			return err
		}
	}
	if err := common.ValidateNonNegativeInt("retryCount", job.RetryCount); err != nil {
		return err
	}
	if err := common.ValidateNonNegativeInt("maxRetryCount", job.MaxRetryCount); err != nil {
		return err
	}
	if job.RetryCount > job.MaxRetryCount {
		return fmt.Errorf("retryCount cannot exceed maxRetryCount")
	}
	if job.SubmittedAt != nil && job.SubmittedAt.IsZero() {
		return fmt.Errorf("submittedAt cannot be zero")
	}
	if job.LastPolledAt != nil && job.LastPolledAt.IsZero() {
		return fmt.Errorf("lastPolledAt cannot be zero")
	}
	if err := common.ValidateOptionalTimestampOrder("createdAt", job.CreatedAt, "submittedAt", job.SubmittedAt); err != nil {
		return err
	}
	if err := common.ValidateOptionalTimestampOrder("createdAt", job.CreatedAt, "completedAt", job.CompletedAt); err != nil {
		return err
	}
	if err := common.ValidateOptionalTimestampOrder("createdAt", job.CreatedAt, "failedAt", job.FailedAt); err != nil {
		return err
	}
	if err := common.RequireTime("createdAt", job.CreatedAt); err != nil {
		return err
	}
	if err := common.RequireTime("updatedAt", job.UpdatedAt); err != nil {
		return err
	}
	if err := common.ValidateTimestampOrder("createdAt", job.CreatedAt, "updatedAt", job.UpdatedAt); err != nil {
		return err
	}
	if err := common.ValidateSchemaVersion("schemaVersion", job.SchemaVersion); err != nil {
		return err
	}
	if job.IsTerminal() {
		if job.Status == common.AIBatchJobStatusCompleted && job.CompletedAt == nil {
			return fmt.Errorf("completed jobs require completedAt")
		}
		if job.Status == common.AIBatchJobStatusFailed && job.FailedAt == nil {
			return fmt.Errorf("failed jobs require failedAt")
		}
	}
	return nil
}

func (job AIBatchJob) IsTerminal() bool {
	switch job.Status {
	case common.AIBatchJobStatusCompleted, common.AIBatchJobStatusFailed, common.AIBatchJobStatusCancelled, common.AIBatchJobStatusTimedOut:
		return true
	default:
		return false
	}
}

func (job AIBatchJob) CanPoll() bool {
	switch job.Status {
	case common.AIBatchJobStatusSubmitted, common.AIBatchJobStatusRunning, common.AIBatchJobStatusPartiallyCompleted:
		return true
	default:
		return false
	}
}

func (job AIBatchJob) CanRetry() bool {
	switch job.Status {
	case common.AIBatchJobStatusFailed, common.AIBatchJobStatusTimedOut:
		return job.RetryCount < job.MaxRetryCount
	default:
		return false
	}
}

func (job AIBatchJob) CanTransitionTo(next common.AIBatchJobStatus) bool {
	if job.Status == next {
		return true
	}
	nextStates, ok := allowedAIBatchJobTransitions[job.Status]
	if !ok {
		return false
	}
	_, ok = nextStates[next]
	return ok
}

func (job *AIBatchJob) TransitionTo(next common.AIBatchJobStatus, at time.Time) error {
	if job == nil {
		return fmt.Errorf("ai batch job is required")
	}
	if !next.IsValid() {
		return fmt.Errorf("invalid next batch job status %q", next)
	}
	if !job.CanTransitionTo(next) {
		return fmt.Errorf("invalid ai batch job transition from %q to %q", job.Status, next)
	}
	if err := common.RequireTime("transitionAt", at); err != nil {
		return err
	}
	job.Status = next
	job.UpdatedAt = at.UTC()
	switch next {
	case common.AIBatchJobStatusSubmitted:
		submittedAt := at.UTC()
		job.SubmittedAt = &submittedAt
	case common.AIBatchJobStatusCompleted:
		completedAt := at.UTC()
		job.CompletedAt = &completedAt
	case common.AIBatchJobStatusFailed:
		failedAt := at.UTC()
		job.FailedAt = &failedAt
	}
	return nil
}

func (job *AIBatchJob) PrepareRetry(at time.Time) error {
	if job == nil {
		return fmt.Errorf("ai batch job is required")
	}
	if !job.CanRetry() {
		return fmt.Errorf("ai batch job cannot be retried from status %q", job.Status)
	}
	if err := common.RequireTime("retryAt", at); err != nil {
		return err
	}
	job.RetryCount++
	job.Status = common.AIBatchJobStatusCreated
	job.ErrorSummary = ""
	job.SubmittedAt = nil
	job.CompletedAt = nil
	job.FailedAt = nil
	job.LastPolledAt = nil
	job.UpdatedAt = at.UTC()
	return nil
}
