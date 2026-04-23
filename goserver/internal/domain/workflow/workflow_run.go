package workflow

import (
	"fmt"
	"time"

	"goserver/internal/domain/common"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

type WorkflowStepRef struct {
	StepRunID    primitive.ObjectID        `bson:"stepRunId,omitempty" json:"stepRunId,omitempty"`
	StepName     common.WorkflowStepName   `bson:"stepName" json:"stepName"`
	Status       common.WorkflowStepStatus `bson:"status" json:"status"`
	StartedAt    *time.Time                `bson:"startedAt,omitempty" json:"startedAt,omitempty"`
	CompletedAt  *time.Time                `bson:"completedAt,omitempty" json:"completedAt,omitempty"`
	ErrorSummary string                    `bson:"errorSummary,omitempty" json:"errorSummary,omitempty"`
}

func (reference WorkflowStepRef) Validate() error {
	if !reference.StepName.IsValid() {
		return fmt.Errorf("invalid stepName %q", reference.StepName)
	}
	if !reference.Status.IsValid() {
		return fmt.Errorf("invalid step status %q", reference.Status)
	}
	if reference.StartedAt != nil && reference.StartedAt.IsZero() {
		return fmt.Errorf("startedAt cannot be zero")
	}
	if reference.CompletedAt != nil && reference.CompletedAt.IsZero() {
		return fmt.Errorf("completedAt cannot be zero")
	}
	if reference.StartedAt != nil && reference.CompletedAt != nil && reference.CompletedAt.Before(reference.StartedAt.UTC()) {
		return fmt.Errorf("completedAt cannot be before startedAt")
	}
	return nil
}

type WorkflowRun struct {
	ID                    primitive.ObjectID       `bson:"_id,omitempty" json:"id,omitempty"`
	BookType              common.BookType          `bson:"bookType" json:"bookType"`
	RunType               common.WorkflowRunType   `bson:"runType" json:"runType"`
	Status                common.WorkflowRunStatus `bson:"status" json:"status"`
	ConfigSnapshotID      primitive.ObjectID       `bson:"configSnapshotId" json:"configSnapshotId"`
	StartedAt             time.Time                `bson:"startedAt" json:"startedAt"`
	CompletedAt           *time.Time               `bson:"completedAt,omitempty" json:"completedAt,omitempty"`
	CompaniesScannedCount int                      `bson:"companiesScannedCount,omitempty" json:"companiesScannedCount,omitempty"`
	ReviewsCreatedCount   int                      `bson:"reviewsCreatedCount,omitempty" json:"reviewsCreatedCount,omitempty"`
	ErrorsCount           int                      `bson:"errorsCount,omitempty" json:"errorsCount,omitempty"`
	Notes                 string                   `bson:"notes,omitempty" json:"notes,omitempty"`
	StepStatuses          []WorkflowStepRef        `bson:"stepStatuses,omitempty" json:"stepStatuses,omitempty"`
	CreatedAt             time.Time                `bson:"createdAt" json:"createdAt"`
	UpdatedAt             time.Time                `bson:"updatedAt" json:"updatedAt"`
	SchemaVersion         int                      `bson:"schemaVersion" json:"schemaVersion"`
}

var allowedWorkflowRunTransitions = map[common.WorkflowRunStatus]map[common.WorkflowRunStatus]struct{}{
	common.WorkflowRunStatusCreated: {
		common.WorkflowRunStatusRunning:   {},
		common.WorkflowRunStatusCancelled: {},
	},
	common.WorkflowRunStatusRunning: {
		common.WorkflowRunStatusWaitingExternal:    {},
		common.WorkflowRunStatusPartiallyCompleted: {},
		common.WorkflowRunStatusCompleted:          {},
		common.WorkflowRunStatusFailed:             {},
		common.WorkflowRunStatusCancelled:          {},
	},
	common.WorkflowRunStatusWaitingExternal: {
		common.WorkflowRunStatusRunning:            {},
		common.WorkflowRunStatusPartiallyCompleted: {},
		common.WorkflowRunStatusCompleted:          {},
		common.WorkflowRunStatusFailed:             {},
		common.WorkflowRunStatusCancelled:          {},
	},
	common.WorkflowRunStatusPartiallyCompleted: {
		common.WorkflowRunStatusRunning:         {},
		common.WorkflowRunStatusWaitingExternal: {},
		common.WorkflowRunStatusCompleted:       {},
		common.WorkflowRunStatusFailed:          {},
		common.WorkflowRunStatusCancelled:       {},
	},
	common.WorkflowRunStatusCompleted: {},
	common.WorkflowRunStatusFailed:    {},
	common.WorkflowRunStatusCancelled: {},
}

func (run WorkflowRun) Validate() error {
	if !run.BookType.IsValid() {
		return fmt.Errorf("invalid bookType %q", run.BookType)
	}
	if !run.RunType.IsValid() {
		return fmt.Errorf("invalid runType %q", run.RunType)
	}
	if !run.Status.IsValid() {
		return fmt.Errorf("invalid status %q", run.Status)
	}
	if err := common.RequireObjectID("configSnapshotId", run.ConfigSnapshotID); err != nil {
		return err
	}
	if err := common.RequireTime("startedAt", run.StartedAt); err != nil {
		return err
	}
	if err := common.ValidateOptionalTimestampOrder("startedAt", run.StartedAt, "completedAt", run.CompletedAt); err != nil {
		return err
	}
	if err := common.ValidateNonNegativeInt("companiesScannedCount", run.CompaniesScannedCount); err != nil {
		return err
	}
	if err := common.ValidateNonNegativeInt("reviewsCreatedCount", run.ReviewsCreatedCount); err != nil {
		return err
	}
	if err := common.ValidateNonNegativeInt("errorsCount", run.ErrorsCount); err != nil {
		return err
	}
	for _, stepStatus := range run.StepStatuses {
		if err := stepStatus.Validate(); err != nil {
			return err
		}
	}
	if err := common.RequireTime("createdAt", run.CreatedAt); err != nil {
		return err
	}
	if err := common.RequireTime("updatedAt", run.UpdatedAt); err != nil {
		return err
	}
	if err := common.ValidateTimestampOrder("createdAt", run.CreatedAt, "updatedAt", run.UpdatedAt); err != nil {
		return err
	}
	if err := common.ValidateSchemaVersion("schemaVersion", run.SchemaVersion); err != nil {
		return err
	}
	if run.IsTerminal() && run.CompletedAt == nil {
		return fmt.Errorf("terminal workflow runs require completedAt")
	}
	return nil
}

func (run WorkflowRun) IsTerminal() bool {
	switch run.Status {
	case common.WorkflowRunStatusCompleted, common.WorkflowRunStatusFailed, common.WorkflowRunStatusCancelled:
		return true
	default:
		return false
	}
}

func (run WorkflowRun) RequiresExternalWait() bool {
	return run.Status == common.WorkflowRunStatusWaitingExternal
}

func (run WorkflowRun) CanTransitionTo(next common.WorkflowRunStatus) bool {
	if run.Status == next {
		return true
	}
	nextStates, ok := allowedWorkflowRunTransitions[run.Status]
	if !ok {
		return false
	}
	_, ok = nextStates[next]
	return ok
}

func (run *WorkflowRun) TransitionTo(next common.WorkflowRunStatus, at time.Time) error {
	if run == nil {
		return fmt.Errorf("workflow run is required")
	}
	if !next.IsValid() {
		return fmt.Errorf("invalid next workflow status %q", next)
	}
	if !run.CanTransitionTo(next) {
		return fmt.Errorf("invalid workflow run transition from %q to %q", run.Status, next)
	}
	if err := common.RequireTime("transitionAt", at); err != nil {
		return err
	}
	run.Status = next
	run.UpdatedAt = at.UTC()
	if next == common.WorkflowRunStatusCompleted || next == common.WorkflowRunStatusFailed || next == common.WorkflowRunStatusCancelled {
		completedAt := at.UTC()
		run.CompletedAt = &completedAt
	}
	return nil
}
