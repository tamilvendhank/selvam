package ai

import (
	"context"
	"time"

	"goserver/internal/platform/domain"
	"goserver/internal/platform/ports"
)

type NoopAIReviewEngine struct{}

func (NoopAIReviewEngine) SubmitReviewBatch(_ context.Context, request ports.AIReviewBatchRequest) (*ports.AIAsyncTask, error) {
	now := time.Now().UTC()
	return &ports.AIAsyncTask{
		Provider:        "noop",
		TaskKind:        "review_batch",
		LocalObjectType: "noop_task",
		LocalObjectID:   "noop",
		Status:          "unavailable",
		ResultAvailable: false,
		SubmittedAt:     &now,
		Metadata: map[string]any{
			"itemCount": len(request.Items),
		},
	}, nil
}

func (NoopAIReviewEngine) RefreshTask(_ context.Context, task ports.AIAsyncTask) (*ports.AIAsyncTask, error) {
	task.Status = "unavailable"
	task.ResultAvailable = false
	return &task, nil
}

type NoopAIBatchEngine struct{}

func (NoopAIBatchEngine) SubmitBatch(_ context.Context, request ports.SubmitBatchRequest) (*ports.BatchSubmissionResult, error) {
	now := time.Now().UTC()
	items := make([]ports.BatchSubmissionItem, 0, len(request.Items))
	for _, item := range request.Items {
		items = append(items, ports.BatchSubmissionItem{
			CorrelationID: item.CorrelationID,
			Status:        domain.BatchItemStatusPending,
			Metadata: map[string]any{
				"referenceId": item.ReferenceID,
			},
		})
	}

	return &ports.BatchSubmissionResult{
		ProviderName:   "noop",
		LocalJobHandle: "noop",
		Status:         domain.BatchJobStatusCreated,
		SubmittedAt:    &now,
		Metadata: map[string]any{
			"itemCount": len(request.Items),
		},
		Items: items,
	}, nil
}

func (NoopAIBatchEngine) GetBatchStatus(_ context.Context, jobHandle string) (*ports.BatchStatusResult, error) {
	now := time.Now().UTC()
	return &ports.BatchStatusResult{
		ProviderName:      "noop",
		ProviderJobHandle: jobHandle,
		Status:            domain.BatchJobStatusCreated,
		LastPolledAt:      &now,
		ResultAvailable:   false,
		Retryable:         false,
	}, nil
}

func (NoopAIBatchEngine) GetBatchResults(_ context.Context, jobHandle string) (*ports.BatchResultsResult, error) {
	return &ports.BatchResultsResult{
		ProviderName:      "noop",
		ProviderJobHandle: jobHandle,
		Status:            domain.BatchJobStatusCreated,
		RawPayload: map[string]any{
			"message": "noop provider does not emit results",
		},
	}, nil
}
