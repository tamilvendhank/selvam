package aijob

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	domainaijob "goserver/internal/domain/aijob"
	domaincommon "goserver/internal/domain/common"
	platformrepo "goserver/internal/platform/repository"
	servicecommon "goserver/internal/service/common"
	workerservice "goserver/internal/service/worker"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

var errPollingSkipped = errors.New("polling skipped")

type pollingRequestOptions struct {
	WorkflowRunID         primitive.ObjectID
	BookType              domaincommon.BookType
	JobType               domaincommon.AIBatchJobType
	PollOnlyStatuses      []domaincommon.AIBatchJobStatus
	Force                 bool
	InitiatedBy           string
	CorrelationID         string
	TreatIneligibleAsSkip bool
}

func (options pollingRequestOptions) Bulk() bool {
	return options.TreatIneligibleAsSkip
}

type pollOneOutcome struct {
	JobRef        servicecommon.BatchJobRef
	Update        BatchStatusUpdate
	StatusChanged bool
	Skipped       bool
}

func (service *batchJobPollingService) maxJobs(requested int) int {
	if requested > 0 && requested < service.config.MaxPageSize {
		return requested
	}
	if requested > service.config.MaxPageSize {
		return service.config.MaxPageSize
	}
	if service.config.DefaultMaxJobs > service.config.MaxPageSize {
		return service.config.MaxPageSize
	}
	return service.config.DefaultMaxJobs
}

func (service *batchJobPollingService) discoverPollableJobIDs(
	ctx context.Context,
	request PollPendingBatchJobsRequest,
) ([]primitive.ObjectID, bool, error) {
	if !request.BatchJobID.IsZero() {
		return []primitive.ObjectID{request.BatchJobID}, false, nil
	}

	limit := service.maxJobs(request.MaxJobs)
	if service.discovery != nil && !request.Force {
		discovered, err := service.discovery.DiscoverPollableBatchJobs(ctx, workerservice.DiscoverPollableBatchJobsRequest{
			DiscoveryRequestBase: workerservice.DiscoveryRequestBase{
				WorkflowRunID: request.WorkflowRunID,
				BookType:      request.BookType,
				JobType:       request.JobType,
				MaxItems:      limit,
			},
			PollOnlyStatuses: request.PollOnlyStatuses,
		})
		if err != nil {
			return nil, false, fmt.Errorf("discover pollable batch jobs: %w", err)
		}
		return jobIDsFromRefs(discovered.BatchJobs, limit), discovered.HasMore, nil
	}

	if service.batchJobs == nil {
		return nil, false, fmt.Errorf("discover pollable batch jobs: batch job repository is required")
	}
	filter := platformrepo.AIBatchJobFilter{}
	if !request.WorkflowRunID.IsZero() {
		filter.WorkflowRunIDs = []primitive.ObjectID{request.WorkflowRunID}
	}
	if request.BookType != "" {
		filter.BookTypes = []domaincommon.BookType{request.BookType}
	}
	if request.JobType != "" {
		filter.JobTypes = []domaincommon.AIBatchJobType{request.JobType}
	}
	if len(request.PollOnlyStatuses) > 0 {
		filter.Statuses = request.PollOnlyStatuses
	}

	options := platformrepo.AIBatchJobListOptions{
		Pagination: platformrepo.PageOptions{PageSize: limit},
		Sort:       platformrepo.AIBatchJobSortOption{By: platformrepo.AIBatchJobSortByLastPolledAt, Order: platformrepo.SortOrderAscending},
	}
	var result *platformrepo.ListResult[*domainaijob.AIBatchJob]
	var err error
	if len(request.PollOnlyStatuses) > 0 {
		result, err = service.batchJobs.List(ctx, filter, options)
	} else {
		result, err = service.batchJobs.FindPollableJobs(ctx, filter, options)
	}
	if err != nil {
		return nil, false, fmt.Errorf("list pollable batch jobs: %w", err)
	}

	ids := make([]primitive.ObjectID, 0, len(result.Items))
	for _, job := range result.Items {
		if job != nil {
			ids = append(ids, job.ID)
		}
	}
	return ids, result.Page.HasMore, nil
}

func validatePollingEligibility(
	job *domainaijob.AIBatchJob,
	options pollingRequestOptions,
	now time.Time,
	minimumPollInterval time.Duration,
) error {
	if job == nil {
		return fmt.Errorf("batch job is required")
	}
	if !options.WorkflowRunID.IsZero() && job.WorkflowRunID != options.WorkflowRunID {
		return fmt.Errorf("%w: workflowRunId filter does not match", errPollingSkipped)
	}
	if options.BookType != "" && job.BookType != options.BookType {
		return fmt.Errorf("%w: bookType filter does not match", errPollingSkipped)
	}
	if options.JobType != "" && job.JobType != options.JobType {
		return fmt.Errorf("%w: jobType filter does not match", errPollingSkipped)
	}
	if len(options.PollOnlyStatuses) > 0 && !statusIn(job.Status, options.PollOnlyStatuses) {
		return fmt.Errorf("%w: status %q not allowed by request", errPollingSkipped, job.Status)
	}
	if !job.CanPoll() {
		return fmt.Errorf("%w: status %q is not pollable", errPollingSkipped, job.Status)
	}
	if strings.TrimSpace(providerPollingHandle(job)) == "" {
		return fmt.Errorf("provider job handle is required for polling")
	}
	if !options.Force && minimumPollInterval > 0 && job.LastPolledAt != nil {
		nextPollAt := job.LastPolledAt.UTC().Add(minimumPollInterval)
		if now.Before(nextPollAt) {
			return fmt.Errorf("%w: next poll allowed at %s", errPollingSkipped, nextPollAt.Format(time.RFC3339))
		}
	}
	return nil
}

