package worker

import (
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

func (service *workerWorkDiscoveryService) limit(requested int) int {
	if requested > 0 && requested < service.config.MaxPageSize {
		return requested
	}
	if requested > service.config.MaxPageSize {
		return service.config.MaxPageSize
	}
	if service.config.DefaultLimit > service.config.MaxPageSize {
		return service.config.MaxPageSize
	}
	return service.config.DefaultLimit
}

func (service *workerWorkDiscoveryService) scanLimit(limit int) int {
	pageSize := limit * scanMultiplier
	if pageSize < limit {
		pageSize = limit
	}
	if pageSize > service.config.MaxPageSize {
		return service.config.MaxPageSize
	}
	return pageSize
}

func (service *workerWorkDiscoveryService) batchJobFilter(base DiscoveryRequestBase) platformrepo.AIBatchJobFilter {
	filter := platformrepo.AIBatchJobFilter{}
	if !base.WorkflowRunID.IsZero() {
		filter.WorkflowRunIDs = []primitive.ObjectID{base.WorkflowRunID}
	}
	if base.BookType != "" {
		filter.BookTypes = []domaincommon.BookType{base.BookType}
	}
	if base.JobType != "" {
		filter.JobTypes = []domaincommon.AIBatchJobType{base.JobType}
	}
	return filter
}

func (service *workerWorkDiscoveryService) batchItemFilter(base DiscoveryRequestBase) platformrepo.AIBatchItemFilter {
	filter := platformrepo.AIBatchItemFilter{}
	if !base.WorkflowRunID.IsZero() {
		filter.WorkflowRunIDs = []primitive.ObjectID{base.WorkflowRunID}
	}
	if base.BookType != "" {
		filter.BookTypes = []domaincommon.BookType{base.BookType}
	}
	if base.ItemType != "" {
		filter.ItemTypes = []domaincommon.AIBatchItemType{base.ItemType}
	}
	return filter
}

func applyReviewRequestFilters(filter *platformrepo.CompanyReviewFilter, base DiscoveryRequestBase, companyID primitive.ObjectID) {
	if !base.WorkflowRunID.IsZero() {
		filter.WorkflowRunIDs = []primitive.ObjectID{base.WorkflowRunID}
	}
	if !companyID.IsZero() {
		filter.CompanyIDs = []primitive.ObjectID{companyID}
	}
	if base.BookType != "" {
		filter.BookTypes = []domaincommon.BookType{base.BookType}
	}
}

func buildDiscoverySummary(operation string, scannedCount int, discoveredCount int) servicecommon.WorkerOperationSummary {
	outcome := servicecommon.ServiceOutcomeSuccess
	message := fmt.Sprintf("discovered %d work item(s)", discoveredCount)
	if discoveredCount == 0 {
		outcome = servicecommon.ServiceOutcomeNoop
		message = "no work discovered"
	}
	return servicecommon.WorkerOperationSummary{
		OperationSummary: servicecommon.OperationSummary{
			Operation:      operation,
			Outcome:        outcome,
			AttemptedCount: scannedCount,
			SuccessCount:   discoveredCount,
			Message:        message,
		},
		DiscoveredCount: discoveredCount,
	}
}

func isSubmittableBatchJob(job *domainaijob.AIBatchJob) bool {
	if job == nil {
		return false
	}
	return job.Status == domaincommon.AIBatchJobStatusCreated
}

func isRetrySubmittableBatchJob(job *domainaijob.AIBatchJob) bool {
	if job == nil {
		return false
	}
	return job.CanRetry()
}

func isPollableBatchJob(job *domainaijob.AIBatchJob, minimumInterval time.Duration, cutoff time.Time) bool {
	if job == nil || !job.CanPoll() {
		return false
	}
	if minimumInterval <= 0 || job.LastPolledAt == nil {
		return true
	}
	return !job.LastPolledAt.UTC().After(cutoff)
}

func isReconciliableBatchJob(job *domainaijob.AIBatchJob, completedOnly bool) bool {
	if job == nil {
		return false
	}
	if completedOnly {
		return job.Status == domaincommon.AIBatchJobStatusCompleted
	}
	switch job.Status {
	case domaincommon.AIBatchJobStatusCompleted,
		domaincommon.AIBatchJobStatusPartiallyCompleted,
		domaincommon.AIBatchJobStatusFailed,
		domaincommon.AIBatchJobStatusTimedOut:
		return true
	default:
		return false
	}
}

func reconciliableStatuses(completedOnly bool) []domaincommon.AIBatchJobStatus {
	if completedOnly {
		return []domaincommon.AIBatchJobStatus{domaincommon.AIBatchJobStatusCompleted}
	}
	return []domaincommon.AIBatchJobStatus{
		domaincommon.AIBatchJobStatusCompleted,
		domaincommon.AIBatchJobStatusPartiallyCompleted,
		domaincommon.AIBatchJobStatusFailed,
		domaincommon.AIBatchJobStatusTimedOut,
	}
}

func isValidatableBatchItem(item *domainaijob.AIBatchItem, revalidate bool) bool {
	if item == nil {
		return false
	}
	if item.Status == domaincommon.AIBatchItemStatusCompleted && item.ValidationStatus == domaincommon.ValidationStatusNotValidated {
		return true
	}
	return revalidate &&
		(item.Status == domaincommon.AIBatchItemStatusCompleted || item.Status == domaincommon.AIBatchItemStatusInvalidOutput) &&
		item.ValidationStatus == domaincommon.ValidationStatusInvalid
}

func isMaterializableBatchItem(item *domainaijob.AIBatchItem) bool {
	return item != nil &&
		item.Status == domaincommon.AIBatchItemStatusCompleted &&
		item.ValidationStatus == domaincommon.ValidationStatusValid &&
		!item.TargetReviewID.IsZero()
}

func isMaterializableReview(review *domainreview.CompanyReview, force bool) bool {
	if review == nil || review.IsFinalized() {
		return false
	}
	if force {
		return review.ReviewStatus == domaincommon.ReviewStatusDraft
	}
	switch review.ReviewLifecycleState {
	case domaincommon.ReviewLifecycleStateAICompletedUnvalidated,
		domaincommon.ReviewLifecycleStateValidationFailed:
		return review.ReviewStatus == domaincommon.ReviewStatusDraft
	default:
		return false
	}
}

func isFinalizableReview(review *domainreview.CompanyReview, force bool) bool {
	if review == nil || review.IsFinalized() {
		return false
	}
	if force {
		return review.ReviewStatus == domaincommon.ReviewStatusDraft
	}
	return review.CanFinalize()
}

func suggestedContinuationStep(bookType domaincommon.BookType) domaincommon.WorkflowStepName {
	if bookType == domaincommon.BookTypeTrading {
		return domaincommon.WorkflowStepNameApproveTradeCandidates
	}
	return domaincommon.WorkflowStepNameEvaluateThesisAndChange
}

func batchJobRefs(jobs []*domainaijob.AIBatchJob) []servicecommon.BatchJobRef {
	refs := make([]servicecommon.BatchJobRef, 0, len(jobs))
	for _, job := range jobs {
		if job == nil {
			continue
		}
		refs = append(refs, servicecommon.BatchJobRef{
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
		})
	}
	return refs
}

func batchItemRefs(items []*domainaijob.AIBatchItem) []servicecommon.BatchItemRef {
	refs := make([]servicecommon.BatchItemRef, 0, len(items))
	for _, item := range items {
		if item == nil {
			continue
		}
		refs = append(refs, servicecommon.BatchItemRef{
			ID:               item.ID,
			BatchJobID:       item.AIBatchJobID,
			WorkflowRunID:    item.WorkflowRunID,
			CompanyID:        item.CompanyID,
			ReviewID:         item.TargetReviewID,
			BookType:         item.BookType,
			ItemType:         item.ItemType,
			Status:           item.Status,
			ValidationStatus: item.ValidationStatus,
			Symbol:           item.Symbol,
		})
	}
	return refs
}

func reviewRefs(reviews []*domainreview.CompanyReview) []servicecommon.ReviewRef {
	refs := make([]servicecommon.ReviewRef, 0, len(reviews))
	for _, review := range reviews {
		if review == nil {
			continue
		}
		refs = append(refs, servicecommon.ReviewRef{
			ID:             review.ID,
			CompanyID:      review.CompanyID,
			WorkflowRunID:  review.WorkflowRunID,
			BookType:       review.BookType,
			Status:         review.ReviewStatus,
			LifecycleState: review.ReviewLifecycleState,
			Symbol:         review.Symbol,
		})
	}
	return refs
}

func makeWorkItemsForBatchJobs(kind servicecommon.WorkItemKind, jobs []servicecommon.BatchJobRef) []servicecommon.WorkItemRef {
	items := make([]servicecommon.WorkItemRef, 0, len(jobs))
	for _, job := range jobs {
		items = append(items, servicecommon.WorkItemRef{
			Kind:          kind,
			ID:            job.ID,
			WorkflowRunID: job.WorkflowRunID,
			BookType:      job.BookType,
			BatchJobID:    job.ID,
			Reason:        string(job.Status),
		})
	}
	return items
}

func makeWorkItemsForBatchItems(kind servicecommon.WorkItemKind, batchItems []servicecommon.BatchItemRef) []servicecommon.WorkItemRef {
	items := make([]servicecommon.WorkItemRef, 0, len(batchItems))
	for _, item := range batchItems {
		items = append(items, servicecommon.WorkItemRef{
			Kind:          kind,
			ID:            item.ID,
			WorkflowRunID: item.WorkflowRunID,
			BookType:      item.BookType,
			BatchJobID:    item.BatchJobID,
			BatchItemID:   item.ID,
			ReviewID:      item.ReviewID,
			CompanyID:     item.CompanyID,
			Reason:        string(item.ValidationStatus),
		})
	}
	return items
}

func makeWorkItemsForReviews(kind servicecommon.WorkItemKind, reviews []servicecommon.ReviewRef) []servicecommon.WorkItemRef {
	items := make([]servicecommon.WorkItemRef, 0, len(reviews))
	for _, review := range reviews {
		items = append(items, servicecommon.WorkItemRef{
			Kind:          kind,
			ID:            review.ID,
			WorkflowRunID: review.WorkflowRunID,
			BookType:      review.BookType,
			ReviewID:      review.ID,
			CompanyID:     review.CompanyID,
			Reason:        string(review.LifecycleState),
		})
	}
	return items
}

func makeWorkItemsForContinuations(continuations []servicecommon.ContinuationRef) []servicecommon.WorkItemRef {
	items := make([]servicecommon.WorkItemRef, 0, len(continuations))
	for _, continuation := range continuations {
		items = append(items, servicecommon.WorkItemRef{
			Kind:          servicecommon.WorkItemKindWorkflowContinuation,
			ID:            continuation.WorkflowRunID,
			WorkflowRunID: continuation.WorkflowRunID,
			BookType:      continuation.BookType,
			Reason:        string(continuation.NextSuggestedStep),
		})
	}
	return items
}

func truncateJobs(jobs []*domainaijob.AIBatchJob, limit int) []*domainaijob.AIBatchJob {
	if len(jobs) <= limit {
		return jobs
	}
	return jobs[:limit]
}

func truncateItems(items []*domainaijob.AIBatchItem, limit int) []*domainaijob.AIBatchItem {
	if len(items) <= limit {
		return items
	}
	return items[:limit]
}

func truncateReviews(reviews []*domainreview.CompanyReview, limit int) []*domainreview.CompanyReview {
	if len(reviews) <= limit {
		return reviews
	}
	return reviews[:limit]
}

func containsJob(jobs []*domainaijob.AIBatchJob, id primitive.ObjectID) bool {
	for _, job := range jobs {
		if job != nil && job.ID == id {
			return true
		}
	}
	return false
}

func uniqueObjectIDs(ids []primitive.ObjectID) []primitive.ObjectID {
	seen := make(map[primitive.ObjectID]struct{}, len(ids))
	unique := make([]primitive.ObjectID, 0, len(ids))
	for _, id := range ids {
		if id.IsZero() {
			continue
		}
		if _, exists := seen[id]; exists {
			continue
		}
		seen[id] = struct{}{}
		unique = append(unique, id)
	}
	return unique
}

func reviewIDsFromReviews(reviews []*domainreview.CompanyReview) map[primitive.ObjectID]struct{} {
	ids := make(map[primitive.ObjectID]struct{}, len(reviews))
	for _, review := range reviews {
		if review != nil && !review.ID.IsZero() {
			ids[review.ID] = struct{}{}
		}
	}
	return ids
}

func limitItemsByReviewIDs(items []*domainaijob.AIBatchItem, reviewIDs map[primitive.ObjectID]struct{}) []*domainaijob.AIBatchItem {
	if len(reviewIDs) == 0 {
		return nil
	}
	filtered := make([]*domainaijob.AIBatchItem, 0, len(items))
	for _, item := range items {
		if item == nil {
			continue
		}
		if _, ok := reviewIDs[item.TargetReviewID]; ok {
			filtered = append(filtered, item)
		}
	}
	return filtered
}

func isWorkflowWaitingOnStep(step *domainworkflow.WorkflowStepRun) bool {
	return step != nil && step.Status == domaincommon.WorkflowStepStatusWaitingExternal
}
