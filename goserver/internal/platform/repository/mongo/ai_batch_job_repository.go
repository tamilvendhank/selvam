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
	"go.mongodb.org/mongo-driver/mongo/options"
)

type AIBatchJobMongoRepository struct {
	collection *mongo.Collection
}

var _ platformrepo.AIBatchJobRepository = (*AIBatchJobMongoRepository)(nil)

func NewAIBatchJobRepository(collection *mongo.Collection) *AIBatchJobMongoRepository {
	return &AIBatchJobMongoRepository{collection: collection}
}

func (repository *AIBatchJobMongoRepository) Create(ctx context.Context, job *aijobpkg.AIBatchJob) (*aijobpkg.AIBatchJob, error) {
	if job == nil {
		return nil, fmt.Errorf("create ai batch job: job is required")
	}

	document := *job
	if document.ID.IsZero() {
		document.ID = newDocumentID()
	}

	if err := document.Validate(); err != nil {
		return nil, fmt.Errorf("create ai batch job: validate job: %w", err)
	}

	if _, err := repository.collection.InsertOne(ctx, &document); err != nil {
		return nil, fmt.Errorf("create ai batch job: %w", mapMongoError(err))
	}

	return &document, nil
}

func (repository *AIBatchJobMongoRepository) GetByID(ctx context.Context, id primitive.ObjectID) (*aijobpkg.AIBatchJob, error) {
	document, err := findOne[aijobpkg.AIBatchJob](ctx, repository.collection, bson.M{"_id": id})
	if err != nil {
		return nil, fmt.Errorf("get ai batch job by id %s: %w", id.Hex(), mapMongoError(err))
	}
	return document, nil
}

func (repository *AIBatchJobMongoRepository) GetByProviderJobHandle(
	ctx context.Context,
	providerName string,
	providerJobHandle string,
) (*aijobpkg.AIBatchJob, error) {
	document, err := findOne[aijobpkg.AIBatchJob](
		ctx,
		repository.collection,
		bson.M{"providerName": providerName, "providerJobHandle": providerJobHandle},
	)
	if err != nil {
		return nil, fmt.Errorf("get ai batch job by provider %s handle %s: %w", providerName, providerJobHandle, mapMongoError(err))
	}
	return document, nil
}

func (repository *AIBatchJobMongoRepository) GetByIdempotencyKey(ctx context.Context, idempotencyKey string) (*aijobpkg.AIBatchJob, error) {
	document, err := findOne[aijobpkg.AIBatchJob](
		ctx,
		repository.collection,
		bson.M{"idempotencyKey": idempotencyKey},
		options.FindOne().SetSort(bson.D{{Key: "createdAt", Value: -1}}),
	)
	if err != nil {
		return nil, fmt.Errorf("get ai batch job by idempotency key %s: %w", idempotencyKey, mapMongoError(err))
	}
	return document, nil
}

func (repository *AIBatchJobMongoRepository) List(
	ctx context.Context,
	filter platformrepo.AIBatchJobFilter,
	options platformrepo.AIBatchJobListOptions,
) (*platformrepo.ListResult[*aijobpkg.AIBatchJob], error) {
	result, err := findPage[aijobpkg.AIBatchJob, *aijobpkg.AIBatchJob](
		ctx,
		repository.collection,
		buildAIBatchJobFilter(filter),
		options.Pagination,
		buildAIBatchJobSort(options.Sort),
		nil,
		func(document *aijobpkg.AIBatchJob) *aijobpkg.AIBatchJob {
			job := *document
			return &job
		},
	)
	if err != nil {
		return nil, fmt.Errorf("list ai batch jobs: %w", mapMongoError(err))
	}
	return result, nil
}

