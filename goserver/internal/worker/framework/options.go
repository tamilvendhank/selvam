package framework

import (
	"fmt"
	"time"
)

type RunLoopOptions struct {
	Interval          time.Duration `json:"interval,omitempty"`
	InitialDelay      time.Duration `json:"initialDelay,omitempty"`
	MaxIterations     int           `json:"maxIterations,omitempty"`
	StopOnError       bool          `json:"stopOnError,omitempty"`
	RecoverPanics     bool          `json:"recoverPanics,omitempty"`
	JitterPct         float64       `json:"jitterPct,omitempty"`
	WorkerTimeout     time.Duration `json:"workerTimeout,omitempty"`
	LogEveryIteration bool          `json:"logEveryIteration,omitempty"`
	NameOverride      string        `json:"nameOverride,omitempty"`
}

func DefaultRunLoopOptions(interval time.Duration) RunLoopOptions {
	return RunLoopOptions{
		Interval:      interval,
		RecoverPanics: true,
	}
}

func RunOnceOptions(timeout time.Duration) RunLoopOptions {
	return RunLoopOptions{
		MaxIterations: 1,
		WorkerTimeout: timeout,
		RecoverPanics: true,
	}
}

func (options RunLoopOptions) Validate() error {
	if options.MaxIterations < 0 {
		return fmt.Errorf("%w: maxIterations must be zero or greater", ErrInvalidWorkerOptions)
	}
	if options.InitialDelay < 0 {
		return fmt.Errorf("%w: initialDelay must be zero or greater", ErrInvalidWorkerOptions)
	}
	if options.WorkerTimeout < 0 {
		return fmt.Errorf("%w: workerTimeout must be zero or greater", ErrInvalidWorkerOptions)
	}
	if options.JitterPct < 0 || options.JitterPct > 100 {
		return fmt.Errorf("%w: jitterPct must be between 0 and 100", ErrInvalidWorkerOptions)
	}
	if options.MaxIterations != 1 && options.Interval <= 0 {
		return fmt.Errorf("%w: interval must be positive for loop mode", ErrInvalidWorkerOptions)
	}
	return nil
}
