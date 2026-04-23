package investing

import (
	"context"

	platformconfig "goserver/internal/platform/config"
	"goserver/internal/platform/domain"
	"goserver/internal/platform/ports"
	platformai "goserver/internal/platform/provider/ai"
	platformservice "goserver/internal/platform/service"
	"goserver/internal/platform/workflow"
)

type Service struct {
	config           platformconfig.AppConfig
	companies        ports.CompanyRepository
	workflowRuns     ports.WorkflowRunRepository
	configService    ports.ConfigService
	scorecardService ports.ScorecardService
	aiReviewEngine   ports.AIReviewEngine
	timeProvider     ports.TimeProvider
}

func NewService(
	config platformconfig.AppConfig,
	companies ports.CompanyRepository,
	workflowRuns ports.WorkflowRunRepository,
	configService ports.ConfigService,
	scorecardService ports.ScorecardService,
	aiReviewEngine ports.AIReviewEngine,
	timeProvider ports.TimeProvider,
) *Service {
	return &Service{
		config:           config,
		companies:        companies,
		workflowRuns:     workflowRuns,
		configService:    configService,
		scorecardService: scorecardService,
		aiReviewEngine:   aiReviewEngine,
		timeProvider:     platformservice.ResolveTimeProviderForWorkflow(timeProvider),
	}
}

func (service *Service) Start(ctx context.Context, request ports.StartInvestingWorkflowRequest) (*domain.WorkflowRun, error) {
	request.DryRun = false
	return service.start(ctx, request)
}

func (service *Service) DryRun(ctx context.Context, request ports.StartInvestingWorkflowRequest) (*domain.WorkflowRun, error) {
	request.DryRun = true
	return service.start(ctx, request)
}

func (service *Service) start(ctx context.Context, request ports.StartInvestingWorkflowRequest) (*domain.WorkflowRun, error) {
	if request.RunType == "" {
		request.RunType = domain.WorkflowRunTypeManual
	}
	if request.Mode == "" {
		request.Mode = domain.InvestingMode(service.config.Investing.DefaultMode)
	}
	if request.IdempotencyKey != "" {
		existing, err := service.workflowRuns.GetByIdempotencyKey(ctx, request.IdempotencyKey)
		if err != nil {
			return nil, err
		}
		if existing != nil {
			return existing, nil
		}
	}

	snapshot, err := service.configService.CreateSnapshot(ctx, domain.BookTypeInvesting, string(request.Mode))
	if err != nil {
		return nil, err
	}

	companies, err := service.companies.List(ctx, ports.CompanyListFilter{
		BookType: domain.BookTypeInvesting,
		Limit:    request.Limit,
	})
	if err != nil {
		return nil, err
	}
	selectedCompanies := filterRequestedCompanies(companies, request.CompanyIDs)

	now := service.timeProvider.Now()
	run := &domain.WorkflowRun{
		BookType:              domain.BookTypeInvesting,
		RunType:               request.RunType,
		Mode:                  string(request.Mode),
		Status:                domain.WorkflowRunStatusRunning,
		StartedAt:             now,
		ConfigSnapshotID:      snapshot.ID,
		CompaniesScannedCount: len(selectedCompanies),
		DryRun:                request.DryRun,
		ReplayFromRunID:       request.ReplayFromRunID,
		IdempotencyKey:        request.IdempotencyKey,
		Notes:                 request.Notes,
		RequestMetadata: map[string]any{
			"requestedBy": request.RequestedBy,
			"companyIds":  request.CompanyIDs,
		},
		StepStatuses:  buildInvestingPendingSteps(),
		SchemaVersion: service.config.SchemaVersion,
		CreatedAt:     now,
		UpdatedAt:     now,
	}

	applyCompletedStep(run, workflow.InvestingStepScanUniverse, map[string]any{
		"requestedCompanyIds": request.CompanyIDs,
		"limit":               request.Limit,
		"bookType":            domain.BookTypeInvesting,
	}, map[string]any{
		"companyIds": companyIDs(selectedCompanies),
		"count":      len(selectedCompanies),
	}, now)
	applyCompletedStep(run, workflow.InvestingStepApplyHardFilters, map[string]any{
		"companyIds": companyIDs(selectedCompanies),
	}, map[string]any{
		"eligibleCompanyIds": companyIDs(selectedCompanies),
		"rejectedCompanyIds": []string{},
	}, now)
	applyCompletedStep(run, workflow.InvestingStepBuildReviewInput, map[string]any{
		"companyIds":       companyIDs(selectedCompanies),
		"configSnapshotId": snapshot.ID,
	}, map[string]any{
		"reviewInputCount": len(selectedCompanies),
	}, now)

	if request.DryRun {
		applySkippedStep(run, workflow.InvestingStepGenerateScorecard, "dry run skips async AI submission")
		markRemainingInvestingStepsSkipped(run, string(workflow.InvestingStepEvaluateThesisAndChange))
		run.Status = domain.WorkflowRunStatusCompleted
		completedAt := now
		run.CompletedAt = &completedAt
		return service.workflowRuns.Create(ctx, run)
	}

	asyncTask, err := service.submitAsyncScorecardStep(ctx, selectedCompanies, snapshot.ID, request.Mode)
	if err != nil {
		markFailedStep(run, workflow.InvestingStepGenerateScorecard, err.Error(), now)
		run.Status = domain.WorkflowRunStatusFailed
		run.ErrorsCount++
		completedAt := now
		run.CompletedAt = &completedAt
		return service.workflowRuns.Create(ctx, run)
	}

	applyWaitingAsyncStep(run, workflow.InvestingStepGenerateScorecard, map[string]any{
		"companyIds":       companyIDs(selectedCompanies),
		"configSnapshotId": snapshot.ID,
	}, map[string]any{
		"asyncOnly": true,
		"mode":      request.Mode,
	}, asyncTask, now)
	run.Status = domain.WorkflowRunStatusWaitingAsync

	return service.workflowRuns.Create(ctx, run)
}