func (repository *AIBatchJobMongoRepository) ListByWorkflowRunID(
	ctx context.Context,
	workflowRunID primitive.ObjectID,
	options platformrepo.AIBatchJobListOptions,
) (*platformrepo.ListResult[*aijobpkg.AIBatchJob], error) {
	return repository.List(ctx, platformrepo.AIBatchJobFilter{WorkflowRunIDs: []primitive.ObjectID{workflowRunID}}, options)
}

func (repository *AIBatchJobMongoRepository) FindPollableJobs(
	ctx context.Context,
	filter platformrepo.AIBatchJobFilter,
	options platformrepo.AIBatchJobListOptions,
) (*platformrepo.ListResult[*aijobpkg.AIBatchJob], error) {
	filter.PollableOnly = true
	return repository.List(ctx, filter, options)
}

func (repository *AIBatchJobMongoRepository) FindSubmittableJobs(
	ctx context.Context,
	filter platformrepo.AIBatchJobFilter,
	options platformrepo.AIBatchJobListOptions,
) (*platformrepo.ListResult[*aijobpkg.AIBatchJob], error) {
	filter.Statuses = []common.AIBatchJobStatus{common.AIBatchJobStatusCreated}
	filter.PollableOnly = false
	filter.RetryableOnly = false
	return repository.List(ctx, filter, options)
}

func (repository *AIBatchJobMongoRepository) UpdateStatus(
	ctx context.Context,
	jobID primitive.ObjectID,
	patch platformrepo.AIBatchJobStatusPatch,
) (*aijobpkg.AIBatchJob, error) {
	current, err := repository.GetByID(ctx, jobID)
	if err != nil {
		return nil, err
	}
	if err := ensureAIBatchJobExpectedStatuses(current, patch.ExpectedCurrentStatuses, "update ai batch job status"); err != nil {
		return nil, err
	}
	if !current.CanTransitionTo(patch.NextStatus) {
		return nil, invalidTransition("update ai batch job status %s cannot transition from %q to %q", jobID.Hex(), current.Status, patch.NextStatus)
	}

	candidate := *current
	transitionAt := mutationTimestamp(patch.Mutation)
	candidate.Status = patch.NextStatus
	candidate.UpdatedAt = transitionAt
	switch patch.NextStatus {
	case common.AIBatchJobStatusSubmitted:
		candidate.SubmittedAt = &transitionAt
	case common.AIBatchJobStatusCompleted:
		candidate.CompletedAt = &transitionAt
	case common.AIBatchJobStatusFailed:
		candidate.FailedAt = &transitionAt
	case common.AIBatchJobStatusTimedOut:
		candidate.FailedAt = &transitionAt
	}
	if patch.ErrorSummary != nil {
		candidate.ErrorSummary = *patch.ErrorSummary
	}

	if err := candidate.Validate(); err != nil {
		return nil, fmt.Errorf("update ai batch job status %s: validate job: %w", jobID.Hex(), err)
	}

	result, err := repository.collection.UpdateOne(
		ctx,
		bson.M{"_id": jobID, "status": current.Status},
		bson.M{"$set": buildAIBatchJobStatusUpdate(&candidate, patch)},
	)
	if err != nil {
		return nil, fmt.Errorf("update ai batch job status %s: %w", jobID.Hex(), mapMongoError(err))
	}
	if result.MatchedCount == 0 {
		return nil, preconditionFailed("update ai batch job status %s stale write rejected", jobID.Hex())
	}
	return &candidate, nil
}

