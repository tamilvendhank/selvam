package aijob

import (
	"context"
	"fmt"
	"time"

	domainaijob "goserver/internal/domain/aijob"
	platformports "goserver/internal/platform/ports"
	platformrepo "goserver/internal/platform/repository"
	servicecommon "goserver/internal/service/common"
	workerservice "goserver/internal/service/worker"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

const (
	defaultReconciliationMaxJobs = 25
	maxReconciliationPageSize    = 500
)

type BatchReconciliationConfig struct {
	DefaultMaxJobs int
	MaxPageSize    int
}

type BatchReconciliationOption func(*batchReconciliationService)

func WithBatchReconciliationConfig(config BatchReconciliationConfig) BatchReconciliationOption {
	return func(service *batchReconciliationService) {
		if config.DefaultMaxJobs > 0 {
			service.config.DefaultMaxJobs = config.DefaultMaxJobs
		}
		if config.MaxPageSize > 0 {
			service.config.MaxPageSize = config.MaxPageSize
		}
	}
}

func WithBatchReconciliationClock(clock servicecommon.ClockPort) BatchReconciliationOption {
	return func(service *batchReconciliationService) {
		if clock != nil {
			service.now = clock.Now
		}
	}
}

type batchReconciliationService struct {
	batchJobs  platformrepo.AIBatchJobRepository
	batchItems platformrepo.AIBatchItemRepository
	discovery  workerservice.WorkerWorkDiscoveryService
	engine     platformports.AIBatchEngine
	config     BatchReconciliationConfig
	now        func() time.Time
}

var _ BatchReconciliationService = (*batchReconciliationService)(nil)

func NewBatchReconciliationService(
	batchJobs platformrepo.AIBatchJobRepository,
	batchItems platformrepo.AIBatchItemRepository,
	discovery workerservice.WorkerWorkDiscoveryService,
	engine platformports.AIBatchEngine,
	options ...BatchReconciliationOption,
) BatchReconciliationService {
	service := &batchReconciliationService{
		batchJobs:  batchJobs,
		batchItems: batchItems,
		discovery:  discovery,
		engine:     engine,
		config: BatchReconciliationConfig{
			DefaultMaxJobs: defaultReconciliationMaxJobs,
			MaxPageSize:    maxReconciliationPageSize,
		},
		now: time.Now,
	}
	for _, option := range options {
		if option != nil {
			option(service)
		}
	}
	if service.config.DefaultMaxJobs <= 0 {
		service.config.DefaultMaxJobs = defaultReconciliationMaxJobs
	}
	if service.config.MaxPageSize <= 0 {
		service.config.MaxPageSize = maxReconciliationPageSize
	}
	return service
}

func (service *batchReconciliationService) ReconcileBatchJob(
	ctx context.Context,
	request ReconcileBatchJobRequest,
) (*ReconcileBatchJobResult, error) {
	if err := request.Validate(); err != nil {
		return nil, err
	}
	outcome, err := service.reconcileOneJob(ctx, request.BatchJobID, reconciliationRequestOptions{
		WorkflowRunID:         request.WorkflowRunID,
		Force:                 request.Force,
		IncludeCompletedItems: request.IncludeCompletedItems,
		InitiatedBy:           request.InitiatedBy,
		CorrelationID:         request.CorrelationID,
	})
	if err != nil {
		return nil, err
	}
	if outcome.Skipped {
		return nil, fmt.Errorf("%w: batch job %s is not reconciliable", servicecommon.ErrNothingToReconcile, request.BatchJobID.Hex())
	}
	return buildSingleReconciliationResult(outcome), nil
}

func (service *batchReconciliationService) ReconcilePendingBatchJobs(
	ctx context.Context,
	request ReconcilePendingBatchJobsRequest,
) (*ReconcilePendingBatchJobsResult, error) {
	if err := request.Validate(); err != nil {
		return nil, err
	}
	jobIDs, err := service.discoverReconciliableJobIDs(ctx, request)
	if err != nil {
		return nil, err
	}
	if len(jobIDs) == 0 {
		return &ReconcilePendingBatchJobsResult{
			Summary: buildReconciliationSummary("reconcile_pending_batch_jobs", 0, 0, 0, 0, 0, 0, 0),
		}, nil
	}

	result := ReconcilePendingBatchJobsResult{}
	skipped := 0
	for _, jobID := range jobIDs {
		outcome, err := service.reconcileOneJob(ctx, jobID, reconciliationRequestOptions{
			WorkflowRunID:         request.WorkflowRunID,
			BookType:              request.BookType,
			JobType:               request.JobType,
			Force:                 request.Force,
			IncludeCompletedItems: request.IncludeCompletedItems,
			InitiatedBy:           request.InitiatedBy,
			CorrelationID:         request.CorrelationID,
			TreatIneligibleAsSkip: true,
		})
		if err != nil {
			result.PartialFailures = append(result.PartialFailures, reconciliationPartialFailure(jobID, err))
			continue
		}
		if outcome.Skipped {
			skipped++
			continue
		}
		mergeReconciliationOutcome(&result, outcome)
	}

	result.Summary = buildReconciliationSummary(
		"reconcile_pending_batch_jobs",
		len(jobIDs),
		len(result.ReconciledJobIDs),
		len(result.PartialFailures),
		result.ItemsCompleted,
		result.ItemsFailed,
		result.ItemsInvalid,
		result.ItemsStillPending,
	)
	result.Summary.SkippedCount = skipped
	return &result, nil
}

func (service *batchReconciliationService) reconcileOneJob(
	ctx context.Context,
	jobID primitive.ObjectID,
	options reconciliationRequestOptions,
) (reconcileOneOutcome, error) {
	context, err := service.loadReconciliationContext(ctx, jobID)
	if err != nil {
		return reconcileOneOutcome{}, err
	}
	if err := validateReconciliationEligibility(context.Job, context.Items, options); err != nil {
		if options.TreatIneligibleAsSkip && isReconciliationSkip(err) {
			return reconcileOneOutcome{Skipped: true}, nil
		}
		return reconcileOneOutcome{}, err
	}

	results, err := service.fetchProviderResults(ctx, context.Job)
	if err != nil {
		return reconcileOneOutcome{}, err
	}
	correlation := correlateProviderResults(context.Items, results.Items)
	outcome := reconcileOneOutcome{
		BatchJobID:         context.Job.ID,
		WorkflowRunID:      context.Job.WorkflowRunID,
		ProviderStatus:     results.Status,
		UnmatchedResults:   correlation.UnmatchedResults,
		DuplicateResults:   correlation.DuplicateResults,
		MissingItemResults: correlation.MissingItems,
	}

	for _, unmatched := range correlation.UnmatchedResults {
		outcome.PartialFailures = append(outcome.PartialFailures, providerResultPartialFailure(context.Job.ID, unmatched, "provider_result_unmatched", "provider returned a result that could not be matched to a persisted batch item"))
	}
	for _, duplicate := range correlation.DuplicateResults {
		outcome.PartialFailures = append(outcome.PartialFailures, providerResultPartialFailure(context.Job.ID, duplicate, "provider_result_duplicate", "provider returned duplicate results for the same batch item"))
	}

	for _, match := range correlation.Matched {
		itemOutcome, err := service.applyItemReconciliation(ctx, match.Item, match.Result, options, results.CompletedAt)
		if err != nil {
			outcome.PartialFailures = append(outcome.PartialFailures, itemReconciliationPartialFailure(context.Job.ID, match.Item.ID, err))
			continue
		}
		mergeItemOutcome(&outcome, itemOutcome)
	}
	outcome.ItemsStillPending += len(correlation.MissingItems)

	if shouldSignalWorkflowAfterReconciliation(outcome) {
		outcome.ReadyWorkflowRunIDs = append(outcome.ReadyWorkflowRunIDs, context.Job.WorkflowRunID)
	}
	if err := service.applyJobReconciliationSummary(ctx, context.Job, results, options); err != nil {
		outcome.PartialFailures = append(outcome.PartialFailures, reconciliationPartialFailure(context.Job.ID, err))
	}
	return outcome, nil
}

func (service *batchReconciliationService) loadReconciliationContext(
	ctx context.Context,
	jobID primitive.ObjectID,
) (reconciliationContext, error) {
	if service.batchJobs == nil {
		return reconciliationContext{}, fmt.Errorf("reconcile batch job %s: batch job repository is required", jobID.Hex())
	}
	if service.batchItems == nil {
		return reconciliationContext{}, fmt.Errorf("reconcile batch job %s: batch item repository is required", jobID.Hex())
	}
	job, err := service.batchJobs.GetByID(ctx, jobID)
	if err != nil {
		return reconciliationContext{}, fmt.Errorf("reconcile batch job %s: load job: %w", jobID.Hex(), err)
	}
	if job == nil {
		return reconciliationContext{}, fmt.Errorf("reconcile batch job %s: %w", jobID.Hex(), platformrepo.ErrNotFound)
	}
	items, err := service.loadBatchItems(ctx, job.ID)
	if err != nil {
		return reconciliationContext{}, err
	}
	return reconciliationContext{Job: job, Items: items}, nil
}

func (service *batchReconciliationService) fetchProviderResults(
	ctx context.Context,
	job *domainaijob.AIBatchJob,
) (providerReconciliationResults, error) {
	if service.engine == nil {
		return providerReconciliationResults{}, fmt.Errorf("reconcile batch job %s: batch engine is required", job.ID.Hex())
	}
	handle := providerPollingHandle(job)
	results, err := service.engine.GetBatchResults(ctx, handle)
	if err != nil {
		return providerReconciliationResults{}, fmt.Errorf("reconcile batch job %s provider handle %q: %w", job.ID.Hex(), handle, err)
	}
	return mapProviderResults(job, results, service.now().UTC())
}
