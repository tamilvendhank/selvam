package investing

import (
	"context"
	"testing"
	"time"

	platformconfig "goserver/internal/platform/config"
	"goserver/internal/platform/domain"
	"goserver/internal/platform/ports"
	platformservice "goserver/internal/platform/service"
	"goserver/internal/platform/testutil"
)

type workflowRunRepoStub struct {
	run *domain.WorkflowRun
}

func (stub *workflowRunRepoStub) Create(_ context.Context, run *domain.WorkflowRun) (*domain.WorkflowRun, error) {
	run.ID = "run-1"
	stub.run = run
	return run, nil
}
func (stub *workflowRunRepoStub) Update(_ context.Context, run *domain.WorkflowRun) (*domain.WorkflowRun, error) {
	stub.run = run
	return run, nil
}
func (stub *workflowRunRepoStub) GetByID(_ context.Context, id string) (*domain.WorkflowRun, error) {
	if stub.run != nil && stub.run.ID == id {
		return stub.run, nil
	}
	return nil, nil
}
func (stub *workflowRunRepoStub) GetByIdempotencyKey(_ context.Context, key string) (*domain.WorkflowRun, error) {
	if stub.run != nil && stub.run.IdempotencyKey == key {
		return stub.run, nil
	}
	return nil, nil
}
func (stub *workflowRunRepoStub) List(context.Context, ports.WorkflowRunListFilter) ([]*domain.WorkflowRun, error) {
	if stub.run == nil {
		return nil, nil
	}
	return []*domain.WorkflowRun{stub.run}, nil
}

type companyRepoStub struct {
	items []*domain.Company
}

func (stub *companyRepoStub) Create(context.Context, *domain.Company) (*domain.Company, error) {
	return nil, nil
}
func (stub *companyRepoStub) Update(context.Context, *domain.Company) (*domain.Company, error) {
	return nil, nil
}
func (stub *companyRepoStub) GetByID(context.Context, string) (*domain.Company, error) {
	return nil, nil
}
func (stub *companyRepoStub) GetBySymbol(context.Context, string) (*domain.Company, error) {
	return nil, nil
}
func (stub *companyRepoStub) List(context.Context, ports.CompanyListFilter) ([]*domain.Company, error) {
	return stub.items, nil
}

type reviewRepoStub struct {
	items map[string]*domain.CompanyReview
}

func (stub *reviewRepoStub) Create(_ context.Context, review *domain.CompanyReview) (*domain.CompanyReview, error) {
	if stub.items == nil {
		stub.items = map[string]*domain.CompanyReview{}
	}
	if review.ID == "" {
		review.ID = "review-" + review.CompanyID
	}
	stub.items[review.ID] = review
	return review, nil
}
func (stub *reviewRepoStub) UpdateMutable(_ context.Context, review *domain.CompanyReview) (*domain.CompanyReview, error) {
	stub.items[review.ID] = review
	return review, nil
}
func (stub *reviewRepoStub) UpdateDraft(ctx context.Context, review *domain.CompanyReview) (*domain.CompanyReview, error) {
	return stub.UpdateMutable(ctx, review)
}
func (stub *reviewRepoStub) Finalize(_ context.Context, reviewID string) (*domain.CompanyReview, error) {
	review := stub.items[reviewID]
	review.ReviewStatus = domain.ReviewStatusFinalized
	return review, nil
}
func (stub *reviewRepoStub) MarkSuperseded(_ context.Context, reviewID string) (*domain.CompanyReview, error) {
	review := stub.items[reviewID]
	review.ReviewStatus = domain.ReviewStatusSuperseded
	return review, nil
}
func (stub *reviewRepoStub) GetByID(_ context.Context, id string) (*domain.CompanyReview, error) {
	return stub.items[id], nil
}
func (stub *reviewRepoStub) GetLatestByCompany(_ context.Context, companyID string, _ domain.BookType) (*domain.CompanyReview, error) {
	for _, review := range stub.items {
		if review.CompanyID == companyID {
			return review, nil
		}
	}
	return nil, nil
}
func (stub *reviewRepoStub) GetLatestComparableByCompany(_ context.Context, companyID string, _ domain.BookType, excludeReviewID string) (*domain.CompanyReview, error) {
	for _, review := range stub.items {
		if review.CompanyID == companyID && review.ID != excludeReviewID && review.ReviewStatus == domain.ReviewStatusFinalized {
			return review, nil
		}
	}
	return nil, nil
}
func (stub *reviewRepoStub) List(_ context.Context, filter ports.CompanyReviewListFilter) ([]*domain.CompanyReview, error) {
	result := make([]*domain.CompanyReview, 0, len(stub.items))
	for _, review := range stub.items {
		if filter.BookType != "" && review.BookType != filter.BookType {
			continue
		}
		result = append(result, review)
	}
	return result, nil
}

type thesisRepoStub struct{}

