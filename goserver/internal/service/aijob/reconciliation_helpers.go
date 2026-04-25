package aijob

import (
	"context"
	"errors"
	"fmt"
	"strings"

	domainaijob "goserver/internal/domain/aijob"
	domaincommon "goserver/internal/domain/common"
	platformrepo "goserver/internal/platform/repository"
	servicecommon "goserver/internal/service/common"
	workerservice "goserver/internal/service/worker"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

var errReconciliationSkipped = errors.New("reconciliation skipped")

type reconciliationRequestOptions struct {
	WorkflowRunID         primitive.ObjectID
	BookType              domaincommon.BookType
	JobType               domaincommon.AIBatchJobType
	Force                 bool
	IncludeCompletedItems bool
	InitiatedBy           string
	CorrelationID         string
	TreatIneligibleAsSkip bool
}

type reconciliationContext struct {
	Job   *domainaijob.AIBatchJob
	Items []*domainaijob.AIBatchItem
}

type reconcileOneOutcome struct {
	BatchJobID          primitive.ObjectID
	WorkflowRunID       primitive.ObjectID
	ProviderStatus      domaincommon.AIBatchJobStatus
	CompletedItems      []ReconciledBatchItemRef
	FailedItems         []ReconciledBatchItemRef
	InvalidItems        []ReconciledBatchItemRef
	MissingItemResults  []*domainaijob.AIBatchItem
	UnmatchedResults    []providerReconciliationItem
	DuplicateResults    []providerReconciliationItem
	ItemsCompleted      int
	ItemsFailed         int
	ItemsInvalid        int
	ItemsStillPending   int
	ReadyValidationIDs  []primitive.ObjectID
	ReadyWorkflowRunIDs []primitive.ObjectID
	PartialFailures     []servicecommon.PartialFailure
	Skipped             bool
}

type itemReconciliationOutcome struct {
	Completed          *ReconciledBatchItemRef
	Failed             *ReconciledBatchItemRef
	Invalid            *ReconciledBatchItemRef
	StillPending       bool
	ReadyForValidation bool
}

func (service *batchReconciliationService) maxReconciliationJobs(requested int) int {
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

func (service *batchReconciliationService) discoverReconciliableJobIDs(
	ctx context.Context,
	request ReconcilePendingBatchJobsRequest,
) ([]primitive.ObjectID, error) {
	if !request.BatchJobID.IsZero() {
		return []primitive.ObjectID{request.BatchJobID}, nil
	}

	limit := service.maxReconciliationJobs(request.MaxJobs)
	if service.discovery != nil && !request.Force {
		discovered, err := service.discovery.DiscoverReconciliableBatchJobs(ctx, workerservice.DiscoverReconciliableBatchJobsRequest{
			DiscoveryRequestBase: workerservice.DiscoveryRequestBase{
				WorkflowRunID: request.WorkflowRunID,
				BookType:      request.BookType,
				JobType:       request.JobType,
				MaxItems:      limit,
			},
		})
		if err != nil {
			return nil, fmt.Errorf("discover reconciliable batch jobs: %w", err)
		}
		return jobIDsFromRefs(discovered.BatchJobs, limit), nil
	}

	if service.batchJobs == nil {
		return nil, fmt.Errorf("discover reconciliable batch jobs: batch job repository is required")
	}
	filter := platformrepo.AIBatchJobFilter{
		Statuses: reconciliableJobStatuses(),
	}
	if !request.WorkflowRunID.IsZero() {
		filter.WorkflowRunIDs = []primitive.ObjectID{request.WorkflowRunID}
	}
	if request.BookType != "" {
		filter.BookTypes = []domaincommon.BookType{request.BookType}
	}
	if request.JobType != "" {
		filter.JobTypes = []domaincommon.AIBatchJobType{request.JobType}
	}
	result, err := service.batchJobs.List(ctx, filter, platformrepo.AIBatchJobListOptions{
		Pagination: platformrepo.PageOptions{PageSize: limit},
		Sort:       platformrepo.AIBatchJobSortOption{By: platformrepo.AIBatchJobSortByUpdatedAt, Order: platformrepo.SortOrderAscending},
	})
	if err != nil {
		return nil, fmt.Errorf("list reconciliable batch jobs: %w", err)
	}
	ids := make([]primitive.ObjectID, 0, len(result.Items))
	for _, job := range result.Items {
		if job != nil {
			ids = append(ids, job.ID)
		}
	}
	return ids, nil
}

func (service *batchReconciliationService) loadBatchItems(
	ctx context.Context,
	batchJobID primitive.ObjectID,
) ([]*domainaijob.AIBatchItem, error) {
	items := make([]*domainaijob.AIBatchItem, 0)
	offset := 0
	for {
		result, err := service.batchItems.ListByBatchJobID(ctx, batchJobID, platformrepo.AIBatchItemListOptions{
			Pagination: platformrepo.PageOptions{PageSize: service.config.MaxPageSize, Offset: offset},
			Sort:       platformrepo.AIBatchItemSortOption{By: platformrepo.AIBatchItemSortByCreatedAt, Order: platformrepo.SortOrderAscending},
		})
		if err != nil {
			return nil, fmt.Errorf("reconcile batch job %s: load batch items: %w", batchJobID.Hex(), err)
		}
		items = append(items, result.Items...)
		if !result.Page.HasMore || len(result.Items) == 0 {
			break
		}
		offset += len(result.Items)
	}
	return items, nil
}

func validateReconciliationEligibility(
	job *domainaijob.AIBatchJob,
	items []*domainaijob.AIBatchItem,
	options reconciliationRequestOptions,
) error {
	if job == nil {
		return fmt.Errorf("batch job is required")
	}
	if !options.WorkflowRunID.IsZero() && job.WorkflowRunID != options.WorkflowRunID {
		return fmt.Errorf("%w: workflowRunId filter does not match", errReconciliationSkipped)
	}
	if options.BookType != "" && job.BookType != options.BookType {
		return fmt.Errorf("%w: bookType filter does not match", errReconciliationSkipped)
	}
	if options.JobType != "" && job.JobType != options.JobType {
		return fmt.Errorf("%w: jobType filter does not match", errReconciliationSkipped)
	}
	if strings.TrimSpace(providerPollingHandle(job)) == "" {
		return fmt.Errorf("provider job handle is required for reconciliation")
	}
	if len(items) == 0 {
		return fmt.Errorf("batch job has no persisted batch items")
	}
	if !options.Force && !isReconciliableJobStatus(job.Status) {
		return fmt.Errorf("%w: status %q is not reconciliable", errReconciliationSkipped, job.Status)
	}
	if !options.Force && !options.IncludeCompletedItems && allItemsAlreadyReconciled(items) {
		return fmt.Errorf("%w: all batch items are already reconciled", errReconciliationSkipped)
	}
	return nil
}

func isReconciliationSkip(err error) bool {
	return errors.Is(err, errReconciliationSkipped)
}

func isReconciliableJobStatus(status domaincommon.AIBatchJobStatus) bool {
	for _, candidate := range reconciliableJobStatuses() {
		if candidate == status {
			return true
		}
	}
	return false
}

func reconciliableJobStatuses() []domaincommon.AIBatchJobStatus {
	return []domaincommon.AIBatchJobStatus{
		domaincommon.AIBatchJobStatusCompleted,
		domaincommon.AIBatchJobStatusPartiallyCompleted,
		domaincommon.AIBatchJobStatusFailed,
		domaincommon.AIBatchJobStatusTimedOut,
		domaincommon.AIBatchJobStatusCancelled,
	}
}

func allItemsAlreadyReconciled(items []*domainaijob.AIBatchItem) bool {
	for _, item := range items {
		if item == nil {
			continue
		}
		if !isItemAlreadyReconciled(item) {
			return false
		}
	}
	return len(items) > 0
}

func isItemAlreadyReconciled(item *domainaijob.AIBatchItem) bool {
	if item == nil {
		return false
	}
	switch item.Status {
	case domaincommon.AIBatchItemStatusCompleted:
		return len(item.ResultPayload) > 0
	case domaincommon.AIBatchItemStatusFailed,
		domaincommon.AIBatchItemStatusInvalidOutput,
		domaincommon.AIBatchItemStatusSkipped:
		return true
	default:
		return false
	}
}

func shouldSignalWorkflowAfterReconciliation(outcome reconcileOneOutcome) bool {
	return !outcome.WorkflowRunID.IsZero() &&
		outcome.ItemsStillPending == 0 &&
		len(outcome.MissingItemResults) == 0
}

func mergeItemOutcome(outcome *reconcileOneOutcome, itemOutcome itemReconciliationOutcome) {
	switch {
	case itemOutcome.Completed != nil:
		outcome.CompletedItems = append(outcome.CompletedItems, *itemOutcome.Completed)
		outcome.ItemsCompleted++
		if itemOutcome.ReadyForValidation {
			outcome.ReadyValidationIDs = append(outcome.ReadyValidationIDs, itemOutcome.Completed.ID)
		}
	case itemOutcome.Failed != nil:
		outcome.FailedItems = append(outcome.FailedItems, *itemOutcome.Failed)
		outcome.ItemsFailed++
	case itemOutcome.Invalid != nil:
		outcome.InvalidItems = append(outcome.InvalidItems, *itemOutcome.Invalid)
		outcome.ItemsInvalid++
	case itemOutcome.StillPending:
		outcome.ItemsStillPending++
	}
}
