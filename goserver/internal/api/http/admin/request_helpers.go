package admin

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	domaincommon "goserver/internal/domain/common"
	platformrepo "goserver/internal/platform/repository"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

const (
	defaultPageSize = 50
	maxPageSize     = 200
)

func decodeOptionalJSON(request *http.Request, out any) error {
	if request.Body == nil {
		return nil
	}
	defer request.Body.Close()
	decoder := json.NewDecoder(request.Body)
	if err := decoder.Decode(out); err != nil {
		if errors.Is(err, io.EOF) {
			return nil
		}
		return badRequestf("request body must be valid JSON")
	}
	return nil
}

func pathMatches(path string, prefix string, suffix string) bool {
	if !strings.HasPrefix(path, prefix) {
		return false
	}
	if suffix != "" && !strings.HasSuffix(path, suffix) {
		return false
	}
	trimmed := strings.TrimPrefix(path, prefix)
	trimmed = strings.TrimSuffix(trimmed, suffix)
	trimmed = strings.Trim(trimmed, "/")
	return trimmed != "" && !strings.Contains(trimmed, "/")
}

func pathParam(path string, prefix string, suffix string) (string, bool) {
	if !pathMatches(path, prefix, suffix) {
		return "", false
	}
	trimmed := strings.TrimPrefix(path, prefix)
	trimmed = strings.TrimSuffix(trimmed, suffix)
	return strings.Trim(trimmed, "/"), true
}

func pathObjectID(request *http.Request, prefix string, suffix string) (primitive.ObjectID, bool, error) {
	raw, ok := pathParam(request.URL.Path, prefix, suffix)
	if !ok {
		return primitive.NilObjectID, false, nil
	}
	id, err := primitive.ObjectIDFromHex(raw)
	if err != nil {
		return primitive.NilObjectID, true, badRequestf("invalid object id %q", raw)
	}
	return id, true, nil
}

func queryObjectID(request *http.Request, name string) (primitive.ObjectID, error) {
	raw := strings.TrimSpace(request.URL.Query().Get(name))
	if raw == "" {
		return primitive.NilObjectID, nil
	}
	id, err := primitive.ObjectIDFromHex(raw)
	if err != nil {
		return primitive.NilObjectID, badRequestf("invalid %s %q", name, raw)
	}
	return id, nil
}

func parsePagination(request *http.Request) (platformrepo.PageOptions, error) {
	limit, err := queryInt(request, "limit", defaultPageSize)
	if err != nil {
		return platformrepo.PageOptions{}, err
	}
	offset, err := queryInt(request, "offset", 0)
	if err != nil {
		return platformrepo.PageOptions{}, err
	}
	if limit <= 0 {
		limit = defaultPageSize
	}
	if limit > maxPageSize {
		limit = maxPageSize
	}
	return platformrepo.PageOptions{PageSize: limit, Offset: offset}, nil
}

func queryInt(request *http.Request, name string, fallback int) (int, error) {
	raw := strings.TrimSpace(request.URL.Query().Get(name))
	if raw == "" {
		return fallback, nil
	}
	value, err := strconv.Atoi(raw)
	if err != nil || value < 0 {
		return 0, badRequestf("%s must be a non-negative integer", name)
	}
	return value, nil
}

func queryBool(request *http.Request, name string) (bool, error) {
	raw := strings.TrimSpace(request.URL.Query().Get(name))
	if raw == "" {
		return false, nil
	}
	value, err := strconv.ParseBool(raw)
	if err != nil {
		return false, badRequestf("%s must be a boolean", name)
	}
	return value, nil
}

func parseTimeRange(request *http.Request, fromName string, toName string) (*platformrepo.TimeRange, error) {
	from, err := queryTime(request, fromName)
	if err != nil {
		return nil, err
	}
	to, err := queryTime(request, toName)
	if err != nil {
		return nil, err
	}
	if from == nil && to == nil {
		return nil, nil
	}
	if from != nil && to != nil && to.Before(*from) {
		return nil, badRequestf("%s cannot be before %s", toName, fromName)
	}
	return &platformrepo.TimeRange{From: from, To: to}, nil
}

