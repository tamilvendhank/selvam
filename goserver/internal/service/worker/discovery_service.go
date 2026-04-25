package worker

import (
	"context"
	"fmt"
	"time"

	domainaijob "goserver/internal/domain/aijob"
	domaincommon "goserver/internal/domain/common"
	domainreview "goserver/internal/domain/review"
	domainworkflow "goserver/internal/domain/workflow"
	platformrepo "goserver/internal/platform/repository"
	servicecommon "goserver/internal/service/common"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

const (
	defaultDiscoveryLimit = 50
	defaultMaxPageSize    = 500
	scanMultiplier        = 3
)

type WorkerWorkDiscoveryConfig struct {
	DefaultLimit                      int
	MaxPageSize                       int
	MinimumPollInterval               time.Duration
	IncludeRetryableFailedSubmissions bool
}

type WorkerWorkDiscoveryOption func(*WorkerWorkDiscoveryConfig)

func WithWorkerWorkDiscoveryConfig(config WorkerWorkDiscoveryConfig) WorkerWorkDiscoveryOption {
	return func(target *WorkerWorkDiscoveryConfig) {
		if config.DefaultLimit > 0 {
			target.DefaultLimit = config.DefaultLimit
		}
		if config.MaxPageSize > 0 {
			target.MaxPageSize = config.MaxPageSize
		}
		if config.MinimumPollInterval > 0 {
			target.MinimumPollInterval = config.MinimumPollInterval
		}
		target.IncludeRetryableFailedSubmissions = config.IncludeRetryableFailedSubmissions
	}
}

func WithMinimumPollInterval(interval time.Duration) WorkerWorkDiscoveryOption {
	return func(config *WorkerWorkDiscoveryConfig) {
		if interval > 0 {
			config.MinimumPollInterval = interval
		}
	}
}

func WithRetryableFailedSubmissions(enabled bool) WorkerWorkDiscoveryOption {
	return func(config *WorkerWorkDiscoveryConfig) {
		config.IncludeRetryableFailedSubmissions = enabled
	}
}

type workerWorkDiscoveryService struct {
	batchJobs     platformrepo.AIBatchJobRepository
	batchItems    platformrepo.AIBatchItemRepository
	reviews       platformrepo.CompanyReviewRepository
	workflowRuns  platformrepo.WorkflowRunRepository
	workflowSteps platformrepo.WorkflowStepRunRepository
	config        WorkerWorkDiscoveryConfig
	now           func() time.Time
}

var _ WorkerWorkDiscoveryService = (*workerWorkDiscoveryService)(nil)

func NewWorkerWorkDiscoveryService(
	batchJobs platformrepo.AIBatchJobRepository,
	batchItems platformrepo.AIBatchItemRepository,
	reviews platformrepo.CompanyReviewRepository,
	workflowRuns platformrepo.WorkflowRunRepository,
	workflowSteps platformrepo.WorkflowStepRunRepository,
	options ...WorkerWorkDiscoveryOption,
) WorkerWorkDiscoveryService {
	config := WorkerWorkDiscoveryConfig{
		DefaultLimit: defaultDiscoveryLimit,
		MaxPageSize:  defaultMaxPageSize,
	}
	for _, option := range options {
		if option != nil {
			option(&config)
		}
	}
	if config.DefaultLimit <= 0 {
		config.DefaultLimit = defaultDiscoveryLimit
	}
	if config.MaxPageSize <= 0 {
		config.MaxPageSize = defaultMaxPageSize
	}

	return &workerWorkDiscoveryService{
		batchJobs:     batchJobs,
		batchItems:    batchItems,
		reviews:       reviews,
		workflowRuns:  workflowRuns,
		workflowSteps: workflowSteps,
		config:        config,
		now:           time.Now,
	}
}

func (service *workerWorkDiscoveryService) DiscoverSubmittableBatchJobs(
	ctx context.Context,
	request DiscoverSubmittableBatchJobsRequest,
) (*DiscoverSubmittableBatchJobsResult, error) {
	if err := request.Validate(); err != nil {
		return nil, err
	}
	if service.batchJobs == nil {
		return nil, fmt.Errorf("discover submittable batch jobs: batch job repository is required")
	}

	limit := service.limit(request.MaxItems)
	filter := service.batchJobFilter(request.DiscoveryRequestBase)
	result, err := service.batchJobs.FindSubmittableJobs(ctx, filter, platformrepo.AIBatchJobListOptions{
		Pagination: platformrepo.PageOptions{PageSize: service.scanLimit(limit)},
		Sort:       platformrepo.AIBatchJobSortOption{By: platformrepo.AIBatchJobSortByCreatedAt, Order: platformrepo.SortOrderAscending},
	})
	if err != nil {
		return nil, fmt.Errorf("discover submittable batch jobs: %w", err)
	}

	jobs := make([]*domainaijob.AIBatchJob, 0, len(result.Items))
	for _, job := range result.Items {
		if isSubmittableBatchJob(job) {
			jobs = append(jobs, job)
		}
	}

	if service.config.IncludeRetryableFailedSubmissions && len(jobs) < limit {
		retryableFilter := service.batchJobFilter(request.DiscoveryRequestBase)
		retryableFilter.RetryableOnly = true
		retryable, err := service.batchJobs.List(ctx, retryableFilter, platformrepo.AIBatchJobListOptions{
			Pagination: platformrepo.PageOptions{PageSize: service.scanLimit(limit - len(jobs))},
			Sort:       platformrepo.AIBatchJobSortOption{By: platformrepo.AIBatchJobSortByUpdatedAt, Order: platformrepo.SortOrderAscending},
		})
		if err != nil {
			return nil, fmt.Errorf("discover retryable submittable batch jobs: %w", err)
		}
		for _, job := range retryable.Items {
			if isRetrySubmittableBatchJob(job) && !containsJob(jobs, job.ID) {
				jobs = append(jobs, job)
			}
		}
		result.Page.HasMore = result.Page.HasMore || retryable.Page.HasMore
	}

	jobs = truncateJobs(jobs, limit)
	refs := batchJobRefs(jobs)
	workItems := makeWorkItemsForBatchJobs(servicecommon.WorkItemKindBatchSubmission, refs)
	return &DiscoverSubmittableBatchJobsResult{
		BatchJobs: refs,
		WorkItems: workItems,
		HasMore:   result.Page.HasMore,
		Summary:   buildDiscoverySummary("discover_submittable_batch_jobs", len(result.Items), len(workItems)),
	}, nil
}

func (service *workerWorkDiscoveryService) DiscoverPollableBatchJobs(
	ctx context.Context,
	request DiscoverPollableBatchJobsRequest,
) (*DiscoverPollableBatchJobsResult, error) {
	if err := request.Validate(); err != nil {
		return nil, err
	}
	if service.batchJobs == nil {
		return nil, fmt.Errorf("discover pollable batch jobs: batch job repository is required")
	}

	limit := service.limit(request.MaxItems)
	filter := service.batchJobFilter(request.DiscoveryRequestBase)
	if len(request.PollOnlyStatuses) > 0 {
		filter.Statuses = request.PollOnlyStatuses
	}
	result, err := service.batchJobs.FindPollableJobs(ctx, filter, platformrepo.AIBatchJobListOptions{
		Pagination: platformrepo.PageOptions{PageSize: service.scanLimit(limit)},
		Sort:       platformrepo.AIBatchJobSortOption{By: platformrepo.AIBatchJobSortByLastPolledAt, Order: platformrepo.SortOrderAscending},
	})
	if err != nil {
		return nil, fmt.Errorf("discover pollable batch jobs: %w", err)
	}

	cutoff := service.now().UTC().Add(-service.config.MinimumPollInterval)
	jobs := make([]*domainaijob.AIBatchJob, 0, len(result.Items))
	for _, job := range result.Items {
		if isPollableBatchJob(job, service.config.MinimumPollInterval, cutoff) {
			jobs = append(jobs, job)
		}
	}

	jobs = truncateJobs(jobs, limit)
	refs := batchJobRefs(jobs)
	workItems := makeWorkItemsForBatchJobs(servicecommon.WorkItemKindBatchPolling, refs)
	return &DiscoverPollableBatchJobsResult{
		BatchJobs: refs,
		WorkItems: workItems,
		HasMore:   result.Page.HasMore,
		Summary:   buildDiscoverySummary("discover_pollable_batch_jobs", len(result.Items), len(workItems)),
	}, nil
}

func (service *workerWorkDiscoveryService) DiscoverReconciliableBatchJobs(
	ctx context.Context,
	request DiscoverReconciliableBatchJobsRequest,
) (*DiscoverReconciliableBatchJobsResult, error) {
	if err := request.Validate(); err != nil {
		return nil, err
	}
	if service.batchJobs == nil {
		return nil, fmt.Errorf("discover reconciliable batch jobs: batch job repository is required")
	}

	limit := service.limit(request.MaxItems)
	filter := service.batchJobFilter(request.DiscoveryRequestBase)
	filter.Statuses = reconciliableStatuses(request.CompletedOnly)
	result, err := service.batchJobs.List(ctx, filter, platformrepo.AIBatchJobListOptions{
		Pagination: platformrepo.PageOptions{PageSize: service.scanLimit(limit)},
		Sort:       platformrepo.AIBatchJobSortOption{By: platformrepo.AIBatchJobSortByUpdatedAt, Order: platformrepo.SortOrderAscending},
	})
	if err != nil {
		return nil, fmt.Errorf("discover reconciliable batch jobs: %w", err)
	}

	jobs := make([]*domainaijob.AIBatchJob, 0, len(result.Items))
	for _, job := range result.Items {
		if isReconciliableBatchJob(job, request.CompletedOnly) {
			jobs = append(jobs, job)
		}
	}

	jobs = truncateJobs(jobs, limit)
	refs := batchJobRefs(jobs)
	workItems := makeWorkItemsForBatchJobs(servicecommon.WorkItemKindBatchReconciliation, refs)
	return &DiscoverReconciliableBatchJobsResult{
		BatchJobs: refs,
		WorkItems: workItems,
		HasMore:   result.Page.HasMore,
		Summary:   buildDiscoverySummary("discover_reconciliable_batch_jobs", len(result.Items), len(workItems)),
	}, nil
}

func (service *workerWorkDiscoveryService) DiscoverValidatableItems(
	ctx context.Context,
	request DiscoverValidatableItemsRequest,
) (*DiscoverValidatableItemsResult, error) {
	if err := request.Validate(); err != nil {
		return nil, err
	}
	if service.batchItems == nil {
		return nil, fmt.Errorf("discover validatable items: batch item repository is required")
	}

	limit := service.limit(request.MaxItems)
	filter := service.batchItemFilter(request.DiscoveryRequestBase)
	if request.Revalidate {
		filter.ValidationStatuses = []domaincommon.ValidationStatus{
			domaincommon.ValidationStatusNotValidated,
			domaincommon.ValidationStatusInvalid,
		}
		filter.Statuses = []domaincommon.AIBatchItemStatus{
			domaincommon.AIBatchItemStatusCompleted,
			domaincommon.AIBatchItemStatusInvalidOutput,
		}
	} else {
		filter.PendingValidationOnly = true
	}

	result, err := service.batchItems.List(ctx, filter, platformrepo.AIBatchItemListOptions{
		Pagination: platformrepo.PageOptions{PageSize: service.scanLimit(limit)},
		Sort:       platformrepo.AIBatchItemSortOption{By: platformrepo.AIBatchItemSortByCompletedAt, Order: platformrepo.SortOrderAscending},
	})
	if err != nil {
		return nil, fmt.Errorf("discover validatable items: %w", err)
	}

	items := make([]*domainaijob.AIBatchItem, 0, len(result.Items))
	for _, item := range result.Items {
		if isValidatableBatchItem(item, request.Revalidate) {
			items = append(items, item)
		}
	}

	items = truncateItems(items, limit)
	refs := batchItemRefs(items)
	workItems := makeWorkItemsForBatchItems(servicecommon.WorkItemKindAIOutputValidation, refs)
	return &DiscoverValidatableItemsResult{
		BatchItems: refs,
		WorkItems:  workItems,
		HasMore:    result.Page.HasMore,
		Summary:    buildDiscoverySummary("discover_validatable_items", len(result.Items), len(workItems)),
	}, nil
}

func (service *workerWorkDiscoveryService) DiscoverMaterializableReviews(
	ctx context.Context,
	request DiscoverMaterializableReviewsRequest,
) (*DiscoverMaterializableReviewsResult, error) {
	if err := request.Validate(); err != nil {
		return nil, err
	}
	if service.batchItems == nil || service.reviews == nil {
		return nil, fmt.Errorf("discover materializable reviews: batch item and review repositories are required")
	}

	limit := service.limit(request.MaxItems)
	itemFilter := service.batchItemFilter(request.DiscoveryRequestBase)
	itemFilter.Statuses = []domaincommon.AIBatchItemStatus{domaincommon.AIBatchItemStatusCompleted}
	itemFilter.ValidationStatuses = []domaincommon.ValidationStatus{domaincommon.ValidationStatusValid}

	itemResult, err := service.batchItems.List(ctx, itemFilter, platformrepo.AIBatchItemListOptions{
		Pagination: platformrepo.PageOptions{PageSize: service.scanLimit(limit)},
		Sort:       platformrepo.AIBatchItemSortOption{By: platformrepo.AIBatchItemSortByCompletedAt, Order: platformrepo.SortOrderAscending},
	})
	if err != nil {
		return nil, fmt.Errorf("discover materializable reviews: list validated batch items: %w", err)
	}

	materializableItems := make([]*domainaijob.AIBatchItem, 0, len(itemResult.Items))
	reviewIDs := make([]primitive.ObjectID, 0, len(itemResult.Items))
	for _, item := range itemResult.Items {
		if isMaterializableBatchItem(item) {
			materializableItems = append(materializableItems, item)
			reviewIDs = append(reviewIDs, item.TargetReviewID)
		}
	}
	reviewIDs = uniqueObjectIDs(reviewIDs)
	if len(reviewIDs) == 0 {
		return &DiscoverMaterializableReviewsResult{
			HasMore: itemResult.Page.HasMore,
			Summary: buildDiscoverySummary("discover_materializable_reviews", len(itemResult.Items), 0),
		}, nil
	}

	reviewFilter := platformrepo.CompanyReviewFilter{
		IDs: reviewIDs,
		LifecycleStates: []domaincommon.ReviewLifecycleState{
			domaincommon.ReviewLifecycleStateAICompletedUnvalidated,
			domaincommon.ReviewLifecycleStateValidationFailed,
		},
		PendingOnly: true,
	}
	applyReviewRequestFilters(&reviewFilter, request.DiscoveryRequestBase, primitive.NilObjectID)
	reviewResult, err := service.reviews.List(ctx, reviewFilter, platformrepo.CompanyReviewListOptions{
		Pagination: platformrepo.PageOptions{PageSize: limit},
		Sort:       platformrepo.CompanyReviewSortOption{By: platformrepo.CompanyReviewSortByUpdatedAt, Order: platformrepo.SortOrderAscending},
	})
	if err != nil {
		return nil, fmt.Errorf("discover materializable reviews: list linked reviews: %w", err)
	}

	reviews := make([]*domainreview.CompanyReview, 0, len(reviewResult.Items))
	for _, review := range reviewResult.Items {
		if isMaterializableReview(review, request.Force) {
			reviews = append(reviews, review)
		}
	}

	reviews = truncateReviews(reviews, limit)
	refs := reviewRefs(reviews)
	itemRefs := batchItemRefs(limitItemsByReviewIDs(materializableItems, reviewIDsFromReviews(reviews)))
	workItems := makeWorkItemsForReviews(servicecommon.WorkItemKindReviewMaterialize, refs)
	return &DiscoverMaterializableReviewsResult{
		Reviews:    refs,
		BatchItems: itemRefs,
		WorkItems:  workItems,
		HasMore:    itemResult.Page.HasMore || reviewResult.Page.HasMore,
		Summary:    buildDiscoverySummary("discover_materializable_reviews", len(itemResult.Items)+len(reviewResult.Items), len(workItems)),
	}, nil
}

func (service *workerWorkDiscoveryService) DiscoverFinalizableReviews(
	ctx context.Context,
	request DiscoverFinalizableReviewsRequest,
) (*DiscoverFinalizableReviewsResult, error) {
	if err := request.Validate(); err != nil {
		return nil, err
	}
	if service.reviews == nil {
		return nil, fmt.Errorf("discover finalizable reviews: review repository is required")
	}

	limit := service.limit(request.MaxItems)
	filter := platformrepo.CompanyReviewFilter{
		LifecycleStates: []domaincommon.ReviewLifecycleState{domaincommon.ReviewLifecycleStateAIValidated},
		PendingOnly:     true,
	}
	applyReviewRequestFilters(&filter, request.DiscoveryRequestBase, request.CompanyID)
	result, err := service.reviews.List(ctx, filter, platformrepo.CompanyReviewListOptions{
		Pagination: platformrepo.PageOptions{PageSize: service.scanLimit(limit)},
		Sort:       platformrepo.CompanyReviewSortOption{By: platformrepo.CompanyReviewSortByUpdatedAt, Order: platformrepo.SortOrderAscending},
	})
	if err != nil {
		return nil, fmt.Errorf("discover finalizable reviews: %w", err)
	}

	reviews := make([]*domainreview.CompanyReview, 0, len(result.Items))
	for _, review := range result.Items {
		if isFinalizableReview(review, request.Force) {
			reviews = append(reviews, review)
		}
	}

	reviews = truncateReviews(reviews, limit)
	refs := reviewRefs(reviews)
	workItems := makeWorkItemsForReviews(servicecommon.WorkItemKindReviewFinalize, refs)
	return &DiscoverFinalizableReviewsResult{
		Reviews:   refs,
		WorkItems: workItems,
		HasMore:   result.Page.HasMore,
		Summary:   buildDiscoverySummary("discover_finalizable_reviews", len(result.Items), len(workItems)),
	}, nil
}

func (service *workerWorkDiscoveryService) DiscoverContinuableWorkflows(
	ctx context.Context,
	request DiscoverContinuableWorkflowsRequest,
) (*DiscoverContinuableWorkflowsResult, error) {
	if err := request.Validate(); err != nil {
		return nil, err
	}
	if service.workflowRuns == nil {
		return nil, fmt.Errorf("discover continuable workflows: workflow run repository is required")
	}

	limit := service.limit(request.MaxItems)
	filter := platformrepo.WorkflowRunFilter{ActiveOnly: true}
	if !request.WorkflowRunID.IsZero() {
		filter.IDs = []primitive.ObjectID{request.WorkflowRunID}
	}
	if request.BookType != "" {
		filter.BookTypes = []domaincommon.BookType{request.BookType}
	}

	result, err := service.workflowRuns.FindResumable(ctx, filter, platformrepo.WorkflowRunListOptions{
		Pagination: platformrepo.PageOptions{PageSize: service.scanLimit(limit)},
		Sort:       platformrepo.WorkflowRunSortOption{By: platformrepo.WorkflowRunSortByUpdatedAt, Order: platformrepo.SortOrderAscending},
	})
	if err != nil {
		return nil, fmt.Errorf("discover continuable workflows: %w", err)
	}

	continuations := make([]servicecommon.ContinuationRef, 0, len(result.Items))
	for _, run := range result.Items {
		ref, eligible, err := service.evaluateContinuableWorkflowCandidate(ctx, run, request.Force)
		if err != nil {
			return nil, err
		}
		if eligible {
			continuations = append(continuations, ref)
		}
		if len(continuations) >= limit {
			break
		}
	}

	workItems := makeWorkItemsForContinuations(continuations)
	return &DiscoverContinuableWorkflowsResult{
		Continuations: continuations,
		WorkItems:     workItems,
		HasMore:       result.Page.HasMore,
		Summary:       buildDiscoverySummary("discover_continuable_workflows", len(result.Items), len(workItems)),
	}, nil
}

func (service *workerWorkDiscoveryService) evaluateContinuableWorkflowCandidate(
	ctx context.Context,
	run *domainworkflow.WorkflowRun,
	force bool,
) (servicecommon.ContinuationRef, bool, error) {
	ref := servicecommon.ContinuationRef{
		WorkflowRunID: run.ID,
		BookType:      run.BookType,
		CurrentStatus: run.Status,
	}
	if run == nil || run.IsTerminal() {
		return ref, false, nil
	}
	if force {
		ref.Ready = true
		ref.NextSuggestedStep = suggestedContinuationStep(run.BookType)
		return ref, true, nil
	}

	blockers, err := service.workflowBlockers(ctx, run.ID)
	if err != nil {
		return ref, false, err
	}
	ref.Blockers = blockers
	ref.Ready = len(blockers) == 0
	ref.NextSuggestedStep = suggestedContinuationStep(run.BookType)
	return ref, ref.Ready, nil
}
