package service

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"

	"goserver/internal/domain"
	"goserver/internal/shared"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

func (service *JobsService) buildSubmissionIteration(
	job *domain.Job,
	iterationNumber int,
	kind, customID, inputText string,
	requestBody map[string]any,
	previousResponseID *string,
	toolOutputs []map[string]any,
) *domain.SubmissionIteration {
	now := time.Now()
	iteration := &domain.SubmissionIteration{
		JobID:            job.ID,
		SubmissionID:     job.SubmissionID,
		SubmissionType:   job.SubmissionType,
		IterationNumber:  iterationNumber,
		Kind:             kind,
		CustomID:         firstNonEmpty(strings.TrimSpace(customID), buildIterationCustomID(job.ID, iterationNumber)),
		InputText:        strings.TrimSpace(inputText),
		RequestBody:      requestBody,
		PreviousResponse: previousResponseID,
		Status:           "preparing",
		ResultText:       "",
		ToolOutputs:      toolOutputs,
		CreatedAt:        now,
		UpdatedAt:        now,
	}

	iteration.ObjectID = primitive.NewObjectID()
	iteration.NormalizeID()
	return iteration
}

func buildIterationCustomID(jobID string, iterationNumber int) string {
	return fmt.Sprintf("job-%s-iter-%d", jobID, iterationNumber)
}

func (service *JobsService) submitIterationsBatch(
	ctx context.Context,
	iterations []*domain.SubmissionIteration,
	metadata map[string]string,
) ([]*domain.SubmissionIteration, error) {
	if len(iterations) == 0 {
		return []*domain.SubmissionIteration{}, nil
	}

	lines := make([]map[string]any, 0, len(iterations))
	for _, iteration := range iterations {
		lines = append(lines, map[string]any{
			"custom_id": iteration.CustomID,
			"method":    "POST",
			"url":       service.config.OpenAI.BatchEndpoint,
			"body":      iteration.RequestBody,
		})
	}

	fileStem := firstNonEmpty(metadata["submission_id"], iterations[0].SubmissionID)
	jsonlPayload, err := service.buildJSONLPayload(lines)
	if err != nil {
		return nil, err
	}

	inputFile, err := service.openai.UploadBatchInputFile(ctx, fileStem+".jsonl", jsonlPayload)
	if err != nil {
		service.markIterationsAsSubmissionFailed(ctx, iterations, err)
		return nil, err
	}

	batch, err := service.openai.CreateBatch(
		ctx,
		inputFile.ID,
		service.config.OpenAI.BatchEndpoint,
		service.config.OpenAI.CompletionWindow,
		metadata,
	)
	if err != nil {
		service.markIterationsAsSubmissionFailed(ctx, iterations, err)
		return nil, err
	}

	updated := make([]*domain.SubmissionIteration, 0, len(iterations))
	for _, iteration := range iterations {
		updatedIteration, updateErr := service.iterationsRepo.Update(ctx, iteration.ID, bson.M{
			"status":        batch.Status,
			"batchId":       batch.ID,
			"inputFileId":   inputFile.ID,
			"outputFileId":  emptyStringToNil(batch.OutputFileID),
			"errorFileId":   emptyStringToNil(batch.ErrorFileID),
			"requestCounts": batch.RequestCounts,
			"lastSyncedAt":  time.Now(),
			"openaiBatch":   buildBatchSnapshot(batch),
		})
		if updateErr != nil {
			return nil, updateErr
		}

		updated = append(updated, updatedIteration)
	}

	return updated, nil
}

