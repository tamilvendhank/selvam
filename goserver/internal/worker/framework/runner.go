package framework

import (
	"context"
	"errors"
	"fmt"
	"math"
	"time"
)

type Runner struct {
	logger Logger
	clock  Clock
}

type RunnerOption func(*Runner)

func WithLogger(logger Logger) RunnerOption {
	return func(runner *Runner) {
		runner.logger = logger
	}
}

func WithClock(clock Clock) RunnerOption {
	return func(runner *Runner) {
		if clock != nil {
			runner.clock = clock
		}
	}
}

func NewRunner(options ...RunnerOption) *Runner {
	runner := &Runner{clock: SystemClock{}}
	for _, option := range options {
		if option != nil {
			option(runner)
		}
	}
	if runner.clock == nil {
		runner.clock = SystemClock{}
	}
	return runner
}

func RunLoop(ctx context.Context, worker Worker, options RunLoopOptions, logger Logger) LoopRunSummary {
	return NewRunner(WithLogger(logger)).RunLoop(ctx, worker, options)
}

func RunOnce(ctx context.Context, worker Worker, options RunLoopOptions, logger Logger) WorkerRunResult {
	return NewRunner(WithLogger(logger)).RunOnce(ctx, worker, options)
}

func (runner *Runner) RunOnce(ctx context.Context, worker Worker, options RunLoopOptions) WorkerRunResult {
	if runner == nil {
		runner = NewRunner()
	}
	options.MaxIterations = 1
	if err := options.Validate(); err != nil {
		result := WorkerRunResult{
			WorkerName: resolvedWorkerName(worker, options),
			Iteration:  1,
			StartedAt:  runner.clock.Now().UTC(),
		}
		result.SetError(err)
		result.Finish(runner.clock)
		runner.logIterationResult(ctx, result, true)
		return result
	}
	result := runner.runIteration(ctx, worker, options, 1)
	runner.logIterationResult(ctx, result, true)
	return result
}

func (runner *Runner) RunLoop(ctx context.Context, worker Worker, options RunLoopOptions) LoopRunSummary {
	if runner == nil {
		runner = NewRunner()
	}
	startedAt := runner.clock.Now().UTC()
	summary := LoopRunSummary{
		WorkerName: resolvedWorkerName(worker, options),
		StartedAt:  startedAt,
		Status:     WorkerStatusRunning,
	}

	if worker == nil {
		return runner.finishLoopSummary(ctx, summary, ErrWorkerNil)
	}
	if err := options.Validate(); err != nil {
		return runner.finishLoopSummary(ctx, summary, err)
	}

	runner.logLoopStart(ctx, summary, options)
	if err := runner.clock.Sleep(ctx, options.InitialDelay); err != nil {
		summary.StoppedByContext = true
		return runner.finishLoopSummary(ctx, summary, fmt.Errorf("%w: %v", ErrWorkerStopped, err))
	}

	for {
		if ctx.Err() != nil {
			summary.StoppedByContext = true
			break
		}
		if options.MaxIterations > 0 && summary.Iterations >= options.MaxIterations {
			summary.StoppedByMaxIterations = true
			break
		}

		iteration := summary.Iterations + 1
		result := runner.runIteration(ctx, worker, options, iteration)
		summary.recordResult(result)
		runner.logIterationResult(ctx, result, options.LogEveryIteration || result.HasFailures())

		if result.HasFailures() && options.StopOnError {
			summary.StoppedByError = true
			break
		}
		if options.MaxIterations > 0 && summary.Iterations >= options.MaxIterations {
			summary.StoppedByMaxIterations = true
			break
		}

		sleepFor := intervalWithJitter(options.Interval, options.JitterPct, iteration, runner.clock)
		if err := runner.clock.Sleep(ctx, sleepFor); err != nil {
			summary.StoppedByContext = true
			break
		}
	}

	return runner.finishLoopSummary(ctx, summary, nil)
}

