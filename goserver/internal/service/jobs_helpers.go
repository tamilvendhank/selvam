package service

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"

	"goserver/internal/domain"
	openaiapi "goserver/internal/openai"
	"goserver/internal/shared"
)

var templatePattern = regexp.MustCompile(`\{([^{}]+)\}`)

var activeBatchStatuses = map[string]struct{}{
	"validating":  {},
	"in_progress": {},
	"finalizing":  {},
	"cancelling":  {},
}

var terminalBatchStatuses = map[string]struct{}{
	"completed":         {},
	"failed":            {},
	"expired":           {},
	"cancelled":         {},
	"submission_failed": {},
}

func normalizePromptInputs(rawPrompts []string) []string {
	prompts := make([]string, 0, len(rawPrompts))
	for _, prompt := range rawPrompts {
		trimmed := strings.TrimSpace(prompt)
		if trimmed != "" {
			prompts = append(prompts, trimmed)
		}
	}

	return prompts
}

func normalizeTemplateRecords(rawRecords any) ([]map[string]any, error) {
	items, ok := rawRecords.([]any)
	if !ok {
		if alreadyTyped, ok := rawRecords.([]map[string]any); ok {
			return alreadyTyped, nil
		}

		return nil, fmt.Errorf("JSON array input must be an array")
	}

	records := make([]map[string]any, 0, len(items))
	for index, item := range items {
		record, ok := item.(map[string]any)
		if !ok {
			return nil, fmt.Errorf("JSON array item %d must be an object", index+1)
		}

		records = append(records, record)
	}

	return records, nil
}

func renderPromptTemplate(template string, record map[string]any, index int) (string, error) {
	var renderErr error

	rendered := templatePattern.ReplaceAllStringFunc(template, func(match string) string {
		tokenMatch := templatePattern.FindStringSubmatch(match)
		if len(tokenMatch) < 2 {
			return match
		}

		variableName := strings.TrimSpace(tokenMatch[1])
		value, ok := resolveTemplateValue(record, variableName)
		if !ok {
			renderErr = fmt.Errorf("missing template variable %q for item %d", variableName, index+1)
			return ""
		}

		return fmt.Sprint(value)
	})

	if renderErr != nil {
		return "", renderErr
	}

	return rendered, nil
}

func resolveTemplateValue(record map[string]any, path string) (any, bool) {
	current := any(record)
	for _, key := range strings.Split(path, ".") {
		currentMap, ok := current.(map[string]any)
		if !ok {
			return nil, false
		}

		next, ok := currentMap[key]
		if !ok {
			return nil, false
		}

		current = next
	}

	return current, true
}

func parseJSONL(text []byte) ([]map[string]any, error) {
	if len(text) == 0 {
		return []map[string]any{}, nil
	}

	lines := strings.Split(strings.TrimSpace(string(text)), "\n")
	records := make([]map[string]any, 0, len(lines))

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			continue
		}

		var payload map[string]any
		if err := json.Unmarshal([]byte(trimmed), &payload); err != nil {
			return nil, err
		}

		records = append(records, payload)
	}

	return records, nil
}

func buildMessageInput(query string, attachedFiles []domain.AttachedFile) []map[string]any {
	content := make([]map[string]any, 0, 1+len(attachedFiles))
	if strings.TrimSpace(query) != "" {
		content = append(content, map[string]any{
			"type": "input_text",
			"text": query,
		})
	}

	for _, attachedFile := range attachedFiles {
		content = append(content, map[string]any{
			"type":    "input_file",
			"file_id": attachedFile.OpenAIFileID,
		})
	}

	return []map[string]any{
		{
			"role":    "user",
			"content": content,
		},
	}
}

type responseRequestSpec struct {
	Query              string
	AttachedFiles      []domain.AttachedFile
	Model              string
	ReasoningEffort    *string
	Instructions       string
	PreviousResponseID string
	InputItems         []map[string]any
	Tools              []map[string]any
}

