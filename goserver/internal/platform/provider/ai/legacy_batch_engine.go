package ai

import (
	"context"
	"fmt"
	"strings"
	"time"

	legacydomain "goserver/internal/domain"
	"goserver/internal/platform/domain"
	"goserver/internal/platform/ports"
	legacyrepo "goserver/internal/repository"
	legacyservice "goserver/internal/service"
)

type LegacyBatchAIReviewEngine struct {
	jobsService *legacyservice.JobsService
	jobsRepo    *legacyrepo.JobsRepository
}

func NewLegacyBatchAIReviewEngine(jobsService *legacyservice.JobsService, jobsRepo *legacyrepo.JobsRepository) *LegacyBatchAIReviewEngine {
	return &LegacyBatchAIReviewEngine{
		jobsService: jobsService,
		jobsRepo:    jobsRepo,
	}
}

func (engine *LegacyBatchAIReviewEngine) SubmitReviewBatch(ctx context.Context, request ports.AIReviewBatchRequest) (*ports.AIAsyncTask, error) {
	if engine == nil || engine.jobsService == nil {
		return (&NoopAIReviewEngine{}).SubmitReviewBatch(ctx, request)
	}
	if len(request.Items) == 0 {
		return nil, fmt.Errorf("at least one batch item is required")
	}

	entries := make([]legacyservice.PromptEntry, 0, len(request.Items))
	for _, item := range request.Items {
		prompt := strings.TrimSpace(item.Prompt)
		if prompt == "" {
			continue
		}
		entries = append(entries, legacyservice.PromptEntry{
			Query:           prompt,
			Model:           request.ModelName,
			ReasoningEffort: item.ReasoningEffort,
			TemplateRecord:  item.Metadata,
		})
	}

	jobs, err := engine.jobsService.SubmitPromptBatchWithFiles(ctx, entries, legacyservice.SubmissionMetadata{
		SubmissionType: "investing_review_async",
	})
	if err != nil {
		return nil, err
	}
	if len(jobs) == 0 {
		return nil, fmt.Errorf("legacy batch engine returned no jobs")
	}

	task := &ports.AIAsyncTask{
		Provider:        "openai-batch",
		TaskKind:        "review_batch",
		LocalObjectType: "submission",
		Status:          "queued",
		Metadata: map[string]any{
			"jobCount": len(jobs),
		},
	}
	for index, job := range jobs {
		jobID, _ := job["id"].(string)
		batchID, _ := job["batchId"].(string)
		submissionID, _ := job["submissionId"].(string)
		status, _ := job["status"].(string)
		task.JobIDs = append(task.JobIDs, jobID)
		if index == 0 {
			task.LocalObjectID = submissionID
			task.SubmissionID = submissionID
			task.RepresentativeJobID = jobID
			task.BatchID = batchID
			task.Status = status
		}
	}

	return task, nil
}

func (engine *LegacyBatchAIReviewEngine) SubmitBatch(ctx context.Context, request ports.SubmitBatchRequest) (*ports.BatchSubmissionResult, error) {
	if engine == nil || engine.jobsService == nil {
		return (&NoopAIBatchEngine{}).SubmitBatch(ctx, request)
	}
	if len(request.Items) == 0 {
		return nil, fmt.Errorf("at least one batch item is required")
	}

	entries := make([]legacyservice.PromptEntry, 0, len(request.Items))
	for _, item := range request.Items {
		prompt := strings.TrimSpace(item.Prompt)
		if prompt == "" {
			continue
		}
		record := map[string]any{
			"correlationId": item.CorrelationID,
			"referenceId":   item.ReferenceID,
			"workflowRunId": request.WorkflowRunID,
			"itemType":      item.ItemType,
		}
		for key, value := range item.InputPayload {
			record[key] = value
		}
		for key, value := range item.TemplateRecord {
			record[key] = value
		}
		for key, value := range item.Metadata {
			record[key] = value
		}
		entries = append(entries, legacyservice.PromptEntry{
			Query:           prompt,
			Model:           request.ModelName,
			ReasoningEffort: item.ReasoningEffort,
			TemplateRecord:  record,
		})
	}

	jobs, err := engine.jobsService.SubmitPromptBatchWithFiles(ctx, entries, legacyservice.SubmissionMetadata{
		SubmissionType: string(request.JobType),
	})
	if err != nil {
		return nil, err
	}
	if len(jobs) == 0 {
		return nil, fmt.Errorf("legacy batch engine returned no jobs")
	}

	result := &ports.BatchSubmissionResult{
		ProviderName: "openai-batch",
		Status:       domain.BatchJobStatusSubmitted,
		Metadata: map[string]any{
			"itemCount": len(jobs),
		},
	}
	for index, job := range jobs {
		jobID, _ := job["id"].(string)
		batchID, _ := job["batchId"].(string)
		submissionID, _ := job["submissionId"].(string)
		status := normalizeLegacyBatchJobStatus(job["status"])
		if submittedAt, ok := job["createdAt"].(time.Time); ok {
			result.SubmittedAt = &submittedAt
		}
		templateRecord, _ := job["templateRecord"].(map[string]any)
		correlationID, _ := templateRecord["correlationId"].(string)
		result.Items = append(result.Items, ports.BatchSubmissionItem{
			CorrelationID:      correlationID,
			ProviderItemHandle: jobID,
			Status:             normalizeLegacyBatchItemStatus(job["status"]),
			Metadata:           map[string]any{"jobId": jobID},
		})
		if index == 0 {
			result.ProviderJobHandle = batchID
			result.LocalJobHandle = submissionID
			result.Status = status
		}
	}

	return result, nil
}

