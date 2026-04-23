package ai

import (
	"context"
	"fmt"
	"strings"

	legacydomain "goserver/internal/domain"
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