func (repository *AIBatchJobMongoRepository) MarkSubmitted(
	ctx context.Context,
	jobID primitive.ObjectID,
	patch platformrepo.AIBatchJobSubmissionPatch,
) (*aijobpkg.AIBatchJob, error) {
	current, err := repository.GetByID(ctx, jobID)
	if err != nil {
		return nil, err
	}
	if err := ensureAIBatchJobExpectedStatuses(current, patch.ExpectedCurrentStatuses, "mark ai batch job submitted"); err != nil {
		return nil, err
	}
	if !current.CanTransitionTo(patch.NewStatus) {
		return nil, invalidTransition("mark ai batch job submitted %s cannot transition from %q to %q", jobID.Hex(), current.Status, patch.NewStatus)
	}

	candidate := *current
	submittedAt := patch.SubmittedAt.UTC()
	candidate.Status = patch.NewStatus
	candidate.ProviderJobHandle = patch.ProviderJobHandle
	candidate.LocalJobHandle = patch.LocalJobHandle
	candidate.SubmissionPayloadRef = patch.SubmissionPayloadRef
	candidate.SubmittedAt = &submittedAt
	candidate.UpdatedAt = submittedAt
	candidate.ErrorSummary = ""

	if err := candidate.Validate(); err != nil {
		return nil, fmt.Errorf("mark ai batch job submitted %s: validate job: %w", jobID.Hex(), err)
	}

	result, err := repository.collection.UpdateOne(
		ctx,
		bson.M{"_id": jobID, "status": current.Status},
		bson.M{"$set": bson.M{
			"status":               candidate.Status,
			"providerJobHandle":    candidate.ProviderJobHandle,
			"localJobHandle":       candidate.LocalJobHandle,
			"submissionPayloadRef": candidate.SubmissionPayloadRef,
			"submittedAt":          candidate.SubmittedAt.UTC(),
			"updatedAt":            candidate.UpdatedAt,
			"errorSummary":         candidate.ErrorSummary,
		}},
	)
	if err != nil {
		return nil, fmt.Errorf("mark ai batch job submitted %s: %w", jobID.Hex(), mapMongoError(err))
	}
	if result.MatchedCount == 0 {
		return nil, preconditionFailed("mark ai batch job submitted %s stale write rejected", jobID.Hex())
	}
	return &candidate, nil
}

func (repository *AIBatchJobMongoRepository) MarkPolled(
	ctx context.Context,
	jobID primitive.ObjectID,
	patch platformrepo.AIBatchJobPollingPatch,
) (*aijobpkg.AIBatchJob, error) {
	current, err := repository.GetByID(ctx, jobID)
	if err != nil {
		return nil, err
	}
	if err := ensureAIBatchJobExpectedStatuses(current, patch.ExpectedCurrentStatuses, "mark ai batch job polled"); err != nil {
		return nil, err
	}
	if !current.CanPoll() {
		return nil, invalidTransition("mark ai batch job polled %s is not pollable from %q", jobID.Hex(), current.Status)
	}

	candidate := *current
	polledAt := patch.LastPolledAt.UTC()
	candidate.LastPolledAt = &polledAt
	candidate.UpdatedAt = polledAt
	if patch.NextStatus != nil {
		if *patch.NextStatus == common.AIBatchJobStatusCompleted || *patch.NextStatus == common.AIBatchJobStatusFailed || *patch.NextStatus == common.AIBatchJobStatusTimedOut {
			return nil, invalidTransition("mark ai batch job polled %s requires terminal transitions through dedicated methods", jobID.Hex())
		}
		if !current.CanTransitionTo(*patch.NextStatus) {
			return nil, invalidTransition("mark ai batch job polled %s cannot transition from %q to %q", jobID.Hex(), current.Status, *patch.NextStatus)
		}
		candidate.Status = *patch.NextStatus
	}
	if patch.ErrorSummary != nil {
		candidate.ErrorSummary = *patch.ErrorSummary
	}

	if err := candidate.Validate(); err != nil {
		return nil, fmt.Errorf("mark ai batch job polled %s: validate job: %w", jobID.Hex(), err)
	}

	set := bson.M{
		"lastPolledAt": candidate.LastPolledAt.UTC(),
		"updatedAt":    candidate.UpdatedAt,
	}
	if patch.NextStatus != nil {
		set["status"] = candidate.Status
	}
	if patch.ErrorSummary != nil {
		set["errorSummary"] = candidate.ErrorSummary
	}
	result, err := repository.collection.UpdateOne(ctx, bson.M{"_id": jobID, "status": current.Status}, bson.M{"$set": set})
	if err != nil {
		return nil, fmt.Errorf("mark ai batch job polled %s: %w", jobID.Hex(), mapMongoError(err))
	}
	if result.MatchedCount == 0 {
		return nil, preconditionFailed("mark ai batch job polled %s stale write rejected", jobID.Hex())
	}
	return &candidate, nil
}

