package continuation

import (
	"context"
	"fmt"
	"time"

	domaincommon "goserver/internal/domain/common"
	platformrepo "goserver/internal/platform/repository"
	servicecommon "goserver/internal/service/common"
	workerservice "goserver/internal/service/worker"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

const (
	defaultContinuationMaxWorkflows = 50
	maxContinuationPageSize         = 500
)

type WorkflowContinuationDecisionConfig struct {
	DefaultMaxWorkflows int
	MaxPageSize         int
	AllowPartialSuccess bool
}

type WorkflowContinuationDecisionOption func(*workflowContinuationDecisionService)

func WithWorkflowContinuationDecisionConfig(config WorkflowContinuationDecisionConfig) WorkflowContinuationDecisionOption {
	return func(service *workflowContinuationDecisionService) {
		if config.DefaultMaxWorkflows > 0 {
			service.config.DefaultMaxWorkflows = config.DefaultMaxWorkflows
		}
		if config.MaxPageSize > 0 {
			service.config.MaxPageSize = config.MaxPageSize
		}
		service.config.AllowPartialSuccess = config.AllowPartialSuccess
	}
}

func WithWorkflowContinuationDecisionClock(clock servicecommon.ClockPort) WorkflowContinuationDecisionOption {
	return func(service *workflowContinuationDecisionService) {
		if clock != nil {
			service.now = clock.Now
		}
	}
}

type workflowContinuationDecisionService struct {
	workflowRuns  platformrepo.WorkflowRunRepository
	workflowSteps platformrepo.WorkflowStepRunRepository
	batchJobs     platformrepo.AIBatchJobRepository
	batchItems    platformrepo.AIBatchItemRepository
	reviews       platformrepo.CompanyReviewRepository
	discovery     workerservice.WorkerWorkDiscoveryService
	config        WorkflowContinuationDecisionConfig
	now           func() time.Time
}

var _ WorkflowContinuationDecisionService = (*workflowContinuationDecisionService)(nil)

func NewWorkflowContinuationDecisionService(
	workflowRuns platformrepo.WorkflowRunRepository,
	workflowSteps platformrepo.WorkflowStepRunRepository,
	batchJobs platformrepo.AIBatchJobRepository,
	batchItems platformrepo.AIBatchItemRepository,
	reviews platformrepo.CompanyReviewRepository,
	discovery workerservice.WorkerWorkDiscoveryService,
	options ...WorkflowContinuationDecisionOption,
) WorkflowContinuationDecisionService {
	service := &workflowContinuationDecisionService{
		workflowRuns:  workflowRuns,
		workflowSteps: workflowSteps,
		batchJobs:     batchJobs,
		batchItems:    batchItems,
		reviews:       reviews,
		discovery:     discovery,
		config: WorkflowContinuationDecisionConfig{
			DefaultMaxWorkflows: defaultContinuationMaxWorkflows,
			MaxPageSize:         maxContinuationPageSize,
			AllowPartialSuccess: true,
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

func (service *workflowContinuationDecisionService) EvaluateWorkflowContinuation(
	ctx context.Context,
	request EvaluateWorkflowContinuationRequest,
) (*EvaluateWorkflowContinuationResult, error) {
	if err := request.Validate(); err != nil {
		return nil, err
	}

	startedAt := service.now().UTC()
	snapshot, err := service.loadContinuationContext(ctx, request.WorkflowRunID)
	if err != nil {
		return nil, fmt.Errorf("evaluate workflow continuation %s: %w", request.WorkflowRunID.Hex(), err)
	}

	result := service.evaluateLoadedWorkflow(snapshot, continuationEvaluationOptions{
		BookType: request.BookType,
		Force:    request.Force,
	})
	completedAt := service.now().UTC()
	result.Summary = buildSingleContinuationSummary(result, startedAt, completedAt)
	return result, nil
}

func (service *workflowContinuationDecisionService) EvaluateManyWorkflowContinuations(
	ctx context.Context,
	request EvaluateManyWorkflowContinuationsRequest,
) (*EvaluateManyWorkflowContinuationsResult, error) {
	if err := request.Validate(); err != nil {
		return nil, err
	}

	startedAt := service.now().UTC()
	workflowRunIDs, hasMore, err := service.discoverCandidateWorkflowRunIDs(ctx, request)
	if err != nil {
		return nil, fmt.Errorf("evaluate workflow continuations: %w", err)
	}

	result := &EvaluateManyWorkflowContinuationsResult{
		Decisions: make([]EvaluateWorkflowContinuationResult, 0, len(workflowRunIDs)),
	}
	for _, workflowRunID := range workflowRunIDs {
		decision, err := service.EvaluateWorkflowContinuation(ctx, EvaluateWorkflowContinuationRequest{
			WorkflowRunID: workflowRunID,
			BookType:      request.BookType,
			Force:         request.Force,
			InitiatedBy:   request.InitiatedBy,
			CorrelationID: request.CorrelationID,
		})
		if err != nil {
			result.FailedEvaluationWorkflowRunIDs = append(result.FailedEvaluationWorkflowRunIDs, workflowRunID)
			result.PartialFailures = append(result.PartialFailures, servicecommon.PartialFailure{
				Scope:         servicecommon.FailureScopeWorkflow,
				WorkflowRunID: workflowRunID,
				ID:            workflowRunID,
				Code:          "workflow_continuation_evaluation_failed",
				Message:       err.Error(),
			})
			continue
		}

		result.Decisions = append(result.Decisions, *decision)
		switch {
		case decision.ReadyToContinueNow():
			result.ReadyWorkflowRunIDs = append(result.ReadyWorkflowRunIDs, workflowRunID)
		case isTerminalReadiness(decision.Readiness):
			result.TerminalWorkflowRunIDs = append(result.TerminalWorkflowRunIDs, workflowRunID)
		default:
			result.BlockedWorkflowRunIDs = append(result.BlockedWorkflowRunIDs, workflowRunID)
		}
	}

	completedAt := service.now().UTC()
	result.Summary = buildBulkContinuationSummary(
		len(workflowRunIDs),
		len(result.ReadyWorkflowRunIDs),
		len(result.BlockedWorkflowRunIDs),
		len(result.TerminalWorkflowRunIDs),
		len(result.PartialFailures),
		hasMore,
		startedAt,
		completedAt,
	)
	return result, nil
}

func (service *workflowContinuationDecisionService) discoverCandidateWorkflowRunIDs(
	ctx context.Context,
	request EvaluateManyWorkflowContinuationsRequest,
) ([]primitive.ObjectID, bool, error) {
	limit := service.maxWorkflows(request.MaxWorkflows)
	if len(request.WorkflowRunIDs) > 0 {
		ids := uniqueObjectIDs(request.WorkflowRunIDs)
		if len(ids) > limit {
			return ids[:limit], true, nil
		}
		return ids, false, nil
	}

	ids := make([]primitive.ObjectID, 0, limit)
	hasMore := false

	if service.discovery != nil {
		discovered, err := service.discovery.DiscoverContinuableWorkflows(ctx, workerservice.DiscoverContinuableWorkflowsRequest{
			DiscoveryRequestBase: workerservice.DiscoveryRequestBase{
				BookType:      request.BookType,
				MaxItems:      limit,
				CorrelationID: request.CorrelationID,
			},
			Force: request.Force,
		})
		if err != nil && service.workflowRuns == nil {
			return nil, false, fmt.Errorf("discover continuable workflows: %w", err)
		}
		if err == nil {
			for _, continuation := range discovered.Continuations {
				ids = appendUniqueObjectID(ids, continuation.WorkflowRunID)
				if len(ids) >= limit {
					return ids, discovered.HasMore || len(discovered.Continuations) > limit, nil
				}
			}
			hasMore = discovered.HasMore
		}
	}

	if service.workflowRuns == nil {
		if len(ids) == 0 {
			return nil, false, fmt.Errorf("workflow run repository is required")
		}
		return ids, hasMore, nil
	}

	filter := platformrepo.WorkflowRunFilter{ActiveOnly: true}
	if request.BookType != "" {
		filter.BookTypes = []domaincommon.BookType{request.BookType}
	}
	list, err := service.workflowRuns.FindResumable(ctx, filter, platformrepo.WorkflowRunListOptions{
		Pagination: platformrepo.PageOptions{PageSize: service.scanLimit(limit)},
		Sort:       platformrepo.WorkflowRunSortOption{By: platformrepo.WorkflowRunSortByUpdatedAt, Order: platformrepo.SortOrderAscending},
	})
	if err != nil {
		return nil, false, fmt.Errorf("list resumable workflows: %w", err)
	}
	for _, run := range list.Items {
		if run == nil {
			continue
		}
		ids = appendUniqueObjectID(ids, run.ID)
		if len(ids) >= limit {
			break
		}
	}
	return ids, hasMore || list.Page.HasMore, nil
}

func (service *workflowContinuationDecisionService) maxWorkflows(requested int) int {
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

func (service *workflowContinuationDecisionService) scanLimit(limit int) int {
	if limit <= 0 {
		return service.config.MaxPageSize
	}
	scanLimit := limit * 3
	if scanLimit < limit {
		scanLimit = limit
	}
	if scanLimit > service.config.MaxPageSize {
		return service.config.MaxPageSize
	}
	return scanLimit
}