func buildBatchRequestBody(spec responseRequestSpec) map[string]any {
	resolvedModel := shared.NormalizeModelName(spec.Model)
	resolvedReasoning := shared.NormalizeReasoningEffort(resolvedModel, shared.DerefString(spec.ReasoningEffort))
	input := spec.InputItems
	if len(input) == 0 {
		input = buildMessageInput(spec.Query, spec.AttachedFiles)
	}

	requestBody := map[string]any{
		"model":        resolvedModel,
		"instructions": spec.Instructions,
		"input":        input,
		"text": map[string]any{
			"format": map[string]any{
				"type": "text",
			},
		},
	}

	if resolvedReasoning != nil {
		requestBody["reasoning"] = map[string]any{
			"effort": *resolvedReasoning,
		}
	}

	if strings.TrimSpace(spec.PreviousResponseID) != "" {
		requestBody["previous_response_id"] = strings.TrimSpace(spec.PreviousResponseID)
	}

	if len(spec.Tools) > 0 {
		requestBody["tools"] = spec.Tools
	}

	return requestBody
}

func buildBatchSnapshot(batch *openaiapi.Batch) map[string]any {
	return map[string]any{
		"id":            batch.ID,
		"status":        batch.Status,
		"endpoint":      batch.Endpoint,
		"inputFileId":   batch.InputFileID,
		"outputFileId":  emptyStringToNil(batch.OutputFileID),
		"errorFileId":   emptyStringToNil(batch.ErrorFileID),
		"requestCounts": batch.RequestCounts,
		"errors":        shared.NormalizeJSONValue(batch.Errors),
		"createdAt":     unixSecondsPointer(batch.CreatedAt),
		"completedAt":   unixSecondsPointer(batch.CompletedAt),
		"failedAt":      unixSecondsPointer(batch.FailedAt),
		"cancelledAt":   unixSecondsPointer(batch.CancelledAt),
		"expiredAt":     unixSecondsPointer(batch.ExpiredAt),
	}
}

func extractResponseText(responseBody map[string]any) string {
	if responseBody == nil {
		return ""
	}

	if outputText, ok := responseBody["output_text"].(string); ok && strings.TrimSpace(outputText) != "" {
		return outputText
	}

	outputItems, ok := responseBody["output"].([]any)
	if !ok {
		return ""
	}

	textParts := make([]string, 0)
	for _, item := range outputItems {
		outputItem, ok := item.(map[string]any)
		if !ok {
			continue
		}

		contentItems, ok := outputItem["content"].([]any)
		if !ok {
			continue
		}

		for _, content := range contentItems {
			contentItem, ok := content.(map[string]any)
			if !ok {
				continue
			}

			if contentType, _ := contentItem["type"].(string); contentType != "output_text" {
				continue
			}

			if text, ok := contentItem["text"].(string); ok && text != "" {
				textParts = append(textParts, text)
			}
		}
	}

	return strings.Join(textParts, "\n\n")
}

func findLineForCustomID(lines []map[string]any, customID string) map[string]any {
	for _, line := range lines {
		if value, _ := line["custom_id"].(string); value == customID {
			return line
		}
	}

	return nil
}

func buildIterationResultSnapshot(iteration *domain.SubmissionIteration, outputLines, errorLines []map[string]any) (string, map[string]any, map[string]any, map[string]any) {
	outputLine := findLineForCustomID(outputLines, iteration.CustomID)
	errorLine := findLineForCustomID(errorLines, iteration.CustomID)
	responseBody := getResponseBody(outputLine)
	extractedText := extractResponseText(responseBody)

	return extractedText, responseBody, outputLine, errorLine
}

func getResponseBody(outputLine map[string]any) map[string]any {
	if outputLine == nil {
		return nil
	}

	responseValue, ok := outputLine["response"].(map[string]any)
	if !ok {
		return nil
	}

	bodyValue, ok := responseValue["body"].(map[string]any)
	if !ok {
		return nil
	}

	return bodyValue
}

func toDateFromUnixSeconds(value int64) *time.Time {
	if value == 0 {
		return nil
	}

	timestamp := time.Unix(value, 0)
	return &timestamp
}

func unixSecondsPointer(value int64) *int64 {
	if value == 0 {
		return nil
	}

	return &value
}

func emptyStringToNil(value string) *string {
	if strings.TrimSpace(value) == "" {
		return nil
	}

	return shared.StringPtr(value)
}

func previewText(value string) string {
	if len(value) <= 180 {
		return value
	}

	return value[:180]
}

func boolInMap(value string, set map[string]struct{}) bool {
	_, ok := set[value]
	return ok
}

func durationLabel(durationMs *int64) string {
	if durationMs == nil {
		return "N/A"
	}

	seconds := *durationMs / 1000
	if seconds < 0 {
		seconds = 0
	}

	return strconv.FormatInt(seconds, 10) + "s"
}

