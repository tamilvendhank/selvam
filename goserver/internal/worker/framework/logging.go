package framework

import "context"

func (runner *Runner) logLoopStart(ctx context.Context, summary LoopRunSummary, options RunLoopOptions) {
	if runner == nil || runner.logger == nil {
		return
	}
	runner.logger.Info(ctx, "worker loop started", map[string]any{
		"workerName":        summary.WorkerName,
		"interval":          options.Interval.String(),
		"initialDelay":      options.InitialDelay.String(),
		"maxIterations":     options.MaxIterations,
		"workerTimeout":     options.WorkerTimeout.String(),
		"stopOnError":       options.StopOnError,
		"recoverPanics":     options.RecoverPanics,
		"logEveryIteration": options.LogEveryIteration,
	})
}

func (runner *Runner) logIterationResult(ctx context.Context, result WorkerRunResult, shouldLog bool) {
	if runner == nil || runner.logger == nil || !shouldLog {
		return
	}
	fields := map[string]any{
		"workerName":      result.WorkerName,
		"iteration":       result.Iteration,
		"success":         result.Success,
		"duration":        result.Duration.String(),
		"processedCount":  result.ProcessedCount,
		"succeededCount":  result.SucceededCount,
		"failedCount":     result.FailedCount,
		"skippedCount":    result.SkippedCount,
		"partialFailures": len(result.PartialFailures),
	}
	if result.ErrorSummary != "" {
		fields["error"] = result.ErrorSummary
	}
	if result.HasFailures() {
		runner.logger.Error(ctx, "worker iteration failed", fields)
		return
	}
	runner.logger.Info(ctx, "worker iteration completed", fields)
}

func (runner *Runner) logLoopFinish(ctx context.Context, summary LoopRunSummary) {
	if runner == nil || runner.logger == nil {
		return
	}
	fields := map[string]any{
		"workerName":             summary.WorkerName,
		"status":                 summary.Status,
		"duration":               summary.Duration.String(),
		"iterations":             summary.Iterations,
		"successfulIterations":   summary.SuccessfulIterations,
		"failedIterations":       summary.FailedIterations,
		"stoppedByContext":       summary.StoppedByContext,
		"stoppedByMaxIterations": summary.StoppedByMaxIterations,
		"stoppedByError":         summary.StoppedByError,
	}
	if summary.LastErrorSummary != "" {
		fields["lastError"] = summary.LastErrorSummary
	}
	if summary.Status == WorkerStatusFailed {
		runner.logger.Error(ctx, "worker loop stopped with failure", fields)
		return
	}
	if summary.Status == WorkerStatusStopped {
		runner.logger.Warn(ctx, "worker loop stopped", fields)
		return
	}
	runner.logger.Info(ctx, "worker loop completed", fields)
}
