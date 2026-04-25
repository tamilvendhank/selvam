package aijob

import (
	"fmt"
	"strings"
	"time"

	domainaijob "goserver/internal/domain/aijob"
	domaincommon "goserver/internal/domain/common"
	platformdomain "goserver/internal/platform/domain"
	platformports "goserver/internal/platform/ports"
)

type providerPollingOutcome struct {
	StatusAfter       domaincommon.AIBatchJobStatus
	ProviderStatus    platformdomain.BatchJobStatus
	PolledAt          time.Time
	CompletedAt       time.Time
	FailedAt          time.Time
	ErrorSummary      *string
	Retryable         bool
	ClearErrorSummary bool
}

func (outcome providerPollingOutcome) InFlightStatusPatch(current domaincommon.AIBatchJobStatus) *domaincommon.AIBatchJobStatus {
	if outcome.StatusAfter == current || isTerminalBatchJobStatus(outcome.StatusAfter) {
		return nil
	}
	next := outcome.StatusAfter
	return &next
}

func mapProviderStatus(
	job *domainaijob.AIBatchJob,
	providerStatus *platformports.BatchStatusResult,
	now time.Time,
) (providerPollingOutcome, error) {
	if job == nil {
		return providerPollingOutcome{}, fmt.Errorf("batch job is required")
	}
	if providerStatus == nil {
		return providerPollingOutcome{}, fmt.Errorf("provider returned nil batch status")
	}
	if !platformdomain.IsValidBatchJobStatus(providerStatus.Status) {
		return providerPollingOutcome{}, fmt.Errorf("provider returned unknown batch status %q", providerStatus.Status)
	}

	polledAt := now.UTC()
	if providerStatus.LastPolledAt != nil && !providerStatus.LastPolledAt.IsZero() {
		polledAt = providerStatus.LastPolledAt.UTC()
	}

	statusAfter := mapPlatformBatchJobStatus(providerStatus.Status)
	if providerStatus.ResultAvailable &&
		(statusAfter == domaincommon.AIBatchJobStatusSubmitted || statusAfter == domaincommon.AIBatchJobStatusRunning) &&
		(providerStatus.ItemsCompletedCount > 0 || providerStatus.ItemsFailedCount > 0) {
		statusAfter = domaincommon.AIBatchJobStatusPartiallyCompleted
	}
	statusAfter = preventPollingStatusRegression(job.Status, statusAfter)

	outcome := providerPollingOutcome{
		StatusAfter:       statusAfter,
		ProviderStatus:    providerStatus.Status,
		PolledAt:          polledAt,
		Retryable:         providerStatus.Retryable,
		ClearErrorSummary: !isProviderFailureStatus(statusAfter),
	}
	if providerStatus.CompletedAt != nil && !providerStatus.CompletedAt.IsZero() {
		outcome.CompletedAt = providerStatus.CompletedAt.UTC()
	}
	if isProviderFailureStatus(statusAfter) {
		outcome.FailedAt = polledAt
		summary := providerErrorSummary(providerStatus)
		outcome.ErrorSummary = &summary
		outcome.ClearErrorSummary = false
	}
	return outcome, nil
}

func mapPlatformBatchJobStatus(status platformdomain.BatchJobStatus) domaincommon.AIBatchJobStatus {
	switch status {
	case platformdomain.BatchJobStatusSubmitted:
		return domaincommon.AIBatchJobStatusSubmitted
	case platformdomain.BatchJobStatusRunning:
		return domaincommon.AIBatchJobStatusRunning
	case platformdomain.BatchJobStatusPartiallyCompleted:
		return domaincommon.AIBatchJobStatusPartiallyCompleted
	case platformdomain.BatchJobStatusCompleted:
		return domaincommon.AIBatchJobStatusCompleted
	case platformdomain.BatchJobStatusFailed:
		return domaincommon.AIBatchJobStatusFailed
	case platformdomain.BatchJobStatusCancelled:
		return domaincommon.AIBatchJobStatusCancelled
	case platformdomain.BatchJobStatusTimedOut:
		return domaincommon.AIBatchJobStatusTimedOut
	default:
		return domaincommon.AIBatchJobStatusCreated
	}
}

func preventPollingStatusRegression(
	current domaincommon.AIBatchJobStatus,
	next domaincommon.AIBatchJobStatus,
) domaincommon.AIBatchJobStatus {
	if isTerminalBatchJobStatus(current) || isTerminalBatchJobStatus(next) {
		return next
	}
	if batchStatusRank(next) < batchStatusRank(current) {
		return current
	}
	return next
}

func batchStatusRank(status domaincommon.AIBatchJobStatus) int {
	switch status {
	case domaincommon.AIBatchJobStatusCreated:
		return 0
	case domaincommon.AIBatchJobStatusSubmitted:
		return 1
	case domaincommon.AIBatchJobStatusRunning:
		return 2
	case domaincommon.AIBatchJobStatusPartiallyCompleted:
		return 3
	case domaincommon.AIBatchJobStatusCompleted:
		return 4
	case domaincommon.AIBatchJobStatusFailed, domaincommon.AIBatchJobStatusTimedOut, domaincommon.AIBatchJobStatusCancelled:
		return 5
	default:
		return 0
	}
}

func isTerminalBatchJobStatus(status domaincommon.AIBatchJobStatus) bool {
	switch status {
	case domaincommon.AIBatchJobStatusCompleted,
		domaincommon.AIBatchJobStatusFailed,
		domaincommon.AIBatchJobStatusTimedOut,
		domaincommon.AIBatchJobStatusCancelled:
		return true
	default:
		return false
	}
}

func isProviderFailureStatus(status domaincommon.AIBatchJobStatus) bool {
	switch status {
	case domaincommon.AIBatchJobStatusFailed,
		domaincommon.AIBatchJobStatusTimedOut,
		domaincommon.AIBatchJobStatusCancelled:
		return true
	default:
		return false
	}
}

func providerErrorSummary(status *platformports.BatchStatusResult) string {
	if status == nil {
		return "provider status unavailable"
	}
	for _, key := range []string{"error", "message", "errorSummary", "statusReason"} {
		if value, ok := status.RawProviderStatus[key]; ok {
			text := strings.TrimSpace(fmt.Sprint(value))
			if text != "" {
				return text
			}
		}
	}
	return fmt.Sprintf("provider reported batch status %q", status.Status)
}
