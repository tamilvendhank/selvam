package continuation

import (
	"context"
	"fmt"

	domaincommon "goserver/internal/domain/common"
	allocationsvc "goserver/internal/service/allocation"
	servicecommon "goserver/internal/service/common"
	projectionsvc "goserver/internal/service/projection"
	reviewsvc "goserver/internal/service/review"
	thesissvc "goserver/internal/service/thesis"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

func (service *workflowContinuationService) executeInvestingStep(
	ctx context.Context,
	execution *continuationExecutionContext,
	stepName domaincommon.WorkflowStepName,
	request ContinueWorkflowRequest,
) (stepExecutionOutcome, error) {
	switch stepName {
	case domaincommon.WorkflowStepNameEvaluateThesisAndChange:
		return service.executeStep(ctx, execution, stepName, request, service.executeEvaluateThesisAndChange)
	case domaincommon.WorkflowStepNameMapActions:
		return service.executeStep(ctx, execution, stepName, request, service.executeMapActions)
	case domaincommon.WorkflowStepNameAssignBuckets:
		return service.executeStep(ctx, execution, stepName, request, service.executeAssignBuckets)
	case domaincommon.WorkflowStepNameBuildCapitalCandidates:
		return service.executeStep(ctx, execution, stepName, request, service.executeBuildCapitalCandidates)
	case domaincommon.WorkflowStepNameAllocateCapital:
		return service.executeStep(ctx, execution, stepName, request, service.executeAllocateCapital)
	case domaincommon.WorkflowStepNamePersistOutputs:
		return service.executeStep(ctx, execution, stepName, request, service.executePersistOutputs)
	case domaincommon.WorkflowStepNamePublishRunSummary:
		return service.executeStep(ctx, execution, stepName, request, service.executePublishRunSummary)
	default:
		return stepExecutionOutcome{StepName: stepName}, fmt.Errorf("unsupported investing continuation step %q", stepName)
	}
}

func (service *workflowContinuationService) executeEvaluateThesisAndChange(
	ctx context.Context,
	execution *continuationExecutionContext,
	request ContinueWorkflowRequest,
) (stepExecutionOutcome, error) {
	if service.thesis == nil {
		return stepExecutionOutcome{}, fmt.Errorf("thesis evaluation service is required")
	}
	result, err := service.thesis.EvaluateThesisForWorkflow(ctx, thesissvc.EvaluateThesisForWorkflowRequest{
		WorkflowRunID: execution.run.ID,
		BookType:      execution.run.BookType,
		MaxReviews:    service.config.MaxReviewsPerStep,
		DryRun:        request.DryRun,
		Force:         request.Force,
		InitiatedBy:   request.InitiatedBy,
		CorrelationID: request.CorrelationID,
	})
	if err != nil {
		return stepExecutionOutcome{}, err
	}
	return stepExecutionOutcome{
		PartialFailures: workflowStepPartialFailures(
			execution.run.ID,
			domaincommon.WorkflowStepNameEvaluateThesisAndChange,
			result.PartialFailures,
			result.HasFailures(),
			result.Summary.Message,
		),
		Metadata: map[string]any{
			"updatedThesisCount": len(result.UpdatedThesisIDs),
			"failedReviewCount":  len(result.FailedReviewIDs),
			"thesisCreated":      result.ThesisCreated,
			"thesisUpdated":      result.ThesisUpdated,
			"thesisBroken":       result.ThesisBroken,
			"thesisUnderReview":  result.ThesisUnderReview,
		},
	}, nil
}

func (service *workflowContinuationService) executeMapActions(
	ctx context.Context,
	execution *continuationExecutionContext,
	request ContinueWorkflowRequest,
) (stepExecutionOutcome, error) {
	if service.actions == nil {
		return stepExecutionOutcome{}, fmt.Errorf("action mapping service is required")
	}
	result, err := service.actions.MapWorkflowActions(ctx, reviewsvc.MapWorkflowActionsRequest{
		WorkflowRunID: execution.run.ID,
		BookType:      execution.run.BookType,
		MaxReviews:    service.config.MaxReviewsPerStep,
		DryRun:        request.DryRun,
		Force:         request.Force,
		InitiatedBy:   request.InitiatedBy,
		CorrelationID: request.CorrelationID,
	})
	if err != nil {
		return stepExecutionOutcome{}, err
	}
	return stepExecutionOutcome{
		PartialFailures: workflowStepPartialFailures(
			execution.run.ID,
			domaincommon.WorkflowStepNameMapActions,
			result.PartialFailures,
			result.HasFailures(),
			result.Summary.Message,
		),
		Metadata: map[string]any{
			"mappedReviewCount": len(result.MappedReviewIDs),
			"failedReviewCount": len(result.FailedReviewIDs),
		},
	}, nil
}

func (service *workflowContinuationService) executeAssignBuckets(
	ctx context.Context,
	execution *continuationExecutionContext,
	request ContinueWorkflowRequest,
) (stepExecutionOutcome, error) {
	if service.buckets == nil {
		return stepExecutionOutcome{}, fmt.Errorf("bucket assignment service is required")
	}
	result, err := service.buckets.AssignBucketsForWorkflow(ctx, reviewsvc.AssignBucketsForWorkflowRequest{
		WorkflowRunID: execution.run.ID,
		BookType:      execution.run.BookType,
		MaxReviews:    service.config.MaxReviewsPerStep,
		DryRun:        request.DryRun,
		Force:         request.Force,
		InitiatedBy:   request.InitiatedBy,
		CorrelationID: request.CorrelationID,
	})
	if err != nil {
		return stepExecutionOutcome{}, err
	}
	return stepExecutionOutcome{
		PartialFailures: workflowStepPartialFailures(
			execution.run.ID,
			domaincommon.WorkflowStepNameAssignBuckets,
			result.PartialFailures,
			result.HasFailures(),
			result.Summary.Message,
		),
		Metadata: map[string]any{
			"assignedReviewCount": len(result.AssignedReviewIDs),
			"failedReviewCount":   len(result.FailedReviewIDs),
		},
	}, nil
}

func (service *workflowContinuationService) executeBuildCapitalCandidates(
	ctx context.Context,
	execution *continuationExecutionContext,
	request ContinueWorkflowRequest,
) (stepExecutionOutcome, error) {
	if service.candidates == nil {
		return stepExecutionOutcome{}, fmt.Errorf("capital candidate builder service is required")
	}
	result, err := service.candidates.BuildCapitalCandidates(ctx, allocationsvc.BuildCapitalCandidatesRequest{
		WorkflowRunID: execution.run.ID,
		BookType:      execution.run.BookType,
		AsOfDate:      service.now().UTC(),
		MaxCandidates: service.config.MaxCandidates,
		DryRun:        request.DryRun,
		Force:         request.Force,
		InitiatedBy:   request.InitiatedBy,
		CorrelationID: request.CorrelationID,
	})
	if err != nil {
		return stepExecutionOutcome{}, err
	}
	execution.candidateRefs = result.RankedCandidateRefs
	return stepExecutionOutcome{
		PartialFailures: workflowStepPartialFailures(
			execution.run.ID,
			domaincommon.WorkflowStepNameBuildCapitalCandidates,
			result.PartialFailures,
			result.HasFailures(),
			result.Summary.Message,
		),
		Metadata: map[string]any{
			"candidateCount":       result.CandidateCount,
			"rankedCandidateCount": len(result.RankedCandidateRefs),
			"skippedCount":         len(result.SkippedCandidates),
			"ineligibleCount":      len(result.IneligibleCandidates),
		},
	}, nil
}

func (service *workflowContinuationService) executeAllocateCapital(
	ctx context.Context,
	execution *continuationExecutionContext,
	request ContinueWorkflowRequest,
) (stepExecutionOutcome, error) {
	if service.allocator == nil {
		return stepExecutionOutcome{}, fmt.Errorf("capital allocation service is required")
	}
	result, err := service.allocator.AllocateCapital(ctx, allocationsvc.AllocateCapitalRequest{
		WorkflowRunID:  execution.run.ID,
		AllocationDate: service.now().UTC(),
		CandidateRefs:  execution.candidateRefs,
		DryRun:         request.DryRun,
		Force:          request.Force,
		InitiatedBy:    request.InitiatedBy,
		CorrelationID:  request.CorrelationID,
	})
	if err != nil {
		return stepExecutionOutcome{}, err
	}
	return stepExecutionOutcome{
		PartialFailures: workflowStepPartialFailures(
			execution.run.ID,
			domaincommon.WorkflowStepNameAllocateCapital,
			result.PartialFailures,
			result.HasFailures(),
			result.Summary.Message,
		),
		Metadata: map[string]any{
			"capitalAllocationRunId": result.CapitalAllocationRunID.Hex(),
			"allocatedCount":         len(result.AllocatedCandidates),
			"blockedCount":           len(result.BlockedCandidates),
			"unallocatedCash":        result.UnallocatedCash,
		},
	}, nil
}

func (service *workflowContinuationService) executePersistOutputs(
	ctx context.Context,
	execution *continuationExecutionContext,
	request ContinueWorkflowRequest,
) (stepExecutionOutcome, error) {
	if service.projections == nil {
		return stepExecutionOutcome{
			Metadata: map[string]any{
				"noop":   true,
				"reason": "downstream services persist their own outputs and no projection service is configured",
			},
		}, nil
	}
	result, err := service.projections.UpdateProjections(ctx, projectionsvc.UpdateProjectionsRequest{
		WorkflowRunID: execution.run.ID,
		BookType:      execution.run.BookType,
		Targets: []projectionsvc.ProjectionTarget{
			projectionsvc.ProjectionTargetCompanyState,
			projectionsvc.ProjectionTargetPosition,
			projectionsvc.ProjectionTargetReview,
			projectionsvc.ProjectionTargetWorkflow,
		},
		DryRun:        request.DryRun,
		Force:         request.Force,
		InitiatedBy:   request.InitiatedBy,
		CorrelationID: request.CorrelationID,
	})
	if err != nil {
		return stepExecutionOutcome{}, err
	}
	return stepExecutionOutcome{
		PartialFailures: workflowStepPartialFailures(
			execution.run.ID,
			domaincommon.WorkflowStepNamePersistOutputs,
			result.PartialFailures,
			result.HasFailures(),
			result.Summary.Message,
		),
		Metadata: map[string]any{
			"updatedProjectionCount": len(result.UpdatedRefs),
			"skippedProjectionCount": len(result.SkippedRefs),
		},
	}, nil
}

func (service *workflowContinuationService) executePublishRunSummary(
	ctx context.Context,
	execution *continuationExecutionContext,
	request ContinueWorkflowRequest,
) (stepExecutionOutcome, error) {
	return stepExecutionOutcome{
		Metadata: map[string]any{
			"published": true,
			"bookType":  execution.run.BookType,
		},
	}, nil
}

func workflowStepPartialFailures(
	workflowRunID primitive.ObjectID,
	stepName domaincommon.WorkflowStepName,
	partials []servicecommon.PartialFailure,
	hasFailures bool,
	summaryMessage string,
) []servicecommon.PartialFailure {
	if len(partials) > 0 {
		return partials
	}
	if !hasFailures {
		return nil
	}
	if summaryMessage == "" {
		summaryMessage = fmt.Sprintf("step %q completed with partial failures", stepName)
	}
	return []servicecommon.PartialFailure{
		{
			Scope:         servicecommon.FailureScopeContinuation,
			WorkflowRunID: workflowRunID,
			ID:            workflowRunID,
			Code:          fmt.Sprintf("%s_partial_failure", stepName),
			Message:       summaryMessage,
		},
	}
}
