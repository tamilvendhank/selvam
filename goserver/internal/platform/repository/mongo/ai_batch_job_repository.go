package mongo

import (
	"context"

	"goserver/internal/platform/domain"
	"goserver/internal/platform/ports"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type AIBatchJobRepository struct {
	collection *mongo.Collection
}

func NewAIBatchJobRepository(collection *mongo.Collection) *AIBatchJobRepository {
	return &AIBatchJobRepository{collection: collection}
}

func (repository *AIBatchJobRepository) Create(ctx context.Context, job *domain.AIBatchJob) (*domain.AIBatchJob, error) {
	if err := job.Validate(); err != nil {
		return nil, err
	}

	document := toAIBatchJobDocument(job)
	document.ObjectID = newDocumentID()
	if _, err := repository.collection.InsertOne(ctx, document); err != nil {
		return nil, err
	}

	return fromAIBatchJobDocument(document), nil
}

func (repository *AIBatchJobRepository) Update(ctx context.Context, job *domain.AIBatchJob) (*domain.AIBatchJob, error) {
	if err := job.Validate(); err != nil {
		return nil, err
	}

	objectID, err := parseObjectID(job.ID)
	if err != nil {
		return nil, err
	}

	document := toAIBatchJobDocument(job)
	document.ObjectID = objectID
	if _, err := repository.collection.ReplaceOne(ctx, bson.M{"_id": objectID}, document); err != nil {
		return nil, err
	}

	return repository.GetByID(ctx, job.ID)
}

func (repository *AIBatchJobRepository) GetByID(ctx context.Context, id string) (*domain.AIBatchJob, error) {
	objectID, err := parseObjectID(id)
	if err != nil {
		return nil, nil
	}

	var document aiBatchJobDocument
	if err := repository.collection.FindOne(ctx, bson.M{"_id": objectID}).Decode(&document); err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, nil
		}
		return nil, err
	}

	return fromAIBatchJobDocument(&document), nil
}

func (repository *AIBatchJobRepository) GetByIdempotencyKey(ctx context.Context, idempotencyKey string) (*domain.AIBatchJob, error) {
	if idempotencyKey == "" {
		return nil, nil
	}

	var document aiBatchJobDocument
	if err := repository.collection.FindOne(
		ctx,
		bson.M{"idempotencyKey": idempotencyKey},
		options.FindOne().SetSort(bson.D{{Key: "createdAt", Value: -1}}),
	).Decode(&document); err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, nil
		}
		return nil, err
	}

	return fromAIBatchJobDocument(&document), nil
}

func (repository *AIBatchJobRepository) List(ctx context.Context, filter ports.AIBatchJobListFilter) ([]*domain.AIBatchJob, error) {
	query := bson.M{}
	if filter.WorkflowRunID != "" {
		query["workflowRunId"] = filter.WorkflowRunID
	}
	if filter.BookType != "" {
		query["bookType"] = filter.BookType
	}
	if filter.Status != "" {
		query["status"] = filter.Status
	}

	documents, err := findAll[aiBatchJobDocument](
		ctx,
		repository.collection,
		query,
		options.Find().
			SetSort(bson.D{{Key: "submittedAt", Value: -1}, {Key: "createdAt", Value: -1}}).
			SetLimit(normalizeLimit(filter.Limit, 100)).
			SetSkip(normalizeSkip(filter.Offset)),
	)
	if err != nil {
		return nil, err
	}

	result := make([]*domain.AIBatchJob, 0, len(documents))
	for index := range documents {
		result = append(result, fromAIBatchJobDocument(&documents[index]))
	}

	return result, nil
}
