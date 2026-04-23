package mongo

import (
	"context"
	"fmt"
	"time"

	"goserver/internal/domain/common"
	workflowpkg "goserver/internal/domain/workflow"
	platformrepo "goserver/internal/platform/repository"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

type WorkflowStepRunMongoRepository struct {
	collection *mongo.Collection
}

var _ platformrepo.WorkflowStepRunRepository = (*WorkflowStepRunMongoRepository)(nil)

func NewWorkflowStepRunRepository(collection *mongo.Collection) *WorkflowStepRunMongoRepository {
	return &WorkflowStepRunMongoRepository{collection: collection}
}

func (repository *WorkflowStepRunMongoRepository) Create(ctx context.Context, stepRun *workflowpkg.WorkflowStepRun) (*workflowpkg.WorkflowStepRun, error) {
	if stepRun == nil {
		return nil, fmt.Errorf("create workflow step run: step run is required")
	}

	document := *stepRun
	if document.ID.IsZero() {
		document.ID = newDocumentID()
	}

	if err := document.Validate(); err != nil {
		return nil, fmt.Errorf("create workflow step run: validate step run: %w", err)
	}

	if _, err := repository.collection.InsertOne(ctx, &document); err != nil {
		return nil, fmt.Errorf("create workflow step run: %w", mapMongoError(err))
	}

	return &document, nil
}

func (repository *WorkflowStepRunMongoRepository) GetByID(ctx context.Context, id primitive.ObjectID) (*workflowpkg.WorkflowStepRun, error) {
	document, err := findOne[workflowpkg.WorkflowStepRun](ctx, repository.collection, bson.M{"_id": id})
	if err != nil {
		return nil, fmt.Errorf("get workflow step run by id %s: %w", id.Hex(), mapMongoError(err))
	}
	return document, nil
}

func (repository *WorkflowStepRunMongoRepository) GetByWorkflowRunAndStepName(
	ctx context.Context,
	workflowRunID primitive.ObjectID,
	stepName common.WorkflowStepName,
) (*workflowpkg.WorkflowStepRun, error) {
	document, err := findOne[workflowpkg.WorkflowStepRun](
		ctx,
		repository.collection,
		bson.M{"workflowRunId": workflowRunID, "stepName": stepName},
	)
	if err != nil {
		return nil, fmt.Errorf("get workflow step run by workflow %s step %s: %w", workflowRunID.Hex(), stepName, mapMongoError(err))
	}
	return document, nil
}

func (repository *WorkflowStepRunMongoRepository) ListByWorkflowRunID(
	ctx context.Context,
	workflowRunID primitive.ObjectID,
	options platformrepo.WorkflowStepRunListOptions,
) (*platformrepo.ListResult[*workflowpkg.WorkflowStepRun], error) {
	return repository.List(ctx, platformrepo.WorkflowStepRunFilter{WorkflowRunIDs: []primitive.ObjectID{workflowRunID}}, options)
}

func (repository *WorkflowStepRunMongoRepository) List(
	ctx context.Context,
	filter platformrepo.WorkflowStepRunFilter,
	options platformrepo.WorkflowStepRunListOptions,
) (*platformrepo.ListResult[*workflowpkg.WorkflowStepRun], error) {
	result, err := findPage[workflowpkg.WorkflowStepRun, *workflowpkg.WorkflowStepRun](
		ctx,
		repository.collection,
		buildWorkflowStepRunFilter(filter),
		options.Pagination,
		buildWorkflowStepRunSort(options.Sort),
		nil,
		func(document *workflowpkg.WorkflowStepRun) *workflowpkg.WorkflowStepRun {
			stepRun := *document
			return &stepRun
		},
	)
	if err != nil {
		return nil, fmt.Errorf("list workflow step runs: %w", mapMongoError(err))
	}
	return result, nil
}

func (repository *WorkflowStepRunMongoRepository) UpdateStatus(
	ctx context.Context,
	stepRunID primitive.ObjectID,
	patch platformrepo.WorkflowStepStatusPatch,
) (*workflowpkg.WorkflowStepRun, error) {
	current, err := repository.GetByID(ctx, stepRunID)
	if err != nil {
		return nil, err
	}

	if len(patch.ExpectedCurrentStatuses) > 0 && !containsWorkflowStepStatus(patch.ExpectedCurrentStatuses, current.Status) {
		return nil, preconditionFailed("update workflow step status %s expected current status %q", stepRunID.Hex(), current.Status)
	}
	if !current.CanTransitionTo(patch.NextStatus) {
		return nil, invalidTransition("update workflow step status %s cannot transition from %q to %q", stepRunID.Hex(), current.Status, patch.NextStatus)
	}

	candidate := *current
	if err := candidate.TransitionTo(patch.NextStatus, mutationTimestamp(patch.Mutation)); err != nil {
		return nil, fmt.Errorf("update workflow step status %s: %w", stepRunID.Hex(), err)
	}
	if patch.ErrorSummary != nil {
		candidate.ErrorSummary = *patch.ErrorSummary
	}

	if err := candidate.Validate(); err != nil {
		return nil, fmt.Errorf("update workflow step status %s: validate step run: %w", stepRunID.Hex(), err)
	}

	result, err := repository.collection.UpdateOne(
		ctx,
		bson.M{"_id": stepRunID, "status": current.Status},
		bson.M{"$set": buildWorkflowStepStatusUpdate(&candidate, patch)},
	)
	if err != nil {
		return nil, fmt.Errorf("update workflow step status %s: %w", stepRunID.Hex(), mapMongoError(err))
	}
	if result.MatchedCount == 0 {
		return nil, preconditionFailed("update workflow step status %s stale write rejected", stepRunID.Hex())
	}

	return &candidate, nil
}

func (repository *WorkflowStepRunMongoRepository) MarkStarted(
	ctx context.Context,
	stepRunID primitive.ObjectID,
	patch platformrepo.WorkflowStepStartPatch,
) (*workflowpkg.WorkflowStepRun, error) {
	current, err := repository.GetByID(ctx, stepRunID)
	if err != nil {
		return nil, err
	}
	if len(patch.ExpectedCurrentStatuses) > 0 && !containsWorkflowStepStatus(patch.ExpectedCurrentStatuses, current.Status) {
		return nil, preconditionFailed("mark workflow step started %s expected current status %q", stepRunID.Hex(), current.Status)
	}
	if !current.CanTransitionTo(common.WorkflowStepStatusRunning) {
		return nil, invalidTransition("mark workflow step started %s cannot transition from %q", stepRunID.Hex(), current.Status)
	}

	candidate := *current
	if err := candidate.TransitionTo(common.WorkflowStepStatusRunning, patch.StartedAt.UTC()); err != nil {
		return nil, fmt.Errorf("mark workflow step started %s: %w", stepRunID.Hex(), err)
	}

	set := bson.M{
		"status":    candidate.Status,
		"startedAt": candidate.StartedAt.UTC(),
		"updatedAt": candidate.UpdatedAt,
	}
	unset := bson.M{}
	candidate.Metadata = applyMetadataPatch("metadata", patch.Metadata, candidate.Metadata, set, unset)
	if err := candidate.Validate(); err != nil {
		return nil, fmt.Errorf("mark workflow step started %s: validate step run: %w", stepRunID.Hex(), err)
	}

	update := bson.M{"$set": set}
	if len(unset) > 0 {
		update["$unset"] = unset
	}
	result, err := repository.collection.UpdateOne(ctx, bson.M{"_id": stepRunID, "status": current.Status}, update)
	if err != nil {
		return nil, fmt.Errorf("mark workflow step started %s: %w", stepRunID.Hex(), mapMongoError(err))
	}
	if result.MatchedCount == 0 {
		return nil, preconditionFailed("mark workflow step started %s stale write rejected", stepRunID.Hex())
	}

	return &candidate, nil
}

func (repository *WorkflowStepRunMongoRepository) MarkCompleted(
	ctx context.Context,
	stepRunID primitive.ObjectID,
	patch platformrepo.WorkflowStepCompletionPatch,
) (*workflowpkg.WorkflowStepRun, error) {
	current, err := repository.GetByID(ctx, stepRunID)
	if err != nil {
		return nil, err
	}
	if len(patch.ExpectedCurrentStatuses) > 0 && !containsWorkflowStepStatus(patch.ExpectedCurrentStatuses, current.Status) {
		return nil, preconditionFailed("mark workflow step completed %s expected current status %q", stepRunID.Hex(), current.Status)
	}
	if !current.CanTransitionTo(common.WorkflowStepStatusCompleted) {
		return nil, invalidTransition("mark workflow step completed %s cannot transition from %q", stepRunID.Hex(), current.Status)
	}

	candidate := *current
	if err := candidate.TransitionTo(common.WorkflowStepStatusCompleted, patch.CompletedAt.UTC()); err != nil {
		return nil, fmt.Errorf("mark workflow step completed %s: %w", stepRunID.Hex(), err)
	}

	set := bson.M{
		"status":      candidate.Status,
		"updatedAt":   candidate.UpdatedAt,
		"completedAt": candidate.CompletedAt.UTC(),
	}
	unset := bson.M{}
	candidate.Metadata = applyMetadataPatch("metadata", patch.Metadata, candidate.Metadata, set, unset)
	if err := candidate.Validate(); err != nil {
		return nil, fmt.Errorf("mark workflow step completed %s: validate step run: %w", stepRunID.Hex(), err)
	}

	update := bson.M{"$set": set}
	if len(unset) > 0 {
		update["$unset"] = unset
	}
	result, err := repository.collection.UpdateOne(ctx, bson.M{"_id": stepRunID, "status": current.Status}, update)
	if err != nil {
		return nil, fmt.Errorf("mark workflow step completed %s: %w", stepRunID.Hex(), mapMongoError(err))
	}
	if result.MatchedCount == 0 {
		return nil, preconditionFailed("mark workflow step completed %s stale write rejected", stepRunID.Hex())
	}

	return &candidate, nil
}

func (repository *WorkflowStepRunMongoRepository) MarkFailed(
	ctx context.Context,
	stepRunID primitive.ObjectID,
	patch platformrepo.WorkflowStepFailurePatch,
) (*workflowpkg.WorkflowStepRun, error) {
	current, err := repository.GetByID(ctx, stepRunID)
	if err != nil {
		return nil, err
	}
	if len(patch.ExpectedCurrentStatuses) > 0 && !containsWorkflowStepStatus(patch.ExpectedCurrentStatuses, current.Status) {
		return nil, preconditionFailed("mark workflow step failed %s expected current status %q", stepRunID.Hex(), current.Status)
	}
	if !current.CanTransitionTo(common.WorkflowStepStatusFailed) {
		return nil, invalidTransition("mark workflow step failed %s cannot transition from %q", stepRunID.Hex(), current.Status)
	}

	candidate := *current
	if err := candidate.TransitionTo(common.WorkflowStepStatusFailed, patch.FailedAt.UTC()); err != nil {
		return nil, fmt.Errorf("mark workflow step failed %s: %w", stepRunID.Hex(), err)
	}
	candidate.ErrorSummary = patch.ErrorSummary

	set := bson.M{
		"status":       candidate.Status,
		"updatedAt":    candidate.UpdatedAt,
		"completedAt":  candidate.CompletedAt.UTC(),
		"errorSummary": candidate.ErrorSummary,
	}
	unset := bson.M{}
	candidate.Metadata = applyMetadataPatch("metadata", patch.Metadata, candidate.Metadata, set, unset)
	if err := candidate.Validate(); err != nil {
		return nil, fmt.Errorf("mark workflow step failed %s: validate step run: %w", stepRunID.Hex(), err)
	}

	update := bson.M{"$set": set}
	if len(unset) > 0 {
		update["$unset"] = unset
	}
	result, err := repository.collection.UpdateOne(ctx, bson.M{"_id": stepRunID, "status": current.Status}, update)
	if err != nil {
		return nil, fmt.Errorf("mark workflow step failed %s: %w", stepRunID.Hex(), mapMongoError(err))
	}
	if result.MatchedCount == 0 {
		return nil, preconditionFailed("mark workflow step failed %s stale write rejected", stepRunID.Hex())
	}

	return &candidate, nil
}

func (repository *WorkflowStepRunMongoRepository) MarkSkipped(
	ctx context.Context,
	stepRunID primitive.ObjectID,
	patch platformrepo.WorkflowStepSkipPatch,
) (*workflowpkg.WorkflowStepRun, error) {
	current, err := repository.GetByID(ctx, stepRunID)
	if err != nil {
		return nil, err
	}
	if len(patch.ExpectedCurrentStatuses) > 0 && !containsWorkflowStepStatus(patch.ExpectedCurrentStatuses, current.Status) {
		return nil, preconditionFailed("mark workflow step skipped %s expected current status %q", stepRunID.Hex(), current.Status)
	}
	if !current.CanTransitionTo(common.WorkflowStepStatusSkipped) {
		return nil, invalidTransition("mark workflow step skipped %s cannot transition from %q", stepRunID.Hex(), current.Status)
	}

	candidate := *current
	if err := candidate.TransitionTo(common.WorkflowStepStatusSkipped, patch.SkippedAt.UTC()); err != nil {
		return nil, fmt.Errorf("mark workflow step skipped %s: %w", stepRunID.Hex(), err)
	}
	candidate.ErrorSummary = patch.Reason

	set := bson.M{
		"status":       candidate.Status,
		"updatedAt":    candidate.UpdatedAt,
		"completedAt":  candidate.CompletedAt.UTC(),
		"errorSummary": candidate.ErrorSummary,
	}
	unset := bson.M{}
	candidate.Metadata = applyMetadataPatch("metadata", patch.Metadata, candidate.Metadata, set, unset)
	if err := candidate.Validate(); err != nil {
		return nil, fmt.Errorf("mark workflow step skipped %s: validate step run: %w", stepRunID.Hex(), err)
	}

	update := bson.M{"$set": set}
	if len(unset) > 0 {
		update["$unset"] = unset
	}
	result, err := repository.collection.UpdateOne(ctx, bson.M{"_id": stepRunID, "status": current.Status}, update)
	if err != nil {
		return nil, fmt.Errorf("mark workflow step skipped %s: %w", stepRunID.Hex(), mapMongoError(err))
	}
	if result.MatchedCount == 0 {
		return nil, preconditionFailed("mark workflow step skipped %s stale write rejected", stepRunID.Hex())
	}

	return &candidate, nil
}

func (repository *WorkflowStepRunMongoRepository) UpdateMetadata(
	ctx context.Context,
	stepRunID primitive.ObjectID,
	patch platformrepo.WorkflowStepMetadataPatch,
) (*workflowpkg.WorkflowStepRun, error) {
	current, err := repository.GetByID(ctx, stepRunID)
	if err != nil {
		return nil, err
	}
	if len(patch.ExpectedCurrentStatuses) > 0 && !containsWorkflowStepStatus(patch.ExpectedCurrentStatuses, current.Status) {
		return nil, preconditionFailed("update workflow step metadata %s expected current status %q", stepRunID.Hex(), current.Status)
	}

	candidate := *current
	set := bson.M{"updatedAt": mutationTimestamp(patch.Mutation)}
	unset := bson.M{}
	candidate.UpdatedAt = set["updatedAt"].(time.Time)
	candidate.Metadata = applyMetadataPatch("metadata", &patch.Metadata, candidate.Metadata, set, unset)

	if err := candidate.Validate(); err != nil {
		return nil, fmt.Errorf("update workflow step metadata %s: validate step run: %w", stepRunID.Hex(), err)
	}

	update := bson.M{"$set": set}
	if len(unset) > 0 {
		update["$unset"] = unset
	}
	result, err := repository.collection.UpdateOne(ctx, bson.M{"_id": stepRunID, "status": current.Status}, update)
	if err != nil {
		return nil, fmt.Errorf("update workflow step metadata %s: %w", stepRunID.Hex(), mapMongoError(err))
	}
	if result.MatchedCount == 0 {
		return nil, preconditionFailed("update workflow step metadata %s stale write rejected", stepRunID.Hex())
	}

	return &candidate, nil
}

func buildWorkflowStepRunFilter(filter platformrepo.WorkflowStepRunFilter) bson.M {
	query := bson.M{}
	addObjectIDFilter(query, "_id", filter.IDs)
	addObjectIDFilter(query, "workflowRunId", filter.WorkflowRunIDs)
	if len(filter.StepNames) > 0 {
		query["stepName"] = bson.M{"$in": filter.StepNames}
	}
	if len(filter.Statuses) > 0 {
		query["status"] = bson.M{"$in": filter.Statuses}
	}
	if filter.ActiveOnly {
		query["status"] = bson.M{"$in": []common.WorkflowStepStatus{
			common.WorkflowStepStatusPending,
			common.WorkflowStepStatusRunning,
			common.WorkflowStepStatusWaitingExternal,
		}}
	}
	if filter.TerminalOnly {
		query["status"] = bson.M{"$in": []common.WorkflowStepStatus{
			common.WorkflowStepStatusCompleted,
			common.WorkflowStepStatusFailed,
			common.WorkflowStepStatusSkipped,
		}}
	}
	addTimeRangeFilter(query, "startedAt", filter.StartedAt)
	addTimeRangeFilter(query, "completedAt", filter.CompletedAt)
	addTimeRangeFilter(query, "createdAt", filter.CreatedAt)
	addTimeRangeFilter(query, "updatedAt", filter.UpdatedAt)
	return query
}

func buildWorkflowStepRunSort(option platformrepo.WorkflowStepRunSortOption) bson.D {
	switch option.By {
	case platformrepo.WorkflowStepRunSortByStepName:
		return bson.D{{Key: "stepName", Value: sortDirection(option.Order, platformrepo.SortOrderAscending)}, {Key: "createdAt", Value: 1}}
	case platformrepo.WorkflowStepRunSortByStartedAt:
		return bson.D{{Key: "startedAt", Value: sortDirection(option.Order, platformrepo.SortOrderAscending)}, {Key: "createdAt", Value: 1}}
	case platformrepo.WorkflowStepRunSortByCompletedAt:
		return bson.D{{Key: "completedAt", Value: sortDirection(option.Order, platformrepo.SortOrderDescending)}, {Key: "createdAt", Value: 1}}
	case platformrepo.WorkflowStepRunSortByUpdatedAt:
		return bson.D{{Key: "updatedAt", Value: sortDirection(option.Order, platformrepo.SortOrderDescending)}, {Key: "createdAt", Value: 1}}
	case platformrepo.WorkflowStepRunSortByCreatedAt, "":
		return bson.D{{Key: "createdAt", Value: sortDirection(option.Order, platformrepo.SortOrderAscending)}, {Key: "stepName", Value: 1}}
	default:
		return bson.D{{Key: "createdAt", Value: 1}, {Key: "stepName", Value: 1}}
	}
}

func buildWorkflowStepStatusUpdate(candidate *workflowpkg.WorkflowStepRun, patch platformrepo.WorkflowStepStatusPatch) bson.M {
	update := bson.M{
		"status":    candidate.Status,
		"updatedAt": candidate.UpdatedAt,
	}
	if candidate.StartedAt != nil {
		update["startedAt"] = candidate.StartedAt.UTC()
	}
	if candidate.CompletedAt != nil {
		update["completedAt"] = candidate.CompletedAt.UTC()
	}
	if patch.ErrorSummary != nil {
		update["errorSummary"] = candidate.ErrorSummary
	}
	return update
}

func containsWorkflowStepStatus(expected []common.WorkflowStepStatus, actual common.WorkflowStepStatus) bool {
	for _, status := range expected {
		if status == actual {
			return true
		}
	}
	return false
}
