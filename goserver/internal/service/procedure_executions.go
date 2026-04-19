package service

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"goserver/internal/domain"
	"goserver/internal/repository"
	"goserver/internal/shared"

	"go.mongodb.org/mongo-driver/bson"
)

type ProcedureExecutionsService struct {
	repo           *repository.ProcedureExecutionsRepository
	proceduresRepo *repository.ProceduresRepository
	jobsService    *JobsService
}

func NewProcedureExecutionsService(
	repo *repository.ProcedureExecutionsRepository,
	proceduresRepo *repository.ProceduresRepository,
	jobsService *JobsService,
) *ProcedureExecutionsService {
	return &ProcedureExecutionsService{
		repo:           repo,
		proceduresRepo: proceduresRepo,
		jobsService:    jobsService,
	}
}

func (service *ProcedureExecutionsService) GetProcedureExecutionsForList(ctx context.Context) ([]map[string]any, error) {
	executions, err := service.repo.List(ctx)
	if err != nil {
		return nil, err
	}

	result := make([]map[string]any, 0, len(executions))
	for _, execution := range executions {
		result = append(result, procedureExecutionViewModel(execution))
	}

	return result, nil
}

func (service *ProcedureExecutionsService) GetProcedureExecutionDetails(ctx context.Context, id string) (map[string]any, error) {
	execution, err := service.repo.GetByID(ctx, id)
	if err != nil || execution == nil {
		return nil, err
	}

	return procedureExecutionViewModel(execution), nil
}

func (service *ProcedureExecutionsService) CreateExecutionDefinition(ctx context.Context, procedureID, prompt string) (map[string]any, error) {
	execution, err := service.createExecution(ctx, procedureID, prompt)
	if err != nil {
		return nil, err
	}

	return procedureExecutionViewModel(execution), nil
}

func (service *ProcedureExecutionsService) CreateAndStartExecution(ctx context.Context, procedureID, prompt string) (map[string]any, error) {
	execution, err := service.createExecution(ctx, procedureID, prompt)
	if err != nil {
		return nil, err
	}

	return service.StartProcedureExecutionByID(ctx, execution.ID)
}

func (service *ProcedureExecutionsService) createExecution(ctx context.Context, procedureID, prompt string) (*domain.ProcedureExecution, error) {
	procedure, err := service.proceduresRepo.GetByID(ctx, procedureID)
	if err != nil {
		return nil, err
	}
	if procedure == nil {
		return nil, fmt.Errorf("procedure not found")
	}

	now := time.Now()
	execution := &domain.ProcedureExecution{
		ProcedureID:      procedure.ID,
		ProcedureName:    procedure.Name,
		InitialPrompt:    strings.TrimSpace(prompt),
		Status:           "draft",
		CurrentStepIndex: nil,
		Steps:            createExecutionSteps(procedure.Steps),
		CreatedAt:        now,
		UpdatedAt:        now,
	}

	createdExecution, err := service.repo.Create(ctx, execution)
	if err != nil {
		return nil, err
	}

	return createdExecution, nil
}

func (service *ProcedureExecutionsService) StartProcedureExecutionByID(ctx context.Context, id string) (map[string]any, error) {
	execution, err := service.repo.GetByID(ctx, id)
	if err != nil || execution == nil {
		return nil, err
	}

	if execution.Status == "completed" {
		return procedureExecutionViewModel(execution), nil
	}

	if getCurrentStepIndex(execution) == -1 {
		nextPendingStepIndex := getNextPendingStepIndex(execution)
		if nextPendingStepIndex != -1 {
			if err := service.startExecutionStep(ctx, execution, nextPendingStepIndex); err != nil {
				return nil, err
			}
		}
	}

	updatedExecution, err := service.persistExecution(ctx, execution)
	if err != nil {
		return nil, err
	}

	return procedureExecutionViewModel(updatedExecution), nil
}

