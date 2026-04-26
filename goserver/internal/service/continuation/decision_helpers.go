package continuation

import (
	"context"
	"fmt"

	domainaijob "goserver/internal/domain/aijob"
	domaincommon "goserver/internal/domain/common"
	domainreview "goserver/internal/domain/review"
	domainworkflow "goserver/internal/domain/workflow"
	platformrepo "goserver/internal/platform/repository"
	servicecommon "goserver/internal/service/common"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

type continuationContext struct {
	run        *domainworkflow.WorkflowRun
	steps      []*domainworkflow.WorkflowStepRun
	batchJobs  []*domainaijob.AIBatchJob
	batchItems []*domainaijob.AIBatchItem
	reviews    []*domainreview.CompanyReview
}

type continuationEvaluationOptions struct {
	BookType domaincommon.BookType
	Force    bool
}

func (service *workflowContinuationDecisionService) loadContinuationContext(
	ctx context.Context,
	workflowRunID primitive.ObjectID,
) (continuationContext, error) {
	if service.workflowRuns == nil {
		return continuationContext{}, fmt.Errorf("workflow run repository is required")
	}

	run, err := service.workflowRuns.GetByID(ctx, workflowRunID)
	if err != nil {
		return continuationContext{}, fmt.Errorf("load workflow run: %w", err)
	}
	if run == nil {
		return continuationContext{}, fmt.Errorf("load workflow run: %w", platformrepo.ErrNotFound)
	}

	snapshot := continuationContext{run: run}
	if service.workflowSteps != nil {
		steps, err := service.workflowSteps.ListByWorkflowRunID(ctx, workflowRunID, platformrepo.WorkflowStepRunListOptions{
			Pagination: platformrepo.PageOptions{PageSize: service.config.MaxPageSize},
			Sort:       platformrepo.WorkflowStepRunSortOption{By: platformrepo.WorkflowStepRunSortByUpdatedAt, Order: platformrepo.SortOrderAscending},
		})
		if err != nil {
			return continuationContext{}, fmt.Errorf("load workflow steps: %w", err)
		}
		if steps != nil {
			snapshot.steps = steps.Items
		}
	}
	if len(snapshot.steps) == 0 {
		snapshot.steps = stepRunsFromWorkflowRefs(run)
	}

	if service.batchJobs != nil {
		jobs, err := service.batchJobs.ListByWorkflowRunID(ctx, workflowRunID, platformrepo.AIBatchJobListOptions{
			Pagination: platformrepo.PageOptions{PageSize: service.config.MaxPageSize},
			Sort:       platformrepo.AIBatchJobSortOption{By: platformrepo.AIBatchJobSortByUpdatedAt, Order: platformrepo.SortOrderAscending},
		})
		if err != nil {
			return continuationContext{}, fmt.Errorf("load batch jobs: %w", err)
		}
		if jobs != nil {
			snapshot.batchJobs = jobs.Items
		}
	}

	if service.batchItems != nil {
		items, err := service.batchItems.List(ctx, platformrepo.AIBatchItemFilter{
			WorkflowRunIDs: []primitive.ObjectID{workflowRunID},
		}, platformrepo.AIBatchItemListOptions{
			Pagination: platformrepo.PageOptions{PageSize: service.config.MaxPageSize},
			Sort:       platformrepo.AIBatchItemSortOption{By: platformrepo.AIBatchItemSortByUpdatedAt, Order: platformrepo.SortOrderAscending},
		})
		if err != nil {
			return continuationContext{}, fmt.Errorf("load batch items: %w", err)
		}
		if items != nil {
			snapshot.batchItems = items.Items
		}
	}

	if service.reviews != nil {
		reviews, err := service.reviews.ListByWorkflowRun(ctx, workflowRunID, platformrepo.CompanyReviewListOptions{
			Pagination: platformrepo.PageOptions{PageSize: service.config.MaxPageSize},
			Sort:       platformrepo.CompanyReviewSortOption{By: platformrepo.CompanyReviewSortByUpdatedAt, Order: platformrepo.SortOrderAscending},
		})
		if err != nil {
			return continuationContext{}, fmt.Errorf("load company reviews: %w", err)
		}
		if reviews != nil {
			snapshot.reviews = reviews.Items
		}
	}

	return snapshot, nil
}

func stepRunsFromWorkflowRefs(run *domainworkflow.WorkflowRun) []*domainworkflow.WorkflowStepRun {
	if run == nil || len(run.StepStatuses) == 0 {
		return nil
	}
	steps := make([]*domainworkflow.WorkflowStepRun, 0, len(run.StepStatuses))
	for _, reference := range run.StepStatuses {
		step := &domainworkflow.WorkflowStepRun{
			ID:            reference.StepRunID,
			WorkflowRunID: run.ID,
			StepName:      reference.StepName,
			Status:        reference.Status,
			StartedAt:     reference.StartedAt,
			CompletedAt:   reference.CompletedAt,
			ErrorSummary:  reference.ErrorSummary,
			CreatedAt:     run.CreatedAt,
			UpdatedAt:     run.UpdatedAt,
			SchemaVersion: domaincommon.SchemaVersion1,
		}
		steps = append(steps, step)
	}
	return steps
}

// evaluateLoadedWorkflow is intentionally read-only. Continuation decisions are
// advisory snapshots because workers can advance jobs, items, reviews, and steps
// immediately after this service reads them.
func (service *workflowContinuationDecisionService) evaluateLoadedWorkflow(
	snapshot continuationContext,
	options continuationEvaluationOptions,
) *EvaluateWorkflowContinuationResult {
	run := snapshot.run
	result := &EvaluateWorkflowContinuationResult{
		WorkflowRunID: run.ID,
		BookType:      run.BookType,
		CurrentStatus: run.Status,
		Counts:        buildContinuationCounts(snapshot),
	}

	if options.BookType != "" && options.BookType != run.BookType {
		result.Blockers = append(result.Blockers, workflowBlocker(
			run.ID,
			"invalid_workflow_state",
			fmt.Sprintf("request bookType %q does not match workflow bookType %q", options.BookType, run.BookType),
			servicecommon.ValidationIssueSeverityError,
			"",
		))
		return service.finishDecision(result, WorkflowContinuationReadinessInvalidState, "")
	}

	sequence := stepSequenceForBook(run.BookType)
	if len(sequence) == 0 {
		result.Blockers = append(result.Blockers, workflowBlocker(
			run.ID,
			"invalid_workflow_state",
			fmt.Sprintf("unsupported workflow bookType %q", run.BookType),
			servicecommon.ValidationIssueSeverityError,
			"",
		))
		return service.finishDecision(result, WorkflowContinuationReadinessInvalidState, "")
	}

	stepsByName := stepStatusByName(snapshot.steps)
	result.NextSuggestedStep = determineNextSuggestedStep(run.BookType, sequence, stepsByName, result.Counts)

	switch run.Status {
	case domaincommon.WorkflowRunStatusCompleted:
		result.Blockers = append(result.Blockers, workflowBlocker(
			run.ID,
			"workflow_terminal",
			"workflow run is already completed",
			servicecommon.ValidationIssueSeverityInfo,
			"",
		))
		return service.finishDecision(result, WorkflowContinuationReadinessAlreadyCompleted, result.NextSuggestedStep)
	case domaincommon.WorkflowRunStatusFailed, domaincommon.WorkflowRunStatusCancelled:
		result.Blockers = append(result.Blockers, workflowBlocker(
			run.ID,
			"workflow_terminal",
			fmt.Sprintf("workflow run is terminal with status %q", run.Status),
			servicecommon.ValidationIssueSeverityError,
			"",
		))
		return service.finishDecision(result, WorkflowContinuationReadinessFailedTerminal, result.NextSuggestedStep)
	}

	if options.Force {
		forced := service.finishDecision(result, WorkflowContinuationReadinessReadyToContinue, result.NextSuggestedStep)
		forced.ContinuationReason = ContinuationReasonForced
		return forced
	}

	blockers := make([]servicecommon.BlockingCondition, 0)
	blockers = append(blockers, detectStepBlockers(snapshot, sequence, stepsByName)...)
	blockers = append(blockers, detectExternalWaitBlockers(snapshot, result.Counts)...)
	blockers = append(blockers, detectValidationBlockers(snapshot, result.Counts, service.config.AllowPartialSuccess)...)
	blockers = append(blockers, detectMaterializationBlockers(snapshot, result.Counts)...)
	blockers = append(blockers, detectFinalizationBlockers(snapshot, result.Counts)...)
	blockers = append(blockers, detectPostAIPrerequisiteBlockers(snapshot, sequence, stepsByName, result.Counts)...)
	result.Blockers = blockers

	readiness := classifyWorkflowReadiness(result.Blockers)
	if readiness == "" {
		readiness = WorkflowContinuationReadinessReadyToContinue
	}
	return service.finishDecision(result, readiness, result.NextSuggestedStep)
}

func (service *workflowContinuationDecisionService) finishDecision(
	result *EvaluateWorkflowContinuationResult,
	readiness WorkflowContinuationReadiness,
	nextStep domaincommon.WorkflowStepName,
) *EvaluateWorkflowContinuationResult {
	result.Readiness = readiness
	result.NextSuggestedStep = nextStep
	result.WaitingOnExternalJobs = hasBlockerCode(result.Blockers,
		"batch_jobs_not_submitted",
		"batch_jobs_still_running",
		"batch_jobs_partially_completed",
		"batch_items_not_reconciled",
	)
	result.WaitingOnValidation = hasBlockerCode(result.Blockers, "batch_items_not_validated")
	result.WaitingOnMaterialization = hasBlockerCode(result.Blockers, "reviews_not_materialized", "no_reviews_created")
	result.WaitingOnFinalization = hasBlockerCode(result.Blockers, "reviews_not_finalized", "review_finalization_preconditions_failed")
	result.ReadyToContinue = readiness == WorkflowContinuationReadinessReadyToContinue && len(result.Blockers) == 0
	result.ContinuationReason = continuationReasonForReadiness(readiness)
	return result
}

func detectStepBlockers(
	snapshot continuationContext,
	sequence []domaincommon.WorkflowStepName,
	stepsByName map[domaincommon.WorkflowStepName]*domainworkflow.WorkflowStepRun,
) []servicecommon.BlockingCondition {
	blockers := make([]servicecommon.BlockingCondition, 0)
	for _, stepName := range sequence {
		step := stepsByName[stepName]
		if step == nil {
			continue
		}
		if isStepFailed(step) {
			blockers = append(blockers, stepBlocker(
				step,
				"required_step_failed",
				fmt.Sprintf("workflow step %q failed", step.StepName),
				servicecommon.ValidationIssueSeverityError,
			))
		}
		if step.Status == domaincommon.WorkflowStepStatusRunning {
			blockers = append(blockers, stepBlocker(
				step,
				"workflow_step_still_running",
				fmt.Sprintf("workflow step %q is still running", step.StepName),
				servicecommon.ValidationIssueSeverityWarning,
			))
		}
	}
	return blockers
}

func detectExternalWaitBlockers(
	snapshot continuationContext,
	counts WorkflowContinuationCounts,
) []servicecommon.BlockingCondition {
	blockers := make([]servicecommon.BlockingCondition, 0)
	bookType := snapshot.run.BookType

	if counts.BatchJobs.Created > 0 {
		blockers = appendOneBlocker(blockers, batchJobBlocker(
			firstBatchJobWithStatus(snapshot.batchJobs, domaincommon.AIBatchJobStatusCreated),
			"batch_jobs_not_submitted",
			"workflow has AI batch jobs waiting to be submitted",
			servicecommon.ValidationIssueSeverityWarning,
			submitBatchStep(bookType),
		))
	}
	if counts.BatchJobs.Submitted+counts.BatchJobs.Running > 0 {
		blockers = appendOneBlocker(blockers, batchJobBlocker(
			firstBatchJobWithAnyStatus(snapshot.batchJobs, domaincommon.AIBatchJobStatusSubmitted, domaincommon.AIBatchJobStatusRunning),
			"batch_jobs_still_running",
			"workflow has AI batch jobs still running at the provider",
			servicecommon.ValidationIssueSeverityInfo,
			pollAndReconcileStep(bookType),
		))
	}
	if counts.BatchJobs.PartiallyCompleted > 0 {
		blockers = appendOneBlocker(blockers, batchJobBlocker(
			firstBatchJobWithStatus(snapshot.batchJobs, domaincommon.AIBatchJobStatusPartiallyCompleted),
			"batch_jobs_partially_completed",
			"workflow has partially completed AI batch jobs that still require polling or reconciliation",
			servicecommon.ValidationIssueSeverityInfo,
			pollAndReconcileStep(bookType),
		))
	}
	if counts.BatchItems.Unreconciled > 0 && counts.BatchJobs.Completed+counts.BatchJobs.Failed+counts.BatchJobs.TimedOut+counts.BatchJobs.Cancelled > 0 {
		blockers = appendOneBlocker(blockers, batchItemBlocker(
			firstUnreconciledBatchItem(snapshot.batchItems),
			"batch_items_not_reconciled",
			"workflow has AI batch items that have not reached a reconciled terminal state",
			servicecommon.ValidationIssueSeverityWarning,
			pollAndReconcileStep(bookType),
		))
	}

	allJobsTerminalFailed := counts.BatchJobs.Total > 0 &&
		counts.BatchJobs.Completed == 0 &&
		counts.BatchJobs.Created+counts.BatchJobs.Submitted+counts.BatchJobs.Running+counts.BatchJobs.PartiallyCompleted == 0 &&
		counts.BatchJobs.Failed+counts.BatchJobs.TimedOut+counts.BatchJobs.Cancelled == counts.BatchJobs.Total
	if allJobsTerminalFailed && counts.BatchItems.Valid+counts.Reviews.Materialized+counts.Reviews.Finalized == 0 {
		blockers = appendOneBlocker(blockers, batchJobBlocker(
			firstFailedBatchJob(snapshot.batchJobs),
			"batch_jobs_failed",
			"all AI batch jobs for the workflow are terminal without usable results",
			servicecommon.ValidationIssueSeverityError,
			pollAndReconcileStep(bookType),
		))
	}

	return blockers
}

func detectValidationBlockers(
	snapshot continuationContext,
	counts WorkflowContinuationCounts,
	allowPartialSuccess bool,
) []servicecommon.BlockingCondition {
	blockers := make([]servicecommon.BlockingCondition, 0)
	bookType := snapshot.run.BookType

	if counts.BatchItems.PendingValidation > 0 {
		blockers = appendOneBlocker(blockers, batchItemBlocker(
			firstPendingValidationBatchItem(snapshot.batchItems),
			"batch_items_not_validated",
			"workflow has completed AI batch items awaiting validation",
			servicecommon.ValidationIssueSeverityWarning,
			validateAIOutputsStep(bookType),
		))
	}

	usableOutputs := counts.BatchItems.Valid + counts.Reviews.Materialized + counts.Reviews.Finalized
	if counts.BatchItems.Invalid+counts.BatchItems.InvalidOutput > 0 && (!allowPartialSuccess || usableOutputs == 0) {
		blockers = appendOneBlocker(blockers, batchItemBlocker(
			firstInvalidBatchItem(snapshot.batchItems),
			"batch_items_invalid",
			"workflow has invalid AI batch outputs and no partial-success path is available",
			servicecommon.ValidationIssueSeverityError,
			validateAIOutputsStep(bookType),
		))
	}
	if counts.BatchItems.Failed > 0 && (!allowPartialSuccess || usableOutputs == 0) {
		blockers = appendOneBlocker(blockers, batchItemBlocker(
			firstFailedBatchItem(snapshot.batchItems),
			"batch_items_failed",
			"workflow has failed AI batch items and no usable successful outputs",
			servicecommon.ValidationIssueSeverityError,
			validateAIOutputsStep(bookType),
		))
	}

	return blockers
}

func detectMaterializationBlockers(
	snapshot continuationContext,
	counts WorkflowContinuationCounts,
) []servicecommon.BlockingCondition {
	blockers := make([]servicecommon.BlockingCondition, 0)
	bookType := snapshot.run.BookType
	stepName := materializeReviewsStep(bookType)

	if counts.BatchItems.Valid > 0 && counts.Reviews.Total == 0 {
		blockers = append(blockers, workflowBlocker(
			snapshot.run.ID,
			"no_reviews_created",
			"workflow has valid AI items but no linked review records",
			servicecommon.ValidationIssueSeverityError,
			stepName,
		))
		return blockers
	}
	if counts.BatchItems.Materializable > 0 {
		blockers = appendOneBlocker(blockers, batchItemBlocker(
			firstMaterializableBatchItem(snapshot.batchItems),
			"reviews_not_materialized",
			"workflow has validated AI items that have not been materialized into reviews",
			servicecommon.ValidationIssueSeverityWarning,
			stepName,
		))
	}
	if counts.Reviews.MaterializationIncomplete > 0 {
		blockers = appendOneBlocker(blockers, reviewBlocker(
			firstReviewWithAnyLifecycle(snapshot.reviews,
				domaincommon.ReviewLifecycleStateAICompletedUnvalidated,
				domaincommon.ReviewLifecycleStateValidationFailed,
			),
			"reviews_not_materialized",
			"workflow has reviews awaiting AI result materialization",
			servicecommon.ValidationIssueSeverityWarning,
			stepName,
		))
	}
	return blockers
}

func detectFinalizationBlockers(
	snapshot continuationContext,
	counts WorkflowContinuationCounts,
) []servicecommon.BlockingCondition {
	blockers := make([]servicecommon.BlockingCondition, 0)
	if counts.Reviews.FinalizationIncomplete == 0 {
		return blockers
	}

	stepName := materializeReviewsStep(snapshot.run.BookType)
	if counts.Reviews.Finalizable == 0 {
		blockers = appendOneBlocker(blockers, reviewBlocker(
			firstReviewWithAnyLifecycle(snapshot.reviews, domaincommon.ReviewLifecycleStateAIValidated),
			"review_finalization_preconditions_failed",
			"workflow has AI-validated reviews that are not finalizable yet",
			servicecommon.ValidationIssueSeverityError,
			stepName,
		))
		return blockers
	}

	blockers = appendOneBlocker(blockers, reviewBlocker(
		firstReviewWithAnyLifecycle(snapshot.reviews, domaincommon.ReviewLifecycleStateAIValidated),
		"reviews_not_finalized",
		"workflow has materialized reviews awaiting finalization",
		servicecommon.ValidationIssueSeverityWarning,
		stepName,
	))
	return blockers
}

func detectPostAIPrerequisiteBlockers(
	snapshot continuationContext,
	sequence []domaincommon.WorkflowStepName,
	stepsByName map[domaincommon.WorkflowStepName]*domainworkflow.WorkflowStepRun,
	counts WorkflowContinuationCounts,
) []servicecommon.BlockingCondition {
	firstIncomplete := firstIncompleteStep(sequence, stepsByName)
	if firstIncomplete == "" || !isStepAtOrAfter(sequence, firstIncomplete, firstPostAIStep(snapshot.run.BookType)) {
		return nil
	}
	if counts.Reviews.Finalized+counts.Reviews.Superseded > 0 {
		return nil
	}
	if counts.BatchItems.Total == 0 && counts.Reviews.Total == 0 {
		return []servicecommon.BlockingCondition{
			workflowBlocker(
				snapshot.run.ID,
				"no_reviews_created",
				"workflow cannot enter post-AI steps because no review records were created",
				servicecommon.ValidationIssueSeverityError,
				firstIncomplete,
			),
		}
	}
	return []servicecommon.BlockingCondition{
		workflowBlocker(
			snapshot.run.ID,
			"reviews_not_finalized",
			"workflow cannot enter post-AI steps until at least one review is finalized",
			servicecommon.ValidationIssueSeverityWarning,
			materializeReviewsStep(snapshot.run.BookType),
		),
	}
}

func classifyWorkflowReadiness(blockers []servicecommon.BlockingCondition) WorkflowContinuationReadiness {
	if len(blockers) == 0 {
		return WorkflowContinuationReadinessReadyToContinue
	}
	if hasBlockerCode(blockers, "invalid_workflow_state") {
		return WorkflowContinuationReadinessInvalidState
	}
	if hasBlockerCode(blockers,
		"required_step_failed",
		"batch_jobs_failed",
		"batch_items_failed",
		"batch_items_invalid",
		"review_finalization_preconditions_failed",
	) {
		return WorkflowContinuationReadinessBlockedByFailures
	}
	if hasBlockerCode(blockers,
		"batch_jobs_not_submitted",
		"batch_jobs_still_running",
		"batch_jobs_partially_completed",
		"batch_items_not_reconciled",
	) {
		return WorkflowContinuationReadinessWaitingExternal
	}
	if hasBlockerCode(blockers, "batch_items_not_validated") {
		return WorkflowContinuationReadinessWaitingValidation
	}
	if hasBlockerCode(blockers, "reviews_not_materialized", "no_reviews_created") {
		return WorkflowContinuationReadinessWaitingMaterialize
	}
	if hasBlockerCode(blockers, "reviews_not_finalized") {
		return WorkflowContinuationReadinessWaitingFinalization
	}
	return WorkflowContinuationReadinessInvalidState
}

func continuationReasonForReadiness(readiness WorkflowContinuationReadiness) ContinuationReason {
	switch readiness {
	case WorkflowContinuationReadinessReadyToContinue:
		return ContinuationReasonAsyncResolved
	case WorkflowContinuationReadinessAlreadyCompleted, WorkflowContinuationReadinessFailedTerminal:
		return ContinuationReasonWorkflowTerminal
	case WorkflowContinuationReadinessBlockedByFailures, WorkflowContinuationReadinessInvalidState:
		return ContinuationReasonPreconditionsFailed
	default:
		return ContinuationReasonStillBlocked
	}
}

func hasBlockerCode(blockers []servicecommon.BlockingCondition, codes ...string) bool {
	if len(blockers) == 0 || len(codes) == 0 {
		return false
	}
	allowed := make(map[string]struct{}, len(codes))
	for _, code := range codes {
		allowed[code] = struct{}{}
	}
	for _, blocker := range blockers {
		if _, ok := allowed[blocker.Code]; ok {
			return true
		}
	}
	return false
}

func firstBatchJobWithStatus(jobs []*domainaijob.AIBatchJob, status domaincommon.AIBatchJobStatus) *domainaijob.AIBatchJob {
	return firstBatchJobWithAnyStatus(jobs, status)
}

func firstBatchJobWithAnyStatus(jobs []*domainaijob.AIBatchJob, statuses ...domaincommon.AIBatchJobStatus) *domainaijob.AIBatchJob {
	for _, job := range jobs {
		if job == nil {
			continue
		}
		for _, status := range statuses {
			if job.Status == status {
				return job
			}
		}
	}
	return nil
}

func firstFailedBatchJob(jobs []*domainaijob.AIBatchJob) *domainaijob.AIBatchJob {
	return firstBatchJobWithAnyStatus(
		jobs,
		domaincommon.AIBatchJobStatusFailed,
		domaincommon.AIBatchJobStatusTimedOut,
		domaincommon.AIBatchJobStatusCancelled,
	)
}

func firstUnreconciledBatchItem(items []*domainaijob.AIBatchItem) *domainaijob.AIBatchItem {
	for _, item := range items {
		if item == nil {
			continue
		}
		switch item.Status {
		case domaincommon.AIBatchItemStatusPending,
			domaincommon.AIBatchItemStatusSubmitted,
			domaincommon.AIBatchItemStatusProcessing:
			return item
		}
	}
	return nil
}

func firstPendingValidationBatchItem(items []*domainaijob.AIBatchItem) *domainaijob.AIBatchItem {
	for _, item := range items {
		if item != nil &&
			item.Status == domaincommon.AIBatchItemStatusCompleted &&
			item.ValidationStatus == domaincommon.ValidationStatusNotValidated {
			return item
		}
	}
	return nil
}

func firstInvalidBatchItem(items []*domainaijob.AIBatchItem) *domainaijob.AIBatchItem {
	for _, item := range items {
		if item == nil {
			continue
		}
		if item.ValidationStatus == domaincommon.ValidationStatusInvalid ||
			item.Status == domaincommon.AIBatchItemStatusInvalidOutput {
			return item
		}
	}
	return nil
}

func firstFailedBatchItem(items []*domainaijob.AIBatchItem) *domainaijob.AIBatchItem {
	for _, item := range items {
		if item != nil && item.Status == domaincommon.AIBatchItemStatusFailed {
			return item
		}
	}
	return nil
}

func firstMaterializableBatchItem(items []*domainaijob.AIBatchItem) *domainaijob.AIBatchItem {
	for _, item := range items {
		if isMaterializableBatchItem(item) {
			return item
		}
	}
	return nil
}

func firstReviewWithAnyLifecycle(
	reviews []*domainreview.CompanyReview,
	states ...domaincommon.ReviewLifecycleState,
) *domainreview.CompanyReview {
	for _, review := range reviews {
		if review == nil {
			continue
		}
		for _, state := range states {
			if review.ReviewLifecycleState == state {
				return review
			}
		}
	}
	return nil
}
