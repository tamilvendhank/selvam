package service

import (
	"context"
	"encoding/json"
	"fmt"

	"goserver/internal/domain"
)

type submissionToolCall struct {
	ID        string
	CallID    string
	Type      string
	Name      string
	Arguments string
	Input     string
	Status    string
	Raw       map[string]any
}

type submissionToolOutput struct {
	Type   string
	CallID string
	Output string
}

type SubmissionToolExecutor interface {
	Execute(ctx context.Context, job *domain.Job, toolCalls []submissionToolCall) ([]submissionToolOutput, error)
}

type UnconfiguredToolExecutor struct{}

func (executor *UnconfiguredToolExecutor) Execute(_ context.Context, _ *domain.Job, toolCalls []submissionToolCall) ([]submissionToolOutput, error) {
	outputs := make([]submissionToolOutput, 0, len(toolCalls))
	for _, toolCall := range toolCalls {
		payload, err := json.Marshal(map[string]any{
			"error": fmt.Sprintf("tool %q is not configured", toolCall.Name),
		})
		if err != nil {
			return nil, err
		}

		outputType := "function_call_output"
		if toolCall.Type == "custom_tool_call" {
			outputType = "custom_tool_call_output"
		}

		outputs = append(outputs, submissionToolOutput{
			Type:   outputType,
			CallID: toolCall.CallID,
			Output: string(payload),
		})
	}

	return outputs, nil
}
