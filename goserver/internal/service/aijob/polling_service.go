package aijob

import (
	"context"
	"fmt"
	"time"

	domainaijob "goserver/internal/domain/aijob"
	domaincommon "goserver/internal/domain/common"
	platformports "goserver/internal/platform/ports"
	platformrepo "goserver/internal/platform/repository"
	servicecommon "goserver/internal/service/common"
	workerservice "goserver/internal/service/worker"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

const (
	defaultPollingMaxJobs = 50
	maxPollingPageSize    = 500
)

type BatchJobPollingConfig struct {
	DefaultMaxJobs      int
	MaxPageSize         int
	MinimumPollInterval time.Duration
}

type BatchJobPollingOption func(*batchJobPollingService)

func WithBatchJobPollingConfig(config BatchJobPollingConfig) BatchJobPollingOption {
	return func(service *batchJobPollingService) {
		if config.DefaultMaxJobs > 0 {
			service.config.DefaultMaxJobs = config.DefaultMaxJobs
		}
		if config.MaxPageSize > 0 {
			service.config.MaxPageSize = config.MaxPageSize
		}
		if config.MinimumPollInterval > 0 {
			service.config.MinimumPollInterval = config.MinimumPollInterval
		}
	}
}

func WithBatchJobPollingClock(clock servicecommon.ClockPort) BatchJobPollingOption {
	return func(service *batchJobPollingService) {
		if clock != nil {
			service.now = clock.Now
		}
	}
}

type batchJobPollingService struct {
	batchJobs platformrepo.AIBatchJobRepository
	discovery workerservice.WorkerWorkDiscoveryService
	engine    platformports.AIBatchEngine
	config    BatchJobPollingConfig
	now       func() time.Time
}

var _ BatchJobPollingService = (*batchJobPollingService)(nil)

func NewBatchJobPollingService(
	batchJobs platformrepo.AIBatchJobRepository,
	discovery workerservice.WorkerWorkDiscoveryService,
	engine platformports.AIBatchEngine,
	options ...BatchJobPollingOption,
) BatchJobPollingService {
	service := &batchJobPollingService{
		batchJobs: batchJobs,
		discovery: discovery,
		engine:    engine,
		config: BatchJobPollingConfig{
			DefaultMaxJobs: defaultPollingMaxJobs,
			MaxPageSize:    maxPollingPageSize,
		},
		now: time.Now,
	}
	for _, option := range options {
		if option != nil {
			option(service)
		}
	}
	if service.config.DefaultMaxJobs <= 0 {
		service.config.DefaultMaxJobs = defaultPollingMaxJobs
	}
	if service.config.MaxPageSize <= 0 {
		service.config.MaxPageSize = maxPollingPageSize
	}
	return service
}

func (service *batchJobPollingService) PollBatchJob(
	ctx context.Context,
	request PollBatchJobRequest,
) (*PollBatchJobResult, error) {
	if err := request.Validate(); err != nil {
		return nil, err
	}
	outcome, err := service.pollOneJob(ctx, request.BatchJobID, pollingRequestOptions{
		WorkflowRunID:    request.WorkflowRunID,
		PollOnlyStatuses: request.PollOnlyStatuses,
		Force:            request.Force,
		InitiatedBy:      request.InitiatedBy,
		CorrelationID:    request.CorrelationID,
	})
	if err != nil {
		return nil, err
	}
	if outcome.Skipped {
		return nil, fmt.Errorf("%w: batch job %s is not pollable", servicecommon.ErrNothingToPoll, request.BatchJobID.Hex())
	}

	result := PollBatchJobResult{
		BatchJobID:      request.BatchJobID,
		PolledJobIDs:    []primitive.ObjectID{request.BatchJobID},
		UpdatedStatuses: []BatchStatusUpdate{outcome.Update},
		Summary:         buildPollingSummary("poll_batch_job", 1, 1, 0, boolToInt(outcome.StatusChanged)),
	}
	appendPolledJobRef(&result.CompletedJobs, &result.StillRunningJobs, &result.FailedJobs, outcome.JobRef)
	return &result, nil
}

func (service *batchJobPollingService) PollPendingBatchJobs(
	ctx context.Context,
	request PollPendingBatchJobsRequest,
) (*PollPendingBatchJobsResult, error) {
	if err := request.Validate(); err != nil {
		return nil, err
	}
	jobIDs, hasMore, err := service.discoverPollableJobIDs(ctx, request)
	if err != nil {
		return nil, err
	}
	if len(jobIDs) == 0 {
		return &PollPendingBatchJobsResult{
			Summary: buildPollingSummary("poll_pending_batch_jobs", 0, 0, 0, 0),
		}, nil
	}

	result := PollPendingBatchJobsResult{}
	statusChanges := 0
	skipped := 0
	for _, jobID := range jobIDs {
		outcome, err := service.pollOneJob(ctx, jobID, pollingRequestOptions{
			WorkflowRunID:         request.WorkflowRunID,
			BookType:              request.BookType,
			JobType:               request.JobType,
			PollOnlyStatuses:      request.PollOnlyStatuses,
			Force:                 request.Force,
			InitiatedBy:           request.InitiatedBy,
			CorrelationID:         request.CorrelationID,
			TreatIneligibleAsSkip: true,
		})
		if err != nil {
			result.PartialFailures = append(result.PartialFailures, pollingPartialFailure(jobID, err))
			continue
		}
		if outcome.Skipped {
			skipped++
			continue
		}
		if outcome.StatusChanged {
			statusChanges++
		}
		result.PolledJobIDs = append(result.PolledJobIDs, jobID)
		result.UpdatedStatuses = append(result.UpdatedStatuses, outcome.Update)
		appendPolledJobRef(&result.CompletedJobs, &result.StillRunningJobs, &result.FailedJobs, outcome.JobRef)
	}

	result.Summary = buildPollingSummary("poll_pending_batch_jobs", len(jobIDs), len(result.PolledJobIDs), len(result.PartialFailures), statusChanges)
	result.Summary.SkippedCount = skipped
	if hasMore {
		result.Summary.Message = fmt.Sprintf("%s; more pollable jobs may be available", result.Summary.Message)
	}
	return &result, nil
}

func (service *batchJobPollingService) pollOneJob(
	ctx context.Context,
	jobID primitive.ObjectID,
	options pollingRequestOptions,
) (pollOneOutcome, error) {
	if service.batchJobs == nil {
		return pollOneOutcome{}, fmt.Errorf("poll batch job %s: batch job repository is required", jobID.Hex())
	}
	if service.engine == nil {
		return pollOneOutcome{}, fmt.Errorf("poll batch job %s: batch engine is required", jobID.Hex())
	}

	job, err := service.batchJobs.GetByID(ctx, jobID)
	if err != nil {
		return pollOneOutcome{}, fmt.Errorf("poll batch job %s: load job: %w", jobID.Hex(), err)
	}
	if job == nil {
		return pollOneOutcome{}, fmt.Errorf("poll batch job %s: %w", jobID.Hex(), platformrepo.ErrNotFound)
	}
	if err := validatePollingEligibility(job, options, service.now().UTC(), service.config.MinimumPollInterval); err != nil {
		if options.Bulk() && isPollingSkip(err) {
			return pollOneOutcome{Skipped: true}, nil
		}
		return pollOneOutcome{}, err
	}

	handle := providerPollingHandle(job)
	providerStatus, err := service.engine.GetBatchStatus(ctx, handle)
	if err != nil {
		return pollOneOutcome{}, fmt.Errorf("poll batch job %s provider handle %q: %w", job.ID.Hex(), handle, err)
	}

	mapped, err := mapProviderStatus(job, providerStatus, service.now().UTC())
	if err != nil {
		return pollOneOutcome{}, fmt.Errorf("poll batch job %s map provider status: %w", job.ID.Hex(), err)
	}

	updated, err := service.applyPollingUpdate(ctx, job, mapped, options)
	if err != nil {
		return pollOneOutcome{}, fmt.Errorf("poll batch job %s persist status %q -> %q: %w", job.ID.Hex(), job.Status, mapped.StatusAfter, err)
	}

	return pollOneOutcome{
		JobRef:        batchJobRef(updated),
		Update:        batchStatusUpdate(job, updated, mapped),
		StatusChanged: job.Status != updated.Status,
	}, nil
}

func (service *batchJobPollingService) applyPollingUpdate(
	ctx context.Context,
	job *domainaijob.AIBatchJob,
	mapped providerPollingOutcome,
	options pollingRequestOptions,
) (*domainaijob.AIBatchJob, error) {
	errorSummary := mapped.ErrorSummary
	clearErrorSummary := ""
	if errorSummary == nil && mapped.ClearErrorSummary {
		errorSummary = &clearErrorSummary
	}

	// Repository preconditions are the safety net for concurrent workers:
	// discovery/poll eligibility is advisory, while writes are guarded by current status.
	polled, err := service.batchJobs.MarkPolled(ctx, job.ID, platformrepo.AIBatchJobPollingPatch{
		LastPolledAt: mapped.PolledAt,
		NextStatus:   mapped.InFlightStatusPatch(job.Status),
		ErrorSummary: errorSummary,
		ExpectedCurrentStatuses: []domaincommon.AIBatchJobStatus{
			job.Status,
		},
		Mutation: platformrepo.MutationMetadata{
			OccurredAt: mapped.PolledAt,
			Actor:      options.InitiatedBy,
			Reason:     "batch status poll",
		},
	})
	if err != nil {
		return nil, err
	}

	switch mapped.StatusAfter {
	case domaincommon.AIBatchJobStatusCompleted:
		completedAt := mapped.CompletedAt
		if completedAt.IsZero() {
			completedAt = mapped.PolledAt
		}
		return service.batchJobs.MarkCompleted(ctx, job.ID, platformrepo.AIBatchJobCompletionPatch{
			CompletedAt: completedAt,
			ExpectedCurrentStatuses: []domaincommon.AIBatchJobStatus{
				polled.Status,
			},
			Mutation: platformrepo.MutationMetadata{
				OccurredAt: completedAt,
				Actor:      options.InitiatedBy,
				Reason:     "provider reported batch completion",
			},
		})
	case domaincommon.AIBatchJobStatusFailed:
		return service.batchJobs.MarkFailed(ctx, job.ID, terminalFailurePatch(polled.Status, mapped, options, "provider reported batch failure"))
	case domaincommon.AIBatchJobStatusTimedOut:
		return service.batchJobs.MarkTimedOut(ctx, job.ID, terminalFailurePatch(polled.Status, mapped, options, "provider reported batch timeout"))
	case domaincommon.AIBatchJobStatusCancelled:
		return service.batchJobs.UpdateStatus(ctx, job.ID, platformrepo.AIBatchJobStatusPatch{
			NextStatus:   domaincommon.AIBatchJobStatusCancelled,
			ErrorSummary: mapped.ErrorSummary,
			ExpectedCurrentStatuses: []domaincommon.AIBatchJobStatus{
				polled.Status,
			},
			Mutation: platformrepo.MutationMetadata{
				OccurredAt: mapped.PolledAt,
				Actor:      options.InitiatedBy,
				Reason:     "provider reported batch cancellation",
			},
		})
	default:
		return polled, nil
	}
}

func terminalFailurePatch(
	currentStatus domaincommon.AIBatchJobStatus,
	mapped providerPollingOutcome,
	options pollingRequestOptions,
	reason string,
) platformrepo.AIBatchJobFailurePatch {
	failedAt := mapped.FailedAt
	if failedAt.IsZero() {
		failedAt = mapped.PolledAt
	}
	errorSummary := "provider reported batch failure"
	if mapped.ErrorSummary != nil && *mapped.ErrorSummary != "" {
		errorSummary = *mapped.ErrorSummary
	}
	return platformrepo.AIBatchJobFailurePatch{
		FailedAt:     failedAt,
		ErrorSummary: errorSummary,
		ExpectedCurrentStatuses: []domaincommon.AIBatchJobStatus{
			currentStatus,
		},
		Mutation: platformrepo.MutationMetadata{
			OccurredAt: failedAt,
			Actor:      options.InitiatedBy,
			Reason:     reason,
		},
	}
}