func (service *JobsService) progressSubmissionIteration(ctx context.Context, iteration *domain.SubmissionIteration) (*domain.SubmissionIteration, error) {
	if iteration == nil || iteration.Status != "completed" {
		return iteration, nil
	}

	toolCalls := extractToolCalls(iteration.ResultResponse)
	if len(toolCalls) == 0 {
		return iteration, nil
	}

	if nextIterationID := shared.DerefString(iteration.NextIterationID); nextIterationID != "" {
		nextIteration, err := service.iterationsRepo.GetByID(ctx, nextIterationID)
		if err != nil || nextIteration == nil {
			return iteration, err
		}

		return nextIteration, nil
	}

	claimed, err := service.iterationsRepo.TryClaimFollowUp(ctx, iteration.ID, time.Now().Add(-service.config.Worker.FollowUpClaimTimeout))
	if err != nil {
		return nil, err
	}
	if !claimed {
		latestIteration, latestErr := service.iterationsRepo.GetLatestByJobID(ctx, iteration.JobID)
		if latestErr != nil {
			return nil, latestErr
		}
		if latestIteration != nil {
			return latestIteration, nil
		}
		return iteration, nil
	}

	job, err := service.repo.GetByID(ctx, iteration.JobID)
	if err != nil || job == nil {
		return nil, err
	}

	toolOutputs, err := service.toolExecutor.Execute(ctx, job, toolCalls)
	if err != nil {
		toolOutputs = buildErroredToolOutputs(toolCalls, err)
	}

	nextIteration := service.buildSubmissionIteration(
		job,
		iteration.IterationNumber+1,
		"tool_result",
		buildIterationCustomID(job.ID, iteration.IterationNumber+1),
		buildToolOutputsText(toolCalls, toolOutputs),
		buildBatchRequestBody(responseRequestSpec{
			Model:              job.Model,
			ReasoningEffort:    job.ReasoningEffort,
			Instructions:       service.config.OpenAI.ResponseInstructions,
			PreviousResponseID: shared.DerefString(iteration.ResponseID),
			InputItems:         buildToolOutputInputItems(toolOutputs),
			Tools:              extractToolsFromRequestBody(iteration.RequestBody),
		}),
		iteration.ResponseID,
		normalizeToolOutputMaps(toolOutputs),
	)

	createdNextIteration, err := service.iterationsRepo.Create(ctx, nextIteration)
	if err != nil {
		return nil, err
	}

	if _, err := service.iterationsRepo.Update(ctx, iteration.ID, bson.M{
		"toolCalls":       normalizeToolCallMaps(toolCalls),
		"nextIterationId": createdNextIteration.ID,
		"followUpState":   "advanced",
	}); err != nil {
		return nil, err
	}

	submittedIterations, err := service.submitIterationsBatch(ctx, []*domain.SubmissionIteration{createdNextIteration}, map[string]string{
		"app":               "webapp",
		"submission_id":     job.SubmissionID,
		"submission_job_id": job.ID,
		"iteration_number":  strconv.Itoa(createdNextIteration.IterationNumber),
		"submission_type":   job.SubmissionType,
	})
	if err != nil {
		failedIteration, failedErr := service.iterationsRepo.GetByID(ctx, createdNextIteration.ID)
		if failedErr != nil {
			return nil, failedErr
		}
		if failedIteration != nil {
			return failedIteration, nil
		}
		return nil, err
	}

	return submittedIterations[0], nil
}

func buildErroredToolOutputs(toolCalls []submissionToolCall, err error) []submissionToolOutput {
	outputs := make([]submissionToolOutput, 0, len(toolCalls))
	for _, toolCall := range toolCalls {
		outputType := "function_call_output"
		if toolCall.Type == "custom_tool_call" {
			outputType = "custom_tool_call_output"
		}

		payload, marshalErr := json.Marshal(map[string]any{
			"error": err.Error(),
		})
		if marshalErr != nil {
			payload = []byte(fmt.Sprintf("{\"error\":%q}", err.Error()))
		}

		outputs = append(outputs, submissionToolOutput{
			Type:   outputType,
			CallID: toolCall.CallID,
			Output: string(payload),
		})
	}

	return outputs
}

func (service *JobsService) syncJobFromIteration(ctx context.Context, jobID string, iteration *domain.SubmissionIteration) (*domain.Job, error) {
	if iteration == nil {
		return service.repo.GetByID(ctx, jobID)
	}

	updatedJob, err := service.repo.Update(ctx, jobID, bson.M{
		"status":             iteration.Status,
		"batchId":            iteration.BatchID,
		"inputFileId":        iteration.InputFileID,
		"outputFileId":       iteration.OutputFileID,
		"errorFileId":        iteration.ErrorFileID,
		"requestCounts":      iteration.RequestCounts,
		"resultText":         iteration.ResultText,
		"resultResponseBody": iteration.ResultResponse,
		"latestOutputLine":   iteration.LatestOutputLine,
		"latestErrorLine":    iteration.LatestErrorLine,
		"lastSyncedAt":       iteration.LastSyncedAt,
		"completedAt":        iteration.CompletedAt,
		"openaiBatch":        iteration.OpenAIBatch,
	})
	if err != nil || updatedJob == nil {
		return updatedJob, err
	}

	if err := service.notifyJobRefreshed(ctx, updatedJob.ID); err != nil {
		return nil, err
	}

	return updatedJob, nil
}