func providerPollingHandle(job *domainaijob.AIBatchJob) string {
	if job == nil {
		return ""
	}
	if strings.TrimSpace(job.ProviderJobHandle) != "" {
		return strings.TrimSpace(job.ProviderJobHandle)
	}
	return strings.TrimSpace(job.LocalJobHandle)
}

func statusIn(status domaincommon.AIBatchJobStatus, statuses []domaincommon.AIBatchJobStatus) bool {
	for _, candidate := range statuses {
		if candidate == status {
			return true
		}
	}
	return false
}

func isPollingSkip(err error) bool {
	return errors.Is(err, errPollingSkipped)
}

func jobIDsFromRefs(refs []servicecommon.BatchJobRef, limit int) []primitive.ObjectID {
	ids := make([]primitive.ObjectID, 0, len(refs))
	for _, ref := range refs {
		if ref.ID.IsZero() {
			continue
		}
		ids = append(ids, ref.ID)
		if len(ids) >= limit {
			break
		}
	}
	return ids
}

func batchJobRef(job *domainaijob.AIBatchJob) servicecommon.BatchJobRef {
	if job == nil {
		return servicecommon.BatchJobRef{}
	}
	return servicecommon.BatchJobRef{
		ID:                job.ID,
		WorkflowRunID:     job.WorkflowRunID,
		BookType:          job.BookType,
		JobType:           job.JobType,
		Status:            job.Status,
		ProviderName:      job.ProviderName,
		ProviderJobHandle: job.ProviderJobHandle,
		LocalJobHandle:    job.LocalJobHandle,
		RetryCount:        job.RetryCount,
		MaxRetryCount:     job.MaxRetryCount,
		SubmittedAt:       job.SubmittedAt,
		LastPolledAt:      job.LastPolledAt,
		CompletedAt:       job.CompletedAt,
		FailedAt:          job.FailedAt,
	}
}

func batchStatusUpdate(
	before *domainaijob.AIBatchJob,
	after *domainaijob.AIBatchJob,
	mapped providerPollingOutcome,
) BatchStatusUpdate {
	update := BatchStatusUpdate{
		PolledAt:  &mapped.PolledAt,
		Retryable: mapped.Retryable,
	}
	if before != nil {
		update.BatchJobID = before.ID
		update.StatusBefore = before.Status
	}
	if after != nil {
		update.BatchJobID = after.ID
		update.StatusAfter = after.Status
	}
	return update
}

func appendPolledJobRef(
	completedJobs *[]servicecommon.BatchJobRef,
	stillRunningJobs *[]servicecommon.BatchJobRef,
	failedJobs *[]servicecommon.BatchJobRef,
	ref servicecommon.BatchJobRef,
) {
	switch ref.Status {
	case domaincommon.AIBatchJobStatusCompleted, domaincommon.AIBatchJobStatusPartiallyCompleted:
		*completedJobs = append(*completedJobs, ref)
	case domaincommon.AIBatchJobStatusFailed, domaincommon.AIBatchJobStatusTimedOut, domaincommon.AIBatchJobStatusCancelled:
		*failedJobs = append(*failedJobs, ref)
	default:
		*stillRunningJobs = append(*stillRunningJobs, ref)
	}
}

func buildPollingSummary(
	operation string,
	attempted int,
	polled int,
	failures int,
	statusChanges int,
) servicecommon.BatchPollingSummary {
	outcome := servicecommon.ServiceOutcomeSuccess
	message := fmt.Sprintf("polled %d batch job(s)", polled)
	switch {
	case attempted == 0:
		outcome = servicecommon.ServiceOutcomeNoop
		message = "no batch jobs to poll"
	case failures > 0 && polled > 0:
		outcome = servicecommon.ServiceOutcomePartial
		message = fmt.Sprintf("polled %d batch job(s) with %d failure(s)", polled, failures)
	case failures > 0:
		outcome = servicecommon.ServiceOutcomeFailed
		message = fmt.Sprintf("failed to poll %d batch job(s)", failures)
	}
	return servicecommon.BatchPollingSummary{
		OperationSummary: servicecommon.OperationSummary{
			Operation:      operation,
			Outcome:        outcome,
			AttemptedCount: attempted,
			SuccessCount:   polled,
			FailureCount:   failures,
			Message:        message,
		},
		PolledCount:       polled,
		StatusChangeCount: statusChanges,
	}
}

func pollingPartialFailure(jobID primitive.ObjectID, err error) servicecommon.PartialFailure {
	retryClass := servicecommon.RetryClassTransient
	retryable := true
	if errors.Is(err, platformrepo.ErrNotFound) || isPollingSkip(err) {
		retryClass = servicecommon.RetryClassNone
		retryable = false
	}
	return servicecommon.PartialFailure{
		Scope:      servicecommon.FailureScopeJob,
		ID:         jobID,
		BatchJobID: jobID,
		Code:       "batch_poll_failed",
		Message:    err.Error(),
		Retry: servicecommon.RetryPolicyHint{
			Retryable:  retryable,
			RetryClass: retryClass,
			Reason:     "polling failure",
		},
	}
}

func boolToInt(value bool) int {
	if value {
		return 1
	}
	return 0
}