func (service *ProcedureExecutionsService) RefreshProcedureExecutionByID(ctx context.Context, id string) (map[string]any, error) {
	execution, err := service.repo.GetByID(ctx, id)
	if err != nil || execution == nil {
		return nil, err
	}

	if execution.Status == "draft" || execution.Status == "completed" || execution.Status == "failed" {
		return procedureExecutionViewModel(execution), nil
	}

	if err := service.progressExecution(ctx, execution); err != nil {
		return nil, err
	}

	updatedExecution, err := service.persistExecution(ctx, execution)
	if err != nil {
		return nil, err
	}

	return procedureExecutionViewModel(updatedExecution), nil
}

func (service *ProcedureExecutionsService) OnJobRefreshed(ctx context.Context, jobID string) error {
	jobID = strings.TrimSpace(jobID)
	if jobID == "" {
		return nil
	}

	executions, err := service.repo.ListRunningByJobID(ctx, jobID)
	if err != nil {
		return err
	}
	if len(executions) == 0 {
		return nil
	}

	job, err := service.jobsService.repo.GetByID(ctx, jobID)
	if err != nil {
		return err
	}

	for _, execution := range executions {
		updated, err := service.progressExecutionWithStoredJob(ctx, execution, jobID, job)
		if err != nil {
			return err
		}
		if !updated {
			continue
		}
		if _, err := service.persistExecution(ctx, execution); err != nil {
			return err
		}
	}

	return nil
}

func (service *ProcedureExecutionsService) RunProgressPass(ctx context.Context) error {
	executions, err := service.repo.ListRunning(ctx)
	if err != nil {
		return err
	}

	for _, execution := range executions {
		updated, err := service.progressExecutionWithStoredState(ctx, execution)
		if err != nil {
			return err
		}
		if !updated {
			continue
		}
		if _, err := service.persistExecution(ctx, execution); err != nil {
			return err
		}
	}

	return nil
}

func (service *ProcedureExecutionsService) persistExecution(ctx context.Context, execution *domain.ProcedureExecution) (*domain.ProcedureExecution, error) {
	return service.repo.Update(ctx, execution.ID, bson.M{
		"status":           execution.Status,
		"currentStepIndex": execution.CurrentStepIndex,
		"startedAt":        execution.StartedAt,
		"completedAt":      execution.CompletedAt,
		"lastRefreshedAt":  execution.LastRefreshedAt,
		"steps":            execution.Steps,
	})
}

func (service *ProcedureExecutionsService) startExecutionStep(ctx context.Context, execution *domain.ProcedureExecution, stepIndex int) error {
	if stepIndex < 0 || stepIndex >= len(execution.Steps) {
		return nil
	}

	step := execution.Steps[stepIndex]
	if step.Status != "pending" {
		return nil
	}

	var previousResultText string
	if stepIndex > 0 {
		previousResultText = execution.Steps[stepIndex-1].ResultText
	}

	stepInput := buildExecutionPrompt(step.Prompt, execution.InitialPrompt, previousResultText)
	jobs, err := service.jobsService.SubmitPromptBatchWithFiles(ctx, []PromptEntry{
		{
			Query:           stepInput,
			Model:           step.Model,
			ReasoningEffort: shared.DerefString(step.ReasoningEffort),
		},
	}, SubmissionMetadata{
		SubmissionType: "procedure_execution",
	})
	if err != nil {
		return err
	}

	if len(jobs) == 0 {
		return fmt.Errorf("failed to create procedure execution step job")
	}

	now := time.Now()
	job := jobs[0]
	jobID, _ := job["id"].(string)
	batchID, _ := job["batchId"].(string)
	lastSyncedAt, _ := job["lastSyncedAt"].(*time.Time)
	resultText, _ := job["resultText"].(string)
	resultResponseBody := shared.MapFromAny(job["resultResponseBody"])
	latestErrorLine := shared.MapFromAny(job["latestErrorLine"])
	latestError := extractErrorPayload(latestErrorLine)

	execution.Status = "running"
	if execution.StartedAt == nil {
		execution.StartedAt = &now
	}
	execution.CurrentStepIndex = shared.IntPtr(stepIndex)
	execution.LastRefreshedAt = &now
	execution.Steps[stepIndex] = domain.ProcedureExecutionStep{
		ID:                 step.ID,
		StepNumber:         step.StepNumber,
		Prompt:             step.Prompt,
		Model:              step.Model,
		ReasoningEffort:    step.ReasoningEffort,
		Status:             "in_progress",
		StepInput:          stepInput,
		JobID:              emptyStringToNil(jobID),
		BatchID:            emptyStringToNil(batchID),
		StartedAt:          &now,
		LastSyncedAt:       lastSyncedAt,
		ResultText:         resultText,
		ResultResponseBody: resultResponseBody,
		LatestError:        latestError,
	}

	return nil
}

