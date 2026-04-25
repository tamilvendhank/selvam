package validation

import (
	"context"
	"fmt"
	"time"

	domainaijob "goserver/internal/domain/aijob"
	domaincommon "goserver/internal/domain/common"
	platformrepo "goserver/internal/platform/repository"
	servicecommon "goserver/internal/service/common"
	workerservice "goserver/internal/service/worker"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

const (
	defaultValidationMaxItems = 50
	maxValidationPageSize     = 500
)

type AIOutputValidationConfig struct {
	DefaultMaxItems int
	MaxPageSize     int
	StrictMode      bool
}

type AIOutputValidationOption func(*aiOutputValidationService)

func WithAIOutputValidationConfig(config AIOutputValidationConfig) AIOutputValidationOption {
	return func(service *aiOutputValidationService) {
		if config.DefaultMaxItems > 0 {
			service.config.DefaultMaxItems = config.DefaultMaxItems
		}
		if config.MaxPageSize > 0 {
			service.config.MaxPageSize = config.MaxPageSize
		}
		service.config.StrictMode = config.StrictMode
	}
}

func WithAIOutputValidationClock(clock servicecommon.ClockPort) AIOutputValidationOption {
	return func(service *aiOutputValidationService) {
		if clock != nil {
			service.now = clock.Now
		}
	}
}

type aiOutputValidationService struct {
	batchItems platformrepo.AIBatchItemRepository
	reviews    platformrepo.CompanyReviewRepository
	discovery  workerservice.WorkerWorkDiscoveryService
	config     AIOutputValidationConfig
	now        func() time.Time
}

var _ AIOutputValidationService = (*aiOutputValidationService)(nil)

func NewAIOutputValidationService(
	batchItems platformrepo.AIBatchItemRepository,
	reviews platformrepo.CompanyReviewRepository,
	discovery workerservice.WorkerWorkDiscoveryService,
	options ...AIOutputValidationOption,
) AIOutputValidationService {
	service := &aiOutputValidationService{
		batchItems: batchItems,
		reviews:    reviews,
		discovery:  discovery,
		config: AIOutputValidationConfig{
			DefaultMaxItems: defaultValidationMaxItems,
			MaxPageSize:     maxValidationPageSize,
		},
		now: time.Now,
	}
	for _, option := range options {
		if option != nil {
			option(service)
		}
	}
	if service.config.DefaultMaxItems <= 0 {
		service.config.DefaultMaxItems = defaultValidationMaxItems
	}
	if service.config.MaxPageSize <= 0 {
		service.config.MaxPageSize = maxValidationPageSize
	}
	return service
}

func (service *aiOutputValidationService) ValidateBatchItemOutput(
	ctx context.Context,
	request ValidateBatchItemOutputRequest,
) (*ValidateBatchItemOutputResult, error) {
	if err := request.Validate(); err != nil {
		return nil, err
	}

	outcome, err := service.validateOneItem(ctx, request.BatchItemID, validationRequestOptions{
		WorkflowRunID: request.WorkflowRunID,
		StrictMode:    request.StrictMode || service.config.StrictMode,
		Revalidate:    request.Revalidate,
		InitiatedBy:   request.InitiatedBy,
		CorrelationID: request.CorrelationID,
	})
	if err != nil {
		return nil, err
	}
	if outcome.Skipped {
		return nil, fmt.Errorf("%w: batch item %s is not validatable", servicecommon.ErrNothingToValidate, request.BatchItemID.Hex())
	}
	return buildSingleValidationResult(outcome), nil
}

func (service *aiOutputValidationService) ValidatePendingAIOutputs(
	ctx context.Context,
	request ValidatePendingAIOutputsRequest,
) (*ValidatePendingAIOutputsResult, error) {
	if err := request.Validate(); err != nil {
		return nil, err
	}

	itemIDs, hasMore, err := service.discoverValidatableItemIDs(ctx, request)
	if err != nil {
		return nil, err
	}
	if len(itemIDs) == 0 {
		return &ValidatePendingAIOutputsResult{
			FieldErrors: map[primitive.ObjectID][]servicecommon.FieldError{},
			Summary:     buildValidationSummary("validate_pending_ai_outputs", 0, 0, 0, 0, 0),
		}, nil
	}

	result := ValidatePendingAIOutputsResult{
		FieldErrors: map[primitive.ObjectID][]servicecommon.FieldError{},
	}
	skipped := 0
	for _, itemID := range itemIDs {
		outcome, err := service.validateOneItem(ctx, itemID, validationRequestOptions{
			WorkflowRunID:         request.WorkflowRunID,
			BookType:              request.BookType,
			ItemType:              request.ItemType,
			StrictMode:            request.StrictMode || service.config.StrictMode,
			Revalidate:            request.Revalidate,
			InitiatedBy:           request.InitiatedBy,
			CorrelationID:         request.CorrelationID,
			TreatIneligibleAsSkip: true,
		})
		if err != nil {
			result.PartialFailures = append(result.PartialFailures, validationPartialFailure(itemID, err))
			continue
		}
		if outcome.Skipped {
			result.SkippedItemIDs = append(result.SkippedItemIDs, itemID)
			skipped++
			continue
		}
		mergeValidationOutcome(&result, outcome)
	}

	result.Summary = buildValidationSummary(
		"validate_pending_ai_outputs",
		len(itemIDs),
		len(result.ValidItemIDs),
		len(result.InvalidItemIDs),
		len(result.PartialFailures),
		len(result.ValidationIssues),
	)
	result.Summary.SkippedCount = skipped
	if hasMore {
		result.Summary.Message = fmt.Sprintf("%s; more validatable items may be available", result.Summary.Message)
	}
	return &result, nil
}

func (service *aiOutputValidationService) validateOneItem(
	ctx context.Context,
	itemID primitive.ObjectID,
	options validationRequestOptions,
) (validateOneOutcome, error) {
	item, err := service.loadValidationContext(ctx, itemID)
	if err != nil {
		return validateOneOutcome{}, err
	}
	if err := validateValidationEligibility(item, options); err != nil {
		if options.TreatIneligibleAsSkip && isValidationSkip(err) {
			return validateOneOutcome{BatchItemID: itemID, Skipped: true}, nil
		}
		return validateOneOutcome{}, err
	}

	report := outputValidationReport{}
	payload := map[string]any(nil)
	if item.Status == domaincommon.AIBatchItemStatusInvalidOutput {
		report.Add(issueError("item_invalid_output", "status", "item is already marked invalid_output by reconciliation", item))
	} else {
		payload, err = parseRawResultPayload(item.ResultPayload)
		if err != nil {
			report.Add(issueError("payload_parse_failed", "resultPayload", err.Error(), item))
		} else {
			report.Merge(service.validatePayloadByItemType(ctx, item, payload, options))
		}
	}

	updated, err := service.applyValidationResult(ctx, item, report, options)
	if err != nil {
		return validateOneOutcome{}, fmt.Errorf("validate batch item %s: persist validation result: %w", item.ID.Hex(), err)
	}

	return validateOneOutcome{
		BatchItemID:            item.ID,
		ReviewID:               item.TargetReviewID,
		ItemType:               item.ItemType,
		ValidationStatusBefore: item.ValidationStatus,
		ValidationStatusAfter:  updated.ValidationStatus,
		Valid:                  updated.ValidationStatus == domaincommon.ValidationStatusValid,
		Invalid:                updated.ValidationStatus == domaincommon.ValidationStatusInvalid,
		Issues:                 report.Issues,
		FieldErrors:            report.FieldErrors(),
	}, nil
}

func (service *aiOutputValidationService) validatePayloadByItemType(
	ctx context.Context,
	item *domainaijob.AIBatchItem,
	payload map[string]any,
	options validationRequestOptions,
) outputValidationReport {
	switch item.ItemType {
	case domaincommon.AIBatchItemTypeCompanyReview:
		return service.validateCompanyReviewPayload(ctx, item, payload, options)
	case domaincommon.AIBatchItemTypeThesisUpdate:
		return validateThesisUpdatePayload(item, payload, options)
	case domaincommon.AIBatchItemTypeChangeSummary:
		return validateChangeSummaryPayload(item, payload, options)
	case domaincommon.AIBatchItemTypeEvidenceSummary:
		return validateEvidenceSummaryPayload(item, payload, options)
	case domaincommon.AIBatchItemTypeTradingCandidateReview:
		return validateTradingCandidateReviewPayload(item, payload, options)
	default:
		report := outputValidationReport{}
		report.Add(issueInvalidEnum("itemType", string(item.ItemType), item))
		return report
	}
}
