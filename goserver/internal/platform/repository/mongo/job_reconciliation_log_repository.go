package mongo

import (
	"context"

	"goserver/internal/platform/domain"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type JobReconciliationLogRepository struct {
	collection *mongo.Collection
}

func NewJobReconciliationLogRepository(collection *mongo.Collection) *JobReconciliationLogRepository {
	return &JobReconciliationLogRepository{collection: collection}
}

func (repository *JobReconciliationLogRepository) Create(ctx context.Context, log *domain.JobReconciliationLog) (*domain.JobReconciliationLog, error) {
	if err := log.Validate(); err != nil {
		return nil, err
	}

	document := toJobReconciliationLogDocument(log)
	document.ObjectID = newDocumentID()
	if _, err := repository.collection.InsertOne(ctx, document); err != nil {
		return nil, err
	}

	return fromJobReconciliationLogDocument(document), nil
}

func (repository *JobReconciliationLogRepository) ListByJobID(ctx context.Context, aiBatchJobID string, limit int) ([]*domain.JobReconciliationLog, error) {
	pageSize := int64(50)
	if limit > 0 {
		pageSize = int64(limit)
	}

	cursor, err := repository.collection.Find(
		ctx,
		bson.M{"aiBatchJobId": aiBatchJobID},
		options.Find().
			SetSort(bson.D{{Key: "polledAt", Value: -1}}).
			SetLimit(pageSize),
	)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var documents []jobReconciliationLogDocument
	if err := cursor.All(ctx, &documents); err != nil {
		return nil, err
	}

	result := make([]*domain.JobReconciliationLog, 0, len(documents))
	for index := range documents {
		result = append(result, fromJobReconciliationLogDocument(&documents[index]))
	}

	return result, nil
}