func (service *ProcedureExecutionsService) refreshActiveExecutionStep(ctx context.Context, execution *domain.ProcedureExecution) error {
	activeStepIndex := getCurrentStepIndex(execution)
	if activeStepIndex == -1 {
		return nil
	}

	activeStep := execution.Steps[activeStepIndex]
	jobID := shared.DerefString(activeStep.JobID)
	if strings.TrimSpace(jobID) == "" {
		return nil
	}

	if _, err := service.jobsService.RefreshJob(ctx, jobID); err != nil {
		return err
	}

	latestExecution, err := service.repo.GetByID(ctx, execution.ID)
	if err != nil {
		return err
	}
	if latestExecution == nil {
		return nil
	}

	*execution = *latestExecution
	return nil
}

func (service *ProcedureExecutionsService) progressExecution(ctx context.Context, execution *domain.ProcedureExecution) error {
	if err := service.refreshActiveExecutionStep(ctx, execution); err != nil {
		return err
	}

	return service.advanceExecutionAfterStepSync(ctx, execution)
}

func (service *ProcedureExecutionsService) progressExecutionWithStoredJob(
	ctx context.Context,
	execution *domain.ProcedureExecution,
	jobID string,
	job *domain.Job,
) (bool, error) {
	activeStepIndex := getCurrentStepIndex(execution)
	if activeStepIndex == -1 {
		return false, nil
	}

	activeStepJobID := strings.TrimSpace(shared.DerefString(execution.Steps[activeStepIndex].JobID))
	if activeStepJobID == "" || activeStepJobID != jobID {
		return false, nil
	}

	service.applyStoredJobToActiveExecutionStep(execution, activeStepIndex, job)
	if err := service.advanceExecutionAfterStepSync(ctx, execution); err != nil {
		return true, err
	}

	return true, nil
}

func (service *ProcedureExecutionsService) progressExecutionWithStoredState(
	ctx context.Context,
	execution *domain.ProcedureExecution,
) (bool, error) {
	if execution == nil || execution.Status != "running" {
		return false, nil
	}

	beforeSignature := executionProgressSignature(execution)
	activeStepIndex := getCurrentStepIndex(execution)
	if activeStepIndex != -1 {
		jobID := strings.TrimSpace(shared.DerefString(execution.Steps[activeStepIndex].JobID))
		if jobID != "" {
			job, err := service.jobsService.repo.GetByID(ctx, jobID)
			if err != nil {
				return false, err
			}

			service.applyStoredJobToActiveExecutionStep(execution, activeStepIndex, job)
		}
	}

	if err := service.advanceExecutionAfterStepSync(ctx, execution); err != nil {
		return false, err
	}

	return beforeSignature != executionProgressSignature(execution), nil
}

func (service *ProcedureExecutionsService) advanceExecutionAfterStepSync(ctx context.Context, execution *domain.ProcedureExecution) error {
	if execution.Status == "failed" {
		return nil
	}

	nextPendingStepIndex := getNextPendingStepIndex(execution)
	if nextPendingStepIndex == -1 && getCurrentStepIndex(execution) == -1 {
		now := time.Now()
		execution.Status = "completed"
		if execution.CompletedAt == nil {
			execution.CompletedAt = &now
		}
		execution.CurrentStepIndex = nil
		return nil
	}

	if getCurrentStepIndex(execution) == -1 && nextPendingStepIndex != -1 {
		return service.startExecutionStep(ctx, execution, nextPendingStepIndex)
	}

	return nil
}

