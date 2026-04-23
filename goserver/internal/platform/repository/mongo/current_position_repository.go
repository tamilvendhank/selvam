package mongo

import (
	"context"
	"time"

	"goserver/internal/platform/domain"
	"goserver/internal/platform/ports"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type PositionRepository struct {
	collection *mongo.Collection
}

func NewPositionRepository(collection *mongo.Collection) *PositionRepository {
	return &PositionRepository{collection: collection}
}

func (repository *PositionRepository) Upsert(ctx context.Context, position *domain.CurrentPosition) (*domain.CurrentPosition, error) {
	if err := position.Validate(); err != nil {
		return nil, err
	}

	now := time.Now().UTC()
	position.UpdatedAt = now
	if position.CreatedAt.IsZero() {
		position.CreatedAt = now
	}

	filter := bson.M{
		"companyId": position.CompanyID,
		"bookType":  position.BookType,
	}
	update := bson.M{"$set": mergeUpdatedAt(bson.M{
		"symbol":                      position.Symbol,
		"quantity":                    position.Quantity,
		"averageCost":                 position.AverageCost,
		"marketPrice":                 position.MarketPrice,
		"marketValue":                 position.MarketValue,
		"positionPctOfBook":           position.PositionPctOfBook,
		"positionPctOfTotalPortfolio": position.PositionPctOfTotalPortfolio,
		"targetPositionPct":           position.TargetPositionPct,
		"maxPositionPct":              position.MaxPositionPct,
		"lastReviewId":                position.LastReviewID,
		"ownedSinceDate":              position.OwnedSinceDate,
		"schemaVersion":               position.SchemaVersion,
		"asOf":                        position.AsOf,
	})}
	update["$setOnInsert"] = bson.M{"createdAt": position.CreatedAt}

	if _, err := repository.collection.UpdateOne(ctx, filter, update, options.Update().SetUpsert(true)); err != nil {
		return nil, err
	}

	if position.ID != "" {
		return repository.GetByID(ctx, position.ID)
	}

	return repository.GetByCompanyAndBook(ctx, position.CompanyID, position.BookType)
}

func (repository *PositionRepository) GetByID(ctx context.Context, id string) (*domain.CurrentPosition, error) {
	objectID, err := parseObjectID(id)
	if err != nil {
		return nil, nil
	}

	var document currentPositionDocument
	if err := repository.collection.FindOne(ctx, bson.M{"_id": objectID}).Decode(&document); err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, nil
		}
		return nil, err
	}

	return fromCurrentPositionDocument(&document), nil
}

func (repository *PositionRepository) GetByCompanyAndBook(ctx context.Context, companyID string, bookType domain.BookType) (*domain.CurrentPosition, error) {
	var document currentPositionDocument
	if err := repository.collection.FindOne(ctx, bson.M{"companyId": companyID, "bookType": bookType}).Decode(&document); err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, nil
		}
		return nil, err
	}

	return fromCurrentPositionDocument(&document), nil
}

func (repository *PositionRepository) List(ctx context.Context, filter ports.PositionListFilter) ([]*domain.CurrentPosition, error) {
	query := bson.M{}
	if filter.BookType != "" {
		query["bookType"] = filter.BookType
	}

	documents, err := findAll[currentPositionDocument](
		ctx,
		repository.collection,
		query,
		options.Find().
			SetSort(bson.D{{Key: "marketValue", Value: -1}, {Key: "updatedAt", Value: -1}}).
			SetLimit(normalizeLimit(filter.Limit, 100)).
			SetSkip(normalizeSkip(filter.Offset)),
	)
	if err != nil {
		return nil, err
	}

	result := make([]*domain.CurrentPosition, 0, len(documents))
	for index := range documents {
		result = append(result, fromCurrentPositionDocument(&documents[index]))
	}

	return result, nil
}