func (repository *AIBatchJobMongoRepository) MarkCompleted(
	ctx context.Context,
	jobID primitive.ObjectID,
	patch platformrepo.AIBatchJobCompletionPatch,
) (*aijobpkg.AIBatchJob, error) {
	current, err := repository.GetByID(ctx, jobID)
	if err != nil {
		return nil, err
	}
	if err := ensureAIBatchJobExpectedStatuses(current, patch.ExpectedCurrentStatuses, "mark ai batch job completed"); err != nil {
		return nil, err
	}
	if !current.CanTransitionTo(common.AIBatchJobStatusCompleted) {
		return nil, invalidTransition("mark ai batch job completed %s cannot transition from %q", jobID.Hex(), current.Status)
	}

	candidate := *current
	completedAt := patch.CompletedAt.UTC()
	candidate.Status = common.AIBatchJobStatusCompleted
	candidate.CompletedAt = &completedAt
	candidate.UpdatedAt = completedAt
	candidate.ResultPayloadRef = patch.ResultPayloadRef
	candidate.ErrorSummary = ""

	if err := candidate.Validate(); err != nil {
		return nil, fmt.Errorf("mark ai batch job completed %s: validate job: %w", jobID.Hex(), err)
	}

	result, err := repository.collection.UpdateOne(
		ctx,
		bson.M{"_id": jobID, "status": current.Status},
		bson.M{"$set": bson.M{
			"status":           candidate.Status,
			"completedAt":      candidate.CompletedAt.UTC(),
			"updatedAt":        candidate.UpdatedAt,
			"resultPayloadRef": candidate.ResultPayloadRef,
			"errorSummary":     candidate.ErrorSummary,
		}},
	)
	if err != nil {
		return nil, fmt.Errorf("mark ai batch job completed %s: %w", jobID.Hex(), mapMongoError(err))
	}
	if result.MatchedCount == 0 {
		return nil, preconditionFailed("mark ai batch job completed %s stale write rejected", jobID.Hex())
	}
	return &candidate, nil
}

func (repository *AIBatchJobMongoRepository) MarkFailed(
	ctx context.Context,
	jobID primitive.ObjectID,
	patch platformrepo.AIBatchJobFailurePatch,
) (*aijobpkg.AIBatchJob, error) {
	return repository.markTerminalFailure(ctx, jobID, patch, common.AIBatchJobStatusFailed)
}

func (repository *AIBatchJobMongoRepository) MarkTimedOut(
	ctx context.Context,
	jobID primitive.ObjectID,
	patch platformrepo.AIBatchJobFailurePatch,
) (*aijobpkg.AIBatchJob, error) {
	return repository.markTerminalFailure(ctx, jobID, patch, common.AIBatchJobStatusTimedOut)
}

