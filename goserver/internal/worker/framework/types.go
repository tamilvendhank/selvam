package framework

import (
	"context"
	"time"
)

type Worker interface {
	Name() string
	RunOnce(ctx context.Context) WorkerRunResult
}

type LoopRunner interface {
	RunLoop(ctx context.Context, worker Worker, options RunLoopOptions) LoopRunSummary
}

type Logger interface {
	Info(ctx context.Context, msg string, fields map[string]any)
	Warn(ctx context.Context, msg string, fields map[string]any)
	Error(ctx context.Context, msg string, fields map[string]any)
}

type Clock interface {
	Now() time.Time
	Sleep(ctx context.Context, duration time.Duration) error
}

type SystemClock struct{}

func (SystemClock) Now() time.Time {
	return time.Now()
}

func (SystemClock) Sleep(ctx context.Context, duration time.Duration) error {
	if duration <= 0 {
		return nil
	}
	timer := time.NewTimer(duration)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}

type WorkerStatus string

const (
	WorkerStatusUnknown   WorkerStatus = ""
	WorkerStatusRunning   WorkerStatus = "running"
	WorkerStatusCompleted WorkerStatus = "completed"
	WorkerStatusStopped   WorkerStatus = "stopped"
	WorkerStatusFailed    WorkerStatus = "failed"
)

type WorkerFailure struct {
	Scope     string `json:"scope,omitempty"`
	EntityID  string `json:"entityId,omitempty"`
	Code      string `json:"code,omitempty"`
	Message   string `json:"message"`
	Retryable bool   `json:"retryable,omitempty"`
}

type WorkerRunResult struct {
	WorkerName      string          `json:"workerName"`
	Iteration       int             `json:"iteration,omitempty"`
	StartedAt       time.Time       `json:"startedAt"`
	CompletedAt     time.Time       `json:"completedAt"`
	Duration        time.Duration   `json:"duration"`
	Success         bool            `json:"success"`
	ProcessedCount  int             `json:"processedCount,omitempty"`
	SucceededCount  int             `json:"succeededCount,omitempty"`
	FailedCount     int             `json:"failedCount,omitempty"`
	SkippedCount    int             `json:"skippedCount,omitempty"`
	PartialFailures []WorkerFailure `json:"partialFailures,omitempty"`
	Error           error           `json:"-"`
	ErrorSummary    string          `json:"errorSummary,omitempty"`
	Metadata        map[string]any  `json:"metadata,omitempty"`
}

type LoopRunSummary struct {
	WorkerName             string           `json:"workerName"`
	StartedAt              time.Time        `json:"startedAt"`
	CompletedAt            time.Time        `json:"completedAt"`
	Duration               time.Duration    `json:"duration"`
	Status                 WorkerStatus     `json:"status,omitempty"`
	Iterations             int              `json:"iterations,omitempty"`
	SuccessfulIterations   int              `json:"successfulIterations,omitempty"`
	FailedIterations       int              `json:"failedIterations,omitempty"`
	LastError              error            `json:"-"`
	LastErrorSummary       string           `json:"lastErrorSummary,omitempty"`
	StoppedByContext       bool             `json:"stoppedByContext,omitempty"`
	StoppedByMaxIterations bool             `json:"stoppedByMaxIterations,omitempty"`
	StoppedByError         bool             `json:"stoppedByError,omitempty"`
	LastResult             *WorkerRunResult `json:"lastResult,omitempty"`
	Metadata               map[string]any   `json:"metadata,omitempty"`
}

type WorkerStatusSnapshot struct {
	WorkerName       string           `json:"workerName"`
	Status           WorkerStatus     `json:"status,omitempty"`
	Iteration        int              `json:"iteration,omitempty"`
	LastStartedAt    *time.Time       `json:"lastStartedAt,omitempty"`
	LastCompletedAt  *time.Time       `json:"lastCompletedAt,omitempty"`
	LastErrorSummary string           `json:"lastErrorSummary,omitempty"`
	LastResult       *WorkerRunResult `json:"lastResult,omitempty"`
	UpdatedAt        time.Time        `json:"updatedAt"`
}
