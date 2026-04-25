package aijob

import (
	"fmt"
	"strings"
	"time"

	domainaijob "goserver/internal/domain/aijob"
	domaincommon "goserver/internal/domain/common"
	platformdomain "goserver/internal/platform/domain"
	platformports "goserver/internal/platform/ports"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

type providerReconciliationResults struct {
	ProviderName      string
	ProviderJobHandle string
	Status            domaincommon.AIBatchJobStatus
	CompletedAt       time.Time
	RawPayload        map[string]any
	Items             []providerReconciliationItem
}

type providerReconciliationItem struct {
	CorrelationID      string
	CorrelationKeys    []string
	Status             domaincommon.AIBatchItemStatus
	OutputPayload      map[string]any
	ErrorSummary       string
	Retryable          bool
	ProviderMetadata   map[string]any
	ProviderItemHandle string
	ResultIndex        int
}

type correlatedProviderResults struct {
	Matched          []matchedProviderResult
	UnmatchedResults []providerReconciliationItem
	DuplicateResults []providerReconciliationItem
	MissingItems     []*domainaijob.AIBatchItem
}

type matchedProviderResult struct {
	Item   *domainaijob.AIBatchItem
	Result providerReconciliationItem
}

func mapProviderResults(
	job *domainaijob.AIBatchJob,
	results *platformports.BatchResultsResult,
	now time.Time,
) (providerReconciliationResults, error) {
	if job == nil {
		return providerReconciliationResults{}, fmt.Errorf("batch job is required")
	}
	if results == nil {
		return providerReconciliationResults{}, fmt.Errorf("provider returned nil batch results")
	}
	if !platformdomain.IsValidBatchJobStatus(results.Status) {
		return providerReconciliationResults{}, fmt.Errorf("provider returned unknown batch status %q", results.Status)
	}

	completedAt := now.UTC()
	if results.CompletedAt != nil && !results.CompletedAt.IsZero() {
		completedAt = results.CompletedAt.UTC()
	}

	mapped := providerReconciliationResults{
		ProviderName:      strings.TrimSpace(results.ProviderName),
		ProviderJobHandle: strings.TrimSpace(results.ProviderJobHandle),
		Status:            mapPlatformBatchJobStatus(results.Status),
		CompletedAt:       completedAt,
		RawPayload:        cloneStringAnyMap(results.RawPayload),
		Items:             make([]providerReconciliationItem, 0, len(results.Items)),
	}
	for index, item := range results.Items {
		mapped.Items = append(mapped.Items, mapProviderResultItem(item, index))
	}
	return mapped, nil
}

func mapProviderResultItem(item platformports.BatchResultItem, index int) providerReconciliationItem {
	status := mapPlatformBatchItemStatus(item.Status)
	errorSummary := strings.TrimSpace(item.ErrorSummary)
	if !platformdomain.IsValidBatchItemStatus(item.Status) {
		status = domaincommon.AIBatchItemStatusInvalidOutput
		errorSummary = fmt.Sprintf("provider returned unknown batch item status %q", item.Status)
	}

	return providerReconciliationItem{
		CorrelationID:      strings.TrimSpace(item.CorrelationID),
		CorrelationKeys:    providerResultCorrelationKeys(item),
		Status:             status,
		OutputPayload:      cloneStringAnyMap(item.OutputPayload),
		ErrorSummary:       errorSummary,
		Retryable:          item.Retryable,
		ProviderMetadata:   cloneStringAnyMap(item.ProviderMetadata),
		ProviderItemHandle: providerResultHandle(item),
		ResultIndex:        index,
	}
}

func mapPlatformBatchItemStatus(status platformdomain.BatchItemStatus) domaincommon.AIBatchItemStatus {
	switch status {
	case platformdomain.BatchItemStatusPending:
		return domaincommon.AIBatchItemStatusPending
	case platformdomain.BatchItemStatusSubmitted:
		return domaincommon.AIBatchItemStatusSubmitted
	case platformdomain.BatchItemStatusProcessing:
		return domaincommon.AIBatchItemStatusProcessing
	case platformdomain.BatchItemStatusCompleted:
		return domaincommon.AIBatchItemStatusCompleted
	case platformdomain.BatchItemStatusFailed:
		return domaincommon.AIBatchItemStatusFailed
	case platformdomain.BatchItemStatusInvalidOutput:
		return domaincommon.AIBatchItemStatusInvalidOutput
	case platformdomain.BatchItemStatusSkipped:
		return domaincommon.AIBatchItemStatusSkipped
	default:
		return domaincommon.AIBatchItemStatusInvalidOutput
	}
}

func correlateProviderResults(
	items []*domainaijob.AIBatchItem,
	results []providerReconciliationItem,
) correlatedProviderResults {
	index, duplicateKeys := buildBatchItemIndex(items)
	correlation := correlatedProviderResults{
		Matched:          make([]matchedProviderResult, 0, len(results)),
		UnmatchedResults: make([]providerReconciliationItem, 0),
		DuplicateResults: make([]providerReconciliationItem, 0),
		MissingItems:     make([]*domainaijob.AIBatchItem, 0),
	}

	matchedItemIDs := make(map[primitive.ObjectID]struct{}, len(results))
	for _, result := range results {
		item, ambiguous := findMatchingBatchItem(result, index, duplicateKeys)
		if item == nil {
			correlation.UnmatchedResults = append(correlation.UnmatchedResults, result)
			continue
		}
		if _, exists := matchedItemIDs[item.ID]; exists || ambiguous {
			correlation.DuplicateResults = append(correlation.DuplicateResults, result)
			continue
		}
		matchedItemIDs[item.ID] = struct{}{}
		correlation.Matched = append(correlation.Matched, matchedProviderResult{Item: item, Result: result})
	}

	for _, item := range items {
		if item == nil {
			continue
		}
		if _, matched := matchedItemIDs[item.ID]; matched {
			continue
		}
		if isItemAlreadyReconciled(item) {
			continue
		}
		correlation.MissingItems = append(correlation.MissingItems, item)
	}
	return correlation
}

func buildBatchItemIndex(items []*domainaijob.AIBatchItem) (map[string]*domainaijob.AIBatchItem, map[string]struct{}) {
	index := make(map[string]*domainaijob.AIBatchItem)
	duplicateKeys := make(map[string]struct{})
	for _, item := range items {
		if item == nil {
			continue
		}
		for _, key := range batchItemCorrelationKeys(item) {
			if _, duplicate := duplicateKeys[key]; duplicate {
				continue
			}
			existing, exists := index[key]
			if exists && existing.ID != item.ID {
				delete(index, key)
				duplicateKeys[key] = struct{}{}
				continue
			}
			index[key] = item
		}
	}
	return index, duplicateKeys
}

func findMatchingBatchItem(
	result providerReconciliationItem,
	index map[string]*domainaijob.AIBatchItem,
	duplicateKeys map[string]struct{},
) (*domainaijob.AIBatchItem, bool) {
	var matched *domainaijob.AIBatchItem
	ambiguous := false
	for _, key := range result.CorrelationKeys {
		if _, duplicate := duplicateKeys[key]; duplicate {
			ambiguous = true
			continue
		}
		item, ok := index[key]
		if !ok {
			continue
		}
		if matched != nil && matched.ID != item.ID {
			ambiguous = true
			continue
		}
		matched = item
	}
	return matched, ambiguous
}

func batchItemCorrelationKeys(item *domainaijob.AIBatchItem) []string {
	keys := make([]string, 0, 16)
	keys = appendCorrelationValue(keys, item.ID)
	keys = appendCorrelationValue(keys, item.TargetReviewID)
	keys = appendCorrelationValue(keys, item.TargetThesisID)
	keys = appendCorrelationValue(keys, item.CompanyID)
	keys = appendCorrelationValue(keys, item.Symbol)
	keys = appendCorrelationValue(keys, item.InputHash)
	for _, field := range correlationMetadataFields() {
		keys = appendCorrelationValue(keys, item.InputPayload[field])
	}
	return uniqueCorrelationKeys(keys)
}

func providerResultCorrelationKeys(item platformports.BatchResultItem) []string {
	keys := make([]string, 0, 16)
	keys = appendCorrelationValue(keys, item.CorrelationID)
	for _, field := range correlationMetadataFields() {
		keys = appendCorrelationValue(keys, item.ProviderMetadata[field])
	}
	for _, field := range correlationPayloadFields() {
		keys = appendCorrelationValue(keys, item.OutputPayload[field])
	}
	return uniqueCorrelationKeys(keys)
}

func correlationMetadataFields() []string {
	return []string{
		"correlationId",
		"correlationID",
		"referenceId",
		"referenceID",
		"batchItemId",
		"batchItemID",
		"aiBatchItemId",
		"aiBatchItemID",
		"itemId",
		"itemID",
		"id",
		"targetReviewId",
		"targetReviewID",
		"reviewId",
		"reviewID",
		"targetThesisId",
		"targetThesisID",
		"thesisId",
		"thesisID",
		"companyId",
		"companyID",
		"symbol",
		"customId",
		"customID",
	}
}

func correlationPayloadFields() []string {
	return []string{
		"correlationId",
		"correlationID",
		"referenceId",
		"referenceID",
		"batchItemId",
		"batchItemID",
		"aiBatchItemId",
		"aiBatchItemID",
		"targetReviewId",
		"targetReviewID",
		"reviewId",
		"reviewID",
		"targetThesisId",
		"targetThesisID",
		"thesisId",
		"thesisID",
		"companyId",
		"companyID",
		"symbol",
		"customId",
		"customID",
	}
}

func providerResultHandle(item platformports.BatchResultItem) string {
	for _, key := range []string{"providerItemHandle", "provider_item_handle", "providerHandle", "providerJobHandle", "jobId", "jobID", "id"} {
		if value, ok := item.ProviderMetadata[key]; ok {
			if text := normalizeProviderText(value); text != "" {
				return text
			}
		}
	}
	return strings.TrimSpace(item.CorrelationID)
}

func appendCorrelationValue(keys []string, value any) []string {
	text := normalizeProviderText(value)
	if text == "" {
		return keys
	}
	return append(keys, strings.ToLower(text))
}

func normalizeProviderText(value any) string {
	switch typed := value.(type) {
	case nil:
		return ""
	case string:
		return strings.TrimSpace(typed)
	case primitive.ObjectID:
		if typed.IsZero() {
			return ""
		}
		return typed.Hex()
	case fmt.Stringer:
		return strings.TrimSpace(typed.String())
	default:
		return strings.TrimSpace(fmt.Sprint(typed))
	}
}

func uniqueCorrelationKeys(keys []string) []string {
	seen := make(map[string]struct{}, len(keys))
	unique := make([]string, 0, len(keys))
	for _, key := range keys {
		key = strings.TrimSpace(strings.ToLower(key))
		if key == "" {
			continue
		}
		if _, exists := seen[key]; exists {
			continue
		}
		seen[key] = struct{}{}
		unique = append(unique, key)
	}
	return unique
}

func cloneStringAnyMap(source map[string]any) map[string]any {
	if len(source) == 0 {
		return nil
	}
	clone := make(map[string]any, len(source))
	for key, value := range source {
		clone[key] = value
	}
	return clone
}
