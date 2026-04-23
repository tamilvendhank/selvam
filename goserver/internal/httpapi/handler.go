package httpapi

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"mime/multipart"
	"net/http"
	"strings"

	"goserver/internal/service"
	"goserver/internal/web"

	"go.uber.org/zap"
)

type Handler struct {
	frontend                   *web.Frontend
	jobsService                *service.JobsService
	proceduresService          *service.ProceduresService
	procedureExecutionsService *service.ProcedureExecutionsService
	platformAPI                http.Handler
	logger                     *zap.Logger
}

func NewHandler(
	frontend *web.Frontend,
	jobsService *service.JobsService,
	proceduresService *service.ProceduresService,
	procedureExecutionsService *service.ProcedureExecutionsService,
	platformAPI http.Handler,
	logger *zap.Logger,
) http.Handler {
	if logger == nil {
		logger = zap.NewNop()
	}

	return &Handler{
		frontend:                   frontend,
		jobsService:                jobsService,
		proceduresService:          proceduresService,
		procedureExecutionsService: procedureExecutionsService,
		platformAPI:                platformAPI,
		logger:                     logger,
	}
}

func (handler *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	defer func() {
		if recovered := recover(); recovered != nil {
			handler.logger.Error(
				"panic while serving request",
				zap.String("method", r.Method),
				zap.String("path", r.URL.Path),
				zap.Any("recovered", recovered),
			)
			handler.handleUnexpectedError(w, r, fmt.Errorf("something went wrong"))
		}
	}()

	if handler.frontend.TryServeStatic(w, r) {
		return
	}

	switch {
	case strings.HasPrefix(r.URL.Path, "/api/v1/") || r.URL.Path == "/api/v1":
		if handler.platformAPI != nil {
			handler.platformAPI.ServeHTTP(w, r)
			return
		}
		http.NotFound(w, r)
		return
	case r.Method == http.MethodGet && r.URL.Path == "/api/submissions":
		handler.listSubmissions(w, r)
	case r.Method == http.MethodPost && r.URL.Path == "/api/submissions":
		handler.createSubmission(w, r)
	case r.Method == http.MethodPost && r.URL.Path == "/api/templated-submissions":
		handler.createTemplatedSubmission(w, r)
	case r.Method == http.MethodPost && r.URL.Path == "/api/submissions/refresh-all":
		handler.refreshAllSubmissions(w, r)
	case r.Method == http.MethodGet && pathMatches(r.URL.Path, "/api/submissions/", ""):
		handler.getSubmission(w, r)
	case r.Method == http.MethodPost && pathMatches(r.URL.Path, "/api/submissions/", "/refresh"):
		handler.refreshSubmission(w, r)
	case r.Method == http.MethodGet && r.URL.Path == "/api/procedures":
		handler.listProcedures(w, r)
	case r.Method == http.MethodPost && r.URL.Path == "/api/procedures":
		handler.createProcedure(w, r)
	case r.Method == http.MethodGet && pathMatches(r.URL.Path, "/api/procedures/", ""):
		handler.getProcedure(w, r)
	case r.Method == http.MethodPut && pathMatches(r.URL.Path, "/api/procedures/", ""):
		handler.updateProcedure(w, r)
	case r.Method == http.MethodGet && r.URL.Path == "/api/procedure-executions":
		handler.listProcedureExecutions(w, r)
	case r.Method == http.MethodPost && r.URL.Path == "/api/procedure-executions":
		handler.createProcedureExecution(w, r)
	case r.Method == http.MethodGet && pathMatches(r.URL.Path, "/api/procedure-executions/", ""):
		handler.getProcedureExecution(w, r)
	case r.Method == http.MethodPost && pathMatches(r.URL.Path, "/api/procedure-executions/", "/start"):
		handler.startProcedureExecution(w, r)
	case r.Method == http.MethodPost && pathMatches(r.URL.Path, "/api/procedure-executions/", "/refresh"):
		handler.refreshProcedureExecution(w, r)
	case r.Method == http.MethodGet && r.URL.Path == "/":
		handler.frontend.ServeIndex(w, http.StatusOK)
	case r.Method == http.MethodGet && r.URL.Path == "/submissions":
		handler.frontend.ServeIndex(w, http.StatusOK)
	case r.Method == http.MethodGet && strings.HasPrefix(r.URL.Path, "/submissions/"):
		handler.frontend.ServeIndex(w, http.StatusOK)
	case r.Method == http.MethodGet && r.URL.Path == "/templated-submissions":
		handler.frontend.ServeIndex(w, http.StatusOK)
	case r.Method == http.MethodGet && r.URL.Path == "/procedures":
		handler.frontend.ServeIndex(w, http.StatusOK)
	case r.Method == http.MethodGet && r.URL.Path == "/procedure-executions":
		handler.frontend.ServeIndex(w, http.StatusOK)
	case r.Method == http.MethodGet && strings.HasPrefix(r.URL.Path, "/procedure-executions/"):
		handler.frontend.ServeIndex(w, http.StatusOK)
	case r.Method == http.MethodGet && r.URL.Path == "/platform":
		handler.frontend.ServeIndex(w, http.StatusOK)
	case r.Method == http.MethodGet && strings.HasPrefix(r.URL.Path, "/platform/"):
		handler.frontend.ServeIndex(w, http.StatusOK)
	case r.Method == http.MethodGet && r.URL.Path == "/jobs":
		http.Redirect(w, r, "/submissions", http.StatusFound)
	case r.Method == http.MethodGet && strings.HasPrefix(r.URL.Path, "/jobs/"):
		http.Redirect(w, r, "/submissions/"+strings.TrimPrefix(r.URL.Path, "/jobs/"), http.StatusFound)
	case r.Method == http.MethodGet && !strings.HasPrefix(r.URL.Path, "/api/"):
		handler.frontend.ServeIndex(w, http.StatusNotFound)
	default:
		http.NotFound(w, r)
	}
}

