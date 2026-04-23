package mongo

import (
	"context"
	"fmt"

	aijobpkg "goserver/internal/domain/aijob"
	"goserver/internal/domain/common"
	platformrepo "goserver/internal/platform/repository"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

type AIBatchItemMongoRepository struct {
	collection *mongo.Collection
}

var _ platformrepo.AIBatchItemRepository = (*AIBatchItemMongoRepository)(nil)

func NewAIBatchItemRepository(collection *mongo.Collection) *AIBatchItemMongoRepository {
	return &AIBatchItemMongoRepository{collection: collection}
}

func (repository *AIBatchItemMongoRepository) Create(ctx context.Context, item *aijobpkg.AIBatchItem) (*aijobpkg.AIBatchItem, error) {
	if item == nil {
		return nil, fmt.Errorf("create ai batch item: item is required")
	}

	document := *item
	if document.ID.IsZero() {
		document.ID = newDocumentID()
	}
	document.Symbol = normalizeSymbol(document.Symbol)

	if err := document.Validate(); err != nil {
		return nil, fmt.Errorf("create ai batch item: validate item: %w", err)
	}

	if _, err := repository.collection.InsertOne(ctx, &document); err != nil {
		return nil, fmt.Errorf("create ai batch item: %w", mapMongoError(err))
	}
	return &document, nil
}

func (repository *AIBatchItemMongoRepository) CreateMany(ctx context.Context, items []*aijobpkg.AIBatchItem) ([]*aijobpkg.AIBatchItem, error) {
	if len(items) == 0 {
		return []*aijobpkg.AIBatchItem{}, nil
	}

	documents := make([]any, 0, len(items))
	result := make([]*aijobpkg.AIBatchItem, 0, len(items))
	for _, item := range items {
		if item == nil {
			return nil, fmt.Errorf("create ai batch items: item is required")
		}
		document := *item
		if document.ID.IsZero() {
			document.ID = newDocumentID()
		}
		document.Symbol = normalizeSymbol(document.Symbol)
		if err := document.Validate(); err != nil {
			return nil, fmt.Errorf("create ai batch items: validate item %s: %w", document.ID.Hex(), err)
		}
		documents = append(documents, document)
		result = append(result, &document)
	}

	if _, err := repository.collection.InsertMany(ctx, documents); err != nil {
		return nil, fmt.Errorf("create ai batch items: %w", mapMongoError(err))
	}
	return result, nil
}

func (repository *AIBatchItemMongoRepository) GetByID(ctx context.Context, id primitive.ObjectID) (*aijobpkg.AIBatchItem, error) {
	document, err := findOne[aijobpkg.AIBatchItem](ctx, repository.collection, bson.M{"_id": id})
	if err != nil {
		return nil, fmt.Errorf("get ai batch item by id %s: %w", id.Hex(), mapMongoError(err))
	}
	return document, nil
}

func (repository *AIBatchItemMongoRepository) ListByBatchJobID(
	ctx context.Context,
	batchJobID primitive.ObjectID,
	options platformrepo.AIBatchItemListOptions,
) (*platformrepo.ListResult[*aijobpkg.AIBatchItem], error) {
	return repository.List(ctx, platformrepo.AIBatchItemFilter{AIBatchJobIDs: []primitive.ObjectID{batchJobID}}, options)
}

func (repository *AIBatchItemMongoRepository) List(
	ctx context.Context,
	filter platformrepo.AIBatchItemFilter,
	options platformrepo.AIBatchItemListOptions,
) (*platformrepo.ListResult[*aijobpkg.AIBatchItem], error) {
	result, err := findPage[aijobpkg.AIBatchItem, *aijobpkg.AIBatchItem](
		ctx,
		repository.collection,
		buildAIBatchItemFilter(filter),
		options.Pagination,
		buildAIBatchItemSort(options.Sort),
		nil,
		func(document *aijobpkg.AIBatchItem) *aijobpkg.AIBatchItem {
			item := *document
			return &item
		},
	)
	if err != nil {
		return nil, fmt.Errorf("list ai batch items: %w", mapMongoError(err))
	}
	return result, nil
}

func (repository *AIBatchItemMongoRepository) FindPendingValidation(
	ctx context.Context,
	filter platformrepo.AIBatchItemFilter,
	options platformrepo.AIBatchItemListOptions,
) (*platformrepo.ListResult[*aijobpkg.AIBatchItem], error) {
	filter.PendingValidationOnly = true
	return repository.List(ctx, filter, options)
}

func (repository *AIBatchItemMongoRepository) FindRetryableItems(
	ctx context.Context,
	filter platformrepo.AIBatchItemFilter,
	options platformrepo.AIBatchItemListOptions,
) (*platformrepo.ListResult[*aijobpkg.AIBatchItem], error) {
	filter.RetryableOnly = true
	return repository.List(ctx, filter, options)
}

func (repository *AIBatchItemMongoRepository) UpdateStatus(
	ctx context.Context,
	itemID primitive.ObjectID,
	patch platformrepo.AIBatchItemStatusPatch,
) (*aijobpkg.AIBatchItem, error) {
	current, err := repository.GetByID(ctx, itemID)
	if err != nil {
		return nil, err
	}
	if err := ensureAIBatchItemExpectedStatuses(current, patch.ExpectedCurrentStatuses, "update ai batch item status"); err != nil {
		return nil, err
	}
	if !current.CanTransitionTo(patch.NextStatus) {
		return nil, invalidTransition("update ai batch item status %s cannot transition from %q to %q", itemID.Hex(), current.Status, patch.NextStatus)
	}

	candidate := *current
	if err := candidate.TransitionTo(patch.NextStatus, mutationTimestamp(patch.Mutation)); err != nil {
		return nil, fmt.Errorf("update ai batch item status %s: %w", itemID.Hex(), err)
	}
	if patch.ErrorSummary != nil {
		candidate.ErrorSummary = *patch.ErrorSummary
	}

	if err := candidate.Validate(); err != nil {
		return nil, fmt.Errorf("update ai batch item status %s: validate item: %w", itemID.Hex(), err)
	}

	result, err := repository.collection.UpdateOne(
		ctx,
		bson.M{"_id": itemID, "status": current.Status},
		bson.M{"$set": buildAIBatchItemStatusUpdate(&candidate, patch)},
	)
	if err != nil {
		return nil, fmt.Errorf("update ai batch item status %s: %w", itemID.Hex(), mapMongoError(err))
	}
	if result.MatchedCount == 0 {
		return nil, preconditionFailed("update ai batch item status %s stale write rejected", itemID.Hex())
	}
	return &candidate, nil
}

func (repository *AIBatchItemMongoRepository) SaveResultPayload(
	ctx context.Context,
	itemID primitive.ObjectID,
	patch platformrepo.AIBatchItemResultPatch,
) (*aijobpkg.AIBatchItem, error) {
	current, err := repository.GetByID(ctx, itemID)
	if err != nil {
		return nil, err
	}
	if current.Status == common.AIBatchItemStatusFailed || current.Status == common.AIBatchItemStatusInvalidOutput || current.Status == common.AIBatchItemStatusSkipped {
		return nil, immutableState("save ai batch item result payload %s rejected for terminal item", itemID.Hex())
	}
	if err := ensureAIBatchItemExpectedStatuses(current, patch.ExpectedCurrentStatuses, "save ai batch item result payload"); err != nil {
		return nil, err
	}

	candidate := *current
	candidate.ResultPayload = cloneMap(patch.ResultPayload)
	candidate.UpdatedAt = mutationTimestamp(patch.Mutation)
	if patch.NextStatus != nil {
		if *patch.NextStatus == common.AIBatchItemStatusCompleted || *patch.NextStatus == common.AIBatchItemStatusFailed || *patch.NextStatus == common.AIBatchItemStatusInvalidOutput || *patch.NextStatus == common.AIBatchItemStatusSkipped {
			return nil, invalidTransition("save ai batch item result payload %s requires terminal transitions through dedicated methods", itemID.Hex())
		}
		if !current.CanTransitionTo(*patch.NextStatus) {
			return nil, invalidTransition("save ai batch item result payload %s cannot transition from %q to %q", itemID.Hex(), current.Status, *patch.NextStatus)
		}
		candidate.Status = *patch.NextStatus
	}
	if patch.ErrorSummary != nil {
		candidate.ErrorSummary = *patch.ErrorSummary
	}

	if err := candidate.Validate(); err != nil {
		return nil, fmt.Errorf("save ai batch item result payload %s: validate item: %w", itemID.Hex(), err)
	}

	set := bson.M{
		"resultPayload": candidate.ResultPayload,
		"updatedAt":     candidate.UpdatedAt,
	}
	if patch.NextStatus != nil {
		set["status"] = candidate.Status
	}
	if patch.ErrorSummary != nil {
		set["errorSummary"] = candidate.ErrorSummary
	}
	result, err := repository.collection.UpdateOne(ctx, bson.M{"_id": itemID, "status": current.Status}, bson.M{"$set": set})
	if err != nil {
		return nil, fmt.Errorf("save ai batch item result payload %s: %w", itemID.Hex(), mapMongoError(err))
	}
	if result.MatchedCount == 0 {
		return nil, preconditionFailed("save ai batch item result payload %s stale write rejected", itemID.Hex())
	}
	return &candidate, nil
}

func (repository *AIBatchItemMongoRepository) SaveValidationResult(
	ctx context.Context,
	itemID primitive.ObjectID,
	patch platformrepo.AIBatchItemValidationPatch,
) (*aijobpkg.AIBatchItem, error) {
	current, err := repository.GetByID(ctx, itemID)
	if err != nil {
		return nil, err
	}
	if current.Status != common.AIBatchItemStatusCompleted && current.Status != common.AIBatchItemStatusInvalidOutput {
		return nil, invalidTransition("save ai batch item validation %s requires completed or invalid_output status", itemID.Hex())
	}
	if err := ensureAIBatchItemExpectedStatuses(current, patch.ExpectedCurrentStatuses, "save ai batch item validation"); err != nil {
		return nil, err
	}

	candidate := *current
	candidate.ValidationStatus = patch.ValidationStatus
	candidate.ValidationErrors = append([]string(nil), patch.ValidationErrors...)
	if patch.TargetReviewID != nil {
		candidate.TargetReviewID = *patch.TargetReviewID
	}
	if patch.TargetThesisID != nil {
		candidate.TargetThesisID = *patch.TargetThesisID
	}
	if patch.TargetEntityVersion != nil {
		candidate.TargetEntityVersion = *patch.TargetEntityVersion
	}
	candidate.UpdatedAt = mutationTimestamp(patch.Mutation)

	if err := candidate.Validate(); err != nil {
		return nil, fmt.Errorf("save ai batch item validation %s: validate item: %w", itemID.Hex(), err)
	}

	result, err := repository.collection.UpdateOne(
		ctx,
		bson.M{"_id": itemID, "status": current.Status},
		bson.M{"$set": bson.M{
			"validationStatus":    candidate.ValidationStatus,
			"validationErrors":    candidate.ValidationErrors,
			"targetReviewId":      candidate.TargetReviewID,
			"targetThesisId":      candidate.TargetThesisID,
			"targetEntityVersion": candidate.TargetEntityVersion,
			"updatedAt":           candidate.UpdatedAt,
		}},
	)
	if err != nil {
		return nil, fmt.Errorf("save ai batch item validation %s: %w", itemID.Hex(), mapMongoError(err))
	}
	if result.MatchedCount == 0 {
		return nil, preconditionFailed("save ai batch item validation %s stale write rejected", itemID.Hex())
	}
	return &candidate, nil
}

func (repository *AIBatchItemMongoRepository) MarkCompleted(
	ctx context.Context,
	itemID primitive.ObjectID,
	patch platformrepo.AIBatchItemCompletionPatch,
) (*aijobpkg.AIBatchItem, error) {
	current, err := repository.GetByID(ctx, itemID)
	if err != nil {
		return nil, err
	}
	if err := ensureAIBatchItemExpectedStatuses(current, patch.ExpectedCurrentStatuses, "mark ai batch item completed"); err != nil {
		return nil, err
	}
	if !current.CanTransitionTo(common.AIBatchItemStatusCompleted) {
		return nil, invalidTransition("mark ai batch item completed %s cannot transition from %q", itemID.Hex(), current.Status)
	}

	candidate := *current
	if err := candidate.TransitionTo(common.AIBatchItemStatusCompleted, patch.CompletedAt.UTC()); err != nil {
		return nil, fmt.Errorf("mark ai batch item completed %s: %w", itemID.Hex(), err)
	}
	candidate.ErrorSummary = ""

	if err := candidate.Validate(); err != nil {
		return nil, fmt.Errorf("mark ai batch item completed %s: validate item: %w", itemID.Hex(), err)
	}

	result, err := repository.collection.UpdateOne(
		ctx,
		bson.M{"_id": itemID, "status": current.Status},
		bson.M{"$set": bson.M{
			"status":       candidate.Status,
			"updatedAt":    candidate.UpdatedAt,
			"completedAt":  candidate.CompletedAt.UTC(),
			"errorSummary": candidate.ErrorSummary,
		}},
	)
	if err != nil {
		return nil, fmt.Errorf("mark ai batch item completed %s: %w", itemID.Hex(), mapMongoError(err))
	}
	if result.MatchedCount == 0 {
		return nil, preconditionFailed("mark ai batch item completed %s stale write rejected", itemID.Hex())
	}
	return &candidate, nil
}

func (repository *AIBatchItemMongoRepository) MarkFailed(
	ctx context.Context,
	itemID primitive.ObjectID,
	patch platformrepo.AIBatchItemFailurePatch,
) (*aijobpkg.AIBatchItem, error) {
	current, err := repository.GetByID(ctx, itemID)
	if err != nil {
		return nil, err
	}
	if err := ensureAIBatchItemExpectedStatuses(current, patch.ExpectedCurrentStatuses, "mark ai batch item failed"); err != nil {
		return nil, err
	}
	if !current.CanTransitionTo(common.AIBatchItemStatusFailed) {
		return nil, invalidTransition("mark ai batch item failed %s cannot transition from %q", itemID.Hex(), current.Status)
	}

	candidate := *current
	if err := candidate.TransitionTo(common.AIBatchItemStatusFailed, patch.FailedAt.UTC()); err != nil {
		return nil, fmt.Errorf("mark ai batch item failed %s: %w", itemID.Hex(), err)
	}
	candidate.ErrorSummary = patch.ErrorSummary

	if err := candidate.Validate(); err != nil {
		return nil, fmt.Errorf("mark ai batch item failed %s: validate item: %w", itemID.Hex(), err)
	}

	result, err := repository.collection.UpdateOne(
		ctx,
		bson.M{"_id": itemID, "status": current.Status},
		bson.M{"$set": bson.M{
			"status":       candidate.Status,
			"updatedAt":    candidate.UpdatedAt,
			"completedAt":  candidate.CompletedAt.UTC(),
			"errorSummary": candidate.ErrorSummary,
		}},
	)
	if err != nil {
		return nil, fmt.Errorf("mark ai batch item failed %s: %w", itemID.Hex(), mapMongoError(err))
	}
	if result.MatchedCount == 0 {
		return nil, preconditionFailed("mark ai batch item failed %s stale write rejected", itemID.Hex())
	}
	return &candidate, nil
}

func (repository *AIBatchItemMongoRepository) MarkInvalidOutput(
	ctx context.Context,
	itemID primitive.ObjectID,
	patch platformrepo.AIBatchItemInvalidOutputPatch,
) (*aijobpkg.AIBatchItem, error) {
	current, err := repository.GetByID(ctx, itemID)
	if err != nil {
		return nil, err
	}
	if err := ensureAIBatchItemExpectedStatuses(current, patch.ExpectedCurrentStatuses, "mark ai batch item invalid"); err != nil {
		return nil, err
	}
	if !current.CanTransitionTo(common.AIBatchItemStatusInvalidOutput) {
		return nil, invalidTransition("mark ai batch item invalid %s cannot transition from %q", itemID.Hex(), current.Status)
	}

	candidate := *current
	if err := candidate.TransitionTo(common.AIBatchItemStatusInvalidOutput, patch.InvalidAt.UTC()); err != nil {
		return nil, fmt.Errorf("mark ai batch item invalid %s: %w", itemID.Hex(), err)
	}
	candidate.ErrorSummary = patch.ErrorSummary
	candidate.ValidationStatus = common.ValidationStatusInvalid
	candidate.ValidationErrors = append([]string(nil), patch.ValidationErrors...)

	if err := candidate.Validate(); err != nil {
		return nil, fmt.Errorf("mark ai batch item invalid %s: validate item: %w", itemID.Hex(), err)
	}

	result, err := repository.collection.UpdateOne(
		ctx,
		bson.M{"_id": itemID, "status": current.Status},
		bson.M{"$set": bson.M{
			"status":           candidate.Status,
			"updatedAt":        candidate.UpdatedAt,
			"completedAt":      candidate.CompletedAt.UTC(),
			"errorSummary":     candidate.ErrorSummary,
			"validationStatus": candidate.ValidationStatus,
			"validationErrors": candidate.ValidationErrors,
		}},
	)
	if err != nil {
		return nil, fmt.Errorf("mark ai batch item invalid %s: %w", itemID.Hex(), mapMongoError(err))
	}
	if result.MatchedCount == 0 {
		return nil, preconditionFailed("mark ai batch item invalid %s stale write rejected", itemID.Hex())
	}
	return &candidate, nil
}

func (repository *AIBatchItemMongoRepository) MarkSkipped(
	ctx context.Context,
	itemID primitive.ObjectID,
	patch platformrepo.AIBatchItemSkipPatch,
) (*aijobpkg.AIBatchItem, error) {
	current, err := repository.GetByID(ctx, itemID)
	if err != nil {
		return nil, err
	}
	if err := ensureAIBatchItemExpectedStatuses(current, patch.ExpectedCurrentStatuses, "mark ai batch item skipped"); err != nil {
		return nil, err
	}
	if !current.CanTransitionTo(common.AIBatchItemStatusSkipped) {
		return nil, invalidTransition("mark ai batch item skipped %s cannot transition from %q", itemID.Hex(), current.Status)
	}

	candidate := *current
	if err := candidate.TransitionTo(common.AIBatchItemStatusSkipped, patch.SkippedAt.UTC()); err != nil {
		return nil, fmt.Errorf("mark ai batch item skipped %s: %w", itemID.Hex(), err)
	}
	candidate.ErrorSummary = patch.Reason

	if err := candidate.Validate(); err != nil {
		return nil, fmt.Errorf("mark ai batch item skipped %s: validate item: %w", itemID.Hex(), err)
	}

	result, err := repository.collection.UpdateOne(
		ctx,
		bson.M{"_id": itemID, "status": current.Status},
		bson.M{"$set": bson.M{
			"status":       candidate.Status,
			"updatedAt":    candidate.UpdatedAt,
			"completedAt":  candidate.CompletedAt.UTC(),
			"errorSummary": candidate.ErrorSummary,
		}},
	)
	if err != nil {
		return nil, fmt.Errorf("mark ai batch item skipped %s: %w", itemID.Hex(), mapMongoError(err))
	}
	if result.MatchedCount == 0 {
		return nil, preconditionFailed("mark ai batch item skipped %s stale write rejected", itemID.Hex())
	}
	return &candidate, nil
}

func (repository *AIBatchItemMongoRepository) PrepareRetry(
	ctx context.Context,
	itemID primitive.ObjectID,
	patch platformrepo.AIBatchItemRetryPatch,
) (*aijobpkg.AIBatchItem, error) {
	current, err := repository.GetByID(ctx, itemID)
	if err != nil {
		return nil, err
	}
	if err := ensureAIBatchItemExpectedStatuses(current, patch.ExpectedCurrentStatuses, "prepare ai batch item retry"); err != nil {
		return nil, err
	}
	if !current.CanRetry() {
		return nil, invalidTransition("prepare ai batch item retry %s cannot retry from %q", itemID.Hex(), current.Status)
	}

	candidate := *current
	if err := candidate.ResetForRetry(patch.RetryAt.UTC()); err != nil {
		return nil, fmt.Errorf("prepare ai batch item retry %s: %w", itemID.Hex(), err)
	}
	candidate.TargetReviewID = primitive.NilObjectID
	candidate.TargetThesisID = primitive.NilObjectID
	candidate.TargetEntityVersion = 0

	if err := candidate.Validate(); err != nil {
		return nil, fmt.Errorf("prepare ai batch item retry %s: validate item: %w", itemID.Hex(), err)
	}

	update := bson.M{
		"$set": bson.M{
			"status":           candidate.Status,
			"validationStatus": candidate.ValidationStatus,
			"errorSummary":     candidate.ErrorSummary,
			"updatedAt":        candidate.UpdatedAt,
		},
		"$unset": bson.M{
			"validationErrors":    "",
			"resultPayload":       "",
			"completedAt":         "",
			"targetReviewId":      "",
			"targetThesisId":      "",
			"targetEntityVersion": "",
		},
	}
	result, err := repository.collection.UpdateOne(
		ctx,
		bson.M{"_id": itemID, "status": current.Status},
		update,
	)
	if err != nil {
		return nil, fmt.Errorf("prepare ai batch item retry %s: %w", itemID.Hex(), mapMongoError(err))
	}
	if result.MatchedCount == 0 {
		return nil, preconditionFailed("prepare ai batch item retry %s stale write rejected", itemID.Hex())
	}
	return &candidate, nil
}

func buildAIBatchItemFilter(filter platformrepo.AIBatchItemFilter) bson.M {
	query := bson.M{}
	addObjectIDFilter(query, "_id", filter.IDs)
	addObjectIDFilter(query, "aiBatchJobId", filter.AIBatchJobIDs)
	addObjectIDFilter(query, "workflowRunId", filter.WorkflowRunIDs)
	addObjectIDFilter(query, "companyId", filter.CompanyIDs)
	addObjectIDFilter(query, "targetReviewId", filter.TargetReviewIDs)
	addObjectIDFilter(query, "targetThesisId", filter.TargetThesisIDs)
	if len(filter.Symbols) > 0 {
		symbols := make([]string, 0, len(filter.Symbols))
		for _, symbol := range filter.Symbols {
			symbols = append(symbols, normalizeSymbol(symbol))
		}
		addStringFilter(query, "symbol", symbols)
	}
	if len(filter.BookTypes) > 0 {
		query["bookType"] = bson.M{"$in": filter.BookTypes}
	}
	if len(filter.ItemTypes) > 0 {
		query["itemType"] = bson.M{"$in": filter.ItemTypes}
	}
	if len(filter.Statuses) > 0 {
		query["status"] = bson.M{"$in": filter.Statuses}
	}
	if len(filter.ValidationStatuses) > 0 {
		query["validationStatus"] = bson.M{"$in": filter.ValidationStatuses}
	}
	if filter.PendingValidationOnly {
		query["status"] = common.AIBatchItemStatusCompleted
		query["validationStatus"] = common.ValidationStatusNotValidated
	}
	if filter.RetryableOnly {
		query["status"] = bson.M{"$in": []common.AIBatchItemStatus{
			common.AIBatchItemStatusFailed,
			common.AIBatchItemStatusInvalidOutput,
		}}
	}
	addTimeRangeFilter(query, "createdAt", filter.CreatedAt)
	addTimeRangeFilter(query, "updatedAt", filter.UpdatedAt)
	addTimeRangeFilter(query, "completedAt", filter.CompletedAt)
	return query
}

func buildAIBatchItemSort(option platformrepo.AIBatchItemSortOption) bson.D {
	switch option.By {
	case platformrepo.AIBatchItemSortByUpdatedAt:
		return bson.D{{Key: "updatedAt", Value: sortDirection(option.Order, platformrepo.SortOrderDescending)}, {Key: "createdAt", Value: -1}}
	case platformrepo.AIBatchItemSortByCompletedAt:
		return bson.D{{Key: "completedAt", Value: sortDirection(option.Order, platformrepo.SortOrderDescending)}, {Key: "createdAt", Value: -1}}
	case platformrepo.AIBatchItemSortBySymbol:
		return bson.D{{Key: "symbol", Value: sortDirection(option.Order, platformrepo.SortOrderAscending)}, {Key: "createdAt", Value: -1}}
	case platformrepo.AIBatchItemSortByCreatedAt, "":
		return bson.D{{Key: "createdAt", Value: sortDirection(option.Order, platformrepo.SortOrderDescending)}, {Key: "symbol", Value: 1}}
	default:
		return bson.D{{Key: "createdAt", Value: -1}, {Key: "symbol", Value: 1}}
	}
}

func buildAIBatchItemStatusUpdate(candidate *aijobpkg.AIBatchItem, patch platformrepo.AIBatchItemStatusPatch) bson.M {
	update := bson.M{
		"status":    candidate.Status,
		"updatedAt": candidate.UpdatedAt,
	}
	if candidate.CompletedAt != nil {
		update["completedAt"] = candidate.CompletedAt.UTC()
	}
	if patch.ErrorSummary != nil {
		update["errorSummary"] = candidate.ErrorSummary
	}
	return update
}

func ensureAIBatchItemExpectedStatuses(current *aijobpkg.AIBatchItem, expected []common.AIBatchItemStatus, operation string) error {
	if len(expected) == 0 {
		return nil
	}
	for _, status := range expected {
		if status == current.Status {
			return nil
		}
	}
	return preconditionFailed("%s %s expected current status %q", operation, current.ID.Hex(), current.Status)
}