func (service *JobsService) markJobsAndIterationsAsSubmissionFailed(
	ctx context.Context,
	jobs []*domain.Job,
	iterations []*domain.SubmissionIteration,
	err error,
) {
	service.markJobsAsSubmissionFailed(ctx, jobs, err)
	service.markIterationsAsSubmissionFailed(ctx, iterations, err)
}

func (service *JobsService) markIterationsAsSubmissionFailed(ctx context.Context, iterations []*domain.SubmissionIteration, err error) {
	now := time.Now()
	for _, iteration := range iterations {
		_, _ = service.iterationsRepo.Update(ctx, iteration.ID, bson.M{
			"status":       "submission_failed",
			"lastSyncedAt": &now,
			"completedAt":  &now,
			"latestErrorLine": map[string]any{
				"error": map[string]any{
					"message": err.Error(),
				},
			},
		})
	}
}

func (service *JobsService) jobDetailViewModel(job *domain.Job, iterations []*domain.SubmissionIteration) map[string]any {
	viewModel := service.jobViewModel(job)
	if viewModel == nil {
		return nil
	}

	iterationViewModels := make([]map[string]any, 0, len(iterations))
	for _, iteration := range iterations {
		iterationViewModels = append(iterationViewModels, iterationViewModel(iteration))
	}

	viewModel["iterations"] = iterationViewModels
	viewModel["iterationCount"] = len(iterationViewModels)
	return viewModel
}

func iterationViewModel(iteration *domain.SubmissionIteration) map[string]any {
	if iteration == nil {
		return nil
	}

	resolvedResultText := strings.TrimSpace(iteration.ResultText)
	if resolvedResultText == "" {
		resolvedResultText = extractResponseText(iteration.ResultResponse)
	}

	inputText := strings.TrimSpace(iteration.InputText)
	if inputText == "" {
		inputText = formatMapAsJSON(iteration.RequestBody)
	}

	return map[string]any{
		"id":                 iteration.ID,
		"jobId":              iteration.JobID,
		"submissionId":       iteration.SubmissionID,
		"submissionType":     iteration.SubmissionType,
		"iterationNumber":    iteration.IterationNumber,
		"kind":               iteration.Kind,
		"customId":           iteration.CustomID,
		"inputText":          inputText,
		"requestBody":        shared.NormalizeJSONValue(iteration.RequestBody),
		"previousResponseId": iteration.PreviousResponse,
		"responseId":         iteration.ResponseID,
		"batchId":            iteration.BatchID,
		"inputFileId":        iteration.InputFileID,
		"outputFileId":       iteration.OutputFileID,
		"errorFileId":        iteration.ErrorFileID,
		"requestCounts":      shared.NormalizeJSONValue(iteration.RequestCounts),
		"status":             iteration.Status,
		"resultText":         resolvedResultText,
		"resultResponseBody": shared.NormalizeJSONValue(iteration.ResultResponse),
		"latestOutputLine":   shared.NormalizeJSONValue(iteration.LatestOutputLine),
		"latestErrorLine":    shared.NormalizeJSONValue(iteration.LatestErrorLine),
		"toolCalls":          shared.NormalizeJSONValue(iteration.ToolCalls),
		"toolOutputs":        shared.NormalizeJSONValue(iteration.ToolOutputs),
		"lastSyncedAt":       iteration.LastSyncedAt,
		"completedAt":        iteration.CompletedAt,
		"openaiBatch":        shared.NormalizeJSONValue(iteration.OpenAIBatch),
		"createdAt":          iteration.CreatedAt,
		"updatedAt":          iteration.UpdatedAt,
		"inputTextLabel": func() string {
			if strings.TrimSpace(inputText) == "" {
				return "N/A"
			}
			return inputText
		}(),
		"createdAtLabel":    shared.FormatDateLabel(&iteration.CreatedAt, "Unknown"),
		"updatedAtLabel":    shared.FormatDateLabel(&iteration.UpdatedAt, "Unknown"),
		"lastSyncedAtLabel": shared.FormatDateLabel(iteration.LastSyncedAt, "Never"),
		"completedAtLabel":  shared.FormatDateLabel(iteration.CompletedAt, ""),
		"canRefresh":        boolInMap(iteration.Status, activeBatchStatuses),
		"isTerminal":        boolInMap(iteration.Status, terminalBatchStatuses),
	}
}

func formatMapAsJSON(value map[string]any) string {
	if len(value) == 0 {
		return ""
	}

	payload, err := json.MarshalIndent(shared.NormalizeJSONValue(value), "", "  ")
	if err != nil {
		return ""
	}

	return string(payload)
}
