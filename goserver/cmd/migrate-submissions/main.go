package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"strings"
	"time"

	"goserver/internal/config"
	"goserver/internal/db"
	"goserver/internal/domain"
	"goserver/internal/logging"
	"goserver/internal/repository"
	"goserver/internal/shared"

	"go.uber.org/zap"
)

type migrationStats struct {
	totalJobs      int
	selectedJobs   int
	migratedJobs   int
	skippedJobs    int
	failedJobs     int
	dryRunMigrated int
}

func main() {
	var (
		dryRun = flag.Bool("dry-run", false, "preview the migration without writing any data")
		jobID  = flag.String("job-id", "", "migrate only a single submission/job by id")
		limit  = flag.Int("limit", 0, "limit the number of jobs processed after filtering")
	)
	flag.Parse()

	bootstrapLogger := logging.NewBootstrap()
	defer logging.Sync(bootstrapLogger)

	cfg, err := config.LoadOffline()
	if err != nil {
		bootstrapLogger.Fatal("failed to load config", zap.Error(err))
	}

	logger, err := logging.New(cfg.Logging)
	if err != nil {
		bootstrapLogger.Fatal("failed to configure logger", zap.Error(err))
	}
	defer logging.Sync(logger)

	migrationLogger := logger.Named("migrate-submissions")

	rootContext := context.Background()

	mongoClient, err := db.Connect(rootContext, cfg)
	if err != nil {
		migrationLogger.Fatal("failed to connect to mongodb", zap.Error(err))
	}
	defer mongoClient.Close(rootContext)

	jobsRepository := repository.NewJobsRepository(mongoClient.Database(), cfg.MongoDB.JobsCollectionName)
	iterationsRepository := repository.NewSubmissionIterationsRepository(mongoClient.Database(), cfg.MongoDB.SubmissionIterationsCollection)

	jobs, err := jobsRepository.List(rootContext)
	if err != nil {
		migrationLogger.Fatal("failed to load jobs", zap.Error(err))
	}

	stats := migrationStats{
		totalJobs: len(jobs),
	}

	selectedJobs := filterJobs(jobs, strings.TrimSpace(*jobID), *limit)
	stats.selectedJobs = len(selectedJobs)

	migrationLogger.Info(
		"starting submissions migration",
		zap.Int("total_jobs", stats.totalJobs),
		zap.Int("selected_jobs", stats.selectedJobs),
		zap.Bool("dry_run", *dryRun),
		zap.String("target_job", strings.TrimSpace(*jobID)),
		zap.String("collection", cfg.MongoDB.SubmissionIterationsCollection),
	)

	for _, job := range selectedJobs {
		iterations, err := iterationsRepository.ListByJobID(rootContext, job.ID)
		if err != nil {
			stats.failedJobs++
			migrationLogger.Error("failed to inspect job", zap.String("job_id", job.ID), zap.Error(err))
			continue
		}

		if len(iterations) > 0 {
			stats.skippedJobs++
			migrationLogger.Info(
				"skipping job with existing iterations",
				zap.String("job_id", job.ID),
				zap.String("summary", summarizeJob(job)),
				zap.Int("iteration_count", len(iterations)),
			)
			continue
		}

		iteration := buildMigratedIteration(job, cfg.OpenAI.ResponseInstructions)
		if *dryRun {
			stats.dryRunMigrated++
			migrationLogger.Info(
				"would migrate job",
				zap.String("job_id", job.ID),
				zap.String("summary", summarizeJob(job)),
				zap.String("status", iteration.Status),
			)
			continue
		}

		if _, err := iterationsRepository.Create(rootContext, iteration); err != nil {
			stats.failedJobs++
			migrationLogger.Error(
				"failed to migrate job",
				zap.String("job_id", job.ID),
				zap.String("summary", summarizeJob(job)),
				zap.Error(err),
			)
			continue
		}

		stats.migratedJobs++
		migrationLogger.Info(
			"migrated job",
			zap.String("job_id", job.ID),
			zap.String("summary", summarizeJob(job)),
			zap.String("iteration_id", iteration.ID),
		)
	}

	migrationLogger.Info(
		"submissions migration finished",
		zap.Int("selected", stats.selectedJobs),
		zap.Int("migrated", stats.migratedJobs),
		zap.Int("dry_run_migrated", stats.dryRunMigrated),
		zap.Int("skipped", stats.skippedJobs),
		zap.Int("failed", stats.failedJobs),
	)

	if stats.failedJobs > 0 {
		os.Exit(1)
	}
}

func filterJobs(jobs []*domain.Job, jobID string, limit int) []*domain.Job {
	filtered := make([]*domain.Job, 0, len(jobs))

	for _, job := range jobs {
		if job == nil {
			continue
		}

		if jobID != "" && job.ID != jobID {
			continue
		}

		filtered = append(filtered, job)
		if limit > 0 && len(filtered) >= limit {
			break
		}
	}

	return filtered
}

