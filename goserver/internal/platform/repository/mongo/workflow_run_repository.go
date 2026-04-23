package mongo

import (
	"context"
	"fmt"
	"strings"

	"goserver/internal/domain/common"
	workflowpkg "goserver/internal/domain/workflow"
	platformrepo "goserver/internal/platform/repository"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

type WorkflowRunMongoRepository struct {
	collection *mongo.Collection
}

var _ platformrepo.WorkflowRunRepository = (*WorkflowRunMongoRepository)(nil)

func NewWorkflowRunRepository(collection *mongo.Collection) *WorkflowRunMongoRepository {
	return &WorkflowRunMongoRepository{collection: collection}
}

func (repository *WorkflowRunMongoRepository) Create(ctx context.Context, run *workflowpkg.WorkflowRun) (*workflowpkg.WorkflowRun, error) {
	if run == nil {
		return nil, fmt.Errorf("create workflow run: run is required")
	}

	document := *run
	if document.ID.IsZero() {
		document.ID = newDocumentID()
	}

	if err := document.Validate(); err != nil {
		return nil, fmt.Errorf("create workflow run: validate run: %w", err)
	}

	if _, err := repository.collection.InsertOne(ctx, &document); err != nil {
		return nil, fmt.Errorf("create workflow run: %w", mapMongoError(err))
	}

	return &document, nil
}

func (repository *WorkflowRunMongoRepository) GetByID(ctx context.Context, id primitive.ObjectID) (*workflowpkg.WorkflowRun, error) {
	document, err := findOne[workflowpkg.WorkflowRun](ctx, repository.collection, bson.M{"_id": id})
	if err != nil {
		return nil, fmt.Errorf("get workflow run by id %s: %w", id.Hex(), mapMongoError(err))
	}
	return document, nil
}

func (repository *WorkflowRunMongoRepository) List(
	ctx context.Context,
	filter platformrepo.WorkflowRunFilter,
	options platformrepo.WorkflowRunListOptions,
) (*platformrepo.ListResult[*workflowpkg.WorkflowRun], error) {
	result, err := findPage[workflowpkg.WorkflowRun, *workflowpkg.WorkflowRun](
		ctx,
		repository.collection,
		buildWorkflowRunFilter(filter),
		options.Pagination,
		buildWorkflowRunSort(options.Sort),
		nil,
		func(document *workflowpkg.WorkflowRun) *workflowpkg.WorkflowRun {
			run := *document
			return &run
		},
	)
	if err != nil {
		return nil, fmt.Errorf("list workflow runs: %w", mapMongoError(err))
	}
	return result, nil
}

func (repository *WorkflowRunMongoRepository) FindResumable(
	ctx context.Context,
	filter platformrepo.WorkflowRunFilter,
	options platformrepo.WorkflowRunListOptions,
) (*platformrepo.ListResult[*workflowpkg.WorkflowRun], error) {
	if len(filter.Statuses) == 0 {
		filter.Statuses = []common.WorkflowRunStatus{
			common.WorkflowRunStatusCreated,
			common.WorkflowRunStatusRunning,
			common.WorkflowRunStatusWaitingExternal,
			common.WorkflowRunStatusPartiallyCompleted,
		}
	}
	filter.TerminalOnly = false
	return repository.List(ctx, filter, options)
}

func (repository *WorkflowRunMongoRepository) UpdateStatus(
	ctx context.Context,
	workflowRunID primitive.ObjectID,
	patch platformrepo.WorkflowRunStatusPatch,
) (*workflowpkg.WorkflowRun, error) {
	current, err := repository.GetByID(ctx, workflowRunID)
	if err != nil {
		return nil, err
	}

	if len(patch.ExpectedCurrentStatuses) > 0 && !containsWorkflowRunStatus(patch.ExpectedCurrentStatuses, current.Status) {
		return nil, preconditionFailed("update workflow run status %s expected current status %q", workflowRunID.Hex(), current.Status)
	}
	if !current.CanTransitionTo(patch.NextStatus) {
		return nil, invalidTransition("update workflow run status %s cannot transition from %q to %q", workflowRunID.Hex(), current.Status, patch.NextStatus)
	}

	candidate := *current
	transitionAt := mutationTimestamp(patch.Mutation)
	if err := candidate.TransitionTo(patch.NextStatus, transitionAt); err != nil {
		return nil, fmt.Errorf("update workflow run status %s: %w", workflowRunID.Hex(), err)
	}
	if patch.Notes != nil {
		candidate.Notes = *patch.Notes
	}

	if err := candidate.Validate(); err != nil {
		return nil, fmt.Errorf("update workflow run status %s: validate run: %w", workflowRunID.Hex(), err)
	}

	result, err := repository.collection.UpdateOne(
		ctx,
		bson.M{"_id": workflowRunID, "status": current.Status},
		bson.M{"$set": buildWorkflowRunStatusUpdate(&candidate, patch)},
	)
	if err != nil {
		return nil, fmt.Errorf("update workflow run status %s: %w", workflowRunID.Hex(), mapMongoError(err))
	}
	if result.MatchedCount == 0 {
		return nil, preconditionFailed("update workflow run status %s stale write rejected", workflowRunID.Hex())
	}

	return &candidate, nil
}

func (repository *WorkflowRunMongoRepository) UpdateProgressCounters(
	ctx context.Context,
	workflowRunID primitive.ObjectID,
	patch platformrepo.WorkflowRunProgressPatch,
) (*workflowpkg.WorkflowRun, error) {
	current, err := repository.GetByID(ctx, workflowRunID)
	if err != nil {
		return nil, err
	}

	if len(patch.ExpectedCurrentStatuses) > 0 && !containsWorkflowRunStatus(patch.ExpectedCurrentStatuses, current.Status) {
		return nil, preconditionFailed("update workflow run progress %s expected current status %q", workflowRunID.Hex(), current.Status)
	}

	candidate := *current
	candidate.CompaniesScannedCount += patch.CompaniesScannedDelta
	candidate.ReviewsCreatedCount += patch.ReviewsCreatedDelta
	candidate.ErrorsCount += patch.ErrorsDelta
	candidate.UpdatedAt = mutationTimestamp(patch.Mutation)
	if patch.Notes != nil {
		candidate.Notes = *patch.Notes
	}

	if err := candidate.Validate(); err != nil {
		return nil, fmt.Errorf("update workflow run progress %s: validate run: %w", workflowRunID.Hex(), err)
	}

	update := bson.M{
		"$set": bson.M{"updatedAt": candidate.UpdatedAt},
	}
	if patch.Notes != nil {
		update["$set"].(bson.M)["notes"] = candidate.Notes
	}
	inc := bson.M{}
	if patch.CompaniesScannedDelta != 0 {
		inc["companiesScannedCount"] = patch.CompaniesScannedDelta
	}
	if patch.ReviewsCreatedDelta != 0 {
		inc["reviewsCreatedCount"] = patch.ReviewsCreatedDelta
	}
	if patch.ErrorsDelta != 0 {
		inc["errorsCount"] = patch.ErrorsDelta
	}
	if len(inc) > 0 {
		update["$inc"] = inc
	}

	result, err := repository.collection.UpdateOne(
		ctx,
		bson.M{"_id": workflowRunID, "status": current.Status},
		update,
	)
	if err != nil {
		return nil, fmt.Errorf("update workflow run progress %s: %w", workflowRunID.Hex(), mapMongoError(err))
	}
	if result.MatchedCount == 0 {
		return nil, preconditionFailed("update workflow run progress %s stale write rejected", workflowRunID.Hex())
	}

	return &candidate, nil
}

func (repository *WorkflowRunMongoRepository) MarkCompleted(
	ctx context.Context,
	workflowRunID primitive.ObjectID,
	patch platformrepo.WorkflowRunCompletionPatch,
) (*workflowpkg.WorkflowRun, error) {
	current, err := repository.GetByID(ctx, workflowRunID)
	if err != nil {
		return nil, err
	}

	if len(patch.ExpectedCurrentStatuses) > 0 && !containsWorkflowRunStatus(patch.ExpectedCurrentStatuses, current.Status) {
		return nil, preconditionFailed("mark workflow run completed %s expected current status %q", workflowRunID.Hex(), current.Status)
	}
	if !current.CanTransitionTo(common.WorkflowRunStatusCompleted) {
		return nil, invalidTransition("mark workflow run completed %s cannot transition from %q", workflowRunID.Hex(), current.Status)
	}

	candidate := *current
	if err := candidate.TransitionTo(common.WorkflowRunStatusCompleted, patch.CompletedAt.UTC()); err != nil {
		return nil, fmt.Errorf("mark workflow run completed %s: %w", workflowRunID.Hex(), err)
	}
	if patch.Notes != nil {
		candidate.Notes = *patch.Notes
	}

	if err := candidate.Validate(); err != nil {
		return nil, fmt.Errorf("mark workflow run completed %s: validate run: %w", workflowRunID.Hex(), err)
	}

	result, err := repository.collection.UpdateOne(
		ctx,
		bson.M{"_id": workflowRunID, "status": current.Status},
		bson.M{"$set": buildWorkflowRunCompletionUpdate(&candidate, patch)},
	)
	if err != nil {
		return nil, fmt.Errorf("mark workflow run completed %s: %w", workflowRunID.Hex(), mapMongoError(err))
	}
	if result.MatchedCount == 0 {
		return nil, preconditionFailed("mark workflow run completed %s stale write rejected", workflowRunID.Hex())
	}

	return &candidate, nil
}

func (repository *WorkflowRunMongoRepository) MarkFailed(
	ctx context.Context,
	workflowRunID primitive.ObjectID,
	patch platformrepo.WorkflowRunFailurePatch,
) (*workflowpkg.WorkflowRun, error) {
	current, err := repository.GetByID(ctx, workflowRunID)
	if err != nil {
		return nil, err
	}

	if len(patch.ExpectedCurrentStatuses) > 0 && !containsWorkflowRunStatus(patch.ExpectedCurrentStatuses, current.Status) {
		return nil, preconditionFailed("mark workflow run failed %s expected current status %q", workflowRunID.Hex(), current.Status)
	}
	if !current.CanTransitionTo(common.WorkflowRunStatusFailed) {
		return nil, invalidTransition("mark workflow run failed %s cannot transition from %q", workflowRunID.Hex(), current.Status)
	}

	candidate := *current
	if err := candidate.TransitionTo(common.WorkflowRunStatusFailed, patch.FailedAt.UTC()); err != nil {
		return nil, fmt.Errorf("mark workflow run failed %s: %w", workflowRunID.Hex(), err)
	}
	candidate.Notes = mergeWorkflowRunFailureNotes(current.Notes, patch.Notes, patch.FailureReason)

	if err := candidate.Validate(); err != nil {
		return nil, fmt.Errorf("mark workflow run failed %s: validate run: %w", workflowRunID.Hex(), err)
	}

	result, err := repository.collection.UpdateOne(
		ctx,
		bson.M{"_id": workflowRunID, "status": current.Status},
		bson.M{"$set": buildWorkflowRunFailureUpdate(&candidate)},
	)
	if err != nil {
		return nil, fmt.Errorf("mark workflow run failed %s: %w", workflowRunID.Hex(), mapMongoError(err))
	}
	if result.MatchedCount == 0 {
		return nil, preconditionFailed("mark workflow run failed %s stale write rejected", workflowRunID.Hex())
	}

	return &candidate, nil
}

func buildWorkflowRunFilter(filter platformrepo.WorkflowRunFilter) bson.M {
	query := bson.M{}
	addObjectIDFilter(query, "_id", filter.IDs)
	addObjectIDFilter(query, "configSnapshotId", filter.ConfigSnapshotIDs)
	if len(filter.BookTypes) > 0 {
		query["bookType"] = bson.M{"$in": filter.BookTypes}
	}
	if len(filter.RunTypes) > 0 {
		query["runType"] = bson.M{"$in": filter.RunTypes}
	}
	if len(filter.Statuses) > 0 {
		query["status"] = bson.M{"$in": filter.Statuses}
	}
	if filter.ActiveOnly {
		query["status"] = bson.M{"$in": []common.WorkflowRunStatus{
			common.WorkflowRunStatusCreated,
			common.WorkflowRunStatusRunning,
			common.WorkflowRunStatusWaitingExternal,
			common.WorkflowRunStatusPartiallyCompleted,
		}}
	}
	if filter.TerminalOnly {
		query["status"] = bson.M{"$in": []common.WorkflowRunStatus{
			common.WorkflowRunStatusCompleted,
			common.WorkflowRunStatusFailed,
			common.WorkflowRunStatusCancelled,
		}}
	}
	addTimeRangeFilter(query, "startedAt", filter.StartedAt)
	addTimeRangeFilter(query, "completedAt", filter.CompletedAt)
	addTimeRangeFilter(query, "createdAt", filter.CreatedAt)
	addTimeRangeFilter(query, "updatedAt", filter.UpdatedAt)
	return query
}

func buildWorkflowRunSort(option platformrepo.WorkflowRunSortOption) bson.D {
	switch option.By {
	case platformrepo.WorkflowRunSortByCompletedAt:
		return bson.D{{Key: "completedAt", Value: sortDirection(option.Order, platformrepo.SortOrderDescending)}, {Key: "startedAt", Value: -1}}
	case platformrepo.WorkflowRunSortByCreatedAt:
		return bson.D{{Key: "createdAt", Value: sortDirection(option.Order, platformrepo.SortOrderDescending)}, {Key: "startedAt", Value: -1}}
	case platformrepo.WorkflowRunSortByUpdatedAt:
		return bson.D{{Key: "updatedAt", Value: sortDirection(option.Order, platformrepo.SortOrderDescending)}, {Key: "startedAt", Value: -1}}
	case platformrepo.WorkflowRunSortByStartedAt, "":
		return bson.D{{Key: "startedAt", Value: sortDirection(option.Order, platformrepo.SortOrderDescending)}, {Key: "createdAt", Value: -1}}
	default:
		return bson.D{{Key: "startedAt", Value: -1}, {Key: "createdAt", Value: -1}}
	}
}

func buildWorkflowRunStatusUpdate(candidate *workflowpkg.WorkflowRun, patch platformrepo.WorkflowRunStatusPatch) bson.M {
	update := bson.M{
		"status":    candidate.Status,
		"updatedAt": candidate.UpdatedAt,
	}
	if patch.Notes != nil {
		update["notes"] = candidate.Notes
	}
	if candidate.CompletedAt != nil {
		update["completedAt"] = candidate.CompletedAt.UTC()
	}
	return update
}

func buildWorkflowRunCompletionUpdate(candidate *workflowpkg.WorkflowRun, patch platformrepo.WorkflowRunCompletionPatch) bson.M {
	update := bson.M{
		"status":      candidate.Status,
		"updatedAt":   candidate.UpdatedAt,
		"completedAt": candidate.CompletedAt.UTC(),
	}
	if patch.Notes != nil {
		update["notes"] = candidate.Notes
	}
	return update
}

func buildWorkflowRunFailureUpdate(candidate *workflowpkg.WorkflowRun) bson.M {
	return bson.M{
		"status":      candidate.Status,
		"updatedAt":   candidate.UpdatedAt,
		"completedAt": candidate.CompletedAt.UTC(),
		"notes":       candidate.Notes,
	}
}

func mergeWorkflowRunFailureNotes(currentNotes string, patchNotes *string, failureReason string) string {
	parts := make([]string, 0, 2)
	if patchNotes != nil && strings.TrimSpace(*patchNotes) != "" {
		parts = append(parts, strings.TrimSpace(*patchNotes))
	} else if strings.TrimSpace(currentNotes) != "" {
		parts = append(parts, strings.TrimSpace(currentNotes))
	}
	if strings.TrimSpace(failureReason) != "" {
		parts = append(parts, "failure_reason: "+strings.TrimSpace(failureReason))
	}
	return strings.Join(parts, "\n")
}

func containsWorkflowRunStatus(expected []common.WorkflowRunStatus, actual common.WorkflowRunStatus) bool {
	for _, status := range expected {
		if status == actual {
			return true
		}
	}
	return false
}