func (repository *AIBatchJobMongoRepository) PrepareRetry(
	ctx context.Context,
	jobID primitive.ObjectID,
	patch platformrepo.AIBatchJobRetryPatch,
) (*aijobpkg.AIBatchJob, error) {
	current, err := repository.GetByID(ctx, jobID)
	if err != nil {
		return nil, err
	}
	if err := ensureAIBatchJobExpectedStatuses(current, patch.ExpectedCurrentStatuses, "prepare ai batch job retry"); err != nil {
		return nil, err
	}
	if !current.CanRetry() {
		return nil, invalidTransition("prepare ai batch job retry %s cannot retry from %q", jobID.Hex(), current.Status)
	}

	candidate := *current
	if err := candidate.PrepareRetry(patch.RetryAt.UTC()); err != nil {
		return nil, fmt.Errorf("prepare ai batch job retry %s: %w", jobID.Hex(), err)
	}
	candidate.ProviderJobHandle = ""
	candidate.LocalJobHandle = ""
	candidate.ResultPayloadRef = nil

	if err := candidate.Validate(); err != nil {
		return nil, fmt.Errorf("prepare ai batch job retry %s: validate job: %w", jobID.Hex(), err)
	}

	update := bson.M{
		"$set": bson.M{
			"status":       candidate.Status,
			"retryCount":   candidate.RetryCount,
			"updatedAt":    candidate.UpdatedAt,
			"errorSummary": candidate.ErrorSummary,
		},
		"$unset": bson.M{
			"providerJobHandle": "",
			"localJobHandle":    "",
			"submittedAt":       "",
			"completedAt":       "",
			"failedAt":          "",
			"lastPolledAt":      "",
			"resultPayloadRef":  "",
		},
	}
	result, err := repository.collection.UpdateOne(
		ctx,
		bson.M{"_id": jobID, "status": current.Status, "retryCount": current.RetryCount},
		update,
	)
	if err != nil {
		return nil, fmt.Errorf("prepare ai batch job retry %s: %w", jobID.Hex(), mapMongoError(err))
	}
	if result.MatchedCount == 0 {
		return nil, preconditionFailed("prepare ai batch job retry %s stale write rejected", jobID.Hex())
	}
	return &candidate, nil
}

func (repository *AIBatchJobMongoRepository) markTerminalFailure(
	ctx context.Context,
	jobID primitive.ObjectID,
	patch platformrepo.AIBatchJobFailurePatch,
	status common.AIBatchJobStatus,
) (*aijobpkg.AIBatchJob, error) {
	current, err := repository.GetByID(ctx, jobID)
	if err != nil {
		return nil, err
	}
	if err := ensureAIBatchJobExpectedStatuses(current, patch.ExpectedCurrentStatuses, "mark ai batch job failure"); err != nil {
		return nil, err
	}
	if !current.CanTransitionTo(status) {
		return nil, invalidTransition("mark ai batch job failure %s cannot transition from %q to %q", jobID.Hex(), current.Status, status)
	}

	candidate := *current
	failedAt := patch.FailedAt.UTC()
	candidate.Status = status
	candidate.FailedAt = &failedAt
	candidate.UpdatedAt = failedAt
	candidate.ErrorSummary = patch.ErrorSummary

	if err := candidate.Validate(); err != nil {
		return nil, fmt.Errorf("mark ai batch job failure %s: validate job: %w", jobID.Hex(), err)
	}

	update := bson.M{"$set": bson.M{
		"status":       candidate.Status,
		"failedAt":     candidate.FailedAt.UTC(),
		"updatedAt":    candidate.UpdatedAt,
		"errorSummary": candidate.ErrorSummary,
	}}
	result, err := repository.collection.UpdateOne(ctx, bson.M{"_id": jobID, "status": current.Status}, update)
	if err != nil {
		return nil, fmt.Errorf("mark ai batch job failure %s: %w", jobID.Hex(), mapMongoError(err))
	}
	if result.MatchedCount == 0 {
		return nil, preconditionFailed("mark ai batch job failure %s stale write rejected", jobID.Hex())
	}
	return &candidate, nil
}

