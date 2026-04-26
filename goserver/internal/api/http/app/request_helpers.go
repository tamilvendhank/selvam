package app

import (
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

func queryBoolPtr(request *http.Request, name string) (*bool, error) {
	raw := strings.TrimSpace(request.URL.Query().Get(name))
	if raw == "" {
		return nil, nil
	}
	value, err := strconv.ParseBool(raw)
	if err != nil {
		return nil, badRequestf("%s must be a boolean", name)
	}
	return &value, nil
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

func queryReviewStatus(request *http.Request) (domaincommon.ReviewStatus, error) {
	value := domaincommon.ReviewStatus(strings.TrimSpace(request.URL.Query().Get("review_status")))
	if value != "" && !value.IsValid() {
		return "", badRequestf("invalid review_status %q", value)
	}
	return value, nil
}

func queryLifecycleState(request *http.Request) (domaincommon.ReviewLifecycleState, error) {
	value := domaincommon.ReviewLifecycleState(strings.TrimSpace(request.URL.Query().Get("review_lifecycle_state")))
	if value != "" && !value.IsValid() {
		return "", badRequestf("invalid review_lifecycle_state %q", value)
	}
	return value, nil
}

func queryActionType(request *http.Request) (domaincommon.InvestingActionType, error) {
	value := domaincommon.InvestingActionType(strings.TrimSpace(request.URL.Query().Get("final_action_after_review")))
	if value != "" && !value.IsValid() {
		return "", badRequestf("invalid final_action_after_review %q", value)
	}
	return value, nil
}

func queryBucket(request *http.Request) (domaincommon.WatchlistBucket, error) {
	value := domaincommon.WatchlistBucket(strings.TrimSpace(request.URL.Query().Get("final_bucket_after_review")))
	if value != "" && !value.IsValid() {
		return "", badRequestf("invalid final_bucket_after_review %q", value)
	}
	return value, nil
}

func queryThesisStatus(request *http.Request) (domaincommon.ThesisStatus, error) {
	value := domaincommon.ThesisStatus(strings.TrimSpace(request.URL.Query().Get("thesis_status")))
	if value != "" && !value.IsValid() {
		return "", badRequestf("invalid thesis_status %q", value)
	}
	return value, nil
}

func oneObjectID(id primitive.ObjectID) []primitive.ObjectID {
	if id.IsZero() {
		return nil
	}
	return []primitive.ObjectID{id}
}

func oneBookType(value domaincommon.BookType) []domaincommon.BookType {
	if value == "" {
		return nil
	}
	return []domaincommon.BookType{value}
}

func oneReviewStatus(value domaincommon.ReviewStatus) []domaincommon.ReviewStatus {
	if value == "" {
		return nil
	}
	return []domaincommon.ReviewStatus{value}
}

func oneLifecycleState(value domaincommon.ReviewLifecycleState) []domaincommon.ReviewLifecycleState {
	if value == "" {
		return nil
	}
	return []domaincommon.ReviewLifecycleState{value}
}

func oneActionType(value domaincommon.InvestingActionType) []domaincommon.InvestingActionType {
	if value == "" {
		return nil
	}
	return []domaincommon.InvestingActionType{value}
}

func oneBucket(value domaincommon.WatchlistBucket) []domaincommon.WatchlistBucket {
	if value == "" {
		return nil
	}
	return []domaincommon.WatchlistBucket{value}
}

func oneThesisStatus(value domaincommon.ThesisStatus) []domaincommon.ThesisStatus {
	if value == "" {
		return nil
	}
	return []domaincommon.ThesisStatus{value}
}
