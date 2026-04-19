package service

import (
	"context"
	"time"

	"go.uber.org/zap"
)

type SubmissionRefreshWorker struct {
	jobsService                *JobsService
	procedureExecutionsService *ProcedureExecutionsService
	interval                   time.Duration
	logger                     *zap.Logger
}

func NewSubmissionRefreshWorker(
	jobsService *JobsService,
	procedureExecutionsService *ProcedureExecutionsService,
	interval time.Duration,
	logger *zap.Logger,
) *SubmissionRefreshWorker {
	if logger == nil {
		logger = zap.NewNop()
	}

	return &SubmissionRefreshWorker{
		jobsService:                jobsService,
		procedureExecutionsService: procedureExecutionsService,
		interval:                   interval,
		logger:                     logger,
	}
}

func (worker *SubmissionRefreshWorker) Start(ctx context.Context) {
	if worker == nil || worker.jobsService == nil || worker.interval <= 0 {
		return
	}

	ticker := time.NewTicker(worker.interval)

	go func() {
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				if err := worker.jobsService.RunRefreshPass(ctx); err != nil && ctx.Err() == nil {
					worker.logger.Error("submission refresh worker failed", zap.Error(err))
				}
				if worker.procedureExecutionsService != nil {
					if err := worker.procedureExecutionsService.RunProgressPass(ctx); err != nil && ctx.Err() == nil {
						worker.logger.Error("procedure execution reconciliation failed", zap.Error(err))
					}
				}
			}
		}
	}()
}
