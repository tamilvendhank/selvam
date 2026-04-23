package mongo

import (
	"context"

	"goserver/internal/platform/domain"
	"goserver/internal/platform/ports"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type AIBatchItemRepository struct {
	collection *mongo.Collection
}

func NewAIBatchItemRepository(collection *mongo.Collection) *AIBatchItemRepository {
	return &AIBatchItemRepository{collection: collection}
}

func (repository *AIBatchItemRepository) Create(ctx context.Context, item *domain.AIBatchItem) (*domain.AIBatchItem, error) {
	if err := item.Validate(); err != nil {
		return nil, err
	}

	document := toAIBatchItemDocument(item)
	document.ObjectID = newDocumentID()
	if _, err := repository.collection.InsertOne(ctx, document); err != nil {
		return nil, err
	}

	return fromAIBatchItemDocument(document), nil
}

func (repository *AIBatchItemRepository) CreateMany(ctx context.Context, items []*domain.AIBatchItem) ([]*domain.AIBatchItem, error) {
	if len(items) == 0 {
		return []*domain.AIBatchItem{}, nil
	}

	documents := make([]any, 0, len(items))
	for _, item := range items {
		if err := item.Validate(); err != nil {
			return nil, err
		}
		document := toAIBatchItemDocument(item)
		document.ObjectID = newDocumentID()
		documents = append(documents, document)
	}
	if _, err := repository.collection.InsertMany(ctx, documents); err != nil {
		return nil, err
	}

	result := make([]*domain.AIBatchItem, 0, len(documents))
	for _, document := range documents {
		result = append(result, fromAIBatchItemDocument(document.(*aiBatchItemDocument)))
	}

	return result, nil
}

func (repository *AIBatchItemRepository) Update(ctx context.Context, item *domain.AIBatchItem) (*domain.AIBatchItem, error) {
	if err := item.Validate(); err != nil {
		return nil, err
	}

	objectID, err := parseObjectID(item.ID)
	if err != nil {
		return nil, err
	}

	document := toAIBatchItemDocument(item)
	document.ObjectID = objectID
	if _, err := repository.collection.ReplaceOne(ctx, bson.M{"_id": objectID}, document); err != nil {
		return nil, err
	}

	return repository.GetByID(ctx, item.ID)
}

func (repository *AIBatchItemRepository) GetByID(ctx context.Context, id string) (*domain.AIBatchItem, error) {
	objectID, err := parseObjectID(id)
	if err != nil {
		return nil, nil
	}

	var document aiBatchItemDocument
	if err := repository.collection.FindOne(ctx, bson.M{"_id": objectID}).Decode(&document); err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, nil
		}
		return nil, err
	}

	return fromAIBatchItemDocument(&document), nil
}

func (repository *AIBatchItemRepository) List(ctx context.Context, filter ports.AIBatchItemListFilter) ([]*domain.AIBatchItem, error) {
	query := bson.M{}
	if filter.AIBatchJobID != "" {
		query["aiBatchJobId"] = filter.AIBatchJobID
	}
	if filter.WorkflowRunID != "" {
		query["workflowRunId"] = filter.WorkflowRunID
	}
	if filter.CompanyID != "" {
		query["companyId"] = filter.CompanyID
	}
	if filter.Status != "" {
		query["status"] = filter.Status
	}
	if filter.ItemType != "" {
		query["itemType"] = filter.ItemType
	}

	documents, err := findAll[aiBatchItemDocument](
		ctx,
		repository.collection,
		query,
		options.Find().
			SetSort(bson.D{{Key: "createdAt", Value: -1}}).
			SetLimit(normalizeLimit(filter.Limit, 200)).
			SetSkip(normalizeSkip(filter.Offset)),
	)
	if err != nil {
		return nil, err
	}

	result := make([]*domain.AIBatchItem, 0, len(documents))
	for index := range documents {
		result = append(result, fromAIBatchItemDocument(&documents[index]))
	}

	return result, nil
}