func (handler *Handler) listSubmissions(w http.ResponseWriter, r *http.Request) {
	jobs, err := handler.jobsService.GetJobsForList(r.Context())
	if err != nil {
		handler.handleUnexpectedError(w, r, err)
		return
	}

	handler.writeJSON(w, http.StatusOK, map[string]any{"jobs": jobs})
}

func (handler *Handler) createSubmission(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseMultipartForm(64 << 20); err != nil {
		handler.writeJSONError(w, http.StatusBadRequest, "Request body must be valid multipart form data.")
		return
	}
	defer cleanupMultipartForm(r.MultipartForm)

	promptEntries, err := getPromptEntriesFromMultipart(r.MultipartForm)
	if err != nil {
		handler.handleUnexpectedError(w, r, err)
		return
	}

	filesByPromptIndex := filesByPromptIndex(r.MultipartForm)
	hasAtLeastOneSubmission := false
	for index, entry := range promptEntries {
		if strings.TrimSpace(entry.Query) != "" || len(filesByPromptIndex[index]) > 0 {
			hasAtLeastOneSubmission = true
			break
		}
	}

	if !hasAtLeastOneSubmission {
		handler.writeJSONError(w, http.StatusBadRequest, "Please add at least one prompt or file before submitting.")
		return
	}

	enrichedEntries := make([]service.PromptEntry, 0, len(promptEntries))
	for index, entry := range promptEntries {
		entry.Files = filesByPromptIndex[index]
		enrichedEntries = append(enrichedEntries, entry)
	}

	jobs, err := handler.jobsService.SubmitPromptBatchWithFiles(r.Context(), enrichedEntries, service.SubmissionMetadata{})
	if err != nil {
		handler.handleUnexpectedError(w, r, err)
		return
	}

	handler.writeJSON(w, http.StatusCreated, map[string]any{"jobs": jobs})
}