func (engine *LegacyBatchAIReviewEngine) RefreshTask(ctx context.Context, task ports.AIAsyncTask) (*ports.AIAsyncTask, error) {
	if engine == nil || engine.jobsService == nil || engine.jobsRepo == nil {
		return (&NoopAIReviewEngine{}).RefreshTask(ctx, task)
	}

	if strings.TrimSpace(task.RepresentativeJobID) != "" {
		if _, err := engine.jobsService.RefreshJob(ctx, task.RepresentativeJobID); err != nil {
			return nil, err
		}
	}

	jobs, err := loadJobsForTask(ctx, engine.jobsRepo, task)
	if err != nil {
		return nil, err
	}
	if len(jobs) == 0 {
		task.Status = "unavailable"
		return &task, nil
	}

	completed := true
	failed := false
	for _, job := range jobs {
		status := strings.TrimSpace(job.Status)
		switch status {
		case "completed":
		case "failed", "cancelled", "expired", "submission_failed":
			failed = true
			completed = false
		default:
			completed = false
		}
	}

	switch {
	case failed:
		task.Status = "failed"
	case completed:
		task.Status = "completed"
		task.ResultAvailable = true
	default:
		task.Status = "in_progress"
	}
	if jobs[0].LastSyncedAt != nil {
		task.LastSyncedAt = jobs[0].LastSyncedAt
	}
	return &task, nil
}

func (engine *LegacyBatchAIReviewEngine) GetBatchStatus(ctx context.Context, jobHandle string) (*ports.BatchStatusResult, error) {
	if engine == nil || engine.jobsRepo == nil {
		return (&NoopAIBatchEngine{}).GetBatchStatus(ctx, jobHandle)
	}

	jobs, err := loadJobsByHandle(ctx, engine.jobsRepo, jobHandle)
	if err != nil {
		return nil, err
	}
	if len(jobs) == 0 {
		return &ports.BatchStatusResult{
			ProviderName:      "openai-batch",
			ProviderJobHandle: jobHandle,
			Status:            domain.BatchJobStatusFailed,
			RawProviderStatus: map[string]any{"error": "provider job handle not found"},
		}, nil
	}

	now := time.Now().UTC()
	result := &ports.BatchStatusResult{
		ProviderName:      "openai-batch",
		ProviderJobHandle: jobHandle,
		LastPolledAt:      &now,
		RawProviderStatus: map[string]any{"jobCount": len(jobs)},
		Retryable:         true,
	}

	completed := 0
	failed := 0
	processing := 0
	allCompleted := true
	anyRunning := false
	for _, job := range jobs {
		status := normalizeLegacyBatchJobStatus(job.Status)
		itemStatus := normalizeLegacyBatchItemStatus(job.Status)
		correlationID, _ := job.TemplateRecord["correlationId"].(string)
		result.Items = append(result.Items, ports.BatchStatusItem{
			CorrelationID: correlationID,
			Status:        itemStatus,
		})
		switch itemStatus {
		case domain.BatchItemStatusCompleted:
			completed++
		case domain.BatchItemStatusFailed, domain.BatchItemStatusInvalidOutput:
			failed++
			allCompleted = false
		case domain.BatchItemStatusSubmitted, domain.BatchItemStatusProcessing:
			processing++
			allCompleted = false
			anyRunning = true
		default:
			allCompleted = false
		}
		if status == domain.BatchJobStatusRunning || status == domain.BatchJobStatusSubmitted {
			anyRunning = true
		}
	}

	result.ItemsCompletedCount = completed
	result.ItemsFailedCount = failed
	result.ItemsProcessingCount = processing
	result.ResultAvailable = completed > 0 || failed > 0
	switch {
	case completed > 0 && failed > 0:
		result.Status = domain.BatchJobStatusPartiallyCompleted
	case allCompleted:
		result.Status = domain.BatchJobStatusCompleted
	case anyRunning:
		result.Status = domain.BatchJobStatusRunning
	case failed == len(jobs):
		result.Status = domain.BatchJobStatusFailed
	default:
		result.Status = domain.BatchJobStatusSubmitted
	}
	if latestCompletedAt(jobs) != nil {
		result.CompletedAt = latestCompletedAt(jobs)
	}

	return result, nil
}

