package validation

import (
	"encoding/json"
	"fmt"
	"math"
	"strconv"
	"strings"

	domainaijob "goserver/internal/domain/aijob"
	domaincommon "goserver/internal/domain/common"
)

func getValue(payload map[string]any, aliases ...string) (any, bool) {
	for _, alias := range aliases {
		if value, ok := payload[alias]; ok {
			return value, true
		}
	}
	return nil, false
}

func getObject(payload map[string]any, aliases ...string) (map[string]any, bool, bool) {
	value, ok := getValue(payload, aliases...)
	if !ok || value == nil {
		return nil, false, false
	}
	object, ok := value.(map[string]any)
	return object, true, ok
}

func getArray(payload map[string]any, aliases ...string) ([]any, bool, bool) {
	value, ok := getValue(payload, aliases...)
	if !ok || value == nil {
		return nil, false, false
	}
	switch typed := value.(type) {
	case []any:
		return typed, true, true
	case []map[string]any:
		items := make([]any, 0, len(typed))
		for _, item := range typed {
			items = append(items, item)
		}
		return items, true, true
	default:
		return nil, true, false
	}
}

func getString(payload map[string]any, aliases ...string) (string, bool, bool) {
	value, ok := getValue(payload, aliases...)
	if !ok || value == nil {
		return "", false, false
	}
	switch typed := value.(type) {
	case string:
		return strings.TrimSpace(typed), true, strings.TrimSpace(typed) != ""
	default:
		text := strings.TrimSpace(fmt.Sprint(typed))
		return text, true, text != ""
	}
}

func getFloat(payload map[string]any, aliases ...string) (float64, bool, bool) {
	value, ok := getValue(payload, aliases...)
	if !ok || value == nil {
		return 0, false, false
	}
	switch typed := value.(type) {
	case float64:
		return typed, true, !math.IsNaN(typed) && !math.IsInf(typed, 0)
	case float32:
		value := float64(typed)
		return value, true, !math.IsNaN(value) && !math.IsInf(value, 0)
	case int:
		return float64(typed), true, true
	case int64:
		return float64(typed), true, true
	case int32:
		return float64(typed), true, true
	case json.Number:
		value, err := typed.Float64()
		return value, true, err == nil
	case string:
		text := strings.TrimSpace(typed)
		if text == "" {
			return 0, true, false
		}
		value, err := strconv.ParseFloat(text, 64)
		return value, true, err == nil
	default:
		return 0, true, false
	}
}

func requiredString(report *outputValidationReport, payload map[string]any, path string, item *domainaijob.AIBatchItem, aliases ...string) string {
	value, present, ok := getString(payload, aliases...)
	if !present {
		report.Add(issueMissingField(path, item))
		return ""
	}
	if !ok {
		report.Add(issueTypeMismatch(path, "a non-empty string", item))
		return ""
	}
	return value
}

func optionalStringEnum(
	report *outputValidationReport,
	payload map[string]any,
	path string,
	item *domainaijob.AIBatchItem,
	allowed func(string) bool,
	aliases ...string,
) string {
	value, present, ok := getString(payload, aliases...)
	if !present || value == "" {
		return ""
	}
	if !ok || !allowed(value) {
		report.Add(issueInvalidEnum(path, value, item))
	}
	return value
}

func requiredFloatInRange(
	report *outputValidationReport,
	payload map[string]any,
	path string,
	min float64,
	max float64,
	item *domainaijob.AIBatchItem,
	aliases ...string,
) float64 {
	value, present, ok := getFloat(payload, aliases...)
	if !present {
		report.Add(issueMissingField(path, item))
		return 0
	}
	if !ok {
		report.Add(issueTypeMismatch(path, "a number", item))
		return 0
	}
	if value < min || value > max {
		report.Add(issueOutOfRange(path, min, max, item))
	}
	return value
}

func optionalFloatInRange(
	report *outputValidationReport,
	payload map[string]any,
	path string,
	min float64,
	max float64,
	item *domainaijob.AIBatchItem,
	aliases ...string,
) (float64, bool) {
	value, present, ok := getFloat(payload, aliases...)
	if !present {
		return 0, false
	}
	if !ok {
		report.Add(issueTypeMismatch(path, "a number", item))
		return 0, true
	}
	if value < min || value > max {
		report.Add(issueOutOfRange(path, min, max, item))
	}
	return value, true
}

func validationSeverityForStrict(strict bool) serviceSeverity {
	if strict {
		return serviceSeverityError
	}
	return serviceSeverityWarning
}

type serviceSeverity string

const (
	serviceSeverityError   serviceSeverity = "error"
	serviceSeverityWarning serviceSeverity = "warning"
)

func addStrictnessIssue(report *outputValidationReport, strict bool, code string, path string, message string, item *domainaijob.AIBatchItem) {
	if validationSeverityForStrict(strict) == serviceSeverityError {
		report.Add(issueError(code, path, message, item))
		return
	}
	report.Add(issueWarning(code, path, message, item))
}

func isValidAction(value string) bool {
	return domaincommon.InvestingActionType(strings.TrimSpace(value)).IsValid()
}

func isValidBucket(value string) bool {
	return domaincommon.WatchlistBucket(strings.TrimSpace(value)).IsValid()
}

func isValidTrendDirection(value string) bool {
	return domaincommon.TrendDirection(strings.TrimSpace(value)).IsValid()
}

func isValidMetricBasis(value string) bool {
	return domaincommon.MetricBasis(strings.TrimSpace(value)).IsValid()
}

func isValidEvidenceStrength(value string) bool {
	return domaincommon.EvidenceStrength(strings.TrimSpace(value)).IsValid()
}

func isValidEvidenceDirection(value string) bool {
	return domaincommon.EvidenceDirection(strings.TrimSpace(value)).IsValid()
}

func isValidEvidenceSourceType(value string) bool {
	return domaincommon.EvidenceSourceType(strings.TrimSpace(value)).IsValid()
}

func isValidSectionActionCap(value string) bool {
	return domaincommon.SectionActionCap(strings.TrimSpace(value)).IsValid()
}
