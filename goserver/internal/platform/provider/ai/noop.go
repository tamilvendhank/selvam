package ai

import (
	"context"
	"time"

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
