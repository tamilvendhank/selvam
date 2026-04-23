package mongo

import (
	"context"

	"goserver/internal/platform/domain"
	"goserver/internal/platform/ports"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type ConfigSnapshotRepository struct {
	collection *mongo.Collection
}

func NewConfigSnapshotRepository(collection *mongo.Collection) *ConfigSnapshotRepository {
	return &ConfigSnapshotRepository{collection: collection}
}

func (repository *ConfigSnapshotRepository) Create(ctx context.Context, snapshot *domain.ConfigSnapshot) (*domain.ConfigSnapshot, error) {
	if err := snapshot.Validate(); err != nil {
		return nil, err
	}

	document := toConfigSnapshotDocument(snapshot)
	document.ObjectID = newDocumentID()
	if _, err := repository.collection.InsertOne(ctx, document); err != nil {
		return nil, err
	}

	return fromConfigSnapshotDocument(document), nil
}

func (repository *ConfigSnapshotRepository) GetByID(ctx context.Context, id string) (*domain.ConfigSnapshot, error) {
	objectID, err := parseObjectID(id)
	if err != nil {
		return nil, nil
	}

	var document configSnapshotDocument
	if err := repository.collection.FindOne(ctx, bson.M{"_id": objectID}).Decode(&document); err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, nil
		}
		return nil, err
	}

	return fromConfigSnapshotDocument(&document), nil
}

func (repository *ConfigSnapshotRepository) List(ctx context.Context, filter ports.ConfigSnapshotListFilter) ([]*domain.ConfigSnapshot, error) {
	query := bson.M{}
	if filter.BookType != "" {
		query["bookType"] = filter.BookType
	}

	documents, err := findAll[configSnapshotDocument](
		ctx,
		repository.collection,
		query,
		options.Find().
			SetSort(bson.D{{Key: "createdAt", Value: -1}}).
			SetLimit(normalizeLimit(filter.Limit, 100)).
			SetSkip(normalizeSkip(filter.Offset)),
	)
	if err != nil {
		return nil, err
	}

	result := make([]*domain.ConfigSnapshot, 0, len(documents))
	for index := range documents {
		result = append(result, fromConfigSnapshotDocument(&documents[index]))
	}

	return result, nil
}