func buildMigratedIteration(job *domain.Job, instructions string) *domain.SubmissionIteration {
	now := time.Now()
	createdAt := job.CreatedAt
	if createdAt.IsZero() {
		createdAt = now
	}

	updatedAt := job.UpdatedAt
	if updatedAt.IsZero() {
		updatedAt = createdAt
	}

	resolvedModel := shared.NormalizeModelName(firstNonEmpty(job.Model, extractResponseModel(job.ResultResponseBody), shared.DefaultModel))
	resolvedReasoning := shared.NormalizeReasoningEffort(resolvedModel, shared.DerefString(job.ReasoningEffort))
	status := firstNonEmpty(strings.TrimSpace(job.Status), deriveStatus(job))
	resultResponseBody := normalizeMap(job.ResultResponseBody)
	latestOutputLine := normalizeMap(job.LatestOutputLine)
	latestErrorLine := normalizeMap(job.LatestErrorLine)
	requestCounts := normalizeMap(job.RequestCounts)
	openAIBatch := normalizeMap(job.OpenAIBatch)
	toolCalls := extractToolCallMaps(resultResponseBody)

	return &domain.SubmissionIteration{
		JobID:            job.ID,
		SubmissionID:     job.SubmissionID,
		SubmissionType:   job.SubmissionType,
		IterationNumber:  1,
		Kind:             "initial",
		CustomID:         firstNonEmpty(strings.TrimSpace(job.CustomID), fmt.Sprintf("job-%s-iter-1", job.ID)),
		InputText:        strings.TrimSpace(job.Query),
		RequestBody:      buildRequestBody(job.Query, job.AttachedFiles, resolvedModel, resolvedReasoning, instructions),
		PreviousResponse: nil,
		ResponseID:       emptyStringToNil(extractResponseID(resultResponseBody)),
		BatchID:          job.BatchID,
		InputFileID:      job.InputFileID,
		OutputFileID:     job.OutputFileID,
		ErrorFileID:      job.ErrorFileID,
		RequestCounts:    requestCounts,
		Status:           status,
		ResultText:       strings.TrimSpace(job.ResultText),
		ResultResponse:   resultResponseBody,
		LatestOutputLine: latestOutputLine,
		LatestErrorLine:  latestErrorLine,
		ToolCalls:        toolCalls,
		ToolOutputs:      nil,
		NextIterationID:  nil,
		FollowUpState:    "",
		FollowUpClaimed:  nil,
		LastSyncedAt:     job.LastSyncedAt,
		CompletedAt:      job.CompletedAt,
		OpenAIBatch:      openAIBatch,
		CreatedAt:        createdAt,
		UpdatedAt:        updatedAt,
	}
}

func buildRequestBody(query string, attachedFiles []domain.AttachedFile, model string, reasoningEffort *string, instructions string) map[string]any {
	content := make([]map[string]any, 0, 1+len(attachedFiles))
	if strings.TrimSpace(query) != "" {
		content = append(content, map[string]any{
			"type": "input_text",
			"text": query,
		})
	}

	for _, attachedFile := range attachedFiles {
		if strings.TrimSpace(attachedFile.OpenAIFileID) == "" {
			continue
		}

		content = append(content, map[string]any{
			"type":    "input_file",
			"file_id": attachedFile.OpenAIFileID,
		})
	}

	requestBody := map[string]any{
		"model":        model,
		"instructions": firstNonEmpty(strings.TrimSpace(instructions), "Answer the user's query clearly and concisely."),
		"input": []map[string]any{
			{
				"role":    "user",
				"content": content,
			},
		},
		"text": map[string]any{
			"format": map[string]any{
				"type": "text",
			},
		},
	}

	if reasoningEffort != nil {
		requestBody["reasoning"] = map[string]any{
			"effort": *reasoningEffort,
		}
	}

	return requestBody
}

func extractResponseID(responseBody map[string]any) string {
	if responseBody == nil {
		return ""
	}

	value, _ := responseBody["id"].(string)
	return strings.TrimSpace(value)
}

func extractResponseModel(responseBody map[string]any) string {
	if responseBody == nil {
		return ""
	}

	value, _ := responseBody["model"].(string)
	return strings.TrimSpace(value)
}

func extractToolCallMaps(responseBody map[string]any) []map[string]any {
	if responseBody == nil {
		return nil
	}

	outputItems, ok := responseBody["output"].([]any)
	if !ok {
		return nil
	}

	toolCalls := make([]map[string]any, 0)
	for _, item := range outputItems {
		payload := normalizeMapFromAny(item)
		if payload == nil {
			continue
		}

		itemType, _ := payload["type"].(string)
		if itemType != "function_call" && itemType != "custom_tool_call" {
			continue
		}

		toolCalls = append(toolCalls, map[string]any{
			"id":        payload["id"],
			"callId":    payload["call_id"],
			"type":      payload["type"],
			"name":      payload["name"],
			"arguments": payload["arguments"],
			"input":     payload["input"],
			"status":    payload["status"],
			"raw":       payload,
		})
	}

	if len(toolCalls) == 0 {
		return nil
	}

	return toolCalls
}

func deriveStatus(job *domain.Job) string {
	if strings.TrimSpace(job.ResultText) != "" || len(job.ResultResponseBody) > 0 {
		return "completed"
	}

	if job.BatchID != nil && strings.TrimSpace(*job.BatchID) != "" {
		return "in_progress"
	}

	return "preparing"
}

func summarizeJob(job *domain.Job) string {
	if job == nil {
		return "unknown"
	}

	if trimmed := strings.TrimSpace(job.Query); trimmed != "" {
		if len(trimmed) > 80 {
			return trimmed[:80]
		}
		return trimmed
	}

	return firstNonEmpty(strings.TrimSpace(job.SubmissionType), "submission")
}

func normalizeMap(value map[string]any) map[string]any {
	if value == nil {
		return nil
	}

	normalized := shared.NormalizeJSONValue(value)
	mapped, _ := normalized.(map[string]any)
	return mapped
}

func normalizeMapFromAny(value any) map[string]any {
	if value == nil {
		return nil
	}

	normalized := shared.NormalizeJSONValue(value)
	mapped, _ := normalized.(map[string]any)
	return mapped
}

func emptyStringToNil(value string) *string {
	if strings.TrimSpace(value) == "" {
		return nil
	}

	return shared.StringPtr(strings.TrimSpace(value))
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}

	return ""
}