func extractResponseID(responseBody map[string]any) string {
	if responseBody == nil {
		return ""
	}

	value, _ := responseBody["id"].(string)
	return strings.TrimSpace(value)
}

func extractToolCalls(responseBody map[string]any) []submissionToolCall {
	if responseBody == nil {
		return nil
	}

	outputItems, ok := responseBody["output"].([]any)
	if !ok {
		return nil
	}

	toolCalls := make([]submissionToolCall, 0)
	for _, item := range outputItems {
		outputItem, ok := item.(map[string]any)
		if !ok {
			continue
		}

		itemType, _ := outputItem["type"].(string)
		if itemType != "function_call" && itemType != "custom_tool_call" {
			continue
		}

		toolCalls = append(toolCalls, submissionToolCall{
			ID:        firstNonEmpty(stringValue(outputItem["id"])),
			CallID:    firstNonEmpty(stringValue(outputItem["call_id"])),
			Type:      itemType,
			Name:      firstNonEmpty(stringValue(outputItem["name"])),
			Arguments: firstNonEmpty(stringValue(outputItem["arguments"])),
			Input:     firstNonEmpty(stringValue(outputItem["input"])),
			Status:    firstNonEmpty(stringValue(outputItem["status"])),
			Raw:       shared.MapFromAny(outputItem),
		})
	}

	return toolCalls
}

func normalizeToolCallMaps(toolCalls []submissionToolCall) []map[string]any {
	if len(toolCalls) == 0 {
		return nil
	}

	output := make([]map[string]any, 0, len(toolCalls))
	for _, toolCall := range toolCalls {
		output = append(output, map[string]any{
			"id":        toolCall.ID,
			"callId":    toolCall.CallID,
			"type":      toolCall.Type,
			"name":      toolCall.Name,
			"arguments": toolCall.Arguments,
			"input":     toolCall.Input,
			"status":    toolCall.Status,
			"raw":       shared.NormalizeJSONValue(toolCall.Raw),
		})
	}

	return output
}

func extractToolsFromRequestBody(requestBody map[string]any) []map[string]any {
	if requestBody == nil {
		return nil
	}

	toolsValue, ok := requestBody["tools"].([]any)
	if !ok {
		if typed, ok := requestBody["tools"].([]map[string]any); ok {
			return typed
		}
		return nil
	}

	tools := make([]map[string]any, 0, len(toolsValue))
	for _, item := range toolsValue {
		tool := shared.MapFromAny(item)
		if tool != nil {
			tools = append(tools, tool)
		}
	}

	return tools
}

func buildToolOutputInputItems(outputs []submissionToolOutput) []map[string]any {
	if len(outputs) == 0 {
		return nil
	}

	items := make([]map[string]any, 0, len(outputs))
	for _, output := range outputs {
		items = append(items, map[string]any{
			"type":    output.Type,
			"call_id": output.CallID,
			"output":  output.Output,
		})
	}

	return items
}

func normalizeToolOutputMaps(outputs []submissionToolOutput) []map[string]any {
	if len(outputs) == 0 {
		return nil
	}

	items := make([]map[string]any, 0, len(outputs))
	for _, output := range outputs {
		items = append(items, map[string]any{
			"type":   output.Type,
			"callId": output.CallID,
			"output": output.Output,
		})
	}

	return items
}

func buildToolOutputsText(toolCalls []submissionToolCall, outputs []submissionToolOutput) string {
	if len(outputs) == 0 {
		return ""
	}

	nameByCallID := make(map[string]string, len(toolCalls))
	for _, toolCall := range toolCalls {
		if strings.TrimSpace(toolCall.CallID) == "" {
			continue
		}

		nameByCallID[toolCall.CallID] = firstNonEmpty(strings.TrimSpace(toolCall.Name), "tool")
	}

	sections := make([]string, 0, len(outputs))
	for _, output := range outputs {
		label := firstNonEmpty(nameByCallID[output.CallID], "tool")
		sections = append(sections, fmt.Sprintf("%s (%s)\n%s", label, output.CallID, formatJSONString(output.Output)))
	}

	return strings.Join(sections, "\n\n")
}

func formatJSONString(value string) string {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return ""
	}

	var payload any
	if err := json.Unmarshal([]byte(trimmed), &payload); err != nil {
		return trimmed
	}

	pretty, err := json.MarshalIndent(payload, "", "  ")
	if err != nil {
		return trimmed
	}

	return string(pretty)
}

func stringValue(value any) string {
	text, _ := value.(string)
	return strings.TrimSpace(text)
}
