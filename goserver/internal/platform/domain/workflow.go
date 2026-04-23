package domain

import (
	"fmt"
	"strings"
	"time"
)

type WorkflowStepError struct {
	Code    string `json:"code,omitempty" bson:"code,omitempty"`
	Message string `json:"message,omitempty" bson:"message,omitempty"`
}

type AsyncTaskReference struct {
	Provider            string          `json:"provider,omitempty" bson:"provider,omitempty"`
	TaskKind            string          `json:"taskKind,omitempty" bson:"taskKind,omitempty"`
	LocalObjectType     string          `json:"localObjectType,omitempty" bson:"localObjectType,omitempty"`
	LocalObjectID       string          `json:"localObjectId,omitempty" bson:"localObjectId,omitempty"`
	SubmissionID        string          `json:"submissionId,omitempty" bson:"submissionId,omitempty"`
	RepresentativeJobID string          `json:"representativeJobId,omitempty" bson:"representativeJobId,omitempty"`
	BatchID             string          `json:"batchId,omitempty" bson:"batchId,omitempty"`
	JobIDs              []string        `json:"jobIds,omitempty" bson:"jobIds,omitempty"`
	Status              AsyncTaskStatus `json:"status,omitempty" bson:"status,omitempty"`
	ResultAvailable     bool            `json:"resultAvailable" bson:"resultAvailable"`
	SubmittedAt         *time.Time      `json:"submittedAt,omitempty" bson:"submittedAt,omitempty"`
	LastSyncedAt        *time.Time      `json:"lastSyncedAt,omitempty" bson:"lastSyncedAt,omitempty"`
	Metadata            map[string]any  `json:"metadata,omitempty" bson:"metadata,omitempty"`
}

func (reference *AsyncTaskReference) Validate() error {
	if reference == nil {
		return nil
	}
	if reference.Status != "" && !IsValidAsyncTaskStatus(reference.Status) {
		return fmt.Errorf("invalid async task status %q", reference.Status)
	}

	return nil
}

type WorkflowStepStatus struct {
	StepName       string                 `json:"stepName" bson:"stepName"`
	Status         WorkflowStepStatusType `json:"status" bson:"status"`
	StartedAt      *time.Time             `json:"startedAt,omitempty" bson:"startedAt,omitempty"`
	CompletedAt    *time.Time             `json:"completedAt,omitempty" bson:"completedAt,omitempty"`
	DurationMs     int64                  `json:"durationMs,omitempty" bson:"durationMs,omitempty"`
	InputSnapshot  map[string]any         `json:"inputSnapshot,omitempty" bson:"inputSnapshot,omitempty"`
	OutputSnapshot map[string]any         `json:"outputSnapshot,omitempty" bson:"outputSnapshot,omitempty"`
	Error          *WorkflowStepError     `json:"error,omitempty" bson:"error,omitempty"`
	AsyncTask      *AsyncTaskReference    `json:"asyncTask,omitempty" bson:"asyncTask,omitempty"`
}

func (status *WorkflowStepStatus) Validate() error {
	if status == nil {
		return nil
	}
	if strings.TrimSpace(status.StepName) == "" {
		return fmt.Errorf("workflow step name is required")
	}
	if !IsValidWorkflowStepStatus(status.Status) {
		return fmt.Errorf("invalid workflow step status %q", status.Status)
	}
	if err := status.AsyncTask.Validate(); err != nil {
		return err
	}

	return nil
}

type WorkflowRun struct {
	ID                    string               `json:"id" bson:"-"`
	BookType              BookType             `json:"bookType" bson:"bookType"`
	RunType               WorkflowRunType      `json:"runType" bson:"runType"`
	Mode                  string               `json:"mode,omitempty" bson:"mode,omitempty"`
	Status                WorkflowRunStatus    `json:"status" bson:"status"`
	StartedAt             time.Time            `json:"startedAt" bson:"startedAt"`
	CompletedAt           *time.Time           `json:"completedAt,omitempty" bson:"completedAt,omitempty"`
	ConfigSnapshotID      string               `json:"configSnapshotId" bson:"configSnapshotId"`
	CompaniesScannedCount int                  `json:"companiesScannedCount,omitempty" bson:"companiesScannedCount,omitempty"`
	ReviewsCreatedCount   int                  `json:"reviewsCreatedCount,omitempty" bson:"reviewsCreatedCount,omitempty"`
	ErrorsCount           int                  `json:"errorsCount,omitempty" bson:"errorsCount,omitempty"`
	DryRun                bool                 `json:"dryRun" bson:"dryRun"`
	ReplayFromRunID       string               `json:"replayFromRunId,omitempty" bson:"replayFromRunId,omitempty"`
	IdempotencyKey        string               `json:"idempotencyKey,omitempty" bson:"idempotencyKey,omitempty"`
	Notes                 string               `json:"notes,omitempty" bson:"notes,omitempty"`
	RequestMetadata       map[string]any       `json:"requestMetadata,omitempty" bson:"requestMetadata,omitempty"`
	StepStatuses          []WorkflowStepStatus `json:"stepStatuses,omitempty" bson:"stepStatuses,omitempty"`
	SchemaVersion         string               `json:"schemaVersion" bson:"schemaVersion"`
	CreatedAt             time.Time            `json:"createdAt" bson:"createdAt"`
	UpdatedAt             time.Time            `json:"updatedAt" bson:"updatedAt"`
}

func (run *WorkflowRun) Validate() error {
	if run == nil {
		return fmt.Errorf("workflow run is required")
	}
	if !IsValidBookType(run.BookType) {
		return fmt.Errorf("invalid workflow run book type %q", run.BookType)
	}
	if !IsValidWorkflowRunType(run.RunType) {
		return fmt.Errorf("invalid workflow run type %q", run.RunType)
	}
	if !IsValidWorkflowRunStatus(run.Status) {
		return fmt.Errorf("invalid workflow run status %q", run.Status)
	}
	if strings.TrimSpace(run.ConfigSnapshotID) == "" {
		return fmt.Errorf("workflow run configSnapshotId is required")
	}
	if strings.TrimSpace(run.SchemaVersion) == "" {
		return fmt.Errorf("workflow run schema version is required")
	}
	for index := range run.StepStatuses {
		if err := run.StepStatuses[index].Validate(); err != nil {
			return err
		}
	}
	if err := ValidateNonZeroTime("workflow run startedAt", run.StartedAt); err != nil {
		return err
	}
	if err := ValidateNonZeroTime("workflow run createdAt", run.CreatedAt); err != nil {
		return err
	}
	if err := ValidateNonZeroTime("workflow run updatedAt", run.UpdatedAt); err != nil {
		return err
	}

	return nil
}
