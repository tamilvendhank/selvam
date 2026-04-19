package service

import (
	"context"
	"encoding/json"
	"fmt"
	"mime/multipart"
	"sort"
	"strconv"
	"strings"
	"time"

	"goserver/internal/config"
	"goserver/internal/domain"
	openaiapi "goserver/internal/openai"
	"goserver/internal/repository"
	"goserver/internal/shared"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type SubmissionMetadata struct {
	SubmissionType string
	PromptTemplate *string
}

type PromptEntry struct {
	Query           string
	Model           string
	ReasoningEffort string
	Files           []*multipart.FileHeader
	TemplateRecord  map[string]any
}

type TemplatedSubmissionInput struct {
	PromptTemplate  string
	Records         any
	Model           string
	ReasoningEffort string
}

type JobRefreshObserver interface {
	OnJobRefreshed(ctx context.Context, jobID string) error
}

type batchRefreshCandidate struct {
	BatchID        string
	PreferredJobID string
	LastSyncedAt   *time.Time
}

type JobsService struct {
	config           config.Config
	repo             *repository.JobsRepository
	iterationsRepo   *repository.SubmissionIterationsRepository
	openai           *openaiapi.Client
	toolExecutor     SubmissionToolExecutor
	refreshObservers []JobRefreshObserver
}

func NewJobsService(
	cfg config.Config,
	repo *repository.JobsRepository,
	iterationsRepo *repository.SubmissionIterationsRepository,
	openaiClient *openaiapi.Client,
	toolExecutor SubmissionToolExecutor,
) *JobsService {
	if toolExecutor == nil {
		toolExecutor = &UnconfiguredToolExecutor{}
	}

	return &JobsService{
		config:         cfg,
		repo:           repo,
		iterationsRepo: iterationsRepo,
		openai:         openaiClient,
		toolExecutor:   toolExecutor,
	}
}

func (service *JobsService) RegisterRefreshObserver(observer JobRefreshObserver) {
	if service == nil || observer == nil {
		return
	}

	service.refreshObservers = append(service.refreshObservers, observer)
}

func (service *JobsService) GetJobsForList(ctx context.Context) ([]map[string]any, error) {
	jobs, err := service.repo.List(ctx)
	if err != nil {
		return nil, err
	}

	result := make([]map[string]any, 0, len(jobs))
	for _, job := range jobs {
		result = append(result, service.jobViewModel(job))
	}

	return result, nil
}

func (service *JobsService) GetJobDetails(ctx context.Context, id string) (map[string]any, error) {
	job, err := service.repo.GetByID(ctx, id)
	if err != nil || job == nil {
		return nil, err
	}

	iterations, err := service.iterationsRepo.ListByJobID(ctx, id)
	if err != nil {
		return nil, err
	}

	return service.jobDetailViewModel(job, iterations), nil
}

func (service *JobsService) SubmitPromptBatchWithFiles(ctx context.Context, promptEntries []PromptEntry, metadata SubmissionMetadata) ([]map[string]any, error) {
	if len(promptEntries) == 0 {
		return nil, fmt.Errorf("at least one prompt is required")
	}

	normalizedEntries := make([]submissionEntry, 0, len(promptEntries))
	uploadedFiles := make([]domain.AttachedFile, 0)

	for _, entry := range promptEntries {
		query := strings.TrimSpace(entry.Query)
		files := entry.Files
		model := shared.NormalizeModelName(firstNonEmpty(entry.Model, service.config.OpenAI.Model))
		reasoningEffort := shared.NormalizeReasoningEffort(model, entry.ReasoningEffort)

		if query == "" && len(files) == 0 {
			continue
		}

		attachedFiles, err := service.uploadOpenAIInputFiles(ctx, files)
		if err != nil {
			service.deleteOpenAIInputFiles(ctx, uploadedFiles)
			return nil, err
		}

		uploadedFiles = append(uploadedFiles, attachedFiles...)
		normalizedEntries = append(normalizedEntries, submissionEntry{
			Query:           query,
			AttachedFiles:   attachedFiles,
			Model:           model,
			ReasoningEffort: reasoningEffort,
			TemplateRecord:  entry.TemplateRecord,
		})
	}

	if len(normalizedEntries) == 0 {
		service.deleteOpenAIInputFiles(ctx, uploadedFiles)
		return nil, fmt.Errorf("at least one prompt or file is required")
	}

	jobs, err := service.submitPromptEntries(ctx, normalizedEntries, metadata)
	if err != nil {
		service.deleteOpenAIInputFiles(ctx, uploadedFiles)
		return nil, err
	}

	return jobs, nil
}

func (service *JobsService) SubmitTemplatedPromptBatch(ctx context.Context, input TemplatedSubmissionInput) ([]map[string]any, error) {
	template := strings.TrimSpace(input.PromptTemplate)
	if template == "" {
		return nil, fmt.Errorf("prompt template is required")
	}

	records, err := normalizeTemplateRecords(input.Records)
	if err != nil {
		return nil, err
	}

	resolvedModel := shared.NormalizeModelName(firstNonEmpty(input.Model, service.config.OpenAI.Model))
	resolvedReasoning := shared.NormalizeReasoningEffort(resolvedModel, input.ReasoningEffort)
	entries := make([]submissionEntry, 0, len(records))

	for index, record := range records {
		query, err := renderPromptTemplate(template, record, index)
		if err != nil {
			return nil, err
		}

		entries = append(entries, submissionEntry{
			Query:           query,
			Model:           resolvedModel,
			ReasoningEffort: resolvedReasoning,
			TemplateRecord:  record,
		})
	}

	return service.submitPromptEntries(ctx, entries, SubmissionMetadata{
		SubmissionType: "templated",
		PromptTemplate: shared.StringPtr(template),
	})
}

func (service *JobsService) RefreshJob(ctx context.Context, id string) (map[string]any, error) {
	job, err := service.repo.GetByID(ctx, id)
	if err != nil || job == nil {
		return nil, err
	}

	latestIteration, err := service.iterationsRepo.GetLatestByJobID(ctx, id)
	if err != nil {
		return nil, err
	}

	if latestIteration == nil || strings.TrimSpace(shared.DerefString(latestIteration.BatchID)) == "" || !boolInMap(latestIteration.Status, activeBatchStatuses) {
		return service.GetJobDetails(ctx, id)
	}

	return service.refreshSharedBatchIterations(ctx, shared.DerefString(latestIteration.BatchID), id)
}

func (service *JobsService) RefreshAllJobs(ctx context.Context) ([]map[string]any, error) {
	if err := service.RunRefreshPass(ctx); err != nil {
		return nil, err
	}

	return service.GetJobsForList(ctx)
}

func (service *JobsService) RunRefreshPass(ctx context.Context) error {
	activeStatuses := make([]string, 0, len(activeBatchStatuses))
	for status := range activeBatchStatuses {
		activeStatuses = append(activeStatuses, status)
	}

	activeIterations, err := service.iterationsRepo.ListByStatuses(ctx, activeStatuses)
	if err != nil {
		return err
	}

	batchEntries := make(map[string]*batchRefreshCandidate)
	for _, iteration := range activeIterations {
		batchID := strings.TrimSpace(shared.DerefString(iteration.BatchID))
		if batchID == "" {
			continue
		}

		candidate, exists := batchEntries[batchID]
		if !exists {
			batchEntries[batchID] = &batchRefreshCandidate{
				BatchID:        batchID,
				PreferredJobID: iteration.JobID,
				LastSyncedAt:   iteration.LastSyncedAt,
			}
			continue
		}

		if candidate.PreferredJobID == "" {
			candidate.PreferredJobID = iteration.JobID
		}
		if laterTime(iteration.LastSyncedAt, candidate.LastSyncedAt) == iteration.LastSyncedAt {
			candidate.LastSyncedAt = iteration.LastSyncedAt
		}
	}

	now := time.Now()
	candidates := make([]*batchRefreshCandidate, 0, len(batchEntries))
	for _, candidate := range batchEntries {
		if service.shouldRefreshBatch(now, candidate.LastSyncedAt) {
			candidates = append(candidates, candidate)
		}
	}

	sort.Slice(candidates, func(i, j int) bool {
		return olderTime(candidates[i].LastSyncedAt, candidates[j].LastSyncedAt)
	})

	if limit := service.config.Worker.MaxBatchesPerPass; limit > 0 && len(candidates) > limit {
		candidates = candidates[:limit]
	}

	for _, candidate := range candidates {
		if _, err := service.refreshSharedBatchIterations(ctx, candidate.BatchID, candidate.PreferredJobID); err != nil {
			return err
		}
	}

	return nil
}

func (service *JobsService) shouldRefreshBatch(now time.Time, lastSyncedAt *time.Time) bool {
	if lastSyncedAt == nil {
		return true
	}
	if service.config.Worker.MinBatchRefreshAge <= 0 {
		return true
	}

	return now.Sub(*lastSyncedAt) >= service.config.Worker.MinBatchRefreshAge
}

func (service *JobsService) notifyJobRefreshed(ctx context.Context, jobID string) error {
	jobID = strings.TrimSpace(jobID)
	if jobID == "" {
		return nil
	}

	for _, observer := range service.refreshObservers {
		if observer == nil {
			continue
		}
		if err := observer.OnJobRefreshed(ctx, jobID); err != nil {
			return err
		}
	}

	return nil
}

func (service *JobsService) submitPromptEntries(ctx context.Context, promptEntries []submissionEntry, metadata SubmissionMetadata) ([]map[string]any, error) {
	if len(promptEntries) == 0 {
		return nil, fmt.Errorf("at least one prompt is required")
	}

	jobs := service.buildSubmissionJobs(promptEntries, metadata)
	createdJobs, err := service.repo.CreateMany(ctx, jobs)
	if err != nil {
		return nil, err
	}

	iterations := make([]*domain.SubmissionIteration, 0, len(createdJobs))
	for _, job := range createdJobs {
		iterations = append(iterations, service.buildSubmissionIteration(
			job,
			1,
			"initial",
			job.CustomID,
			strings.TrimSpace(job.Query),
			buildBatchRequestBody(responseRequestSpec{
				Query:           job.Query,
				AttachedFiles:   job.AttachedFiles,
				Model:           job.Model,
				ReasoningEffort: job.ReasoningEffort,
				Instructions:    service.config.OpenAI.ResponseInstructions,
			}),
			nil,
			nil,
		))
	}

	createdIterations, err := service.iterationsRepo.CreateMany(ctx, iterations)
	if err != nil {
		service.markJobsAsSubmissionFailed(ctx, createdJobs, err)
		return nil, err
	}

	submittedIterations, err := service.submitIterationsBatch(ctx, createdIterations, map[string]string{
		"app":             "webapp",
		"submission_id":   createdJobs[0].SubmissionID,
		"job_count":       strconv.Itoa(len(createdJobs)),
		"submission_type": firstNonEmpty(metadata.SubmissionType, "manual"),
	})
	if err != nil {
		service.markJobsAndIterationsAsSubmissionFailed(ctx, createdJobs, createdIterations, err)
		return nil, err
	}

	viewModels := make([]map[string]any, 0, len(submittedIterations))
	for _, iteration := range submittedIterations {
		updatedJob, syncErr := service.syncJobFromIteration(ctx, iteration.JobID, iteration)
		if syncErr != nil {
			return nil, syncErr
		}

		viewModels = append(viewModels, service.jobViewModel(updatedJob))
	}

	return viewModels, nil
}

func (service *JobsService) refreshSharedBatchIterations(ctx context.Context, batchID, preferredJobID string) (map[string]any, error) {
	iterations, err := service.iterationsRepo.ListByBatchID(ctx, batchID)
	if err != nil {
		return nil, err
	}
	if len(iterations) == 0 {
		return nil, nil
	}

	batch, err := service.openai.RetrieveBatch(ctx, batchID)
	if err != nil {
		return nil, err
	}

	outputLines := []map[string]any{}
	if strings.TrimSpace(batch.OutputFileID) != "" {
		content, err := service.openai.LoadFileContent(ctx, batch.OutputFileID)
		if err != nil {
			return nil, err
		}

		outputLines, err = parseJSONL(content)
		if err != nil {
			return nil, err
		}
	}

	errorLines := []map[string]any{}
	if strings.TrimSpace(batch.ErrorFileID) != "" {
		content, err := service.openai.LoadFileContent(ctx, batch.ErrorFileID)
		if err != nil {
			return nil, err
		}

		errorLines, err = parseJSONL(content)
		if err != nil {
			return nil, err
		}
	}

	var selectedJobID string
	for _, iteration := range iterations {
		resultText, responseBody, outputLine, errorLine := buildIterationResultSnapshot(iteration, outputLines, errorLines)
		updatedIteration, err := service.iterationsRepo.Update(ctx, iteration.ID, bson.M{
			"status":             batch.Status,
			"outputFileId":       emptyStringToNil(batch.OutputFileID),
			"errorFileId":        emptyStringToNil(batch.ErrorFileID),
			"requestCounts":      batch.RequestCounts,
			"lastSyncedAt":       time.Now(),
			"completedAt":        toDateFromUnixSeconds(batch.CompletedAt),
			"openaiBatch":        buildBatchSnapshot(batch),
			"resultText":         resultText,
			"resultResponseBody": responseBody,
			"latestOutputLine":   outputLine,
			"latestErrorLine":    errorLine,
			"responseId":         emptyStringToNil(extractResponseID(responseBody)),
			"toolCalls":          normalizeToolCallMaps(extractToolCalls(responseBody)),
		})
		if err != nil {
			return nil, err
		}

		latestIteration, err := service.progressSubmissionIteration(ctx, updatedIteration)
		if err != nil {
			return nil, err
		}
		if latestIteration == nil {
			latestIteration = updatedIteration
		}

		updatedJob, err := service.syncJobFromIteration(ctx, latestIteration.JobID, latestIteration)
		if err != nil {
			return nil, err
		}

		if updatedJob.ID == preferredJobID || selectedJobID == "" {
			selectedJobID = updatedJob.ID
		}
	}

	if selectedJobID == "" {
		return nil, nil
	}

	return service.GetJobDetails(ctx, selectedJobID)
}

func (service *JobsService) buildSubmissionJobs(promptEntries []submissionEntry, metadata SubmissionMetadata) []*domain.Job {
	now := time.Now()
	submissionID := fmt.Sprintf("submission-%d", now.UnixMilli())
	submissionType := firstNonEmpty(metadata.SubmissionType, "manual")
	jobs := make([]*domain.Job, 0, len(promptEntries))

	for index, entry := range promptEntries {
		job := &domain.Job{
			Query:           entry.Query,
			SubmissionID:    submissionID,
			SubmissionIndex: index + 1,
			SubmissionSize:  len(promptEntries),
			SubmissionType:  submissionType,
			PromptTemplate:  metadata.PromptTemplate,
			TemplateRecord:  entry.TemplateRecord,
			AttachedFiles:   entry.AttachedFiles,
			Model:           shared.NormalizeModelName(firstNonEmpty(entry.Model, service.config.OpenAI.Model)),
			ReasoningEffort: shared.NormalizeReasoningEffort(firstNonEmpty(entry.Model, service.config.OpenAI.Model), shared.DerefString(entry.ReasoningEffort)),
			Status:          "preparing",
			ResultText:      "",
			CreatedAt:       now,
			UpdatedAt:       now,
		}

		job.ObjectID = primitive.NewObjectID()
		job.NormalizeID()
		job.CustomID = "job-" + job.ID
		jobs = append(jobs, job)
	}

	return jobs
}

func (service *JobsService) buildJSONLPayload(lines []map[string]any) ([]byte, error) {
	chunks := make([]string, 0, len(lines))
	for _, line := range lines {
		payload, err := json.Marshal(line)
		if err != nil {
			return nil, err
		}

		chunks = append(chunks, string(payload))
	}

	return []byte(strings.Join(chunks, "\n") + "\n"), nil
}

func (service *JobsService) uploadOpenAIInputFiles(ctx context.Context, files []*multipart.FileHeader) ([]domain.AttachedFile, error) {
	uploadedFiles := make([]domain.AttachedFile, 0, len(files))

	for _, fileHeader := range files {
		file, err := fileHeader.Open()
		if err != nil {
			service.deleteOpenAIInputFiles(ctx, uploadedFiles)
			return nil, err
		}

		uploadedFile, err := service.openai.UploadUserFile(ctx, fileHeader.Filename, fileHeader.Header.Get("Content-Type"), file)
		file.Close()
		if err != nil {
			service.deleteOpenAIInputFiles(ctx, uploadedFiles)
			return nil, err
		}

		if err := service.openai.WaitForFileProcessing(ctx, uploadedFile.ID); err != nil {
			ignoreErr := service.openai.DeleteFile(ctx, uploadedFile.ID)
			_ = ignoreErr
			service.deleteOpenAIInputFiles(ctx, uploadedFiles)
			return nil, err
		}

		size := fileHeader.Size
		uploadedFiles = append(uploadedFiles, domain.AttachedFile{
			OpenAIFileID: uploadedFile.ID,
			OriginalName: fileHeader.Filename,
			MimeType:     firstNonEmpty(fileHeader.Header.Get("Content-Type"), "application/octet-stream"),
			Size:         &size,
		})
	}

	return uploadedFiles, nil
}

func (service *JobsService) deleteOpenAIInputFiles(ctx context.Context, files []domain.AttachedFile) {
	for _, file := range files {
		if strings.TrimSpace(file.OpenAIFileID) == "" {
			continue
		}

		_ = service.openai.DeleteFile(ctx, file.OpenAIFileID)
	}
}

func (service *JobsService) markJobsAsSubmissionFailed(ctx context.Context, jobs []*domain.Job, err error) {
	now := time.Now()
	for _, job := range jobs {
		_, _ = service.repo.Update(ctx, job.ID, bson.M{
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

func (service *JobsService) jobViewModel(job *domain.Job) map[string]any {
	if job == nil {
		return nil
	}

	resolvedResultText := strings.TrimSpace(job.ResultText)
	if resolvedResultText == "" {
		resolvedResultText = extractResponseText(shared.MapFromAny(job.ResultResponseBody))
	}

	normalizedQuery := strings.TrimSpace(job.Query)
	modelFromResponse := ""
	if job.ResultResponseBody != nil {
		if value, ok := job.ResultResponseBody["model"].(string); ok {
			modelFromResponse = value
		}
	}

	model := shared.NormalizeModelName(firstNonEmpty(job.Model, modelFromResponse, service.config.OpenAI.Model))
	reasoningEffort := shared.NormalizeReasoningEffort(model, shared.DerefString(job.ReasoningEffort))
	isCompleted := job.Status == "completed"
	isActive := boolInMap(job.Status, activeBatchStatuses)
	isTerminal := boolInMap(job.Status, terminalBatchStatuses)

	return map[string]any{
		"id":                 job.ID,
		"query":              normalizedQuery,
		"customId":           job.CustomID,
		"submissionId":       job.SubmissionID,
		"submissionIndex":    job.SubmissionIndex,
		"submissionSize":     job.SubmissionSize,
		"submissionType":     job.SubmissionType,
		"promptTemplate":     job.PromptTemplate,
		"templateRecord":     shared.NormalizeJSONValue(job.TemplateRecord),
		"attachedFiles":      shared.NormalizeJSONValue(job.AttachedFiles),
		"model":              model,
		"reasoningEffort":    reasoningEffort,
		"status":             job.Status,
		"batchId":            job.BatchID,
		"inputFileId":        job.InputFileID,
		"outputFileId":       job.OutputFileID,
		"errorFileId":        job.ErrorFileID,
		"requestCounts":      shared.NormalizeJSONValue(job.RequestCounts),
		"resultText":         resolvedResultText,
		"resultResponseBody": shared.NormalizeJSONValue(job.ResultResponseBody),
		"latestOutputLine":   shared.NormalizeJSONValue(job.LatestOutputLine),
		"latestErrorLine":    shared.NormalizeJSONValue(job.LatestErrorLine),
		"lastSyncedAt":       job.LastSyncedAt,
		"completedAt":        job.CompletedAt,
		"openaiBatch":        shared.NormalizeJSONValue(job.OpenAIBatch),
		"createdAt":          job.CreatedAt,
		"updatedAt":          job.UpdatedAt,
		"queryLabel": func() string {
			if normalizedQuery == "" {
				return "N/A"
			}
			return normalizedQuery
		}(),
		"reasoningEffortLabel": func() string {
			if reasoningEffort == nil {
				return "N/A"
			}
			return *reasoningEffort
		}(),
		"createdAtLabel":    shared.FormatDateLabel(&job.CreatedAt, "Unknown"),
		"updatedAtLabel":    shared.FormatDateLabel(&job.UpdatedAt, "Unknown"),
		"lastSyncedAtLabel": shared.FormatDateLabel(job.LastSyncedAt, "Never"),
		"completedAtLabel":  shared.FormatDateLabel(job.CompletedAt, ""),
		"previewText":       previewText(resolvedResultText),
		"canRefresh":        isActive,
		"canViewResults":    isCompleted && resolvedResultText != "",
		"isCompleted":       isCompleted,
		"isActive":          isActive,
		"isTerminal":        isTerminal,
	}
}

type submissionEntry struct {
	Query           string
	AttachedFiles   []domain.AttachedFile
	Model           string
	ReasoningEffort *string
	TemplateRecord  map[string]any
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}

	return ""
}

func laterTime(first, second *time.Time) *time.Time {
	if first == nil {
		return second
	}
	if second == nil {
		return first
	}
	if first.After(*second) {
		return first
	}

	return second
}

func olderTime(first, second *time.Time) bool {
	if first == nil {
		return second != nil
	}
	if second == nil {
		return false
	}

	return first.Before(*second)
}
