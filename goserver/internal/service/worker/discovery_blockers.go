package worker

import (
	"context"
	"fmt"

	domaincommon "goserver/internal/domain/common"
	platformrepo "goserver/internal/platform/repository"
	servicecommon "goserver/internal/service/common"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

func (service *workerWorkDiscoveryService) workflowBlockers(
	ctx context.Context,
	workflowRunID primitive.ObjectID,
) ([]servicecommon.BlockingCondition, error) {
	blockers := make([]servicecommon.BlockingCondition, 0)

	if service.batchJobs != nil {
		jobs, err := service.batchJobs.List(ctx, platformrepo.AIBatchJobFilter{
			WorkflowRunIDs: []primitive.ObjectID{workflowRunID},
			Statuses: []domaincommon.AIBatchJobStatus{
				domaincommon.AIBatchJobStatusCreated,
				domaincommon.AIBatchJobStatusSubmitted,
				domaincommon.AIBatchJobStatusRunning,
				domaincommon.AIBatchJobStatusPartiallyCompleted,
			},
		}, platformrepo.AIBatchJobListOptions{
			Pagination: platformrepo.PageOptions{PageSize: 1},
			Sort:       platformrepo.AIBatchJobSortOption{By: platformrepo.AIBatchJobSortByUpdatedAt, Order: platformrepo.SortOrderAscending},
		})
		if err != nil {
			return nil, fmt.Errorf("discover continuable workflows: inspect batch jobs for workflow %s: %w", workflowRunID.Hex(), err)
		}
		if len(jobs.Items) > 0 {
			job := jobs.Items[0]
			blockers = append(blockers, servicecommon.BlockingCondition{
				Scope:         servicecommon.FailureScopeJob,
				ID:            job.ID,
				WorkflowRunID: workflowRunID,
				BatchJobID:    job.ID,
				Code:          "batch_job_unresolved",
				Message:       "workflow still has unresolved batch jobs",
			})
		}
	}

	if service.batchItems != nil {
		items, err := service.batchItems.FindPendingValidation(ctx, platformrepo.AIBatchItemFilter{
			WorkflowRunIDs: []primitive.ObjectID{workflowRunID},
		}, platformrepo.AIBatchItemListOptions{
			Pagination: platformrepo.PageOptions{PageSize: 1},
			Sort:       platformrepo.AIBatchItemSortOption{By: platformrepo.AIBatchItemSortByCompletedAt, Order: platformrepo.SortOrderAscending},
		})
		if err != nil {
			return nil, fmt.Errorf("discover continuable workflows: inspect batch items for workflow %s: %w", workflowRunID.Hex(), err)
		}
		if len(items.Items) > 0 {
			item := items.Items[0]
			blockers = append(blockers, servicecommon.BlockingCondition{
				Scope:         servicecommon.FailureScopeItem,
				ID:            item.ID,
				WorkflowRunID: workflowRunID,
				BatchJobID:    item.AIBatchJobID,
				BatchItemID:   item.ID,
				ReviewID:      item.TargetReviewID,
				Code:          "batch_item_pending_validation",
				Message:       "workflow has AI batch items awaiting validation",
			})
		}
	}

	if service.reviews != nil {
		materializable, err := service.reviews.List(ctx, platformrepo.CompanyReviewFilter{
			WorkflowRunIDs: []primitive.ObjectID{workflowRunID},
			LifecycleStates: []domaincommon.ReviewLifecycleState{
				domaincommon.ReviewLifecycleStateAICompletedUnvalidated,
				domaincommon.ReviewLifecycleStateValidationFailed,
			},
			PendingOnly: true,
		}, platformrepo.CompanyReviewListOptions{
			Pagination: platformrepo.PageOptions{PageSize: 1},
			Sort:       platformrepo.CompanyReviewSortOption{By: platformrepo.CompanyReviewSortByUpdatedAt, Order: platformrepo.SortOrderAscending},
		})
		if err != nil {
			return nil, fmt.Errorf("discover continuable workflows: inspect materializable reviews for workflow %s: %w", workflowRunID.Hex(), err)
		}
		if len(materializable.Items) > 0 {
			review := materializable.Items[0]
			blockers = append(blockers, servicecommon.BlockingCondition{
				Scope:         servicecommon.FailureScopeReview,
				ID:            review.ID,
				WorkflowRunID: workflowRunID,
				ReviewID:      review.ID,
				Code:          "review_pending_materialization",
				Message:       "workflow has reviews awaiting materialization",
			})
		}

		finalizable, err := service.reviews.List(ctx, platformrepo.CompanyReviewFilter{
			WorkflowRunIDs:  []primitive.ObjectID{workflowRunID},
			LifecycleStates: []domaincommon.ReviewLifecycleState{domaincommon.ReviewLifecycleStateAIValidated},
			PendingOnly:     true,
		}, platformrepo.CompanyReviewListOptions{
			Pagination: platformrepo.PageOptions{PageSize: 1},
			Sort:       platformrepo.CompanyReviewSortOption{By: platformrepo.CompanyReviewSortByUpdatedAt, Order: platformrepo.SortOrderAscending},
		})
		if err != nil {
			return nil, fmt.Errorf("discover continuable workflows: inspect finalizable reviews for workflow %s: %w", workflowRunID.Hex(), err)
		}
		if len(finalizable.Items) > 0 {
			review := finalizable.Items[0]
			blockers = append(blockers, servicecommon.BlockingCondition{
				Scope:         servicecommon.FailureScopeReview,
				ID:            review.ID,
				WorkflowRunID: workflowRunID,
				ReviewID:      review.ID,
				Code:          "review_pending_finalization",
				Message:       "workflow has reviews awaiting finalization",
			})
		}
	}

	if service.workflowSteps != nil {
		steps, err := service.workflowSteps.List(ctx, platformrepo.WorkflowStepRunFilter{
			WorkflowRunIDs: []primitive.ObjectID{workflowRunID},
			Statuses:       []domaincommon.WorkflowStepStatus{domaincommon.WorkflowStepStatusWaitingExternal},
		}, platformrepo.WorkflowStepRunListOptions{
			Pagination: platformrepo.PageOptions{PageSize: 1},
			Sort:       platformrepo.WorkflowStepRunSortOption{By: platformrepo.WorkflowStepRunSortByUpdatedAt, Order: platformrepo.SortOrderAscending},
		})
		if err != nil {
			return nil, fmt.Errorf("discover continuable workflows: inspect workflow steps for workflow %s: %w", workflowRunID.Hex(), err)
		}
		if len(steps.Items) > 0 && isWorkflowWaitingOnStep(steps.Items[0]) && len(blockers) > 0 {
			blockers[0].WaitingOnStep = steps.Items[0].StepName
		}
	}

	return blockers, nil
}