func (handler *Handler) createTemplatedSubmission(w http.ResponseWriter, r *http.Request) {
	var payload struct {
		Records         any    `json:"records"`
		PromptTemplate  string `json:"promptTemplate"`
		Model           string `json:"model"`
		ReasoningEffort string `json:"reasoningEffort"`
	}

	if err := decodeJSONBody(r, &payload); err != nil {
		handler.writeJSONError(w, http.StatusBadRequest, "Request body must be valid JSON.")
		return
	}

	jobs, err := handler.jobsService.SubmitTemplatedPromptBatch(r.Context(), service.TemplatedSubmissionInput{
		Records:         payload.Records,
		PromptTemplate:  payload.PromptTemplate,
		Model:           payload.Model,
		ReasoningEffort: payload.ReasoningEffort,
	})
	if err != nil {
		handler.handleUnexpectedError(w, r, err)
		return
	}

	handler.writeJSON(w, http.StatusCreated, map[string]any{"jobs": jobs})
}

func (handler *Handler) refreshAllSubmissions(w http.ResponseWriter, r *http.Request) {
	jobs, err := handler.jobsService.RefreshAllJobs(r.Context())
	if err != nil {
		handler.handleUnexpectedError(w, r, err)
		return
	}
	if err := handler.procedureExecutionsService.RunProgressPass(r.Context()); err != nil {
		handler.handleUnexpectedError(w, r, err)
		return
	}

	handler.writeJSON(w, http.StatusOK, map[string]any{"jobs": jobs})
}

func (handler *Handler) getSubmission(w http.ResponseWriter, r *http.Request) {
	id, ok := pathParam(r.URL.Path, "/api/submissions/", "")
	if !ok {
		http.NotFound(w, r)
		return
	}

	job, err := handler.jobsService.GetJobDetails(r.Context(), id)
	if err != nil {
		handler.handleUnexpectedError(w, r, err)
		return
	}
	if job == nil {
		handler.writeJSONError(w, http.StatusNotFound, "Job not found.")
		return
	}

	handler.writeJSON(w, http.StatusOK, map[string]any{"job": job})
}

func (handler *Handler) refreshSubmission(w http.ResponseWriter, r *http.Request) {
	id, ok := pathParam(r.URL.Path, "/api/submissions/", "/refresh")
	if !ok {
		http.NotFound(w, r)
		return
	}

	job, err := handler.jobsService.RefreshJob(r.Context(), id)
	if err != nil {
		handler.handleUnexpectedError(w, r, err)
		return
	}
	if job == nil {
		handler.writeJSONError(w, http.StatusNotFound, "Job not found.")
		return
	}
	if err := handler.procedureExecutionsService.OnJobRefreshed(r.Context(), id); err != nil {
		handler.handleUnexpectedError(w, r, err)
		return
	}

	handler.writeJSON(w, http.StatusOK, map[string]any{"job": job})
}

func (handler *Handler) listProcedures(w http.ResponseWriter, r *http.Request) {
	procedures, err := handler.proceduresService.GetProceduresForList(r.Context())
	if err != nil {
		handler.handleUnexpectedError(w, r, err)
		return
	}

	handler.writeJSON(w, http.StatusOK, map[string]any{"procedures": procedures})
}

func (handler *Handler) getProcedure(w http.ResponseWriter, r *http.Request) {
	id, ok := pathParam(r.URL.Path, "/api/procedures/", "")
	if !ok {
		http.NotFound(w, r)
		return
	}

	procedure, err := handler.proceduresService.GetProcedureDetails(r.Context(), id)
	if err != nil {
		handler.handleUnexpectedError(w, r, err)
		return
	}
	if procedure == nil {
		handler.writeJSONError(w, http.StatusNotFound, "Procedure not found.")
		return
	}

	handler.writeJSON(w, http.StatusOK, map[string]any{"procedure": procedure})
}

func (handler *Handler) createProcedure(w http.ResponseWriter, r *http.Request) {
	var payload struct {
		Name  string           `json:"name"`
		Steps []map[string]any `json:"steps"`
	}

	if err := decodeJSONBody(r, &payload); err != nil {
		handler.writeJSONError(w, http.StatusBadRequest, "Request body must be valid JSON.")
		return
	}

	procedure, err := handler.proceduresService.CreateProcedureDefinition(r.Context(), payload.Name, payload.Steps)
	if err != nil {
		handler.handleUnexpectedError(w, r, err)
		return
	}

	handler.writeJSON(w, http.StatusCreated, map[string]any{"procedure": procedure})
}

