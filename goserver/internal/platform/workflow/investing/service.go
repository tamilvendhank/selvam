package investing

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"

	platformconfig "goserver/internal/platform/config"
	"goserver/internal/platform/domain"
	"goserver/internal/platform/ports"
	platformservice "goserver/internal/platform/service"
	"goserver/internal/platform/workflow"
)

type Service struct {
	config                 platformconfig.AppConfig
	companies              ports.CompanyRepository
	reviews                ports.CompanyReviewRepository
	theses                 ports.ThesisRepository
	workflowRuns           ports.WorkflowRunRepository
	workflowSteps          ports.WorkflowStepRunRepository
	configService          ports.ConfigService
	scorecardService       ports.ScorecardService
	actionMappingService   ports.ActionMappingService
	changeDetectionService ports.ChangeDetectionService
	thesisService          ports.ThesisService
	batchJobs              ports.AIBatchJobRepository
	batchItems             ports.AIBatchItemRepository
	reconciliationLogs     ports.JobReconciliationLogRepository
	aiBatchEngine          ports.AIBatchEngine
	timeProvider           ports.TimeProvider
}

func NewService(
	config platformconfig.AppConfig,
	companies ports.CompanyRepository,
	reviews ports.CompanyReviewRepository,
	theses ports.ThesisRepository,
	workflowRuns ports.WorkflowRunRepository,
	workflowSteps ports.WorkflowStepRunRepository,
	configService ports.ConfigService,
	scorecardService ports.ScorecardService,
	actionMappingService ports.ActionMappingService,
	changeDetectionService ports.ChangeDetectionService,
	thesisService ports.ThesisService,
	batchJobs ports.AIBatchJobRepository,
	batchItems ports.AIBatchItemRepository,
	reconciliationLogs ports.JobReconciliationLogRepository,
	aiBatchEngine ports.AIBatchEngine,
	timeProvider ports.TimeProvider,
) *Service {
	if aiBatchEngine == nil {
		aiBatchEngine = &noopBatchEngineAdapter{}
	}

	return &Service{
		config:                 config,
		companies:              companies,
		reviews:                reviews,
		theses:                 theses,
		workflowRuns:           workflowRuns,
		workflowSteps:          workflowSteps,
		configService:          configService,
		scorecardService:       scorecardService,
		actionMappingService:   actionMappingService,
		changeDetectionService: changeDetectionService,
		thesisService:          thesisService,
		batchJobs:              batchJobs,
		batchItems:             batchItems,
		reconciliationLogs:     reconciliationLogs,
		aiBatchEngine:          aiBatchEngine,
		timeProvider:           platformservice.ResolveTimeProviderForWorkflow(timeProvider),
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

func (service *Service) Resume(ctx context.Context, workflowRunID string) (*domain.WorkflowRun, error) {
	return service.Reconcile(ctx, workflowRunID)
}

func (service *Service) Reconcile(ctx context.Context, workflowRunID string) (*domain.WorkflowRun, error) {
	run, err := service.workflowRuns.GetByID(ctx, workflowRunID)
	if err != nil {
		return nil, err
	}
	if run == nil {
		return nil, platformservice.ErrNotFound
	}

	jobs, err := service.batchJobs.List(ctx, ports.AIBatchJobListFilter{
		WorkflowRunID: workflowRunID,
		Limit:         100,
	})
	if err != nil {
		return nil, err
	}
	if len(jobs) == 0 {
		return service.completeRunIfReady(ctx, run)
	}

	for _, job := range jobs {
		switch job.Status {
		case domain.BatchJobStatusCreated:
			if err := service.submitExistingBatchJob(ctx, run, job); err != nil {
				return nil, err
			}
		case domain.BatchJobStatusSubmitted, domain.BatchJobStatusRunning, domain.BatchJobStatusPartiallyCompleted:
			if err := service.pollAndReconcileBatchJob(ctx, run, job); err != nil {
				return nil, err
			}
		case domain.BatchJobStatusCompleted:
			if err := service.reconcileBatchResults(ctx, run, job, nil); err != nil {
				return nil, err
			}
		}
	}

	return service.completeRunIfReady(ctx, run)
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
	applyCompletedStep(run, workflow.InvestingStepBuildReviewInputs, map[string]any{
		"companyIds":       companyIDs(selectedCompanies),
		"configSnapshotId": snapshot.ID,
	}, map[string]any{
		"reviewInputCount": len(selectedCompanies),
	}, now)

	if request.DryRun {
		applySkippedStep(run, workflow.InvestingStepCreatePendingReviewRecords, "dry run skips async review shell creation")
		markRemainingInvestingStepsSkipped(run, string(workflow.InvestingStepCreateBatchJob))
		run.Status = domain.WorkflowRunStatusCompleted
		completedAt := now
		run.CompletedAt = &completedAt
		created, err := service.workflowRuns.Create(ctx, run)
		if err != nil {
			return nil, err
		}
		if err := service.syncStepRuns(ctx, created); err != nil {
			return nil, err
		}
		return created, nil
	}

	run, err = service.workflowRuns.Create(ctx, run)
	if err != nil {
		return nil, err
	}
	if err := service.syncStepRuns(ctx, run); err != nil {
		return nil, err
	}

	batchJob := &domain.AIBatchJob{
		JobType:       domain.BatchJobTypeInvestingReview,
		WorkflowRunID: run.ID,
		BookType:      domain.BookTypeInvesting,
		ProviderName:  service.config.AsyncAI.Provider,
		Status:        domain.BatchJobStatusCreated,
		SubmissionPayloadRef: map[string]any{
			"configSnapshotId": snapshot.ID,
			"mode":             request.Mode,
			"companyIds":       companyIDs(selectedCompanies),
		},
		MaxRetryCount:  3,
		IdempotencyKey: buildBatchIdempotencyKey(run.IdempotencyKey, run.ID),
		SchemaVersion:  service.config.SchemaVersion,
		CreatedAt:      now,
		UpdatedAt:      now,
	}
	batchJob, err = service.batchJobs.Create(ctx, batchJob)
	if err != nil {
		return nil, err
	}
	run.RequestMetadata["createdBatchJobIds"] = []string{batchJob.ID}
	applyCompletedStep(run, workflow.InvestingStepCreateBatchJob, map[string]any{
		"workflowRunId": run.ID,
	}, map[string]any{
		"aiBatchJobId": batchJob.ID,
	}, now)

	type builtInput struct {
		company      *domain.Company
		review       *domain.CompanyReview
		asyncItem    ports.AIReviewBatchItem
		inputPayload map[string]any
		inputHash    string
	}

	builtInputs := make([]builtInput, 0, len(selectedCompanies))
	for _, company := range selectedCompanies {
		asyncItem, err := service.scorecardService.BuildAsyncReviewItem(ctx, company, snapshot.ID, request.Mode)
		if err != nil {
			return nil, err
		}
		inputPayload := buildInputPayload(company, run, snapshot.ID, request.Mode, asyncItem)
		inputHash, err := hashPayload(inputPayload)
		if err != nil {
			return nil, err
		}
		reviewShell := &domain.CompanyReview{
			CompanyID:        company.ID,
			Symbol:           company.Symbol,
			BookType:         domain.BookTypeInvesting,
			ReviewDate:       now,
			ReviewPeriodType: domain.ReviewPeriodManual,
			WorkflowRunID:    run.ID,
			ConfigSnapshotID: snapshot.ID,
			ReviewStatus:     domain.ReviewStatusPendingInput,
			Mode:             request.Mode,
			ReviewerType:     domain.ReviewerTypeAI,
			AIModelName:      service.config.AsyncAI.Model,
			AIPromptVersion:  service.config.AsyncAI.PromptVersion,
			SchemaVersion:    service.config.SchemaVersion,
			SourceBatchJobID: batchJob.ID,
			InputSnapshot:    inputPayload,
			InputHash:        inputHash,
			ValidationStatus: domain.ValidationStatusNotValidated,
			ReviewMetadata: map[string]any{
				"asyncOnly": true,
			},
			CreatedAt: now,
			UpdatedAt: now,
		}
		reviewShell, err = service.reviews.Create(ctx, reviewShell)
		if err != nil {
			return nil, err
		}
		builtInputs = append(builtInputs, builtInput{
			company:      company,
			review:       reviewShell,
			asyncItem:    asyncItem,
			inputPayload: inputPayload,
			inputHash:    inputHash,
		})
	}
	applyCompletedStep(run, workflow.InvestingStepCreatePendingReviewRecords, map[string]any{
		"aiBatchJobId": batchJob.ID,
	}, map[string]any{
		"reviewShellCount": len(builtInputs),
	}, now)

	batchItems := make([]*domain.AIBatchItem, 0, len(builtInputs))
	for _, input := range builtInputs {
		batchItems = append(batchItems, &domain.AIBatchItem{
			AIBatchJobID:     batchJob.ID,
			WorkflowRunID:    run.ID,
			CompanyID:        input.company.ID,
			Symbol:           input.company.Symbol,
			BookType:         domain.BookTypeInvesting,
			ItemType:         domain.BatchItemTypeCompanyReview,
			InputPayload:     input.inputPayload,
			InputHash:        input.inputHash,
			Status:           domain.BatchItemStatusPending,
			ValidationStatus: domain.ValidationStatusNotValidated,
			TargetReviewID:   input.review.ID,
			CreatedAt:        now,
			UpdatedAt:        now,
		})
	}
	createdItems, err := service.batchItems.CreateMany(ctx, batchItems)
	if err != nil {
		return nil, err
	}

	submitItems := make([]ports.SubmitBatchItem, 0, len(createdItems))
	for index, item := range createdItems {
		review := builtInputs[index].review
		review.SourceBatchItemID = item.ID
		review.ReviewStatus = domain.ReviewStatusPendingAI
		review.UpdatedAt = now
		if _, err := service.reviews.UpdateMutable(ctx, review); err != nil {
			return nil, err
		}
		submitItems = append(submitItems, ports.SubmitBatchItem{
			CorrelationID:   item.ID,
			ReferenceID:     builtInputs[index].company.ID,
			ItemType:        domain.BatchItemTypeCompanyReview,
			Prompt:          builtInputs[index].asyncItem.Prompt,
			InputPayload:    builtInputs[index].inputPayload,
			TemplateRecord:  builtInputs[index].asyncItem.TemplateRecord,
			Model:           builtInputs[index].asyncItem.Model,
			ReasoningEffort: builtInputs[index].asyncItem.ReasoningEffort,
			Metadata:        builtInputs[index].asyncItem.Metadata,
		})
	}

	submission, err := service.aiBatchEngine.SubmitBatch(ctx, ports.SubmitBatchRequest{
		JobType:              domain.BatchJobTypeInvestingReview,
		BookType:             domain.BookTypeInvesting,
		WorkflowRunID:        run.ID,
		IdempotencyKey:       batchJob.IdempotencyKey,
		PromptVersion:        service.config.AsyncAI.PromptVersion,
		ModelName:            service.config.AsyncAI.Model,
		ResponseInstructions: service.config.AsyncAI.ResponseInstructions,
		Items:                submitItems,
	})
	if err != nil {
		batchJob.Status = domain.BatchJobStatusFailed
		batchJob.ErrorSummary = err.Error()
		batchJob.FailedAt = &now
		batchJob.UpdatedAt = now
		if _, updateErr := service.batchJobs.Update(ctx, batchJob); updateErr != nil {
			return nil, updateErr
		}
		markFailedStep(run, workflow.InvestingStepSubmitBatchJob, err.Error(), now)
		run.Status = domain.WorkflowRunStatusFailed
		run.ErrorsCount = len(createdItems)
		completedAt := now
		run.CompletedAt = &completedAt
		run.UpdatedAt = now
		run, err = service.workflowRuns.Update(ctx, run)
		if err != nil {
			return nil, err
		}
		if err := service.syncStepRuns(ctx, run); err != nil {
			return nil, err
		}
		return run, nil
	}

	if submission.ProviderName != "" {
		batchJob.ProviderName = submission.ProviderName
	}
	batchJob.ProviderJobHandle = submission.ProviderJobHandle
	batchJob.LocalJobHandle = submission.LocalJobHandle
	batchJob.Status = submission.Status
	batchJob.SubmittedAt = submission.SubmittedAt
	batchJob.UpdatedAt = now
	if submission.Metadata != nil {
		batchJob.ResultPayloadRef = submission.Metadata
	}
	batchJob, err = service.batchJobs.Update(ctx, batchJob)
	if err != nil {
		return nil, err
	}

	submissionStatuses := make(map[string]ports.BatchSubmissionItem, len(submission.Items))
	for _, item := range submission.Items {
		submissionStatuses[item.CorrelationID] = item
	}
	for _, item := range createdItems {
		if submitted, ok := submissionStatuses[item.ID]; ok {
			item.Status = submitted.Status
			item.UpdatedAt = now
			if _, err := service.batchItems.Update(ctx, item); err != nil {
				return nil, err
			}
		}
	}

	applyCompletedStep(run, workflow.InvestingStepSubmitBatchJob, map[string]any{
		"aiBatchJobId": batchJob.ID,
	}, map[string]any{
		"providerJobHandle":  batchJob.ProviderJobHandle,
		"submittedItemCount": len(submission.Items),
	}, now)
	applyWaitingAsyncStep(run, workflow.InvestingStepWaitForAsyncResults, map[string]any{
		"aiBatchJobId": batchJob.ID,
	}, map[string]any{
		"pendingItemCount": len(createdItems),
	}, &domain.AsyncTaskReference{
		Provider:        batchJob.ProviderName,
		TaskKind:        string(batchJob.JobType),
		LocalObjectType: "ai_batch_job",
		LocalObjectID:   batchJob.ID,
		SubmissionID:    batchJob.LocalJobHandle,
		BatchID:         batchJob.ProviderJobHandle,
		Status:          domain.AsyncTaskStatusQueued,
		SubmittedAt:     batchJob.SubmittedAt,
	}, now)
	run.Status = domain.WorkflowRunStatusWaitingAsync
	run.UpdatedAt = now

	run, err = service.workflowRuns.Update(ctx, run)
	if err != nil {
		return nil, err
	}
	if err := service.syncStepRuns(ctx, run); err != nil {
		return nil, err
	}

	return run, nil
}

func (service *Service) submitExistingBatchJob(ctx context.Context, run *domain.WorkflowRun, job *domain.AIBatchJob) error {
	items, err := service.batchItems.List(ctx, ports.AIBatchItemListFilter{
		AIBatchJobID: job.ID,
		Limit:        500,
	})
	if err != nil {
		return err
	}
	if len(items) == 0 {
		return nil
	}

	submitItems := make([]ports.SubmitBatchItem, 0, len(items))
	for _, item := range items {
		if item.Status == domain.BatchItemStatusCompleted || item.Status == domain.BatchItemStatusSkipped {
			continue
		}
		submitItems = append(submitItems, batchSubmitItemFromPayload(item))
	}
	if len(submitItems) == 0 {
		return nil
	}

	result, err := service.aiBatchEngine.SubmitBatch(ctx, ports.SubmitBatchRequest{
		JobType:              job.JobType,
		BookType:             job.BookType,
		WorkflowRunID:        job.WorkflowRunID,
		IdempotencyKey:       job.IdempotencyKey,
		PromptVersion:        service.config.AsyncAI.PromptVersion,
		ModelName:            service.config.AsyncAI.Model,
		ResponseInstructions: service.config.AsyncAI.ResponseInstructions,
		Items:                submitItems,
	})
	if err != nil {
		return err
	}

	now := service.timeProvider.Now()
	job.ProviderName = result.ProviderName
	job.ProviderJobHandle = result.ProviderJobHandle
	job.LocalJobHandle = result.LocalJobHandle
	job.Status = result.Status
	job.SubmittedAt = result.SubmittedAt
	job.UpdatedAt = now
	if _, err := service.batchJobs.Update(ctx, job); err != nil {
		return err
	}
	applyCompletedStep(run, workflow.InvestingStepSubmitBatchJob, map[string]any{
		"aiBatchJobId": job.ID,
	}, map[string]any{
		"providerJobHandle":  job.ProviderJobHandle,
		"submittedItemCount": len(result.Items),
	}, now)
	applyWaitingAsyncStep(run, workflow.InvestingStepWaitForAsyncResults, map[string]any{
		"aiBatchJobId": job.ID,
	}, map[string]any{
		"pendingItemCount": len(submitItems),
	}, &domain.AsyncTaskReference{
		Provider:        job.ProviderName,
		TaskKind:        string(job.JobType),
		LocalObjectType: "ai_batch_job",
		LocalObjectID:   job.ID,
		SubmissionID:    job.LocalJobHandle,
		BatchID:         job.ProviderJobHandle,
		Status:          domain.AsyncTaskStatusQueued,
		SubmittedAt:     job.SubmittedAt,
	}, now)
	run.Status = domain.WorkflowRunStatusWaitingAsync
	run.UpdatedAt = now
	if _, err := service.workflowRuns.Update(ctx, run); err != nil {
		return err
	}
	return service.syncStepRuns(ctx, run)
}

func (service *Service) pollAndReconcileBatchJob(ctx context.Context, run *domain.WorkflowRun, job *domain.AIBatchJob) error {
	statusResult, err := service.aiBatchEngine.GetBatchStatus(ctx, job.ProviderJobHandle)
	if err != nil {
		return err
	}

	now := service.timeProvider.Now()
	statusBefore := job.Status
	job.Status = statusResult.Status
	job.LastPolledAt = statusResult.LastPolledAt
	job.CompletedAt = statusResult.CompletedAt
	job.UpdatedAt = now
	if statusResult.Status == domain.BatchJobStatusFailed {
		job.FailedAt = &now
	}
	job.ResultPayloadRef = statusResult.RawProviderStatus
	job, err = service.batchJobs.Update(ctx, job)
	if err != nil {
		return err
	}

	if service.reconciliationLogs != nil {
		_, _ = service.reconciliationLogs.Create(ctx, &domain.JobReconciliationLog{
			AIBatchJobID:             job.ID,
			PolledAt:                 now,
			StatusBefore:             statusBefore,
			StatusAfter:              statusResult.Status,
			ItemsCompletedDelta:      statusResult.ItemsCompletedCount,
			ItemsFailedDelta:         statusResult.ItemsFailedCount,
			RawProviderStatusSummary: statusResult.RawProviderStatus,
			CreatedAt:                now,
		})
	}

	applyWaitingAsyncStep(run, workflow.InvestingStepWaitForAsyncResults, map[string]any{
		"aiBatchJobId": job.ID,
	}, map[string]any{
		"itemsCompletedCount":  statusResult.ItemsCompletedCount,
		"itemsFailedCount":     statusResult.ItemsFailedCount,
		"itemsProcessingCount": statusResult.ItemsProcessingCount,
	}, &domain.AsyncTaskReference{
		Provider:        job.ProviderName,
		TaskKind:        string(job.JobType),
		LocalObjectType: "ai_batch_job",
		LocalObjectID:   job.ID,
		SubmissionID:    job.LocalJobHandle,
		BatchID:         job.ProviderJobHandle,
		Status:          domain.AsyncTaskStatusInProgress,
		SubmittedAt:     job.SubmittedAt,
		LastSyncedAt:    statusResult.LastPolledAt,
		ResultAvailable: statusResult.ResultAvailable,
	}, now)

	if _, err := service.workflowRuns.Update(ctx, run); err != nil {
		return err
	}
	if err := service.syncStepRuns(ctx, run); err != nil {
		return err
	}
	if !statusResult.ResultAvailable && statusResult.Status != domain.BatchJobStatusCompleted && statusResult.Status != domain.BatchJobStatusPartiallyCompleted && statusResult.Status != domain.BatchJobStatusFailed {
		return nil
	}

	return service.reconcileBatchResults(ctx, run, job, statusResult)
}

func (service *Service) reconcileBatchResults(ctx context.Context, run *domain.WorkflowRun, job *domain.AIBatchJob, statusResult *ports.BatchStatusResult) error {
	results, err := service.aiBatchEngine.GetBatchResults(ctx, job.ProviderJobHandle)
	if err != nil {
		return err
	}
	items, err := service.batchItems.List(ctx, ports.AIBatchItemListFilter{
		AIBatchJobID: job.ID,
		Limit:        500,
	})
	if err != nil {
		return err
	}
	itemsByID := make(map[string]*domain.AIBatchItem, len(items))
	for _, item := range items {
		itemsByID[item.ID] = item
	}

	now := service.timeProvider.Now()
	for _, result := range results.Items {
		item := itemsByID[result.CorrelationID]
		if item == nil {
			continue
		}
		item.Status = result.Status
		item.ResultPayload = result.OutputPayload
		item.ErrorSummary = firstNonEmptyString(item.ErrorSummary, result.ErrorSummary)
		item.UpdatedAt = now
		if result.Status == domain.BatchItemStatusCompleted || result.Status == domain.BatchItemStatusFailed || result.Status == domain.BatchItemStatusInvalidOutput || result.Status == domain.BatchItemStatusSkipped {
			item.CompletedAt = &now
		}

		review, err := service.reviews.GetByID(ctx, item.TargetReviewID)
		if err != nil {
			return err
		}
		switch result.Status {
		case domain.BatchItemStatusCompleted:
			if review != nil && review.IsMutable() {
				review.ReviewStatus = domain.ReviewStatusAICompletedUnvalidated
				review.RawAIResultPayload = result.OutputPayload
				review.ValidationStatus = domain.ValidationStatusNotValidated
				review.UpdatedAt = now
				if _, err := service.reviews.UpdateMutable(ctx, review); err != nil {
					return err
				}
			}
			if err := service.materializeReviewResult(ctx, item, review); err != nil {
				item.Status = domain.BatchItemStatusInvalidOutput
				item.ValidationStatus = domain.ValidationStatusInvalid
				item.ValidationErrors = []string{err.Error()}
				item.ErrorSummary = err.Error()
				item.UpdatedAt = now
				if _, updateErr := service.batchItems.Update(ctx, item); updateErr != nil {
					return updateErr
				}
			}
		case domain.BatchItemStatusFailed:
			item.ValidationStatus = domain.ValidationStatusInvalid
			item.ValidationErrors = []string{firstNonEmptyString(result.ErrorSummary, "provider returned failed item")}
			if _, err := service.batchItems.Update(ctx, item); err != nil {
				return err
			}
			if review != nil && review.IsMutable() {
				review.ReviewStatus = domain.ReviewStatusValidationFailed
				review.ValidationStatus = domain.ValidationStatusInvalid
				review.ValidationErrors = item.ValidationErrors
				review.RawAIResultPayload = result.OutputPayload
				review.UpdatedAt = now
				if _, err := service.reviews.UpdateMutable(ctx, review); err != nil {
					return err
				}
			}
		default:
			if _, err := service.batchItems.Update(ctx, item); err != nil {
				return err
			}
		}
	}

	job.Status = results.Status
	job.CompletedAt = results.CompletedAt
	job.LastPolledAt = &now
	job.ResultPayloadRef = results.RawPayload
	job.UpdatedAt = now
	if statusResult != nil && statusResult.Status == domain.BatchJobStatusFailed {
		job.FailedAt = &now
	}
	if _, err := service.batchJobs.Update(ctx, job); err != nil {
		return err
	}

	applyCompletedStep(run, workflow.InvestingStepPollAndReconcileBatchResults, map[string]any{
		"aiBatchJobId": job.ID,
	}, map[string]any{
		"status":      results.Status,
		"itemCount":   len(results.Items),
		"completedAt": results.CompletedAt,
	}, now)

	return nil
}

func (service *Service) materializeReviewResult(ctx context.Context, item *domain.AIBatchItem, review *domain.CompanyReview) error {
	if item == nil || review == nil {
		return fmt.Errorf("materialization requires both ai batch item and review shell")
	}

	structured, err := extractStructuredReviewPayload(item.ResultPayload)
	if err != nil {
		return err
	}

	candidate, err := buildReviewCandidate(review, structured)
	if err != nil {
		return err
	}

	previousReview, err := service.reviews.GetLatestComparableByCompany(ctx, candidate.CompanyID, candidate.BookType, candidate.ID)
	if err != nil {
		return err
	}
	activeThesis, err := service.theses.GetActiveByCompanyID(ctx, candidate.CompanyID)
	if err != nil {
		return err
	}

	changeLog, err := service.changeDetectionService.CompareReviews(ctx, candidate, previousReview, activeThesis)
	if err != nil {
		return err
	}
	candidate.ChangeLog = changeLog

	decision, err := service.actionMappingService.MapReview(ctx, candidate, activeThesis, previousReview)
	if err != nil {
		return err
	}
	candidate.DecisionAction = decision
	if candidate.FinalActionAfterReview == "" && decision != nil {
		candidate.FinalActionAfterReview = decision.ActionType
	}
	if candidate.FinalBucketAfterReview == "" && decision != nil {
		candidate.FinalBucketAfterReview = decision.BucketAfterAction
	}
	if candidate.ActionRationaleSummary == "" && decision != nil {
		candidate.ActionRationaleSummary = decision.ActionReasonPrimary
	}
	if candidate.WhatChangedSummary == "" && changeLog != nil {
		candidate.WhatChangedSummary = changeLog.ChangeSummary
	}

	if err := service.scorecardService.ValidateReview(ctx, candidate); err != nil {
		return err
	}

	candidate.ReviewStatus = domain.ReviewStatusAIValidated
	candidate.ValidationStatus = domain.ValidationStatusValid
	candidate.ValidationErrors = nil
	updatedReview, err := service.reviews.UpdateMutable(ctx, candidate)
	if err != nil {
		return err
	}
	finalizedReview, err := service.reviews.Finalize(ctx, updatedReview.ID)
	if err != nil {
		return err
	}
	if previousReview != nil && previousReview.ReviewStatus == domain.ReviewStatusFinalized {
		if _, err := service.reviews.MarkSuperseded(ctx, previousReview.ID); err != nil {
			return err
		}
	}

	thesis, err := service.thesisService.BuildOrUpdateFromReview(ctx, finalizedReview)
	if err != nil {
		return err
	}

	item.ValidationStatus = domain.ValidationStatusValid
	item.ValidationErrors = nil
	item.TargetReviewID = finalizedReview.ID
	if thesis != nil {
		item.TargetThesisID = thesis.ID
		item.TargetEntityVersion = thesis.ThesisVersion
	}
	item.UpdatedAt = service.timeProvider.Now()
	if _, err := service.batchItems.Update(ctx, item); err != nil {
		return err
	}

	return nil
}

func (service *Service) completeRunIfReady(ctx context.Context, run *domain.WorkflowRun) (*domain.WorkflowRun, error) {
	items, err := service.batchItems.List(ctx, ports.AIBatchItemListFilter{
		WorkflowRunID: run.ID,
		Limit:         1000,
	})
	if err != nil {
		return nil, err
	}
	if len(items) == 0 {
		return run, nil
	}

	now := service.timeProvider.Now()
	pendingCount := 0
	finalizedCount := 0
	failedCount := 0
	for _, item := range items {
		switch item.Status {
		case domain.BatchItemStatusCompleted, domain.BatchItemStatusSkipped:
			finalizedCount++
		case domain.BatchItemStatusFailed, domain.BatchItemStatusInvalidOutput:
			failedCount++
		default:
			pendingCount++
		}
	}

	if pendingCount > 0 {
		run.Status = domain.WorkflowRunStatusWaitingAsync
		run.UpdatedAt = now
		updated, err := service.workflowRuns.Update(ctx, run)
		if err != nil {
			return nil, err
		}
		if err := service.syncStepRuns(ctx, updated); err != nil {
			return nil, err
		}
		return updated, nil
	}

	reviews, err := service.reviews.List(ctx, ports.CompanyReviewListFilter{
		BookType: domain.BookTypeInvesting,
		Limit:    1000,
	})
	if err != nil {
		return nil, err
	}

	buyCandidates := 0
	for _, review := range reviews {
		if review == nil || review.WorkflowRunID != run.ID {
			continue
		}
		if review.FinalActionAfterReview == domain.ActionBuy {
			buyCandidates++
		}
	}

	applyCompletedStep(run, workflow.InvestingStepWaitForAsyncResults, map[string]any{
		"workflowRunId": run.ID,
	}, map[string]any{
		"pendingItemCount": 0,
	}, now)
	applyCompletedStep(run, workflow.InvestingStepValidateAIOutputs, nil, map[string]any{
		"failedItemCount": failedCount,
	}, now)
	applyCompletedStep(run, workflow.InvestingStepMaterializeFinalReviews, nil, map[string]any{
		"finalizedReviewCount": finalizedCount,
	}, now)
	applyCompletedStep(run, workflow.InvestingStepEvaluateThesisAndChange, nil, map[string]any{
		"finalizedReviewCount": finalizedCount,
	}, now)
	applyCompletedStep(run, workflow.InvestingStepMapActions, nil, map[string]any{
		"finalizedReviewCount": finalizedCount,
	}, now)
	applyCompletedStep(run, workflow.InvestingStepAssignBuckets, nil, map[string]any{
		"finalizedReviewCount": finalizedCount,
	}, now)
	applyCompletedStep(run, workflow.InvestingStepBuildCapitalCandidates, nil, map[string]any{
		"buyCandidateCount": buyCandidates,
	}, now)
	applyCompletedStep(run, workflow.InvestingStepAllocateCapital, nil, map[string]any{
		"allocationPlanned": false,
	}, now)
	applyCompletedStep(run, workflow.InvestingStepPersistOutputs, nil, map[string]any{
		"persisted": true,
	}, now)
	applyCompletedStep(run, workflow.InvestingStepPublishRunSummary, nil, map[string]any{
		"partialCompletion": failedCount > 0 && finalizedCount > 0,
	}, now)

	run.ReviewsCreatedCount = finalizedCount
	run.ErrorsCount = failedCount
	if finalizedCount == 0 && failedCount > 0 {
		run.Status = domain.WorkflowRunStatusFailed
	} else {
		run.Status = domain.WorkflowRunStatusCompleted
	}
	run.CompletedAt = &now
	run.UpdatedAt = now
	run, err = service.workflowRuns.Update(ctx, run)
	if err != nil {
		return nil, err
	}
	if err := service.syncStepRuns(ctx, run); err != nil {
		return nil, err
	}

	return run, nil
}

func (service *Service) syncStepRuns(ctx context.Context, run *domain.WorkflowRun) error {
	if service.workflowSteps == nil || run == nil {
		return nil
	}

	for _, step := range run.StepStatuses {
		metadata := map[string]any{}
		if step.InputSnapshot != nil {
			metadata["inputSnapshot"] = step.InputSnapshot
		}
		if step.OutputSnapshot != nil {
			metadata["outputSnapshot"] = step.OutputSnapshot
		}
		if step.AsyncTask != nil {
			metadata["asyncTask"] = step.AsyncTask
		}
		if step.Error != nil {
			metadata["error"] = step.Error
		}
		_, err := service.workflowSteps.Upsert(ctx, &domain.WorkflowStepRun{
			WorkflowRunID: run.ID,
			StepName:      step.StepName,
			Status:        normalizeWorkflowStepStatus(step.Status),
			StartedAt:     step.StartedAt,
			CompletedAt:   step.CompletedAt,
			ErrorSummary:  errorMessage(step.Error),
			Metadata:      metadata,
			CreatedAt:     run.CreatedAt,
			UpdatedAt:     run.UpdatedAt,
		})
		if err != nil {
			return err
		}
	}

	return nil
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

func buildInputPayload(
	company *domain.Company,
	run *domain.WorkflowRun,
	configSnapshotID string,
	mode domain.InvestingMode,
	item ports.AIReviewBatchItem,
) map[string]any {
	payload := map[string]any{
		"companyId":        company.ID,
		"symbol":           company.Symbol,
		"exchange":         company.Exchange,
		"companyName":      company.CompanyName,
		"workflowRunId":    run.ID,
		"configSnapshotId": configSnapshotID,
		"schemaVersion":    run.SchemaVersion,
		"aiPromptVersion":  asString(item.Metadata["promptVersion"]),
		"modelName":        firstNonEmptyString(item.Model, run.Mode),
		"mode":             mode,
		"prompt":           item.Prompt,
		"referenceId":      item.ReferenceID,
		"templateRecord":   item.TemplateRecord,
		"metadata":         item.Metadata,
		"reasoningEffort":  item.ReasoningEffort,
		"itemType":         domain.BatchItemTypeCompanyReview,
		"numericFeatures": map[string]any{
			"marketCapBucket": company.MarketCapBucket,
			"statusActive":    company.StatusActive,
		},
		"textSourceMetadata": []map[string]any{},
	}

	return payload
}

func batchSubmitItemFromPayload(item *domain.AIBatchItem) ports.SubmitBatchItem {
	templateRecord, _ := item.InputPayload["templateRecord"].(map[string]any)
	metadata, _ := item.InputPayload["metadata"].(map[string]any)

	return ports.SubmitBatchItem{
		CorrelationID:   item.ID,
		ReferenceID:     asString(item.InputPayload["referenceId"]),
		ItemType:        item.ItemType,
		Prompt:          asString(item.InputPayload["prompt"]),
		InputPayload:    item.InputPayload,
		TemplateRecord:  templateRecord,
		Model:           asString(item.InputPayload["modelName"]),
		ReasoningEffort: asString(item.InputPayload["reasoningEffort"]),
		Metadata:        metadata,
	}
}

func extractStructuredReviewPayload(raw map[string]any) (map[string]any, error) {
	if raw == nil {
		return nil, fmt.Errorf("raw ai result payload is required")
	}
	for _, key := range []string{"review", "structuredReview", "parsedReview"} {
		if value, ok := raw[key].(map[string]any); ok {
			return value, nil
		}
	}
	if responseBody, ok := raw["resultResponseBody"].(map[string]any); ok {
		for _, key := range []string{"output_parsed", "parsed", "review"} {
			if value, ok := responseBody[key].(map[string]any); ok {
				return value, nil
			}
		}
	}
	if resultText := strings.TrimSpace(asString(raw["resultText"])); resultText != "" {
		var parsed map[string]any
		if err := json.Unmarshal([]byte(resultText), &parsed); err == nil {
			if review, ok := parsed["review"].(map[string]any); ok {
				return review, nil
			}
			if _, ok := parsed["sections"]; ok {
				return parsed, nil
			}
		}
	}
	if _, ok := raw["sections"]; ok {
		return raw, nil
	}

	return nil, fmt.Errorf("ai result payload does not include a parseable structured review")
}

func buildReviewCandidate(reviewShell *domain.CompanyReview, structured map[string]any) (*domain.CompanyReview, error) {
	payload, err := json.Marshal(structured)
	if err != nil {
		return nil, fmt.Errorf("marshal structured review: %w", err)
	}

	var parsed domain.CompanyReview
	if err := json.Unmarshal(payload, &parsed); err != nil {
		return nil, fmt.Errorf("unmarshal structured review: %w", err)
	}

	candidate := *reviewShell
	candidate.WeightedTotalScore = parsed.WeightedTotalScore
	candidate.HardGateFailed = parsed.HardGateFailed
	candidate.HardGateFailureReasons = parsed.HardGateFailureReasons
	candidate.ConfidenceScore = parsed.ConfidenceScore
	candidate.FinalBucketAfterReview = parsed.FinalBucketAfterReview
	candidate.FinalActionAfterReview = parsed.FinalActionAfterReview
	candidate.ActionRationaleSummary = parsed.ActionRationaleSummary
	candidate.WhatChangedSummary = parsed.WhatChangedSummary
	candidate.ReviewerType = domain.ReviewerTypeAI
	candidate.AIModelName = firstNonEmptyString(parsed.AIModelName, reviewShell.AIModelName)
	candidate.AIPromptVersion = firstNonEmptyString(parsed.AIPromptVersion, reviewShell.AIPromptVersion)
	candidate.ReviewMetadata = mergeMaps(reviewShell.ReviewMetadata, parsed.ReviewMetadata)
	candidate.Sections = parsed.Sections
	candidate.PositionSnapshot = firstNonNilPositionSnapshot(parsed.PositionSnapshot, reviewShell.PositionSnapshot)
	candidate.ChangeLog = parsed.ChangeLog
	candidate.DecisionAction = parsed.DecisionAction
	candidate.ReviewStatus = domain.ReviewStatusAIValidated
	candidate.ValidationStatus = domain.ValidationStatusValid
	candidate.ValidationErrors = nil
	candidate.RawAIResultPayload = reviewShell.RawAIResultPayload
	candidate.UpdatedAt = reviewShell.UpdatedAt

	return &candidate, nil
}

func hashPayload(value map[string]any) (string, error) {
	encoded, err := json.Marshal(value)
	if err != nil {
		return "", err
	}
	sum := sha256.Sum256(encoded)
	return hex.EncodeToString(sum[:]), nil
}

func mergeMaps(base map[string]any, overlay map[string]any) map[string]any {
	if len(base) == 0 && len(overlay) == 0 {
		return nil
	}
	merged := map[string]any{}
	for key, value := range base {
		merged[key] = value
	}
	for key, value := range overlay {
		merged[key] = value
	}
	return merged
}

func firstNonNilPositionSnapshot(values ...*domain.PositionSnapshot) *domain.PositionSnapshot {
	for _, value := range values {
		if value != nil {
			return value
		}
	}
	return nil
}

func normalizeWorkflowStepStatus(status domain.WorkflowStepStatusType) domain.WorkflowStepStatusType {
	if status == domain.WorkflowStepStatusWaitingAsync {
		return domain.WorkflowStepStatusWaitingExternal
	}
	return status
}

func errorMessage(stepError *domain.WorkflowStepError) string {
	if stepError == nil {
		return ""
	}
	return stepError.Message
}

func firstNonEmptyString(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}

func asString(value any) string {
	if value == nil {
		return ""
	}
	text, _ := value.(string)
	return text
}

func buildBatchIdempotencyKey(runKey string, workflowRunID string) string {
	if strings.TrimSpace(runKey) != "" {
		return runKey + "::investing-review-batch"
	}
	return workflowRunID + "::investing-review-batch"
}

type noopBatchEngineAdapter struct{}

func (noopBatchEngineAdapter) SubmitBatch(ctx context.Context, request ports.SubmitBatchRequest) (*ports.BatchSubmissionResult, error) {
	return (&platformserviceBatchNoop{}).SubmitBatch(ctx, request)
}

func (noopBatchEngineAdapter) GetBatchStatus(ctx context.Context, jobHandle string) (*ports.BatchStatusResult, error) {
	return (&platformserviceBatchNoop{}).GetBatchStatus(ctx, jobHandle)
}

func (noopBatchEngineAdapter) GetBatchResults(ctx context.Context, jobHandle string) (*ports.BatchResultsResult, error) {
	return (&platformserviceBatchNoop{}).GetBatchResults(ctx, jobHandle)
}

type platformserviceBatchNoop struct{}

func (platformserviceBatchNoop) SubmitBatch(_ context.Context, request ports.SubmitBatchRequest) (*ports.BatchSubmissionResult, error) {
	items := make([]ports.BatchSubmissionItem, 0, len(request.Items))
	for _, item := range request.Items {
		items = append(items, ports.BatchSubmissionItem{
			CorrelationID: item.CorrelationID,
			Status:        domain.BatchItemStatusPending,
		})
	}
	return &ports.BatchSubmissionResult{
		ProviderName: "noop",
		Status:       domain.BatchJobStatusCreated,
		Items:        items,
	}, nil
}

func (platformserviceBatchNoop) GetBatchStatus(_ context.Context, jobHandle string) (*ports.BatchStatusResult, error) {
	return &ports.BatchStatusResult{
		ProviderName:      "noop",
		ProviderJobHandle: jobHandle,
		Status:            domain.BatchJobStatusCreated,
	}, nil
}

func (platformserviceBatchNoop) GetBatchResults(_ context.Context, jobHandle string) (*ports.BatchResultsResult, error) {
	return &ports.BatchResultsResult{
		ProviderName:      "noop",
		ProviderJobHandle: jobHandle,
		Status:            domain.BatchJobStatusCreated,
	}, nil
}