func queryTime(request *http.Request, name string) (*time.Time, error) {
	raw := strings.TrimSpace(request.URL.Query().Get(name))
	if raw == "" {
		return nil, nil
	}
	for _, layout := range []string{time.RFC3339, "2006-01-02"} {
		parsed, err := time.Parse(layout, raw)
		if err == nil {
			utc := parsed.UTC()
			return &utc, nil
		}
	}
	return nil, badRequestf("%s must be RFC3339 or YYYY-MM-DD", name)
}

func queryBookType(request *http.Request) (domaincommon.BookType, error) {
	value := domaincommon.BookType(strings.TrimSpace(request.URL.Query().Get("book_type")))
	if value != "" && !value.IsValid() {
		return "", badRequestf("invalid book_type %q", value)
	}
	return value, nil
}

func queryWorkflowRunType(request *http.Request) (domaincommon.WorkflowRunType, error) {
	value := domaincommon.WorkflowRunType(strings.TrimSpace(request.URL.Query().Get("run_type")))
	if value != "" && !value.IsValid() {
		return "", badRequestf("invalid run_type %q", value)
	}
	return value, nil
}

func queryWorkflowRunStatus(request *http.Request) (domaincommon.WorkflowRunStatus, error) {
	value := domaincommon.WorkflowRunStatus(strings.TrimSpace(request.URL.Query().Get("status")))
	if value != "" && !value.IsValid() {
		return "", badRequestf("invalid status %q", value)
	}
	return value, nil
}

func queryWorkflowStepStatus(request *http.Request) (domaincommon.WorkflowStepStatus, error) {
	value := domaincommon.WorkflowStepStatus(strings.TrimSpace(request.URL.Query().Get("status")))
	if value != "" && !value.IsValid() {
		return "", badRequestf("invalid status %q", value)
	}
	return value, nil
}

func queryBatchJobType(request *http.Request) (domaincommon.AIBatchJobType, error) {
	value := domaincommon.AIBatchJobType(strings.TrimSpace(request.URL.Query().Get("job_type")))
	if value != "" && !value.IsValid() {
		return "", badRequestf("invalid job_type %q", value)
	}
	return value, nil
}

func queryBatchJobStatus(request *http.Request) (domaincommon.AIBatchJobStatus, error) {
	value := domaincommon.AIBatchJobStatus(strings.TrimSpace(request.URL.Query().Get("status")))
	if value != "" && !value.IsValid() {
		return "", badRequestf("invalid status %q", value)
	}
	return value, nil
}

func queryBatchItemType(request *http.Request) (domaincommon.AIBatchItemType, error) {
	value := domaincommon.AIBatchItemType(strings.TrimSpace(request.URL.Query().Get("item_type")))
	if value != "" && !value.IsValid() {
		return "", badRequestf("invalid item_type %q", value)
	}
	return value, nil
}

func queryBatchItemStatus(request *http.Request) (domaincommon.AIBatchItemStatus, error) {
	value := domaincommon.AIBatchItemStatus(strings.TrimSpace(request.URL.Query().Get("status")))
	if value != "" && !value.IsValid() {
		return "", badRequestf("invalid status %q", value)
	}
	return value, nil
}

func queryValidationStatus(request *http.Request) (domaincommon.ValidationStatus, error) {
	value := domaincommon.ValidationStatus(strings.TrimSpace(request.URL.Query().Get("validation_status")))
	if value != "" && !value.IsValid() {
		return "", badRequestf("invalid validation_status %q", value)
	}
	return value, nil
}

func oneObjectID(id primitive.ObjectID) []primitive.ObjectID {
	if id.IsZero() {
		return nil
	}
	return []primitive.ObjectID{id}
}