func (runner *Runner) runIteration(
	ctx context.Context,
	worker Worker,
	options RunLoopOptions,
	iteration int,
) (result WorkerRunResult) {
	if runner == nil {
		runner = NewRunner()
	}
	workerName := resolvedWorkerName(worker, options)
	startedAt := runner.clock.Now().UTC()
	result = WorkerRunResult{
		WorkerName: workerName,
		Iteration:  iteration,
		StartedAt:  startedAt,
	}
	if worker == nil {
		result.SetError(ErrWorkerNil)
		result.Finish(runner.clock)
		return result
	}
	if ctx.Err() != nil {
		result.SetError(fmt.Errorf("%w: %v", ErrWorkerStopped, ctx.Err()))
		result.Finish(runner.clock)
		return result
	}

	if options.RecoverPanics {
		defer func() {
			if recovered := recover(); recovered != nil {
				result = WorkerRunResult{
					WorkerName: workerName,
					Iteration:  iteration,
					StartedAt:  startedAt,
				}
				result.SetError(panicError(workerName, recovered))
				result.Finish(runner.clock)
			}
		}()
	}

	iterationContext := ctx
	cancel := func() {}
	if options.WorkerTimeout > 0 {
		iterationContext, cancel = context.WithTimeout(ctx, options.WorkerTimeout)
	}
	defer cancel()

	result = worker.RunOnce(iterationContext)
	result = normalizeResult(result, workerName, iteration, startedAt, runner.clock)
	if errors.Is(iterationContext.Err(), context.DeadlineExceeded) {
		result.SetError(ErrWorkerTimeout)
	} else if ctx.Err() != nil && !result.HasFailures() {
		result.SetError(fmt.Errorf("%w: %v", ErrWorkerStopped, ctx.Err()))
	}
	return result
}

func (summary *LoopRunSummary) recordResult(result WorkerRunResult) {
	summary.Iterations++
	if result.HasFailures() {
		summary.FailedIterations++
		if result.Error != nil {
			summary.LastError = result.Error
		}
		if result.ErrorSummary != "" {
			summary.LastErrorSummary = result.ErrorSummary
		}
	} else {
		summary.SuccessfulIterations++
	}
	copyResult := result
	summary.LastResult = &copyResult
}

func (runner *Runner) finishLoopSummary(ctx context.Context, summary LoopRunSummary, err error) LoopRunSummary {
	completedAt := runner.clock.Now().UTC()
	summary.CompletedAt = completedAt
	summary.Duration = completedAt.Sub(summary.StartedAt)
	if err != nil {
		summary.LastError = err
		summary.LastErrorSummary = err.Error()
	}
	switch {
	case summary.StoppedByContext:
		summary.Status = WorkerStatusStopped
	case summary.LastError != nil || summary.StoppedByError:
		summary.Status = WorkerStatusFailed
	default:
		summary.Status = WorkerStatusCompleted
	}
	runner.logLoopFinish(ctx, summary)
	return summary
}

func resolvedWorkerName(worker Worker, options RunLoopOptions) string {
	if options.NameOverride != "" {
		return options.NameOverride
	}
	if worker == nil {
		return "unknown_worker"
	}
	if name := worker.Name(); name != "" {
		return name
	}
	return "unnamed_worker"
}

func intervalWithJitter(base time.Duration, jitterPct float64, iteration int, clock Clock) time.Duration {
	if base <= 0 || jitterPct <= 0 {
		return base
	}
	maxOffset := int64(float64(base) * jitterPct / 100)
	if maxOffset <= 0 {
		return base
	}
	if clock == nil {
		clock = SystemClock{}
	}
	seed := clock.Now().UnixNano() + int64(iteration)*7919
	seed = int64(math.Abs(float64(seed)))
	offset := seed%(2*maxOffset+1) - maxOffset
	jittered := base + time.Duration(offset)
	if jittered <= 0 {
		return base
	}
	return jittered
}