func (handler *Handler) updateProcedure(w http.ResponseWriter, r *http.Request) {
	id, ok := pathParam(r.URL.Path, "/api/procedures/", "")
	if !ok {
		http.NotFound(w, r)
		return
	}

	var payload struct {
		Name  string           `json:"name"`
		Steps []map[string]any `json:"steps"`
	}

	if err := decodeJSONBody(r, &payload); err != nil {
		handler.writeJSONError(w, http.StatusBadRequest, "Request body must be valid JSON.")
		return
	}

	procedure, err := handler.proceduresService.UpdateProcedureDefinition(r.Context(), id, payload.Name, payload.Steps)
	if err != nil {
		handler.handleUnexpectedError(w, r, err)
		return
	}
	if procedure == nil {
		handler.writeJSONError(w, http.StatusNotFound, "Procedure not found.")
		return
	}

	handler.writeJSON(w, http.StatusOK, map[string]any{"procedure": procedure})
}

func (handler *Handler) listProcedureExecutions(w http.ResponseWriter, r *http.Request) {
	executions, err := handler.procedureExecutionsService.GetProcedureExecutionsForList(r.Context())
	if err != nil {
		handler.handleUnexpectedError(w, r, err)
		return
	}

	handler.writeJSON(w, http.StatusOK, map[string]any{"executions": executions})
}

func (handler *Handler) getProcedureExecution(w http.ResponseWriter, r *http.Request) {
	id, ok := pathParam(r.URL.Path, "/api/procedure-executions/", "")
	if !ok {
		http.NotFound(w, r)
		return
	}

	execution, err := handler.procedureExecutionsService.GetProcedureExecutionDetails(r.Context(), id)
	if err != nil {
		handler.handleUnexpectedError(w, r, err)
		return
	}
	if execution == nil {
		handler.writeJSONError(w, http.StatusNotFound, "Execution not found.")
		return
	}

	handler.writeJSON(w, http.StatusOK, map[string]any{"execution": execution})
}

func (handler *Handler) createProcedureExecution(w http.ResponseWriter, r *http.Request) {
	var payload struct {
		ProcedureID string `json:"procedureId"`
		Prompt      string `json:"prompt"`
	}

	if err := decodeJSONBody(r, &payload); err != nil {
		handler.writeJSONError(w, http.StatusBadRequest, "Request body must be valid JSON.")
		return
	}

	execution, err := handler.procedureExecutionsService.CreateAndStartExecution(r.Context(), payload.ProcedureID, payload.Prompt)
	if err != nil {
		handler.handleUnexpectedError(w, r, err)
		return
	}

	handler.writeJSON(w, http.StatusCreated, map[string]any{"execution": execution})
}

func (handler *Handler) startProcedureExecution(w http.ResponseWriter, r *http.Request) {
	id, ok := pathParam(r.URL.Path, "/api/procedure-executions/", "/start")
	if !ok {
		http.NotFound(w, r)
		return
	}

	execution, err := handler.procedureExecutionsService.StartProcedureExecutionByID(r.Context(), id)
	if err != nil {
		handler.handleUnexpectedError(w, r, err)
		return
	}
	if execution == nil {
		handler.writeJSONError(w, http.StatusNotFound, "Execution not found.")
		return
	}

	handler.writeJSON(w, http.StatusOK, map[string]any{"execution": execution})
}

func (handler *Handler) refreshProcedureExecution(w http.ResponseWriter, r *http.Request) {
	id, ok := pathParam(r.URL.Path, "/api/procedure-executions/", "/refresh")
	if !ok {
		http.NotFound(w, r)
		return
	}

	execution, err := handler.procedureExecutionsService.RefreshProcedureExecutionByID(r.Context(), id)
	if err != nil {
		handler.handleUnexpectedError(w, r, err)
		return
	}
	if execution == nil {
		handler.writeJSONError(w, http.StatusNotFound, "Execution not found.")
		return
	}

	handler.writeJSON(w, http.StatusOK, map[string]any{"execution": execution})
}