func (service *ProcedureExecutionsService) applyStoredJobToActiveExecutionStep(
	execution *domain.ProcedureExecution,
	activeStepIndex int,
	job *domain.Job,
) {
	if execution == nil || activeStepIndex < 0 || activeStepIndex >= len(execution.Steps) {
		return
	}

	now := time.Now()
	execution.LastRefreshedAt = &now
	if job == nil {
		execution.Status = "failed"
		execution.CurrentStepIndex = shared.IntPtr(activeStepIndex)
		execution.Steps[activeStepIndex].Status = "failed"
		execution.Steps[activeStepIndex].LatestError = map[string]any{
			"message": "Linked batch job could not be found.",
		}
		return
	}

	execution.Steps[activeStepIndex].BatchID = job.BatchID
	execution.Steps[activeStepIndex].LastSyncedAt = job.LastSyncedAt
	execution.Steps[activeStepIndex].ResultText = job.ResultText
	execution.Steps[activeStepIndex].ResultResponseBody = job.ResultResponseBody
	execution.Steps[activeStepIndex].LatestError = extractErrorPayload(job.LatestErrorLine)

	switch job.Status {
	case "completed":
		execution.Steps[activeStepIndex].Status = "completed"
		execution.Steps[activeStepIndex].CompletedAt = job.CompletedAt
		execution.Steps[activeStepIndex].ExecutionDuration = calculateExecutionDuration(execution.Steps[activeStepIndex].StartedAt, job.CompletedAt)
		execution.CurrentStepIndex = nil
	case "failed", "expired", "cancelled", "submission_failed":
		execution.Status = "failed"
		execution.CurrentStepIndex = shared.IntPtr(activeStepIndex)
		execution.Steps[activeStepIndex].Status = "failed"
		completedAt := job.CompletedAt
		if completedAt == nil {
			completedAt = &now
		}
		execution.Steps[activeStepIndex].CompletedAt = completedAt
		execution.Steps[activeStepIndex].ExecutionDuration = calculateExecutionDuration(execution.Steps[activeStepIndex].StartedAt, completedAt)
	default:
		execution.Steps[activeStepIndex].Status = "in_progress"
	}
}

func createExecutionSteps(steps []domain.ProcedureStep) []domain.ProcedureExecutionStep {
	output := make([]domain.ProcedureExecutionStep, 0, len(steps))
	for index, step := range steps {
		output = append(output, domain.ProcedureExecutionStep{
			ID:              firstNonEmpty(step.ID, fmt.Sprintf("step-%d", index+1)),
			StepNumber:      index + 1,
			Prompt:          step.Prompt,
			Model:           step.Model,
			ReasoningEffort: step.ReasoningEffort,
			Status:          "pending",
			StepInput:       "",
			ResultText:      "",
		})
	}

	return output
}

func buildExecutionPrompt(stepPrompt, executionPrompt, previousResultText string) string {
	parts := []string{stepPrompt}
	if trimmed := strings.TrimSpace(executionPrompt); trimmed != "" {
		parts = append(parts, "Initial input:\n"+trimmed)
	}
	if trimmed := strings.TrimSpace(previousResultText); trimmed != "" {
		parts = append(parts, "Previous step output:\n"+trimmed)
	}

	return strings.Join(parts, "\n\n")
}

func getCurrentStepIndex(execution *domain.ProcedureExecution) int {
	for index, step := range execution.Steps {
		if step.Status == "in_progress" {
			return index
		}
	}

	return -1
}

func getNextPendingStepIndex(execution *domain.ProcedureExecution) int {
	for index, step := range execution.Steps {
		if step.Status == "pending" {
			return index
		}
	}

	return -1
}

func calculateExecutionDuration(startedAt, completedAt *time.Time) *int64 {
	if startedAt == nil || completedAt == nil {
		return nil
	}

	duration := completedAt.Sub(*startedAt).Milliseconds()
	return &duration
}

