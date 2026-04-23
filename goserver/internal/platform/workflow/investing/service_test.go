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

type aiReviewEngineStub struct{}

func (aiReviewEngineStub) SubmitReviewBatch(_ context.Context, request ports.AIReviewBatchRequest) (*ports.AIAsyncTask, error) {
	now := time.Now().UTC()
	return &ports.AIAsyncTask{
		Provider:            "stub",
		TaskKind:            "review_batch",
		LocalObjectType:     "submission",
		LocalObjectID:       "submission-1",
		SubmissionID:        "submission-1",
		RepresentativeJobID: "job-1",
		BatchID:             "batch-1",
		JobIDs:              []string{"job-1"},
		Status:              "queued",
		SubmittedAt:         &now,
		Metadata: map[string]any{
			"itemCount": len(request.Items),
		},
	}, nil
}
func (aiReviewEngineStub) RefreshTask(_ context.Context, task ports.AIAsyncTask) (*ports.AIAsyncTask, error) {
	return &task, nil
}

func TestDryRunCompletesWithoutAsyncSubmission(t *testing.T) {
	config := platformconfig.Default()
	service := NewService(
		config,
		&companyRepoStub{items: []*domain.Company{testutil.SampleCompany()}},
		&workflowRunRepoStub{},
		configServiceStub{},
		platformservice.NewScorecardService(config),
		aiReviewEngineStub{},
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
		&workflowRunRepoStub{},
		configServiceStub{},
		platformservice.NewScorecardService(config),
		aiReviewEngineStub{},
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
