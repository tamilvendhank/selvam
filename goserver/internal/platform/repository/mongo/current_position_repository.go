package mongo

import (
	"context"
	"fmt"
	"time"

	"goserver/internal/domain/common"
	positionpkg "goserver/internal/domain/position"
	platformrepo "goserver/internal/platform/repository"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type CurrentPositionMongoRepository struct {
	collection *mongo.Collection
}

var _ platformrepo.CurrentPositionRepository = (*CurrentPositionMongoRepository)(nil)

func NewCurrentPositionRepository(collection *mongo.Collection) *CurrentPositionMongoRepository {
	return &CurrentPositionMongoRepository{collection: collection}
}

func NewPositionRepository(collection *mongo.Collection) *CurrentPositionMongoRepository {
	return NewCurrentPositionRepository(collection)
}

func (repository *CurrentPositionMongoRepository) Upsert(ctx context.Context, position *positionpkg.CurrentPosition) (*positionpkg.CurrentPosition, error) {
	if position == nil {
		return nil, fmt.Errorf("upsert current position: position is required")
	}

	document := *position
	if document.ID.IsZero() {
		document.ID = newDocumentID()
	}
	if err := document.Validate(); err != nil {
		return nil, fmt.Errorf("upsert current position: validate position: %w", err)
	}

	filter := bson.M{
		"companyId": document.CompanyID,
		"bookType":  document.BookType,
	}
	update := bson.M{
		"$set": bson.M{
			"companyId":                     document.CompanyID,
			"bookType":                      document.BookType,
			"isOpen":                        document.IsOpen,
			"quantity":                      document.Quantity,
			"averageCost":                   document.AverageCost,
			"currentMarketValue":            document.CurrentMarketValue,
			"currentPositionPctOfBook":      document.CurrentPositionPctOfBook,
			"currentPositionPctOfPortfolio": document.CurrentPositionPctOfPortfolio,
			"lastUpdatedAt":                 document.LastUpdatedAt,
			"schemaVersion":                 document.SchemaVersion,
		},
		"$setOnInsert": bson.M{
			"_id": document.ID,
		},
	}

	if _, err := repository.collection.UpdateOne(ctx, filter, update, options.Update().SetUpsert(true)); err != nil {
		return nil, fmt.Errorf("upsert current position for company %s book %s: %w", document.CompanyID.Hex(), document.BookType, mapMongoError(err))
	}

	return repository.GetByCompanyAndBook(ctx, document.CompanyID, document.BookType)
}

func (repository *CurrentPositionMongoRepository) GetByID(ctx context.Context, id primitive.ObjectID) (*positionpkg.CurrentPosition, error) {
	document, err := findOne[positionpkg.CurrentPosition](ctx, repository.collection, bson.M{"_id": id})
	if err != nil {
		return nil, fmt.Errorf("get current position by id %s: %w", id.Hex(), mapMongoError(err))
	}
	return document, nil
}

func (repository *CurrentPositionMongoRepository) GetByCompanyAndBook(
	ctx context.Context,
	companyID primitive.ObjectID,
	bookType common.BookType,
) (*positionpkg.CurrentPosition, error) {
	document, err := findOne[positionpkg.CurrentPosition](ctx, repository.collection, bson.M{"companyId": companyID, "bookType": bookType})
	if err != nil {
		return nil, fmt.Errorf("get current position by company %s book %s: %w", companyID.Hex(), bookType, mapMongoError(err))
	}
	return document, nil
}

func (repository *CurrentPositionMongoRepository) List(
	ctx context.Context,
	filter platformrepo.CurrentPositionFilter,
	options platformrepo.CurrentPositionListOptions,
) (*platformrepo.ListResult[*positionpkg.CurrentPosition], error) {
	result, err := findPage[positionpkg.CurrentPosition, *positionpkg.CurrentPosition](
		ctx,
		repository.collection,
		buildCurrentPositionFilter(filter),
		options.Pagination,
		buildCurrentPositionSort(options.Sort),
		nil,
		func(document *positionpkg.CurrentPosition) *positionpkg.CurrentPosition {
			position := *document
			return &position
		},
	)
	if err != nil {
		return nil, fmt.Errorf("list current positions: %w", mapMongoError(err))
	}
	return result, nil
}

func (repository *CurrentPositionMongoRepository) ListOpenByBook(
	ctx context.Context,
	bookType common.BookType,
	options platformrepo.CurrentPositionListOptions,
) (*platformrepo.ListResult[*positionpkg.CurrentPosition], error) {
	isOpen := true
	return repository.List(ctx, platformrepo.CurrentPositionFilter{
		BookTypes: []common.BookType{bookType},
		IsOpen:    &isOpen,
	}, options)
}

func (repository *CurrentPositionMongoRepository) UpdateSnapshot(
	ctx context.Context,
	positionID primitive.ObjectID,
	patch platformrepo.CurrentPositionPatch,
) (*positionpkg.CurrentPosition, error) {
	current, err := repository.GetByID(ctx, positionID)
	if err != nil {
		return nil, err
	}
	if patch.ExpectedLastUpdatedAt != nil && !current.LastUpdatedAt.Equal(patch.ExpectedLastUpdatedAt.UTC()) {
		return nil, preconditionFailed("update current position %s expected lastUpdatedAt %s", positionID.Hex(), patch.ExpectedLastUpdatedAt.UTC().Format(time.RFC3339Nano))
	}

	candidate := *current
	if patch.IsOpen != nil {
		candidate.IsOpen = *patch.IsOpen
	}
	if patch.Quantity != nil {
		candidate.Quantity = *patch.Quantity
	}
	if patch.AverageCost != nil {
		candidate.AverageCost = *patch.AverageCost
	}
	if patch.CurrentMarketValue != nil {
		candidate.CurrentMarketValue = *patch.CurrentMarketValue
	}
	if patch.CurrentPositionPctOfBook != nil {
		candidate.CurrentPositionPctOfBook = *patch.CurrentPositionPctOfBook
	}
	if patch.CurrentPositionPctOfPortfolio != nil {
		candidate.CurrentPositionPctOfPortfolio = *patch.CurrentPositionPctOfPortfolio
	}
	candidate.LastUpdatedAt = patch.LastUpdatedAt.UTC()

	if err := candidate.Validate(); err != nil {
		return nil, fmt.Errorf("update current position %s: validate position: %w", positionID.Hex(), err)
	}

	result, err := repository.collection.UpdateOne(
		ctx,
		bson.M{"_id": positionID, "lastUpdatedAt": current.LastUpdatedAt},
		bson.M{"$set": buildCurrentPositionUpdate(&candidate, patch)},
	)
	if err != nil {
		return nil, fmt.Errorf("update current position %s: %w", positionID.Hex(), mapMongoError(err))
	}
	if result.MatchedCount == 0 {
		return nil, preconditionFailed("update current position %s stale write rejected", positionID.Hex())
	}

	return &candidate, nil
}

func (repository *CurrentPositionMongoRepository) ClosePosition(
	ctx context.Context,
	positionID primitive.ObjectID,
	patch platformrepo.CurrentPositionClosePatch,
) (*positionpkg.CurrentPosition, error) {
	current, err := repository.GetByID(ctx, positionID)
	if err != nil {
		return nil, err
	}
	if !current.IsOpen {
		return nil, invalidTransition("close current position %s is already closed", positionID.Hex())
	}
	if patch.ExpectedLastUpdatedAt != nil && !current.LastUpdatedAt.Equal(patch.ExpectedLastUpdatedAt.UTC()) {
		return nil, preconditionFailed("close current position %s expected lastUpdatedAt %s", positionID.Hex(), patch.ExpectedLastUpdatedAt.UTC().Format(time.RFC3339Nano))
	}

	candidate := *current
	candidate.IsOpen = false
	candidate.Quantity = 0
	candidate.CurrentMarketValue = 0
	candidate.CurrentPositionPctOfBook = 0
	candidate.CurrentPositionPctOfPortfolio = 0
	candidate.LastUpdatedAt = patch.ClosedAt.UTC()

	if err := candidate.Validate(); err != nil {
		return nil, fmt.Errorf("close current position %s: validate position: %w", positionID.Hex(), err)
	}

	update := bson.M{"$set": bson.M{
		"isOpen":                        candidate.IsOpen,
		"quantity":                      candidate.Quantity,
		"currentMarketValue":            candidate.CurrentMarketValue,
		"currentPositionPctOfBook":      candidate.CurrentPositionPctOfBook,
		"currentPositionPctOfPortfolio": candidate.CurrentPositionPctOfPortfolio,
		"lastUpdatedAt":                 candidate.LastUpdatedAt,
		"closedAt":                      patch.ClosedAt.UTC(),
		"closeReason":                   patch.Reason,
		"closedBy":                      patch.ClosedBy,
	}}
	result, err := repository.collection.UpdateOne(
		ctx,
		bson.M{"_id": positionID, "isOpen": true, "lastUpdatedAt": current.LastUpdatedAt},
		update,
	)
	if err != nil {
		return nil, fmt.Errorf("close current position %s: %w", positionID.Hex(), mapMongoError(err))
	}
	if result.MatchedCount == 0 {
		return nil, preconditionFailed("close current position %s stale write rejected", positionID.Hex())
	}

	return &candidate, nil
}

func buildCurrentPositionFilter(filter platformrepo.CurrentPositionFilter) bson.M {
	query := bson.M{}
	addObjectIDFilter(query, "_id", filter.IDs)
	addObjectIDFilter(query, "companyId", filter.CompanyIDs)
	if len(filter.BookTypes) > 0 {
		query["bookType"] = bson.M{"$in": filter.BookTypes}
	}
	addBoolFilter(query, "isOpen", filter.IsOpen)
	addTimeRangeFilter(query, "lastUpdatedAt", filter.LastUpdatedAt)
	return query
}

func buildCurrentPositionSort(option platformrepo.CurrentPositionSortOption) bson.D {
	switch option.By {
	case platformrepo.CurrentPositionSortByCurrentPositionPctOfBook:
		return bson.D{{Key: "currentPositionPctOfBook", Value: sortDirection(option.Order, platformrepo.SortOrderDescending)}, {Key: "lastUpdatedAt", Value: -1}}
	case platformrepo.CurrentPositionSortByLastUpdatedAt:
		return bson.D{{Key: "lastUpdatedAt", Value: sortDirection(option.Order, platformrepo.SortOrderDescending)}, {Key: "currentMarketValue", Value: -1}}
	case platformrepo.CurrentPositionSortByCurrentMarketValue, "":
		return bson.D{{Key: "currentMarketValue", Value: sortDirection(option.Order, platformrepo.SortOrderDescending)}, {Key: "lastUpdatedAt", Value: -1}}
	default:
		return bson.D{{Key: "currentMarketValue", Value: -1}, {Key: "lastUpdatedAt", Value: -1}}
	}
}

func buildCurrentPositionUpdate(candidate *positionpkg.CurrentPosition, patch platformrepo.CurrentPositionPatch) bson.M {
	update := bson.M{
		"lastUpdatedAt": candidate.LastUpdatedAt,
	}
	if patch.IsOpen != nil {
		update["isOpen"] = candidate.IsOpen
	}
	if patch.Quantity != nil {
		update["quantity"] = candidate.Quantity
	}
	if patch.AverageCost != nil {
		update["averageCost"] = candidate.AverageCost
	}
	if patch.CurrentMarketValue != nil {
		update["currentMarketValue"] = candidate.CurrentMarketValue
	}
	if patch.CurrentPositionPctOfBook != nil {
		update["currentPositionPctOfBook"] = candidate.CurrentPositionPctOfBook
	}
	if patch.CurrentPositionPctOfPortfolio != nil {
		update["currentPositionPctOfPortfolio"] = candidate.CurrentPositionPctOfPortfolio
	}
	return update
}
