package continuation

import (
	"context"
	"fmt"
	"time"

	domaincommon "goserver/internal/domain/common"
	platformrepo "goserver/internal/platform/repository"
	allocationsvc "goserver/internal/service/allocation"
	servicecommon "goserver/internal/service/common"
	projectionsvc "goserver/internal/service/projection"
	reviewsvc "goserver/internal/service/review"
	thesissvc "goserver/internal/service/thesis"
	workerservice "goserver/internal/service/worker"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

type WorkflowContinuationExecutionConfig struct {
	DefaultMaxWorkflows int
	MaxPageSize         int
	MaxReviewsPerStep   int
	MaxCandidates       int
}

type WorkflowContinuationExecutionOption func(*workflowContinuationService)

func WithWorkflowContinuationExecutionConfig(config WorkflowContinuationExecutionConfig) WorkflowContinuationExecutionOption {
	return func(service *workflowContinuationService) {
		if config.DefaultMaxWorkflows > 0 {
			service.config.DefaultMaxWorkflows = config.DefaultMaxWorkflows
		}
		if config.MaxPageSize > 0 {
			service.config.MaxPageSize = config.MaxPageSize
		}
		if config.MaxReviewsPerStep > 0 {
			service.config.MaxReviewsPerStep = config.MaxReviewsPerStep
		}
		if config.MaxCandidates > 0 {
			service.config.MaxCandidates = config.MaxCandidates
		}
	}
}

func WithWorkflowContinuationExecutionClock(clock servicecommon.ClockPort) WorkflowContinuationExecutionOption {
	return func(service *workflowContinuationService) {
		if clock != nil {
			service.now = clock.Now
		}
	}
}

type workflowContinuationService struct {
	workflowRuns  platformrepo.WorkflowRunRepository
	workflowSteps platformrepo.WorkflowStepRunRepository
	discovery     workerservice.WorkerWorkDiscoveryService
	decision      WorkflowContinuationDecisionService
	thesis        thesissvc.ThesisEvaluationService
	actions       reviewsvc.ActionMappingService
	buckets       reviewsvc.BucketAssignmentService
	candidates    allocationsvc.CapitalCandidateBuilderService
	allocator     allocationsvc.CapitalAllocationService
	projections   projectionsvc.ProjectionUpdateService
	config        WorkflowContinuationExecutionConfig
	now           func() time.Time
}

var _ WorkflowContinuationService = (*workflowContinuationService)(nil)

func NewWorkflowContinuationService(
	workflowRuns platformrepo.WorkflowRunRepository,
	workflowSteps platformrepo.WorkflowStepRunRepository,
	discovery workerservice.WorkerWorkDiscoveryService,
	decision WorkflowContinuationDecisionService,
	thesis thesissvc.ThesisEvaluationService,
	actions reviewsvc.ActionMappingService,
	buckets reviewsvc.BucketAssignmentService,
	candidates allocationsvc.CapitalCandidateBuilderService,
	allocator allocationsvc.CapitalAllocationService,
	projections projectionsvc.ProjectionUpdateService,
	options ...WorkflowContinuationExecutionOption,
) WorkflowContinuationService {
	service := &workflowContinuationService{
		workflowRuns:  workflowRuns,
		workflowSteps: workflowSteps,
		discovery:     discovery,
		decision:      decision,
		thesis:        thesis,
		actions:       actions,
		buckets:       buckets,
		candidates:    candidates,
		allocator:     allocator,
		projections:   projections,
		config: WorkflowContinuationExecutionConfig{
			DefaultMaxWorkflows: defaultContinuationMaxWorkflows,
			MaxPageSize:         maxContinuationPageSize,
		},
		now: time.Now,
	}
	for _, option := range options {
		if option != nil {
			option(service)
		}
	}
	if service.config.DefaultMaxWorkflows <= 0 {
		service.config.DefaultMaxWorkflows = defaultContinuationMaxWorkflows
	}
	if service.config.MaxPageSize <= 0 {
		service.config.MaxPageSize = maxContinuationPageSize
	}
	return service
}

func (service *workflowContinuationService) ContinueWorkflow(
	ctx context.Context,
	request ContinueWorkflowRequest,
) (*ContinueWorkflowResult, error) {
	if err := request.Validate(); err != nil {
		return nil, err
	}

	startedAt := service.now().UTC()
	result, err := service.continueOneWorkflow(ctx, request)
	if err != nil {
		return nil, err
	}
	completedAt := service.now().UTC()
	result.Summary = buildSingleContinuationExecutionSummary(result, startedAt, completedAt)
	return result, nil
}

func (service *workflowContinuationService) ContinueEligibleWorkflows(
	ctx context.Context,
	request ContinueEligibleWorkflowsRequest,
) (*ContinueEligibleWorkflowsResult, error) {
	if err := request.Validate(); err != nil {
		return nil, err
	}

	startedAt := service.now().UTC()
	workflowRunIDs, decisions, hasMore, err := service.discoverExecutableWorkflowRunIDs(ctx, request)
	if err != nil {
		return nil, fmt.Errorf("continue eligible workflows: %w", err)
	}

	result := &ContinueEligibleWorkflowsResult{
		Decisions: decisions,
	}
	for _, workflowRunID := range workflowRunIDs {
		single, err := service.ContinueWorkflow(ctx, ContinueWorkflowRequest{
			WorkflowRunID:    workflowRunID,
			BookType:         request.BookType,
			DryRun:           request.DryRun,
			Force:            request.Force,
			AllowedStepRange: request.AllowedStepRange,
			InitiatedBy:      request.InitiatedBy,
			CorrelationID:    request.CorrelationID,
		})
		if err != nil {
			result.FailedWorkflowRunIDs = append(result.FailedWorkflowRunIDs, workflowRunID)
			result.PartialFailures = append(result.PartialFailures, servicecommon.PartialFailure{
				Scope:         servicecommon.FailureScopeContinuation,
				WorkflowRunID: workflowRunID,
				ID:            workflowRunID,
				Code:          "workflow_continuation_failed",
				Message:       err.Error(),
			})
			continue
		}
		result.Decisions = appendDecisionIfMissing(result.Decisions, single)
		result.PartialFailures = append(result.PartialFailures, single.PartialFailures...)
		if single.Blocked {
			result.StillBlockedWorkflowRunIDs = append(result.StillBlockedWorkflowRunIDs, workflowRunID)
			continue
		}
		if single.Failed {
			result.FailedWorkflowRunIDs = append(result.FailedWorkflowRunIDs, workflowRunID)
			continue
		}
		if single.Continued || single.DryRun {
			result.ContinuedWorkflowRunIDs = append(result.ContinuedWorkflowRunIDs, workflowRunID)
		}
		if single.Completed {
			result.CompletedWorkflowRunIDs = append(result.CompletedWorkflowRunIDs, workflowRunID)
		}
	}

	completedAt := service.now().UTC()
	result.Summary = buildBulkContinuationExecutionSummary(
		len(workflowRunIDs),
		len(result.ContinuedWorkflowRunIDs),
		len(result.CompletedWorkflowRunIDs),
		len(result.StillBlockedWorkflowRunIDs),
		len(result.FailedWorkflowRunIDs),
		len(result.PartialFailures),
		request.DryRun,
		hasMore,
		startedAt,
		completedAt,
	)
	return result, nil
}

func (service *workflowContinuationService) discoverExecutableWorkflowRunIDs(
	ctx context.Context,
	request ContinueEligibleWorkflowsRequest,
) ([]primitive.ObjectID, []EvaluateWorkflowContinuationResult, bool, error) {
	if !request.WorkflowRunID.IsZero() {
		return []primitive.ObjectID{request.WorkflowRunID}, nil, false, nil
	}
	if service.decision != nil {
		decisions, err := service.decision.EvaluateManyWorkflowContinuations(ctx, EvaluateManyWorkflowContinuationsRequest{
			BookType:      request.BookType,
			MaxWorkflows:  request.MaxWorkflows,
			Force:         request.Force,
			InitiatedBy:   request.InitiatedBy,
			CorrelationID: request.CorrelationID,
		})
		if err == nil {
			return uniqueObjectIDs(decisions.ReadyWorkflowRunIDs), decisions.Decisions, false, nil
		}
		if service.workflowRuns == nil && service.discovery == nil {
			return nil, nil, false, fmt.Errorf("evaluate continuation candidates: %w", err)
		}
	}
	if service.discovery != nil {
		limit := service.maxWorkflows(request.MaxWorkflows)
		discovered, err := service.discovery.DiscoverContinuableWorkflows(ctx, workerservice.DiscoverContinuableWorkflowsRequest{
			DiscoveryRequestBase: workerservice.DiscoveryRequestBase{
				BookType:      request.BookType,
				MaxItems:      limit,
				CorrelationID: request.CorrelationID,
			},
			Force: request.Force,
		})
		if err != nil {
			return nil, nil, false, fmt.Errorf("discover continuable workflows: %w", err)
		}
		ids := make([]primitive.ObjectID, 0, len(discovered.Continuations))
		for _, continuation := range discovered.Continuations {
			if continuation.ReadyToContinue() {
				ids = appendUniqueObjectID(ids, continuation.WorkflowRunID)
			}
		}
		return ids, nil, discovered.HasMore, nil
	}
	if service.workflowRuns == nil {
		return nil, nil, false, fmt.Errorf("workflow run repository or continuation decision service is required")
	}
	limit := service.maxWorkflows(request.MaxWorkflows)
	filter := platformrepo.WorkflowRunFilter{ActiveOnly: true}
	if request.BookType != "" {
		filter.BookTypes = []domaincommon.BookType{request.BookType}
	}
	runs, err := service.workflowRuns.FindResumable(ctx, filter, platformrepo.WorkflowRunListOptions{
		Pagination: platformrepo.PageOptions{PageSize: service.scanLimit(limit)},
		Sort:       platformrepo.WorkflowRunSortOption{By: platformrepo.WorkflowRunSortByUpdatedAt, Order: platformrepo.SortOrderAscending},
	})
	if err != nil {
		return nil, nil, false, fmt.Errorf("list resumable workflows: %w", err)
	}
	ids := make([]primitive.ObjectID, 0, len(runs.Items))
	for _, run := range runs.Items {
		if run == nil {
			continue
		}
		ids = appendUniqueObjectID(ids, run.ID)
		if len(ids) >= limit {
			break
		}
	}
	return ids, nil, runs.Page.HasMore, nil
}

func (service *workflowContinuationService) maxWorkflows(requested int) int {
	if requested > 0 && requested < service.config.MaxPageSize {
		return requested
	}
	if requested > service.config.MaxPageSize {
		return service.config.MaxPageSize
	}
	if service.config.DefaultMaxWorkflows > service.config.MaxPageSize {
		return service.config.MaxPageSize
	}
	return service.config.DefaultMaxWorkflows
}

func (service *workflowContinuationService) scanLimit(limit int) int {
	if limit <= 0 {
		return service.config.MaxPageSize
	}
	scanLimit := limit * 3
	if scanLimit > service.config.MaxPageSize {
		return service.config.MaxPageSize
	}
	return scanLimit
}
