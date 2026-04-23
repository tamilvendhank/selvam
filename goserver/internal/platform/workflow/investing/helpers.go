package investing

import (
	"time"

	"goserver/internal/platform/domain"
	"goserver/internal/platform/ports"
	"goserver/internal/platform/workflow"
)

func buildInvestingPendingSteps() []domain.WorkflowStepStatus {
	steps := make([]domain.WorkflowStepStatus, 0, len(workflow.InvestingStepNames()))
	for _, name := range workflow.InvestingStepNames() {
		steps = append(steps, domain.WorkflowStepStatus{
			StepName: name,
			Status:   domain.WorkflowStepStatusPending,
		})
	}

	return steps
}

func applyCompletedStep(run *domain.WorkflowRun, step workflow.StepName, input map[string]any, output map[string]any, now time.Time) {
	for index := range run.StepStatuses {
		if run.StepStatuses[index].StepName != string(step) {
			continue
		}
		run.StepStatuses[index].Status = domain.WorkflowStepStatusCompleted
		run.StepStatuses[index].StartedAt = &now
		run.StepStatuses[index].CompletedAt = &now
		run.StepStatuses[index].InputSnapshot = input
		run.StepStatuses[index].OutputSnapshot = output
		return
	}
}

func applyWaitingAsyncStep(run *domain.WorkflowRun, step workflow.StepName, input map[string]any, output map[string]any, task *domain.AsyncTaskReference, now time.Time) {
	for index := range run.StepStatuses {
		if run.StepStatuses[index].StepName != string(step) {
			continue
		}
		run.StepStatuses[index].Status = domain.WorkflowStepStatusWaitingAsync
		run.StepStatuses[index].StartedAt = &now
		run.StepStatuses[index].InputSnapshot = input
		run.StepStatuses[index].OutputSnapshot = output
		run.StepStatuses[index].AsyncTask = task
		return
	}
}

func applySkippedStep(run *domain.WorkflowRun, step workflow.StepName, reason string) {
	for index := range run.StepStatuses {
		if run.StepStatuses[index].StepName != string(step) {
			continue
		}
		run.StepStatuses[index].Status = domain.WorkflowStepStatusSkipped
		run.StepStatuses[index].OutputSnapshot = map[string]any{
			"reason": reason,
		}
		return
	}
}

func markRemainingInvestingStepsSkipped(run *domain.WorkflowRun, startStep string) {
	skip := false
	for index := range run.StepStatuses {
		if run.StepStatuses[index].StepName == startStep {
			skip = true
		}
		if skip && run.StepStatuses[index].Status == domain.WorkflowStepStatusPending {
			run.StepStatuses[index].Status = domain.WorkflowStepStatusSkipped
		}
	}
}

func markFailedStep(run *domain.WorkflowRun, step workflow.StepName, message string, now time.Time) {
	for index := range run.StepStatuses {
		if run.StepStatuses[index].StepName != string(step) {
			continue
		}
		run.StepStatuses[index].Status = domain.WorkflowStepStatusFailed
		run.StepStatuses[index].StartedAt = &now
		run.StepStatuses[index].CompletedAt = &now
		run.StepStatuses[index].Error = &domain.WorkflowStepError{
			Code:    "step_failed",
			Message: message,
		}
		return
	}
}

func fromAsyncTask(task *ports.AIAsyncTask) *domain.AsyncTaskReference {
	if task == nil {
		return nil
	}

	return &domain.AsyncTaskReference{
		Provider:            task.Provider,
		TaskKind:            task.TaskKind,
		LocalObjectType:     task.LocalObjectType,
		LocalObjectID:       task.LocalObjectID,
		SubmissionID:        task.SubmissionID,
		RepresentativeJobID: task.RepresentativeJobID,
		BatchID:             task.BatchID,
		JobIDs:              task.JobIDs,
		Status:              domain.AsyncTaskStatus(task.Status),
		ResultAvailable:     task.ResultAvailable,
		SubmittedAt:         task.SubmittedAt,
		LastSyncedAt:        task.LastSyncedAt,
		Metadata:            task.Metadata,
	}
}
