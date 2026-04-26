package async

import (
	"context"
	"errors"
	"fmt"

	servicecommon "goserver/internal/service/common"
	"goserver/internal/worker/framework"
)

const (
	BatchSubmissionWorkerName       = "batch_submission_worker"
	BatchPollingWorkerName          = "batch_polling_worker"
	BatchReconciliationWorkerName   = "batch_reconciliation_worker"
	AIOutputValidationWorkerName    = "ai_output_validation_worker"
	ReviewMaterializationWorkerName = "review_materialization_worker"
	ReviewFinalizationWorkerName    = "review_finalization_worker"
	WorkflowContinuationWorkerName  = "workflow_continuation_worker"
)

type workerBase struct {
	base    framework.BaseWorker
	options AsyncWorkerOptions
}

func newWorkerBase(name string, options AsyncWorkerOptions, clock framework.Clock) workerBase {
	return workerBase{
		base:    framework.NewBaseWorker(name, clock),
		options: options,
	}
}

func (worker workerBase) Name() string {
	return worker.base.Name()
}

func (worker workerBase) validate(ctx context.Context) error {
	if err := ctx.Err(); err != nil {
		return fmt.Errorf("%w: %v", framework.ErrWorkerStopped, err)
	}
	return worker.options.Validate()
}

func dependencyFailure(workerName string, dependency string) framework.WorkerRunResult {
	return framework.ResultFromError(workerName, fmt.Errorf("%s is required", dependency))
}

func invalidOptionsFailure(workerName string, err error) framework.WorkerRunResult {
	return framework.ResultFromError(workerName, fmt.Errorf("%w: %v", framework.ErrInvalidWorkerOptions, err))
}

func validationFailure(workerName string, err error) framework.WorkerRunResult {
	if errors.Is(err, framework.ErrWorkerStopped) {
		return framework.ResultFromError(workerName, err)
	}
	return invalidOptionsFailure(workerName, err)
}

func expectedNoopError(err error) bool {
	return errors.Is(err, servicecommon.ErrNothingToSubmit) ||
		errors.Is(err, servicecommon.ErrNothingToPoll) ||
		errors.Is(err, servicecommon.ErrNothingToReconcile) ||
		errors.Is(err, servicecommon.ErrNothingToValidate) ||
		errors.Is(err, servicecommon.ErrNothingToMaterialize) ||
		errors.Is(err, servicecommon.ErrNothingToFinalize)
}

func noopResult(workerName string, err error, metadata map[string]any) framework.WorkerRunResult {
	result := framework.NewSuccessResult(workerName, 0, metadata)
	if err != nil {
		if result.Metadata == nil {
			result.Metadata = map[string]any{}
		}
		result.Metadata["noop_reason"] = err.Error()
	}
	return result
}

func mergeMetadata(base map[string]any, extra map[string]any) map[string]any {
	if len(base) == 0 && len(extra) == 0 {
		return nil
	}
	merged := make(map[string]any, len(base)+len(extra))
	for key, value := range base {
		merged[key] = value
	}
	for key, value := range extra {
		merged[key] = value
	}
	return merged
}

func maxInt(left int, right int) int {
	if left > right {
		return left
	}
	return right
}

func effectiveMaxItems(options AsyncWorkerOptions) int {
	if options.MaxItemsPerRun > 0 {
		return options.MaxItemsPerRun
	}
	return options.MaxReviewsPerRun
}
