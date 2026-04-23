package mongo

import (
	"context"

	"goserver/internal/platform/domain"
	"goserver/internal/platform/ports"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type CapitalAllocationRepository struct {
	collection *mongo.Collection
}

func NewCapitalAllocationRepository(collection *mongo.Collection) *CapitalAllocationRepository {
	return &CapitalAllocationRepository{collection: collection}
}

func (repository *CapitalAllocationRepository) Create(ctx context.Context, run *domain.CapitalAllocationRun) (*domain.CapitalAllocationRun, error) {
	if err := run.Validate(); err != nil {
		return nil, err
	}

	document := toCapitalAllocationRunDocument(run)
	document.ObjectID = newDocumentID()
	if _, err := repository.collection.InsertOne(ctx, document); err != nil {
		return nil, err
	}

	return fromCapitalAllocationRunDocument(document), nil
}

func (repository *CapitalAllocationRepository) GetByID(ctx context.Context, id string) (*domain.CapitalAllocationRun, error) {
	objectID, err := parseObjectID(id)
	if err != nil {
		return nil, nil
	}

	var document capitalAllocationRunDocument
	if err := repository.collection.FindOne(ctx, bson.M{"_id": objectID}).Decode(&document); err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, nil
		}
		return nil, err
	}

	return fromCapitalAllocationRunDocument(&document), nil
}

func (repository *CapitalAllocationRepository) List(ctx context.Context, filter ports.CapitalAllocationListFilter) ([]*domain.CapitalAllocationRun, error) {
	query := bson.M{}
	if filter.BookType != "" {
		query["bookType"] = filter.BookType
	}

	documents, err := findAll[capitalAllocationRunDocument](
		ctx,
		repository.collection,
		query,
		options.Find().
			SetSort(bson.D{{Key: "allocationDate", Value: -1}, {Key: "createdAt", Value: -1}}).
			SetLimit(normalizeLimit(filter.Limit, 100)).
			SetSkip(normalizeSkip(filter.Offset)),
	)
	if err != nil {
		return nil, err
	}

	result := make([]*domain.CapitalAllocationRun, 0, len(documents))
	for index := range documents {
		result = append(result, fromCapitalAllocationRunDocument(&documents[index]))
	}

	return result, nil
}