func (thesisRepoStub) Create(_ context.Context, thesis *domain.InvestmentThesis) (*domain.InvestmentThesis, error) {
	thesis.ID = "thesis-1"
	return thesis, nil
}
func (thesisRepoStub) Update(_ context.Context, thesis *domain.InvestmentThesis) (*domain.InvestmentThesis, error) {
	return thesis, nil
}
func (thesisRepoStub) GetByID(context.Context, string) (*domain.InvestmentThesis, error) {
	return nil, nil
}
func (thesisRepoStub) GetActiveByCompanyID(context.Context, string) (*domain.InvestmentThesis, error) {
	return nil, nil
}
func (thesisRepoStub) ListByCompanyID(context.Context, string) ([]*domain.InvestmentThesis, error) {
	return nil, nil
}

type workflowStepRunRepoStub struct{}

func (workflowStepRunRepoStub) Create(_ context.Context, run *domain.WorkflowStepRun) (*domain.WorkflowStepRun, error) {
	run.ID = run.StepName
	return run, nil
}
func (workflowStepRunRepoStub) Upsert(_ context.Context, run *domain.WorkflowStepRun) (*domain.WorkflowStepRun, error) {
	run.ID = run.StepName
	return run, nil
}
func (workflowStepRunRepoStub) GetByID(context.Context, string) (*domain.WorkflowStepRun, error) {
	return nil, nil
}
func (workflowStepRunRepoStub) GetByWorkflowRunAndStep(context.Context, string, string) (*domain.WorkflowStepRun, error) {
	return nil, nil
}
func (workflowStepRunRepoStub) List(context.Context, ports.WorkflowStepRunListFilter) ([]*domain.WorkflowStepRun, error) {
	return nil, nil
}

type configServiceStub struct{}

func (configServiceStub) CurrentConfig(context.Context) (map[string]any, error) {
	return map[string]any{}, nil
}
func (configServiceStub) CreateSnapshot(context.Context, domain.BookType, string) (*domain.ConfigSnapshot, error) {
	return &domain.ConfigSnapshot{ID: "snapshot-1"}, nil
}
func (configServiceStub) ListSnapshots(context.Context, ports.ConfigSnapshotListFilter) ([]*domain.ConfigSnapshot, error) {
	return nil, nil
}
func (configServiceStub) GetSnapshot(context.Context, string) (*domain.ConfigSnapshot, error) {
	return nil, nil
}

type aiBatchJobRepoStub struct {
	job *domain.AIBatchJob
}

func (stub *aiBatchJobRepoStub) Create(_ context.Context, job *domain.AIBatchJob) (*domain.AIBatchJob, error) {
	job.ID = "batch-job-1"
	stub.job = job
	return job, nil
}
func (stub *aiBatchJobRepoStub) Update(_ context.Context, job *domain.AIBatchJob) (*domain.AIBatchJob, error) {
	stub.job = job
	return job, nil
}
func (stub *aiBatchJobRepoStub) GetByID(_ context.Context, id string) (*domain.AIBatchJob, error) {
	if stub.job != nil && stub.job.ID == id {
		return stub.job, nil
	}
	return nil, nil
}
func (stub *aiBatchJobRepoStub) GetByIdempotencyKey(context.Context, string) (*domain.AIBatchJob, error) {
	return nil, nil
}
func (stub *aiBatchJobRepoStub) List(context.Context, ports.AIBatchJobListFilter) ([]*domain.AIBatchJob, error) {
	if stub.job == nil {
		return nil, nil
	}
	return []*domain.AIBatchJob{stub.job}, nil
}

type aiBatchItemRepoStub struct {
	items map[string]*domain.AIBatchItem
}

func (stub *aiBatchItemRepoStub) Create(_ context.Context, item *domain.AIBatchItem) (*domain.AIBatchItem, error) {
	if stub.items == nil {
		stub.items = map[string]*domain.AIBatchItem{}
	}
	item.ID = "item-1"
	stub.items[item.ID] = item
	return item, nil
}
func (stub *aiBatchItemRepoStub) CreateMany(_ context.Context, items []*domain.AIBatchItem) ([]*domain.AIBatchItem, error) {
	if stub.items == nil {
		stub.items = map[string]*domain.AIBatchItem{}
	}
	for index, item := range items {
		item.ID = "item-" + item.CompanyID
		if item.ID == "item-" {
			item.ID = "item-" + time.Now().UTC().Format("150405") + "-" + string(rune(index+'0'))
		}
		stub.items[item.ID] = item
	}
	return items, nil
}
func (stub *aiBatchItemRepoStub) Update(_ context.Context, item *domain.AIBatchItem) (*domain.AIBatchItem, error) {
	stub.items[item.ID] = item
	return item, nil
}
func (stub *aiBatchItemRepoStub) GetByID(_ context.Context, id string) (*domain.AIBatchItem, error) {
	return stub.items[id], nil
}
func (stub *aiBatchItemRepoStub) List(_ context.Context, filter ports.AIBatchItemListFilter) ([]*domain.AIBatchItem, error) {
	result := make([]*domain.AIBatchItem, 0, len(stub.items))
	for _, item := range stub.items {
		if filter.AIBatchJobID != "" && item.AIBatchJobID != filter.AIBatchJobID {
			continue
		}
		if filter.WorkflowRunID != "" && item.WorkflowRunID != filter.WorkflowRunID {
			continue
		}
		result = append(result, item)
	}
	return result, nil
}

