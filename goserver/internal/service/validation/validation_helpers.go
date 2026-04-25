package validation

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	domainaijob "goserver/internal/domain/aijob"
	domaincommon "goserver/internal/domain/common"
	platformrepo "goserver/internal/platform/repository"
	servicecommon "goserver/internal/service/common"
	workerservice "goserver/internal/service/worker"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

var errValidationSkipped = errors.New("validation skipped")

type validationRequestOptions struct {
	WorkflowRunID         primitive.ObjectID
	BookType              domaincommon.BookType
	ItemType              domaincommon.AIBatchItemType
	StrictMode            bool
	Revalidate            bool
	InitiatedBy           string
	CorrelationID         string
	TreatIneligibleAsSkip bool
}

type validateOneOutcome struct {
	BatchItemID            primitive.ObjectID
	ReviewID               primitive.ObjectID
	ItemType               domaincommon.AIBatchItemType
	ValidationStatusBefore domaincommon.ValidationStatus
	ValidationStatusAfter  domaincommon.ValidationStatus
	Valid                  bool
	Invalid                bool
	Issues                 []servicecommon.ValidationIssue
	FieldErrors            []servicecommon.FieldError
	Skipped                bool
}

func (service *aiOutputValidationService) maxValidationItems(requested int) int {
	if requested > 0 && requested < service.config.MaxPageSize {
		return requested
	}
	if requested > service.config.MaxPageSize {
		return service.config.MaxPageSize
	}
	if service.config.DefaultMaxItems > service.config.MaxPageSize {
		return service.config.MaxPageSize
	}
	return service.config.DefaultMaxItems
}

func (service *aiOutputValidationService) discoverValidatableItemIDs(
	ctx context.Context,
	request ValidatePendingAIOutputsRequest,
) ([]primitive.ObjectID, bool, error) {
	if !request.BatchItemID.IsZero() {
		return []primitive.ObjectID{request.BatchItemID}, false, nil
	}

	limit := service.maxValidationItems(request.MaxItems)
	if service.discovery != nil {
		discovered, err := service.discovery.DiscoverValidatableItems(ctx, workerservice.DiscoverValidatableItemsRequest{
			DiscoveryRequestBase: workerservice.DiscoveryRequestBase{
				WorkflowRunID: request.WorkflowRunID,
				BookType:      request.BookType,
				ItemType:      request.ItemType,
				MaxItems:      limit,
			},
			StrictMode: request.StrictMode || service.config.StrictMode,
			Revalidate: request.Revalidate,
		})
		if err != nil {
			return nil, false, fmt.Errorf("discover validatable items: %w", err)
		}
		return itemIDsFromRefs(discovered.BatchItems, limit), discovered.HasMore, nil
	}

	if service.batchItems == nil {
		return nil, false, fmt.Errorf("discover validatable items: batch item repository is required")
	}
	filter := platformrepo.AIBatchItemFilter{}
	if !request.WorkflowRunID.IsZero() {
		filter.WorkflowRunIDs = []primitive.ObjectID{request.WorkflowRunID}
	}
	if request.BookType != "" {
		filter.BookTypes = []domaincommon.BookType{request.BookType}
	}
	if request.ItemType != "" {
		filter.ItemTypes = []domaincommon.AIBatchItemType{request.ItemType}
	}
	if request.Revalidate {
		filter.Statuses = []domaincommon.AIBatchItemStatus{
			domaincommon.AIBatchItemStatusCompleted,
			domaincommon.AIBatchItemStatusInvalidOutput,
		}
		filter.ValidationStatuses = []domaincommon.ValidationStatus{
			domaincommon.ValidationStatusNotValidated,
			domaincommon.ValidationStatusValid,
			domaincommon.ValidationStatusInvalid,
		}
	} else {
		filter.PendingValidationOnly = true
	}

	result, err := service.batchItems.List(ctx, filter, platformrepo.AIBatchItemListOptions{
		Pagination: platformrepo.PageOptions{PageSize: limit},
		Sort:       platformrepo.AIBatchItemSortOption{By: platformrepo.AIBatchItemSortByCompletedAt, Order: platformrepo.SortOrderAscending},
	})
	if err != nil {
		return nil, false, fmt.Errorf("list validatable items: %w", err)
	}
	ids := make([]primitive.ObjectID, 0, len(result.Items))
	for _, item := range result.Items {
		if item != nil {
			ids = append(ids, item.ID)
		}
	}
	return ids, result.Page.HasMore, nil
}

