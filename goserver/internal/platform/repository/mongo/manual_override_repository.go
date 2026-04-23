package mongo

import (
	"context"

	"goserver/internal/platform/domain"
	"goserver/internal/platform/ports"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type ManualOverrideRepository struct {
	collection *mongo.Collection
}

func NewManualOverrideRepository(collection *mongo.Collection) *ManualOverrideRepository {
	return &ManualOverrideRepository{collection: collection}
}

func (repository *ManualOverrideRepository) Create(ctx context.Context, override *domain.ManualOverride) (*domain.ManualOverride, error) {
	if err := override.Validate(); err != nil {
		return nil, err
	}

	document := toManualOverrideDocument(override)
	document.ObjectID = newDocumentID()
	if _, err := repository.collection.InsertOne(ctx, document); err != nil {
		return nil, err
	}

	return fromManualOverrideDocument(document), nil
}

func (repository *ManualOverrideRepository) GetByID(ctx context.Context, id string) (*domain.ManualOverride, error) {
	objectID, err := parseObjectID(id)
	if err != nil {
		return nil, nil
	}

	var document manualOverrideDocument
	if err := repository.collection.FindOne(ctx, bson.M{"_id": objectID}).Decode(&document); err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, nil
		}
		return nil, err
	}

	return fromManualOverrideDocument(&document), nil
}

func (repository *ManualOverrideRepository) List(ctx context.Context, filter ports.ManualOverrideListFilter) ([]*domain.ManualOverride, error) {
	query := bson.M{}
	if filter.CompanyID != "" {
		query["companyId"] = filter.CompanyID
	}
	if filter.ReviewID != "" {
		query["reviewId"] = filter.ReviewID
	}
	if filter.BookType != "" {
		query["bookType"] = filter.BookType
	}

	documents, err := findAll[manualOverrideDocument](
		ctx,
		repository.collection,
		query,
		options.Find().
			SetSort(bson.D{{Key: "overrideDate", Value: -1}, {Key: "createdAt", Value: -1}}).
			SetLimit(normalizeLimit(filter.Limit, 100)).
			SetSkip(normalizeSkip(filter.Offset)),
	)
	if err != nil {
		return nil, err
	}

	result := make([]*domain.ManualOverride, 0, len(documents))
	for index := range documents {
		result = append(result, fromManualOverrideDocument(&documents[index]))
	}

	return result, nil
}
