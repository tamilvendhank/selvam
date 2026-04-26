package framework

import (
	"fmt"
	"time"
)

func NewSuccessResult(workerName string, processedCount int, metadata map[string]any) WorkerRunResult {
	now := time.Now().UTC()
	return WorkerRunResult{
		WorkerName:     workerName,
		StartedAt:      now,
		CompletedAt:    now,
		Success:        true,
		ProcessedCount: processedCount,
		SucceededCount: processedCount,
		Metadata:       metadata,
	}
}

func NewFailureResult(workerName string, err error, metadata map[string]any) WorkerRunResult {
	now := time.Now().UTC()
	result := WorkerRunResult{
		WorkerName:  workerName,
		StartedAt:   now,
		CompletedAt: now,
		Success:     false,
		FailedCount: 1,
		Metadata:    metadata,
	}
	result.SetError(err)
	return result
}

func ResultFromError(workerName string, err error) WorkerRunResult {
	return NewFailureResult(workerName, err, nil)
}

func (result WorkerRunResult) HasFailures() bool {
	return result.Error != nil || result.ErrorSummary != "" || result.FailedCount > 0 || len(result.PartialFailures) > 0
}

func (result WorkerRunResult) IsPartialSuccess() bool {
	return result.Success && result.HasFailures()
}

func (result *WorkerRunResult) AddFailure(failure WorkerFailure) {
	if failure.Message == "" {
		failure.Message = "worker failure"
	}
	result.PartialFailures = append(result.PartialFailures, failure)
	result.FailedCount++
	if result.ErrorSummary == "" {
		result.ErrorSummary = failure.Message
	}
}

func (result *WorkerRunResult) SetError(err error) {
	if err == nil {
		return
	}
	result.Error = err
	result.ErrorSummary = err.Error()
	result.Success = false
	if result.FailedCount == 0 && len(result.PartialFailures) == 0 {
		result.FailedCount = 1
	}
}

func (result *WorkerRunResult) Finish(clock Clock) {
	if clock == nil {
		clock = SystemClock{}
	}
	if result.StartedAt.IsZero() {
		result.StartedAt = clock.Now().UTC()
	}
	result.CompletedAt = clock.Now().UTC()
	result.Duration = result.CompletedAt.Sub(result.StartedAt)
	if result.HasFailures() {
		result.Success = false
		return
	}
	result.Success = true
	if result.ProcessedCount > 0 && result.SucceededCount == 0 {
		result.SucceededCount = result.ProcessedCount - result.SkippedCount
		if result.SucceededCount < 0 {
			result.SucceededCount = 0
		}
	}
}

func normalizeResult(result WorkerRunResult, workerName string, iteration int, startedAt time.Time, clock Clock) WorkerRunResult {
	if result.WorkerName == "" {
		result.WorkerName = workerName
	}
	if result.Iteration == 0 {
		result.Iteration = iteration
	}
	if result.StartedAt.IsZero() {
		result.StartedAt = startedAt.UTC()
	}
	if result.CompletedAt.IsZero() {
		result.Finish(clock)
	} else {
		result.CompletedAt = result.CompletedAt.UTC()
		result.Duration = result.CompletedAt.Sub(result.StartedAt)
		if result.HasFailures() {
			result.Success = false
		}
	}
	return result
}

func panicError(workerName string, recovered any) error {
	return fmt.Errorf("worker %q panic: %v", workerName, recovered)
}

func (summary LoopRunSummary) StatusSnapshot(clock Clock) WorkerStatusSnapshot {
	if clock == nil {
		clock = SystemClock{}
	}
	snapshot := WorkerStatusSnapshot{
		WorkerName:       summary.WorkerName,
		Status:           summary.Status,
		Iteration:        summary.Iterations,
		LastErrorSummary: summary.LastErrorSummary,
		LastResult:       summary.LastResult,
		UpdatedAt:        clock.Now().UTC(),
	}
	if summary.LastResult != nil {
		startedAt := summary.LastResult.StartedAt
		completedAt := summary.LastResult.CompletedAt
		if !startedAt.IsZero() {
			snapshot.LastStartedAt = &startedAt
		}
		if !completedAt.IsZero() {
			snapshot.LastCompletedAt = &completedAt
		}
	}
	return snapshot
}