func (engine *LegacyBatchAIReviewEngine) GetBatchResults(ctx context.Context, jobHandle string) (*ports.BatchResultsResult, error) {
	if engine == nil || engine.jobsRepo == nil {
		return (&NoopAIBatchEngine{}).GetBatchResults(ctx, jobHandle)
	}

	jobs, err := loadJobsByHandle(ctx, engine.jobsRepo, jobHandle)
	if err != nil {
		return nil, err
	}

	result := &ports.BatchResultsResult{
		ProviderName:      "openai-batch",
		ProviderJobHandle: jobHandle,
		Status:            domain.BatchJobStatusRunning,
		RawPayload: map[string]any{
			"jobCount": len(jobs),
		},
	}
	completed := 0
	failed := 0
	for _, job := range jobs {
		correlationID, _ := job.TemplateRecord["correlationId"].(string)
		itemStatus := normalizeLegacyBatchItemStatus(job.Status)
		item := ports.BatchResultItem{
			CorrelationID: correlationID,
			Status:        itemStatus,
			OutputPayload: map[string]any{
				"resultText":         job.ResultText,
				"resultResponseBody": job.ResultResponseBody,
				"latestOutputLine":   job.LatestOutputLine,
				"latestErrorLine":    job.LatestErrorLine,
				"templateRecord":     job.TemplateRecord,
				"jobId":              job.ID,
				"submissionId":       job.SubmissionID,
				"customId":           job.CustomID,
			},
			ProviderMetadata: map[string]any{
				"jobId":        job.ID,
				"submissionId": job.SubmissionID,
			},
		}
		if itemStatus == domain.BatchItemStatusFailed {
			item.ErrorSummary = extractLegacyError(job)
			failed++
		}
		if itemStatus == domain.BatchItemStatusCompleted {
			completed++
		}
		result.Items = append(result.Items, item)
	}

	switch {
	case completed > 0 && failed > 0:
		result.Status = domain.BatchJobStatusPartiallyCompleted
	case completed == len(jobs) && len(jobs) > 0:
		result.Status = domain.BatchJobStatusCompleted
	case failed == len(jobs) && len(jobs) > 0:
		result.Status = domain.BatchJobStatusFailed
	default:
		result.Status = domain.BatchJobStatusRunning
	}
	result.CompletedAt = latestCompletedAt(jobs)

	return result, nil
}

func loadJobsForTask(ctx context.Context, repository *legacyrepo.JobsRepository, task ports.AIAsyncTask) ([]*legacydomain.Job, error) {
	if repository == nil {
		return nil, nil
	}
	if strings.TrimSpace(task.BatchID) != "" {
		return repository.ListByBatchID(ctx, task.BatchID)
	}
	if strings.TrimSpace(task.RepresentativeJobID) != "" {
		job, err := repository.GetByID(ctx, task.RepresentativeJobID)
		if err != nil || job == nil {
			return nil, err
		}
		return []*legacydomain.Job{job}, nil
	}

	return nil, nil
}

func loadJobsByHandle(ctx context.Context, repository *legacyrepo.JobsRepository, jobHandle string) ([]*legacydomain.Job, error) {
	if repository == nil || strings.TrimSpace(jobHandle) == "" {
		return nil, nil
	}
	if jobs, err := repository.ListByBatchID(ctx, jobHandle); err == nil && len(jobs) > 0 {
		return jobs, nil
	}
	if job, err := repository.GetByID(ctx, jobHandle); err != nil {
		return nil, err
	} else if job != nil {
		return []*legacydomain.Job{job}, nil
	}

	return nil, nil
}

func normalizeLegacyBatchJobStatus(value any) domain.BatchJobStatus {
	status := strings.TrimSpace(fmt.Sprint(value))
	switch status {
	case "completed":
		return domain.BatchJobStatusCompleted
	case "failed", "submission_failed":
		return domain.BatchJobStatusFailed
	case "cancelled", "expired":
		return domain.BatchJobStatusCancelled
	case "in_progress", "processing":
		return domain.BatchJobStatusRunning
	case "queued", "validating":
		return domain.BatchJobStatusSubmitted
	default:
		return domain.BatchJobStatusCreated
	}
}

func normalizeLegacyBatchItemStatus(value any) domain.BatchItemStatus {
	status := strings.TrimSpace(fmt.Sprint(value))
	switch status {
	case "completed":
		return domain.BatchItemStatusCompleted
	case "failed", "cancelled", "expired", "submission_failed":
		return domain.BatchItemStatusFailed
	case "in_progress", "processing":
		return domain.BatchItemStatusProcessing
	case "queued", "validating":
		return domain.BatchItemStatusSubmitted
	default:
		return domain.BatchItemStatusPending
	}
}

func latestCompletedAt(jobs []*legacydomain.Job) *time.Time {
	var latest *time.Time
	for _, job := range jobs {
		if job == nil || job.CompletedAt == nil {
			continue
		}
		if latest == nil || latest.Before(*job.CompletedAt) {
			completedAt := *job.CompletedAt
			latest = &completedAt
		}
	}

	return latest
}

func extractLegacyError(job *legacydomain.Job) string {
	if job == nil {
		return ""
	}
	if job.LatestErrorLine == nil {
		return ""
	}
	if errorValue, ok := job.LatestErrorLine["error"].(map[string]any); ok {
		if message, ok := errorValue["message"].(string); ok {
			return message
		}
	}
	return ""
}
