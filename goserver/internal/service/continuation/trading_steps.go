package continuation

import (
	"context"
	"fmt"

	domaincommon "goserver/internal/domain/common"
	projectionsvc "goserver/internal/service/projection"
)

func (service *workflowContinuationService) executeTradingStep(
	ctx context.Context,
	execution *continuationExecutionContext,
	stepName domaincommon.WorkflowStepName,
	request ContinueWorkflowRequest,
) (stepExecutionOutcome, error) {
	switch stepName {
	case domaincommon.WorkflowStepNameApproveTradeCandidates:
		return service.skipStep(ctx, execution, stepName, request, "trading candidate approval service is not configured")
	case domaincommon.WorkflowStepNamePersistTradingReview:
		if service.projections == nil {
			return service.skipStep(ctx, execution, stepName, request, "trading persistence service is not configured")
		}
		return service.executeStep(ctx, execution, stepName, request, service.executePersistTradingReview)
	case domaincommon.WorkflowStepNamePublishTradingRunSummary:
		return service.executeStep(ctx, execution, stepName, request, service.executePublishTradingRunSummary)
	default:
		return stepExecutionOutcome{StepName: stepName}, fmt.Errorf("unsupported trading continuation step %q", stepName)
	}
}

func (service *workflowContinuationService) executePersistTradingReview(
	ctx context.Context,
	execution *continuationExecutionContext,
	request ContinueWorkflowRequest,
) (stepExecutionOutcome, error) {
	result, err := service.projections.UpdateProjections(ctx, projectionsvc.UpdateProjectionsRequest{
		WorkflowRunID: execution.run.ID,
		BookType:      execution.run.BookType,
		Targets: []projectionsvc.ProjectionTarget{
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
			domaincommon.WorkflowStepNamePersistTradingReview,
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

func (service *workflowContinuationService) executePublishTradingRunSummary(
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
