package worker

import (
	"context"
	"sync"
	"time"

	"goserver/internal/platform/domain"
	"goserver/internal/platform/ports"
)

type LoopWorker interface {
	RunOnce(ctx context.Context) error
}

type Supervisor struct {
	interval time.Duration
	workers  []LoopWorker
}

func NewSupervisor(interval time.Duration, workers ...LoopWorker) *Supervisor {
	return &Supervisor{
		interval: interval,
		workers:  workers,
	}
}

func (supervisor *Supervisor) Start(ctx context.Context) {
	if supervisor == nil || supervisor.interval <= 0 || len(supervisor.workers) == 0 {
		return
	}

	go func() {
		ticker := time.NewTicker(supervisor.interval)
		defer ticker.Stop()

		_ = supervisor.runPass(ctx)
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				_ = supervisor.runPass(ctx)
			}
		}
	}()
}

func (supervisor *Supervisor) runPass(ctx context.Context) error {
	for _, worker := range supervisor.workers {
		if err := worker.RunOnce(ctx); err != nil {
			return err
		}
	}

	return nil
}

type BatchSubmissionWorker struct {
	jobs     ports.AIBatchJobRepository
	workflows ports.WorkflowService
	limit    int
	mu       sync.Mutex
}

func NewBatchSubmissionWorker(jobs ports.AIBatchJobRepository, workflows ports.WorkflowService, limit int) *BatchSubmissionWorker {
	return &BatchSubmissionWorker{jobs: jobs, workflows: workflows, limit: limit}
}

func (worker *BatchSubmissionWorker) RunOnce(ctx context.Context) error {
	worker.mu.Lock()
	defer worker.mu.Unlock()

	return worker.reconcileRuns(ctx, []domain.BatchJobStatus{domain.BatchJobStatusCreated}, true)
}

type BatchPollingWorker struct {
	jobs      ports.AIBatchJobRepository
	workflows ports.WorkflowService
	limit     int
	mu        sync.Mutex
}

func NewBatchPollingWorker(jobs ports.AIBatchJobRepository, workflows ports.WorkflowService, limit int) *BatchPollingWorker {
	return &BatchPollingWorker{jobs: jobs, workflows: workflows, limit: limit}
}

func (worker *BatchPollingWorker) RunOnce(ctx context.Context) error {
	worker.mu.Lock()
	defer worker.mu.Unlock()

	return worker.reconcileRuns(ctx, []domain.BatchJobStatus{
		domain.BatchJobStatusSubmitted,
		domain.BatchJobStatusRunning,
	}, false)
}

type ResultReconciliationWorker struct {
	jobs      ports.AIBatchJobRepository
	workflows ports.WorkflowService
	limit     int
	mu        sync.Mutex
}

func NewResultReconciliationWorker(jobs ports.AIBatchJobRepository, workflows ports.WorkflowService, limit int) *ResultReconciliationWorker {
	return &ResultReconciliationWorker{jobs: jobs, workflows: workflows, limit: limit}
}

func (worker *ResultReconciliationWorker) RunOnce(ctx context.Context) error {
	worker.mu.Lock()
	defer worker.mu.Unlock()

	return worker.reconcileRuns(ctx, []domain.BatchJobStatus{
		domain.BatchJobStatusCompleted,
		domain.BatchJobStatusPartiallyCompleted,
		domain.BatchJobStatusFailed,
	}, false)
}

type WorkflowContinuationWorker struct {
	runs      ports.WorkflowRunRepository
	workflows ports.WorkflowService
	limit     int
	mu        sync.Mutex
}

func NewWorkflowContinuationWorker(runs ports.WorkflowRunRepository, workflows ports.WorkflowService, limit int) *WorkflowContinuationWorker {
	return &WorkflowContinuationWorker{runs: runs, workflows: workflows, limit: limit}
}

func (worker *WorkflowContinuationWorker) RunOnce(ctx context.Context) error {
	worker.mu.Lock()
	defer worker.mu.Unlock()

	runs, err := worker.runs.List(ctx, ports.WorkflowRunListFilter{
		Status: domain.WorkflowRunStatusWaitingAsync,
		Limit:  worker.limit,
	})
	if err != nil {
		return err
	}
	for _, run := range runs {
		if run == nil {
			continue
		}
		if _, err := worker.workflows.ReconcileWorkflow(ctx, run.ID); err != nil {
			return err
		}
	}

	return nil
}

func (worker *BatchSubmissionWorker) reconcileRuns(ctx context.Context, statuses []domain.BatchJobStatus, resume bool) error {
	return reconcileJobsByStatuses(ctx, worker.jobs, worker.workflows, worker.limit, statuses, resume)
}

func (worker *BatchPollingWorker) reconcileRuns(ctx context.Context, statuses []domain.BatchJobStatus, resume bool) error {
	return reconcileJobsByStatuses(ctx, worker.jobs, worker.workflows, worker.limit, statuses, resume)
}

func (worker *ResultReconciliationWorker) reconcileRuns(ctx context.Context, statuses []domain.BatchJobStatus, resume bool) error {
	return reconcileJobsByStatuses(ctx, worker.jobs, worker.workflows, worker.limit, statuses, resume)
}

func reconcileJobsByStatuses(
	ctx context.Context,
	jobs ports.AIBatchJobRepository,
	workflows ports.WorkflowService,
	limit int,
	statuses []domain.BatchJobStatus,
	resume bool,
) error {
	seen := map[string]struct{}{}
	for _, status := range statuses {
		list, err := jobs.List(ctx, ports.AIBatchJobListFilter{
			Status: status,
			Limit:  limit,
		})
		if err != nil {
			return err
		}
		for _, job := range list {
			if job == nil || job.WorkflowRunID == "" {
				continue
			}
			if _, ok := seen[job.WorkflowRunID]; ok {
				continue
			}
			seen[job.WorkflowRunID] = struct{}{}
			if resume {
				if _, err := workflows.ResumeWorkflow(ctx, job.WorkflowRunID); err != nil {
					return err
				}
				continue
			}
			if _, err := workflows.ReconcileWorkflow(ctx, job.WorkflowRunID); err != nil {
				return err
			}
		}
	}

	return nil
}