func buildAIBatchJobFilter(filter platformrepo.AIBatchJobFilter) bson.M {
	query := bson.M{}
	addObjectIDFilter(query, "_id", filter.IDs)
	addObjectIDFilter(query, "workflowRunId", filter.WorkflowRunIDs)
	if len(filter.JobTypes) > 0 {
		query["jobType"] = bson.M{"$in": filter.JobTypes}
	}
	if len(filter.BookTypes) > 0 {
		query["bookType"] = bson.M{"$in": filter.BookTypes}
	}
	if len(filter.Statuses) > 0 {
		query["status"] = bson.M{"$in": filter.Statuses}
	}
	if filter.ProviderName != "" {
		query["providerName"] = filter.ProviderName
	}
	if filter.ProviderJobHandle != "" {
		query["providerJobHandle"] = filter.ProviderJobHandle
	}
	if filter.LocalJobHandle != "" {
		query["localJobHandle"] = filter.LocalJobHandle
	}
	if filter.IdempotencyKey != "" {
		query["idempotencyKey"] = filter.IdempotencyKey
	}
	if filter.PollableOnly {
		query["status"] = bson.M{"$in": []common.AIBatchJobStatus{
			common.AIBatchJobStatusSubmitted,
			common.AIBatchJobStatusRunning,
			common.AIBatchJobStatusPartiallyCompleted,
		}}
	}
	if filter.RetryableOnly {
		query["status"] = bson.M{"$in": []common.AIBatchJobStatus{
			common.AIBatchJobStatusFailed,
			common.AIBatchJobStatusTimedOut,
		}}
		query["$expr"] = bson.M{"$lt": bson.A{"$retryCount", "$maxRetryCount"}}
	}
	addTimeRangeFilter(query, "submittedAt", filter.SubmittedAt)
	addTimeRangeFilter(query, "lastPolledAt", filter.LastPolledAt)
	addTimeRangeFilter(query, "completedAt", filter.CompletedAt)
	addTimeRangeFilter(query, "failedAt", filter.FailedAt)
	addTimeRangeFilter(query, "createdAt", filter.CreatedAt)
	addTimeRangeFilter(query, "updatedAt", filter.UpdatedAt)
	return query
}

func buildAIBatchJobSort(option platformrepo.AIBatchJobSortOption) bson.D {
	switch option.By {
	case platformrepo.AIBatchJobSortByLastPolledAt:
		return bson.D{{Key: "lastPolledAt", Value: sortDirection(option.Order, platformrepo.SortOrderDescending)}, {Key: "submittedAt", Value: -1}}
	case platformrepo.AIBatchJobSortByCreatedAt:
		return bson.D{{Key: "createdAt", Value: sortDirection(option.Order, platformrepo.SortOrderDescending)}, {Key: "submittedAt", Value: -1}}
	case platformrepo.AIBatchJobSortByUpdatedAt:
		return bson.D{{Key: "updatedAt", Value: sortDirection(option.Order, platformrepo.SortOrderDescending)}, {Key: "submittedAt", Value: -1}}
	case platformrepo.AIBatchJobSortBySubmittedAt, "":
		return bson.D{{Key: "submittedAt", Value: sortDirection(option.Order, platformrepo.SortOrderDescending)}, {Key: "createdAt", Value: -1}}
	default:
		return bson.D{{Key: "submittedAt", Value: -1}, {Key: "createdAt", Value: -1}}
	}
}

func buildAIBatchJobStatusUpdate(candidate *aijobpkg.AIBatchJob, patch platformrepo.AIBatchJobStatusPatch) bson.M {
	update := bson.M{
		"status":    candidate.Status,
		"updatedAt": candidate.UpdatedAt,
	}
	if candidate.SubmittedAt != nil {
		update["submittedAt"] = candidate.SubmittedAt.UTC()
	}
	if candidate.CompletedAt != nil {
		update["completedAt"] = candidate.CompletedAt.UTC()
	}
	if candidate.FailedAt != nil {
		update["failedAt"] = candidate.FailedAt.UTC()
	}
	if patch.ErrorSummary != nil {
		update["errorSummary"] = candidate.ErrorSummary
	}
	return update
}

func ensureAIBatchJobExpectedStatuses(current *aijobpkg.AIBatchJob, expected []common.AIBatchJobStatus, operation string) error {
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
