package validation

import "context"

type AIOutputValidationService interface {
	ValidateBatchItemOutput(ctx context.Context, request ValidateBatchItemOutputRequest) (*ValidateBatchItemOutputResult, error)
	ValidatePendingAIOutputs(ctx context.Context, request ValidatePendingAIOutputsRequest) (*ValidatePendingAIOutputsResult, error)
}
