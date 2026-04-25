package validation

import (
	"fmt"
	"strings"

	domainaijob "goserver/internal/domain/aijob"
	servicecommon "goserver/internal/service/common"
)

type outputValidationReport struct {
	Issues []servicecommon.ValidationIssue
}

func (report *outputValidationReport) Add(issue servicecommon.ValidationIssue) {
	if strings.TrimSpace(issue.Message) == "" {
		return
	}
	report.Issues = append(report.Issues, issue)
}

func (report *outputValidationReport) Merge(other outputValidationReport) {
	report.Issues = append(report.Issues, other.Issues...)
}

func (report outputValidationReport) IsValid() bool {
	return report.ErrorCount() == 0
}

func (report outputValidationReport) ErrorCount() int {
	count := 0
	for _, issue := range report.Issues {
		if issue.Severity == servicecommon.ValidationIssueSeverityError {
			count++
		}
	}
	return count
}

func (report outputValidationReport) WarningCount() int {
	count := 0
	for _, issue := range report.Issues {
		if issue.Severity == servicecommon.ValidationIssueSeverityWarning {
			count++
		}
	}
	return count
}

func (report outputValidationReport) ErrorMessages() []string {
	errors := make([]string, 0, report.ErrorCount())
	for _, issue := range report.Issues {
		if issue.Severity != servicecommon.ValidationIssueSeverityError {
			continue
		}
		if issue.FieldPath != "" {
			errors = append(errors, fmt.Sprintf("%s: %s", issue.FieldPath, issue.Message))
		} else {
			errors = append(errors, issue.Message)
		}
	}
	return errors
}

func (report outputValidationReport) FieldErrors() []servicecommon.FieldError {
	fieldErrors := make([]servicecommon.FieldError, 0, report.ErrorCount())
	for _, issue := range report.Issues {
		if issue.Severity != servicecommon.ValidationIssueSeverityError {
			continue
		}
		fieldErrors = append(fieldErrors, servicecommon.FieldError{
			FieldPath: issue.FieldPath,
			Code:      issue.Code,
			Message:   issue.Message,
		})
	}
	return fieldErrors
}

func issueError(code, fieldPath, message string, item *domainaijob.AIBatchItem) servicecommon.ValidationIssue {
	return issue(servicecommon.ValidationIssueSeverityError, code, fieldPath, message, item)
}

func issueWarning(code, fieldPath, message string, item *domainaijob.AIBatchItem) servicecommon.ValidationIssue {
	return issue(servicecommon.ValidationIssueSeverityWarning, code, fieldPath, message, item)
}

func issue(
	severity servicecommon.ValidationIssueSeverity,
	code string,
	fieldPath string,
	message string,
	item *domainaijob.AIBatchItem,
) servicecommon.ValidationIssue {
	validationIssue := servicecommon.ValidationIssue{
		Severity:  severity,
		Code:      code,
		Message:   message,
		FieldPath: fieldPath,
	}
	if item != nil {
		validationIssue.BatchItemID = item.ID
		validationIssue.ReviewID = item.TargetReviewID
		validationIssue.CompanyID = item.CompanyID
	}
	return validationIssue
}

func issueMissingField(fieldPath string, item *domainaijob.AIBatchItem) servicecommon.ValidationIssue {
	return issueError("missing_field", fieldPath, "field is required", item)
}

func issueTypeMismatch(fieldPath string, expected string, item *domainaijob.AIBatchItem) servicecommon.ValidationIssue {
	return issueError("type_mismatch", fieldPath, fmt.Sprintf("field must be %s", expected), item)
}

func issueInvalidEnum(fieldPath string, value string, item *domainaijob.AIBatchItem) servicecommon.ValidationIssue {
	return issueError("invalid_enum", fieldPath, fmt.Sprintf("unsupported value %q", value), item)
}

func issueOutOfRange(fieldPath string, min float64, max float64, item *domainaijob.AIBatchItem) servicecommon.ValidationIssue {
	return issueError("out_of_range", fieldPath, fmt.Sprintf("field must be between %.4g and %.4g", min, max), item)
}

func issueDuplicate(fieldPath string, value string, item *domainaijob.AIBatchItem) servicecommon.ValidationIssue {
	return issueError("duplicate_value", fieldPath, fmt.Sprintf("duplicate value %q", value), item)
}

func issueUnknownSubScore(fieldPath string, value string, item *domainaijob.AIBatchItem, strict bool) servicecommon.ValidationIssue {
	if strict {
		return issueError("unknown_sub_score", fieldPath, fmt.Sprintf("unknown sub-score %q", value), item)
	}
	return issueWarning("unknown_sub_score", fieldPath, fmt.Sprintf("unknown sub-score %q", value), item)
}

func issueUnknownSection(fieldPath string, value string, item *domainaijob.AIBatchItem, strict bool) servicecommon.ValidationIssue {
	if strict {
		return issueError("unknown_section", fieldPath, fmt.Sprintf("unknown section %q", value), item)
	}
	return issueWarning("unknown_section", fieldPath, fmt.Sprintf("unknown section %q", value), item)
}
