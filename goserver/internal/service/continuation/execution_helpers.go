package continuation

import (
	"context"
	"errors"
	"fmt"
	"time"

	domaincommon "goserver/internal/domain/common"
	domainworkflow "goserver/internal/domain/workflow"
	platformrepo "goserver/internal/platform/repository"
	allocationsvc "goserver/internal/service/allocation"
	servicecommon "goserver/internal/service/common"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

type continuationExecutionContext struct {
	run           *domainworkflow.WorkflowRun
	steps         map[domaincommon.WorkflowStepName]*domainworkflow.WorkflowStepRun
	candidateRefs []allocationsvc.CapitalCandidateRef
}

type stepExecutionOutcome struct {
	StepName        domaincommon.WorkflowStepName
	Executed        bool
	Skipped         bool
	Failed          bool
	PartialFailures []servicecommon.PartialFailure
	Metadata        map[string]any
}

type stepExecutionFunc func(context.Context, *continuationExecutionContext, ContinueWorkflowRequest) (stepExecutionOutcome, error)

func (service *workflowContinuationService) continueOneWorkflow(
	ctx context.Context,
	request ContinueWorkflowRequest,
) (*ContinueWorkflowResult, error) {
	if service.decision == nil {
		return nil, fmt.Errorf("workflow continuation decision service is required")
	}

	decision, err := service.decision.EvaluateWorkflowContinuation(ctx, EvaluateWorkflowContinuationRequest{
		WorkflowRunID: request.WorkflowRunID,
		BookType:      request.BookType,
		Force:         request.Force,
		InitiatedBy:   request.InitiatedBy,
		CorrelationID: request.CorrelationID,
	})
	if err != nil {
		return nil, fmt.Errorf("evaluate continuation readiness: %w", err)
	}

	result := &ContinueWorkflowResult{
		WorkflowRunID:           request.WorkflowRunID,
		BookType:                decision.BookType,
		CurrentStatus:           decision.CurrentStatus,
		Readiness:               decision.Readiness,
		DryRun:                  request.DryRun,
		NextSuggestedStep:       decision.NextSuggestedStep,
		Blockers:                decision.Blockers,
		PartialFailures:         nil,
		ContinuedWorkflowRunIDs: nil,
	}
	if !decision.ReadyToContinueNow() {
		result.Blocked = true
		result.StillBlockedWorkflowRunIDs = []primitive.ObjectID{request.WorkflowRunID}
		return result, nil
	}

	execution, err := service.loadExecutionContext(ctx, request.WorkflowRunID)
	if err != nil {
		return nil, fmt.Errorf("load continuation execution context: %w", err)
	}
	if execution.run.IsTerminal() && !request.Force {
		result.Blocked = true
		result.Blockers = append(result.Blockers, workflowBlocker(
			request.WorkflowRunID,
			"workflow_terminal",
			fmt.Sprintf("workflow is terminal with status %q", execution.run.Status),
			servicecommon.ValidationIssueSeverityWarning,
			"",
		))
		result.StillBlockedWorkflowRunIDs = []primitive.ObjectID{request.WorkflowRunID}
		return result, nil
	}
	if decision.NextSuggestedStep != "" && !containsStep(postAIStepSequence(execution.run.BookType), decision.NextSuggestedStep) {
		result.Blocked = true
		result.Blockers = append(result.Blockers, workflowBlocker(
			request.WorkflowRunID,
			"invalid_workflow_state",
			fmt.Sprintf("next suggested step %q is not a post-AI continuation step", decision.NextSuggestedStep),
			servicecommon.ValidationIssueSeverityWarning,
			decision.NextSuggestedStep,
		))
		result.StillBlockedWorkflowRunIDs = []primitive.ObjectID{request.WorkflowRunID}
		return result, nil
	}

	steps := executableStepsFromDecision(decision, execution, request.AllowedStepRange)
	result.PlannedSteps = steps
	if request.DryRun {
		result.Continued = len(steps) > 0
		result.ContinuedWorkflowRunIDs = []primitive.ObjectID{request.WorkflowRunID}
		result.Completed = len(steps) == 0 || service.allPostAIStepsDone(execution)
		if result.Completed {
			result.CompletedWorkflowRunIDs = []primitive.ObjectID{request.WorkflowRunID}
		}
		return result, nil
	}

	if err := service.markWorkflowRunning(ctx, execution.run, request); err != nil {
		return nil, fmt.Errorf("mark workflow running: %w", err)
	}

	for _, stepName := range steps {
		outcome, err := service.executeContinuationStep(ctx, execution, stepName, request)
		result.PartialFailures = append(result.PartialFailures, outcome.PartialFailures...)
		if outcome.Skipped {
			result.SkippedSteps = append(result.SkippedSteps, stepName)
		}
		if outcome.Executed {
			result.ExecutedSteps = append(result.ExecutedSteps, stepName)
			result.Continued = true
		}
		if err != nil {
			result.Failed = true
			result.FailedSteps = append(result.FailedSteps, stepName)
			result.FailedWorkflowRunIDs = []primitive.ObjectID{request.WorkflowRunID}
			result.PartialFailures = append(result.PartialFailures, stepPartialFailure(request.WorkflowRunID, stepName, err))
			_ = service.markWorkflowPartiallyCompleted(ctx, execution.run, request, "continuation step failed")
			result.NextSuggestedStep = stepName
			return result, nil
		}
	}

	result.ContinuedWorkflowRunIDs = []primitive.ObjectID{request.WorkflowRunID}
	result.Completed = service.allPostAIStepsDone(execution)
	if result.Completed {
		if err := service.markWorkflowCompleted(ctx, execution.run, request); err != nil {
			return nil, fmt.Errorf("mark workflow completed: %w", err)
		}
		result.CompletedWorkflowRunIDs = []primitive.ObjectID{request.WorkflowRunID}
		result.NextSuggestedStep = ""
		return result, nil
	}

	if err := service.markWorkflowPartiallyCompleted(ctx, execution.run, request, "continuation executed allowed step range"); err != nil {
		return nil, fmt.Errorf("mark workflow partially completed: %w", err)
	}
	result.NextSuggestedStep = firstIncompleteStep(postAIStepSequence(execution.run.BookType), execution.steps)
	return result, nil
}

func (service *workflowContinuationService) loadExecutionContext(
	ctx context.Context,
	workflowRunID primitive.ObjectID,
) (*continuationExecutionContext, error) {
	if service.workflowRuns == nil {
		return nil, fmt.Errorf("workflow run repository is required")
	}
	run, err := service.workflowRuns.GetByID(ctx, workflowRunID)
	if err != nil {
		return nil, fmt.Errorf("load workflow run: %w", err)
	}
	if run == nil {
		return nil, fmt.Errorf("load workflow run: %w", platformrepo.ErrNotFound)
	}

	stepMap := map[domaincommon.WorkflowStepName]*domainworkflow.WorkflowStepRun{}
	if service.workflowSteps != nil {
		steps, err := service.workflowSteps.ListByWorkflowRunID(ctx, workflowRunID, platformrepo.WorkflowStepRunListOptions{
			Pagination: platformrepo.PageOptions{PageSize: service.config.MaxPageSize},
			Sort:       platformrepo.WorkflowStepRunSortOption{By: platformrepo.WorkflowStepRunSortByUpdatedAt, Order: platformrepo.SortOrderAscending},
		})
		if err != nil {
			return nil, fmt.Errorf("load workflow steps: %w", err)
		}
		if steps != nil {
			stepMap = stepStatusByName(steps.Items)
		}
	}
	if len(stepMap) == 0 {
		stepMap = stepStatusByName(stepRunsFromWorkflowRefs(run))
	}
	return &continuationExecutionContext{
		run:   run,
		steps: stepMap,
	}, nil
}

func executableStepsFromDecision(
	decision *EvaluateWorkflowContinuationResult,
	execution *continuationExecutionContext,
	stepRange servicecommon.StepRange,
) []domaincommon.WorkflowStepName {
	if decision == nil || execution == nil || execution.run == nil {
		return nil
	}
	sequence := postAIStepSequence(execution.run.BookType)
	if len(sequence) == 0 {
		return nil
	}
	start := decision.NextSuggestedStep
	if start == "" || !containsStep(sequence, start) {
		start = firstIncompleteStep(sequence, execution.steps)
	}
	if start == "" {
		return nil
	}
	remaining := remainingSteps(sequence, start)
	return applyAllowedStepRange(remaining, stepRange)
}

func remainingSteps(sequence []domaincommon.WorkflowStepName, start domaincommon.WorkflowStepName) []domaincommon.WorkflowStepName {
	index := stepIndex(sequence, start)
	if index < 0 {
		return nil
	}
	remaining := make([]domaincommon.WorkflowStepName, len(sequence[index:]))
	copy(remaining, sequence[index:])
	return remaining
}

func applyAllowedStepRange(
	steps []domaincommon.WorkflowStepName,
	stepRange servicecommon.StepRange,
) []domaincommon.WorkflowStepName {
	if len(steps) == 0 || (stepRange.Start == "" && stepRange.End == "") {
		return steps
	}
	filtered := make([]domaincommon.WorkflowStepName, 0, len(steps))
	inRange := stepRange.Start == ""
	for _, stepName := range steps {
		if stepRange.Start != "" && stepName == stepRange.Start {
			inRange = true
		}
		if inRange {
			filtered = append(filtered, stepName)
		}
		if stepRange.End != "" && stepName == stepRange.End {
			break
		}
	}
	return filtered
}

func postAIStepSequence(bookType domaincommon.BookType) []domaincommon.WorkflowStepName {
	sequence := stepSequenceForBook(bookType)
	first := firstPostAIStep(bookType)
	index := stepIndex(sequence, first)
	if index < 0 {
		return nil
	}
	postAI := make([]domaincommon.WorkflowStepName, len(sequence[index:]))
	copy(postAI, sequence[index:])
	return postAI
}

func containsStep(steps []domaincommon.WorkflowStepName, stepName domaincommon.WorkflowStepName) bool {
	return stepIndex(steps, stepName) >= 0
}

func shouldSkipCompletedStep(step *domainworkflow.WorkflowStepRun, force bool) bool {
	return !force && isStepComplete(step)
}

func canRetryFailedStep(step *domainworkflow.WorkflowStepRun, force bool) bool {
	return step == nil || step.Status != domaincommon.WorkflowStepStatusFailed || force
}

func (service *workflowContinuationService) allPostAIStepsDone(execution *continuationExecutionContext) bool {
	for _, stepName := range postAIStepSequence(execution.run.BookType) {
		if !isStepComplete(execution.steps[stepName]) {
			return false
		}
	}
	return true
}

func (service *workflowContinuationService) markWorkflowRunning(
	ctx context.Context,
	run *domainworkflow.WorkflowRun,
	request ContinueWorkflowRequest,
) error {
	if service.workflowRuns == nil || run == nil {
		return nil
	}
	if run.Status == domaincommon.WorkflowRunStatusRunning {
		return nil
	}
	if run.IsTerminal() {
		return fmt.Errorf("cannot continue terminal workflow status %q", run.Status)
	}
	updated, err := service.workflowRuns.UpdateStatus(ctx, run.ID, platformrepo.WorkflowRunStatusPatch{
		NextStatus: domaincommon.WorkflowRunStatusRunning,
		ExpectedCurrentStatuses: []domaincommon.WorkflowRunStatus{
			domaincommon.WorkflowRunStatusCreated,
			domaincommon.WorkflowRunStatusRunning,
			domaincommon.WorkflowRunStatusWaitingExternal,
			domaincommon.WorkflowRunStatusPartiallyCompleted,
		},
		Mutation: mutationMetadata(service.now().UTC(), request.InitiatedBy, "workflow continuation started"),
	})
	if err != nil {
		if errors.Is(err, platformrepo.ErrPreconditionFailed) {
			return nil
		}
		return err
	}
	if updated != nil {
		*run = *updated
	}
	return nil
}

func (service *workflowContinuationService) markWorkflowPartiallyCompleted(
	ctx context.Context,
	run *domainworkflow.WorkflowRun,
	request ContinueWorkflowRequest,
	reason string,
) error {
	if service.workflowRuns == nil || run == nil || run.IsTerminal() {
		return nil
	}
	note := reason
	updated, err := service.workflowRuns.UpdateStatus(ctx, run.ID, platformrepo.WorkflowRunStatusPatch{
		NextStatus: domaincommon.WorkflowRunStatusPartiallyCompleted,
		Notes:      &note,
		ExpectedCurrentStatuses: []domaincommon.WorkflowRunStatus{
			domaincommon.WorkflowRunStatusRunning,
			domaincommon.WorkflowRunStatusWaitingExternal,
			domaincommon.WorkflowRunStatusPartiallyCompleted,
		},
		Mutation: mutationMetadata(service.now().UTC(), request.InitiatedBy, reason),
	})
	if err != nil {
		if errors.Is(err, platformrepo.ErrPreconditionFailed) {
			return nil
		}
		return err
	}
	if updated != nil {
		*run = *updated
	}
	return nil
}

func (service *workflowContinuationService) markWorkflowCompleted(
	ctx context.Context,
	run *domainworkflow.WorkflowRun,
	request ContinueWorkflowRequest,
) error {
	if service.workflowRuns == nil || run == nil {
		return nil
	}
	if run.Status == domaincommon.WorkflowRunStatusCompleted {
		return nil
	}
	if run.Status == domaincommon.WorkflowRunStatusFailed || run.Status == domaincommon.WorkflowRunStatusCancelled {
		return fmt.Errorf("cannot complete terminal workflow status %q", run.Status)
	}
	note := "workflow continuation completed"
	updated, err := service.workflowRuns.MarkCompleted(ctx, run.ID, platformrepo.WorkflowRunCompletionPatch{
		CompletedAt: service.now().UTC(),
		Notes:       &note,
		ExpectedCurrentStatuses: []domaincommon.WorkflowRunStatus{
			domaincommon.WorkflowRunStatusRunning,
			domaincommon.WorkflowRunStatusWaitingExternal,
			domaincommon.WorkflowRunStatusPartiallyCompleted,
		},
		Mutation: mutationMetadata(service.now().UTC(), request.InitiatedBy, "workflow continuation completed"),
	})
	if err != nil {
		if errors.Is(err, platformrepo.ErrPreconditionFailed) {
			return nil
		}
		return err
	}
	if updated != nil {
		*run = *updated
	}
	return nil
}

func mutationMetadata(at time.Time, actor string, reason string) platformrepo.MutationMetadata {
	return platformrepo.MutationMetadata{
		OccurredAt: at,
		Actor:      actor,
		Reason:     reason,
	}
}