func extractErrorPayload(latestErrorLine map[string]any) map[string]any {
	if latestErrorLine == nil {
		return nil
	}

	if errorPayload, ok := latestErrorLine["error"].(map[string]any); ok {
		return errorPayload
	}

	return latestErrorLine
}

func executionProgressSignature(execution *domain.ProcedureExecution) string {
	if execution == nil {
		return ""
	}

	payload, err := json.Marshal(struct {
		Status           string                          `json:"status"`
		CurrentStepIndex *int                            `json:"currentStepIndex"`
		StartedAt        *time.Time                      `json:"startedAt"`
		CompletedAt      *time.Time                      `json:"completedAt"`
		LastRefreshedAt  *time.Time                      `json:"lastRefreshedAt"`
		Steps            []domain.ProcedureExecutionStep `json:"steps"`
	}{
		Status:           execution.Status,
		CurrentStepIndex: execution.CurrentStepIndex,
		StartedAt:        execution.StartedAt,
		CompletedAt:      execution.CompletedAt,
		LastRefreshedAt:  execution.LastRefreshedAt,
		Steps:            execution.Steps,
	})
	if err != nil {
		return ""
	}

	return string(payload)
}

func procedureExecutionViewModel(execution *domain.ProcedureExecution) map[string]any {
	if execution == nil {
		return nil
	}

	currentStepNumber := any(nil)
	if execution.CurrentStepIndex != nil && *execution.CurrentStepIndex >= 0 {
		currentStepNumber = *execution.CurrentStepIndex + 1
	}

	steps := make([]map[string]any, 0, len(execution.Steps))
	for _, step := range execution.Steps {
		steps = append(steps, map[string]any{
			"id":                  step.ID,
			"stepNumber":          step.StepNumber,
			"prompt":              step.Prompt,
			"model":               step.Model,
			"reasoningEffort":     step.ReasoningEffort,
			"status":              step.Status,
			"stepInput":           step.StepInput,
			"jobId":               step.JobID,
			"batchId":             step.BatchID,
			"startedAt":           step.StartedAt,
			"completedAt":         step.CompletedAt,
			"lastSyncedAt":        step.LastSyncedAt,
			"executionDurationMs": step.ExecutionDuration,
			"resultText":          step.ResultText,
			"resultResponseBody":  shared.NormalizeJSONValue(step.ResultResponseBody),
			"latestError":         shared.NormalizeJSONValue(step.LatestError),
			"canRefresh":          step.Status == "in_progress" && step.JobID != nil && *step.JobID != "",
			"createdInputLabel":   firstNonEmpty(strings.TrimSpace(step.StepInput), "N/A"),
			"startedAtLabel":      shared.FormatDateLabel(step.StartedAt, "Not started"),
			"completedAtLabel":    shared.FormatDateLabel(step.CompletedAt, ""),
			"durationLabel":       durationLabel(step.ExecutionDuration),
		})
	}

	return map[string]any{
		"id":               execution.ID,
		"procedureId":      execution.ProcedureID,
		"procedureName":    execution.ProcedureName,
		"initialPrompt":    execution.InitialPrompt,
		"status":           execution.Status,
		"currentStepIndex": execution.CurrentStepIndex,
		"startedAt":        execution.StartedAt,
		"completedAt":      execution.CompletedAt,
		"lastRefreshedAt":  execution.LastRefreshedAt,
		"steps":            steps,
		"createdAt":        execution.CreatedAt,
		"updatedAt":        execution.UpdatedAt,
		"createdAtLabel":   shared.FormatDateLabel(&execution.CreatedAt, "Unknown"),
		"updatedAtLabel":   shared.FormatDateLabel(&execution.UpdatedAt, "Unknown"),
		"startedAtLabel":   shared.FormatDateLabel(execution.StartedAt, ""),
		"completedAtLabel": shared.FormatDateLabel(execution.CompletedAt, ""),
		"initialPromptLabel": func() string {
			if trimmed := strings.TrimSpace(execution.InitialPrompt); trimmed != "" {
				return trimmed
			}
			return "N/A"
		}(),
		"currentStepNumber": currentStepNumber,
	}
}