func (handler *Handler) handleUnexpectedError(w http.ResponseWriter, r *http.Request, err error) {
	handler.logger.Error(
		"request error",
		zap.String("method", r.Method),
		zap.String("path", r.URL.Path),
		zap.Error(err),
	)
	if strings.HasPrefix(r.URL.Path, "/api/") {
		handler.writeJSONError(w, http.StatusInternalServerError, err.Error())
		return
	}

	handler.frontend.ServeIndex(w, http.StatusInternalServerError)
}

func (handler *Handler) writeJSON(w http.ResponseWriter, statusCode int, payload any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(statusCode)
	if err := json.NewEncoder(w).Encode(payload); err != nil {
		handler.logger.Error(
			"failed to encode json response",
			zap.Int("status_code", statusCode),
			zap.Error(err),
		)
	}
}

func (handler *Handler) writeJSONError(w http.ResponseWriter, statusCode int, message string) {
	handler.writeJSON(w, statusCode, map[string]any{
		"error": message,
	})
}

func decodeJSONBody(r *http.Request, out any) error {
	if r.Body == nil {
		return errors.New("request body is empty")
	}

	decoder := json.NewDecoder(r.Body)
	decoder.UseNumber()
	return decoder.Decode(out)
}

func pathMatches(path, prefix, suffix string) bool {
	_, ok := pathParam(path, prefix, suffix)
	return ok
}

func pathParam(path, prefix, suffix string) (string, bool) {
	if !strings.HasPrefix(path, prefix) {
		return "", false
	}

	rest := strings.TrimPrefix(path, prefix)
	if suffix != "" {
		if !strings.HasSuffix(rest, suffix) {
			return "", false
		}
		rest = strings.TrimSuffix(rest, suffix)
	}

	rest = strings.Trim(rest, "/")
	if rest == "" || strings.Contains(rest, "/") {
		return "", false
	}

	return rest, true
}

func cleanupMultipartForm(form *multipart.Form) {
	if form != nil {
		_ = form.RemoveAll()
	}
}

func filesByPromptIndex(form *multipart.Form) map[int][]*multipart.FileHeader {
	files := make(map[int][]*multipart.FileHeader)
	if form == nil {
		return files
	}

	for fieldName, headers := range form.File {
		index, ok := promptFileIndex(fieldName)
		if !ok {
			continue
		}

		files[index] = append(files[index], headers...)
	}

	return files
}

func getPromptEntriesFromMultipart(form *multipart.Form) ([]service.PromptEntry, error) {
	if form == nil {
		return []service.PromptEntry{}, nil
	}

	rawPayload := firstFormValue(form.Value, "submissionPayload", "submissionPayload[]")
	if strings.TrimSpace(rawPayload) != "" {
		var payload []map[string]any
		if err := json.Unmarshal([]byte(rawPayload), &payload); err != nil {
			return nil, err
		}

		entries := make([]service.PromptEntry, 0, len(payload))
		for _, entry := range payload {
			query, _ := entry["query"].(string)
			model, _ := entry["model"].(string)
			reasoningEffort, _ := entry["reasoningEffort"].(string)
			entries = append(entries, service.PromptEntry{
				Query:           query,
				Model:           model,
				ReasoningEffort: reasoningEffort,
			})
		}

		return entries, nil
	}

	prompts := form.Value["prompts"]
	entries := make([]service.PromptEntry, 0, len(prompts))
	for _, prompt := range prompts {
		entries = append(entries, service.PromptEntry{Query: prompt})
	}

	return entries, nil
}

func firstFormValue(values map[string][]string, keys ...string) string {
	for _, key := range keys {
		if items := values[key]; len(items) > 0 {
			return items[0]
		}
	}

	return ""
}

func promptFileIndex(fieldName string) (int, bool) {
	if !strings.HasPrefix(fieldName, "promptFiles-") {
		return 0, false
	}

	value := strings.TrimPrefix(fieldName, "promptFiles-")
	index := 0
	for _, char := range value {
		if char < '0' || char > '9' {
			return 0, false
		}
		index = index*10 + int(char-'0')
	}

	return index, true
}

func requestContext(r *http.Request) context.Context {
	if r == nil {
		return context.Background()
	}

	return r.Context()
}