func (service *Service) submitAsyncScorecardStep(ctx context.Context, companies []*domain.Company, snapshotID string, mode domain.InvestingMode) (*domain.AsyncTaskReference, error) {
	if service.aiReviewEngine == nil {
		unavailable, err := (&platformai.NoopAIReviewEngine{}).SubmitReviewBatch(ctx, ports.AIReviewBatchRequest{})
		if err != nil {
			return nil, err
		}
		return fromAsyncTask(unavailable), nil
	}

	items := make([]ports.AIReviewBatchItem, 0, len(companies))
	for _, company := range companies {
		item, err := service.scorecardService.BuildAsyncReviewItem(ctx, company, snapshotID, mode)
		if err != nil {
			return nil, err
		}
		items = append(items, item)
	}

	task, err := service.aiReviewEngine.SubmitReviewBatch(ctx, ports.AIReviewBatchRequest{
		BookType:             string(domain.BookTypeInvesting),
		PromptVersion:        service.config.AsyncAI.PromptVersion,
		ModelName:            service.config.AsyncAI.Model,
		ResponseInstructions: service.config.AsyncAI.ResponseInstructions,
		Items:                items,
	})
	if err != nil {
		return nil, err
	}

	return fromAsyncTask(task), nil
}

func filterRequestedCompanies(companies []*domain.Company, requested []string) []*domain.Company {
	if len(requested) == 0 {
		return companies
	}

	set := map[string]struct{}{}
	for _, id := range requested {
		set[id] = struct{}{}
	}

	filtered := make([]*domain.Company, 0, len(requested))
	for _, company := range companies {
		if _, exists := set[company.ID]; exists {
			filtered = append(filtered, company)
		}
	}

	return filtered
}

func companyIDs(companies []*domain.Company) []string {
	ids := make([]string, 0, len(companies))
	for _, company := range companies {
		ids = append(ids, company.ID)
	}

	return ids
}