type reconciliationLogRepoStub struct{}

func (reconciliationLogRepoStub) Create(_ context.Context, log *domain.JobReconciliationLog) (*domain.JobReconciliationLog, error) {
	log.ID = "log-1"
	return log, nil
}
func (reconciliationLogRepoStub) ListByJobID(context.Context, string, int) ([]*domain.JobReconciliationLog, error) {
	return nil, nil
}

type aiBatchEngineStub struct{}

func (aiBatchEngineStub) SubmitBatch(_ context.Context, request ports.SubmitBatchRequest) (*ports.BatchSubmissionResult, error) {
	now := time.Now().UTC()
	items := make([]ports.BatchSubmissionItem, 0, len(request.Items))
	for _, item := range request.Items {
		items = append(items, ports.BatchSubmissionItem{
			CorrelationID: item.CorrelationID,
			Status:        domain.BatchItemStatusSubmitted,
		})
	}
	return &ports.BatchSubmissionResult{
		ProviderName:      "stub",
		ProviderJobHandle: "provider-job-1",
		LocalJobHandle:    "local-job-1",
		Status:            domain.BatchJobStatusSubmitted,
		SubmittedAt:       &now,
		Items:             items,
	}, nil
}
func (aiBatchEngineStub) GetBatchStatus(context.Context, string) (*ports.BatchStatusResult, error) {
	now := time.Now().UTC()
	return &ports.BatchStatusResult{
		ProviderName:      "stub",
		ProviderJobHandle: "provider-job-1",
		Status:            domain.BatchJobStatusRunning,
		LastPolledAt:      &now,
	}, nil
}
func (aiBatchEngineStub) GetBatchResults(context.Context, string) (*ports.BatchResultsResult, error) {
	return &ports.BatchResultsResult{
		ProviderName:      "stub",
		ProviderJobHandle: "provider-job-1",
		Status:            domain.BatchJobStatusCompleted,
	}, nil
}

func TestDryRunCompletesWithoutAsyncSubmission(t *testing.T) {
	config := platformconfig.Default()
	service := NewService(
		config,
		&companyRepoStub{items: []*domain.Company{testutil.SampleCompany()}},
		&reviewRepoStub{items: map[string]*domain.CompanyReview{}},
		thesisRepoStub{},
		&workflowRunRepoStub{},
		workflowStepRunRepoStub{},
		configServiceStub{},
		platformservice.NewScorecardService(config),
		platformservice.NewActionMappingService(config),
		platformservice.NewChangeDetectionService(config),
		platformservice.NewThesisService(thesisRepoStub{}, nil),
		&aiBatchJobRepoStub{},
		&aiBatchItemRepoStub{items: map[string]*domain.AIBatchItem{}},
		reconciliationLogRepoStub{},
		aiBatchEngineStub{},
		nil,
	)

	run, err := service.DryRun(context.Background(), ports.StartInvestingWorkflowRequest{Mode: domain.InvestingModeBalanced})
	if err != nil {
		t.Fatalf("DryRun returned error: %v", err)
	}
	if run.Status != domain.WorkflowRunStatusCompleted {
		t.Fatalf("expected completed dry run, got %s", run.Status)
	}
}

func TestStartWaitingAsync(t *testing.T) {
	config := platformconfig.Default()
	service := NewService(
		config,
		&companyRepoStub{items: []*domain.Company{testutil.SampleCompany()}},
		&reviewRepoStub{items: map[string]*domain.CompanyReview{}},
		thesisRepoStub{},
		&workflowRunRepoStub{},
		workflowStepRunRepoStub{},
		configServiceStub{},
		platformservice.NewScorecardService(config),
		platformservice.NewActionMappingService(config),
		platformservice.NewChangeDetectionService(config),
		platformservice.NewThesisService(thesisRepoStub{}, nil),
		&aiBatchJobRepoStub{},
		&aiBatchItemRepoStub{items: map[string]*domain.AIBatchItem{}},
		reconciliationLogRepoStub{},
		aiBatchEngineStub{},
		nil,
	)

	run, err := service.Start(context.Background(), ports.StartInvestingWorkflowRequest{Mode: domain.InvestingModeBalanced})
	if err != nil {
		t.Fatalf("Start returned error: %v", err)
	}
	if run.Status != domain.WorkflowRunStatusWaitingAsync {
		t.Fatalf("expected waiting_async run, got %s", run.Status)
	}
}