func (service *aiOutputValidationService) loadValidationContext(
	ctx context.Context,
	itemID primitive.ObjectID,
) (*domainaijob.AIBatchItem, error) {
	if service.batchItems == nil {
		return nil, fmt.Errorf("validate batch item %s: batch item repository is required", itemID.Hex())
	}
	item, err := service.batchItems.GetByID(ctx, itemID)
	if err != nil {
		return nil, fmt.Errorf("validate batch item %s: load item: %w", itemID.Hex(), err)
	}
	if item == nil {
		return nil, fmt.Errorf("validate batch item %s: %w", itemID.Hex(), platformrepo.ErrNotFound)
	}
	return item, nil
}

func validateValidationEligibility(item *domainaijob.AIBatchItem, options validationRequestOptions) error {
	if item == nil {
		return fmt.Errorf("batch item is required")
	}
	if !options.WorkflowRunID.IsZero() && item.WorkflowRunID != options.WorkflowRunID {
		return fmt.Errorf("%w: workflowRunId filter does not match", errValidationSkipped)
	}
	if options.BookType != "" && item.BookType != options.BookType {
		return fmt.Errorf("%w: bookType filter does not match", errValidationSkipped)
	}
	if options.ItemType != "" && item.ItemType != options.ItemType {
		return fmt.Errorf("%w: itemType filter does not match", errValidationSkipped)
	}
	if item.Status != domaincommon.AIBatchItemStatusCompleted && item.Status != domaincommon.AIBatchItemStatusInvalidOutput {
		return fmt.Errorf("%w: status %q is not output-ready", errValidationSkipped, item.Status)
	}
	if item.ValidationStatus != domaincommon.ValidationStatusNotValidated && !options.Revalidate {
		return fmt.Errorf("%w: validationStatus %q already evaluated", errValidationSkipped, item.ValidationStatus)
	}
	if item.Status == domaincommon.AIBatchItemStatusCompleted && len(item.ResultPayload) == 0 {
		return fmt.Errorf("completed batch item has no result payload")
	}
	return nil
}

func parseRawResultPayload(payload map[string]any) (map[string]any, error) {
	if len(payload) == 0 {
		return nil, fmt.Errorf("result payload is required")
	}
	if looksLikeStructuredOutput(payload) {
		return cloneMap(payload), nil
	}
	for _, key := range []string{"parsed", "review", "result", "data", "output", "content", "message"} {
		value, ok := payload[key]
		if !ok {
			continue
		}
		parsed, err := parsePayloadValue(value)
		if err == nil && len(parsed) > 0 {
			return parsed, nil
		}
	}
	return cloneMap(payload), nil
}

func parsePayloadValue(value any) (map[string]any, error) {
	switch typed := value.(type) {
	case nil:
		return nil, fmt.Errorf("payload value is empty")
	case map[string]any:
		return cloneMap(typed), nil
	case string:
		text := strings.TrimSpace(typed)
		if text == "" {
			return nil, fmt.Errorf("payload value is blank")
		}
		var decoded map[string]any
		if err := json.Unmarshal([]byte(text), &decoded); err != nil {
			return nil, err
		}
		return decoded, nil
	default:
		marshaled, err := json.Marshal(typed)
		if err != nil {
			return nil, err
		}
		var decoded map[string]any
		if err := json.Unmarshal(marshaled, &decoded); err != nil {
			return nil, err
		}
		return decoded, nil
	}
}

func looksLikeStructuredOutput(payload map[string]any) bool {
	for _, key := range []string{
		"sections",
		"weighted_total_score",
		"weightedTotalScore",
		"final_action_after_review",
		"finalActionAfterReview",
		"thesis_update",
		"change_summary",
		"evidence_summary",
		"trading_candidate_review",
	} {
		if _, ok := payload[key]; ok {
			return true
		}
	}
	return false
}

func isValidationSkip(err error) bool {
	return errors.Is(err, errValidationSkipped)
}

func itemIDsFromRefs(refs []servicecommon.BatchItemRef, limit int) []primitive.ObjectID {
	ids := make([]primitive.ObjectID, 0, len(refs))
	for _, ref := range refs {
		if ref.ID.IsZero() {
			continue
		}
		ids = append(ids, ref.ID)
		if len(ids) >= limit {
			break
		}
	}
	return ids
}

func cloneMap(source map[string]any) map[string]any {
	if len(source) == 0 {
		return nil
	}
	clone := make(map[string]any, len(source))
	for key, value := range source {
		clone[key] = value
	}
	return clone
}

func uniqueObjectIDs(ids []primitive.ObjectID) []primitive.ObjectID {
	seen := make(map[primitive.ObjectID]struct{}, len(ids))
	unique := make([]primitive.ObjectID, 0, len(ids))
	for _, id := range ids {
		if id.IsZero() {
			continue
		}
		if _, exists := seen[id]; exists {
			continue
		}
		seen[id] = struct{}{}
		unique = append(unique, id)
	}
	return unique
}
