package framework

import "context"

type BaseWorker struct {
	name  string
	clock Clock
}

func NewBaseWorker(name string, clock Clock) BaseWorker {
	if clock == nil {
		clock = SystemClock{}
	}
	return BaseWorker{name: name, clock: clock}
}

func (worker BaseWorker) Name() string {
	return worker.name
}

func (worker BaseWorker) NewResult() WorkerRunResult {
	now := worker.clock.Now().UTC()
	return WorkerRunResult{
		WorkerName: worker.name,
		StartedAt:  now,
		Metadata:   map[string]any{},
	}
}

func (worker BaseWorker) FinishResult(result WorkerRunResult) WorkerRunResult {
	result.Finish(worker.clock)
	return result
}

type WorkerFunc struct {
	name string
	fn   func(ctx context.Context) WorkerRunResult
}

func NewWorkerFunc(name string, fn func(ctx context.Context) WorkerRunResult) WorkerFunc {
	return WorkerFunc{name: name, fn: fn}
}

func (worker WorkerFunc) Name() string {
	return worker.name
}

func (worker WorkerFunc) RunOnce(ctx context.Context) WorkerRunResult {
	if worker.fn == nil {
		return ResultFromError(worker.name, ErrWorkerNil)
	}
	return worker.fn(ctx)
}
