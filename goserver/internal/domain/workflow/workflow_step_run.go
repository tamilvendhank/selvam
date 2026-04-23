package workflow

import (
	"fmt"
	"time"

	"goserver/internal/domain/common"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

type WorkflowStepRun struct {
	ID            primitive.ObjectID        `bson:"_id,omitempty" json:"id,omitempty"`
	WorkflowRunID primitive.ObjectID        `bson:"workflowRunId" json:"workflowRunId"`
	StepName      common.WorkflowStepName   `bson:"stepName" json:"stepName"`
	Status        common.WorkflowStepStatus `bson:"status" json:"status"`
	StartedAt     *time.Time                `bson:"startedAt,omitempty" json:"startedAt,omitempty"`
	CompletedAt   *time.Time                `bson:"completedAt,omitempty" json:"completedAt,omitempty"`
	ErrorSummary  string                    `bson:"errorSummary,omitempty" json:"errorSummary,omitempty"`
	Metadata      map[string]any            `bson:"metadata,omitempty" json:"metadata,omitempty"`
	CreatedAt     time.Time                 `bson:"createdAt" json:"createdAt"`
	UpdatedAt     time.Time                 `bson:"updatedAt" json:"updatedAt"`
	SchemaVersion int                       `bson:"schemaVersion" json:"schemaVersion"`
}

var allowedWorkflowStepTransitions = map[common.WorkflowStepStatus]map[common.WorkflowStepStatus]struct{}{
	common.WorkflowStepStatusPending: {
		common.WorkflowStepStatusRunning: {},
		common.WorkflowStepStatusSkipped: {},
	},
	common.WorkflowStepStatusRunning: {
		common.WorkflowStepStatusWaitingExternal: {},
		common.WorkflowStepStatusCompleted:       {},
		common.WorkflowStepStatusFailed:          {},
	},
	common.WorkflowStepStatusWaitingExternal: {
		common.WorkflowStepStatusRunning:   {},
		common.WorkflowStepStatusCompleted: {},
		common.WorkflowStepStatusFailed:    {},
	},
	common.WorkflowStepStatusCompleted: {},
	common.WorkflowStepStatusFailed:    {},
	common.WorkflowStepStatusSkipped:   {},
}

func (stepRun WorkflowStepRun) Validate() error {
	if err := common.RequireObjectID("workflowRunId", stepRun.WorkflowRunID); err != nil {
		return err
	}
	if !stepRun.StepName.IsValid() {
		return fmt.Errorf("invalid stepName %q", stepRun.StepName)
	}
	if !stepRun.Status.IsValid() {
		return fmt.Errorf("invalid status %q", stepRun.Status)
	}
	if stepRun.StartedAt != nil && stepRun.StartedAt.IsZero() {
		return fmt.Errorf("startedAt cannot be zero")
	}
	if stepRun.CompletedAt != nil && stepRun.CompletedAt.IsZero() {
		return fmt.Errorf("completedAt cannot be zero")
	}
	if stepRun.StartedAt != nil && stepRun.CompletedAt != nil && stepRun.CompletedAt.Before(stepRun.StartedAt.UTC()) {
		return fmt.Errorf("completedAt cannot be before startedAt")
	}
	if err := common.RequireTime("createdAt", stepRun.CreatedAt); err != nil {
		return err
	}
	if err := common.RequireTime("updatedAt", stepRun.UpdatedAt); err != nil {
		return err
	}
	if err := common.ValidateTimestampOrder("createdAt", stepRun.CreatedAt, "updatedAt", stepRun.UpdatedAt); err != nil {
		return err
	}
	if err := common.ValidateSchemaVersion("schemaVersion", stepRun.SchemaVersion); err != nil {
		return err
	}
	if stepRun.IsTerminal() && stepRun.CompletedAt == nil {
		return fmt.Errorf("terminal workflow step runs require completedAt")
	}
	return nil
}

func (stepRun WorkflowStepRun) IsTerminal() bool {
	switch stepRun.Status {
	case common.WorkflowStepStatusCompleted, common.WorkflowStepStatusFailed, common.WorkflowStepStatusSkipped:
		return true
	default:
		return false
	}
}

func (stepRun WorkflowStepRun) RequiresExternalWait() bool {
	return stepRun.Status == common.WorkflowStepStatusWaitingExternal
}

func (stepRun WorkflowStepRun) CanTransitionTo(next common.WorkflowStepStatus) bool {
	if stepRun.Status == next {
		return true
	}
	nextStates, ok := allowedWorkflowStepTransitions[stepRun.Status]
	if !ok {
		return false
	}
	_, ok = nextStates[next]
	return ok
}

func (stepRun *WorkflowStepRun) TransitionTo(next common.WorkflowStepStatus, at time.Time) error {
	if stepRun == nil {
		return fmt.Errorf("workflow step run is required")
	}
	if !next.IsValid() {
		return fmt.Errorf("invalid next workflow step status %q", next)
	}
	if !stepRun.CanTransitionTo(next) {
		return fmt.Errorf("invalid workflow step transition from %q to %q", stepRun.Status, next)
	}
	if err := common.RequireTime("transitionAt", at); err != nil {
		return err
	}
	if stepRun.StartedAt == nil && next != common.WorkflowStepStatusPending && next != common.WorkflowStepStatusSkipped {
		startedAt := at.UTC()
		stepRun.StartedAt = &startedAt
	}
	if next == common.WorkflowStepStatusCompleted || next == common.WorkflowStepStatusFailed || next == common.WorkflowStepStatusSkipped {
		completedAt := at.UTC()
		stepRun.CompletedAt = &completedAt
	}
	stepRun.Status = next
	stepRun.UpdatedAt = at.UTC()
	return nil
}
