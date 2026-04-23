package mongo

import (
	"context"
	"fmt"

	"goserver/internal/domain/common"
	configpkg "goserver/internal/domain/config"
	platformrepo "goserver/internal/platform/repository"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type ConfigSnapshotMongoRepository struct {
	collection *mongo.Collection
}

var _ platformrepo.ConfigSnapshotRepository = (*ConfigSnapshotMongoRepository)(nil)

func NewConfigSnapshotRepository(collection *mongo.Collection) *ConfigSnapshotMongoRepository {
	return &ConfigSnapshotMongoRepository{collection: collection}
}

func (repository *ConfigSnapshotMongoRepository) Create(ctx context.Context, snapshot *configpkg.ConfigSnapshot) (*configpkg.ConfigSnapshot, error) {
	if snapshot == nil {
		return nil, fmt.Errorf("create config snapshot: snapshot is required")
	}

	document := *snapshot
	if document.ID.IsZero() {
		document.ID = newDocumentID()
	}

	if err := document.Validate(); err != nil {
		return nil, fmt.Errorf("create config snapshot: validate snapshot: %w", err)
	}

	if _, err := repository.collection.InsertOne(ctx, &document); err != nil {
		return nil, fmt.Errorf("create config snapshot: %w", mapMongoError(err))
	}

	return &document, nil
}

func (repository *ConfigSnapshotMongoRepository) GetByID(ctx context.Context, id primitive.ObjectID) (*configpkg.ConfigSnapshot, error) {
	document, err := findOne[configpkg.ConfigSnapshot](ctx, repository.collection, bson.M{"_id": id})
	if err != nil {
		return nil, fmt.Errorf("get config snapshot by id %s: %w", id.Hex(), mapMongoError(err))
	}
	return document, nil
}

func (repository *ConfigSnapshotMongoRepository) List(
	ctx context.Context,
	filter platformrepo.ConfigSnapshotFilter,
	options platformrepo.ConfigSnapshotListOptions,
) (*platformrepo.ListResult[*configpkg.ConfigSnapshot], error) {
	result, err := findPage[configpkg.ConfigSnapshot, *configpkg.ConfigSnapshot](
		ctx,
		repository.collection,
		buildConfigSnapshotFilter(filter),
		options.Pagination,
		buildConfigSnapshotSort(options.Sort),
		nil,
		func(document *configpkg.ConfigSnapshot) *configpkg.ConfigSnapshot {
			snapshot := *document
			return &snapshot
		},
	)
	if err != nil {
		return nil, fmt.Errorf("list config snapshots: %w", mapMongoError(err))
	}
	return result, nil
}

func (repository *ConfigSnapshotMongoRepository) GetLatestByBookType(
	ctx context.Context,
	bookType common.BookType,
	lookup platformrepo.LatestConfigSnapshotLookup,
) (*configpkg.ConfigSnapshot, error) {
	filter := bson.M{"bookType": bookType}
	if lookup.Mode != nil {
		filter["mode"] = *lookup.Mode
	}

	document, err := findOne[configpkg.ConfigSnapshot](
		ctx,
		repository.collection,
		filter,
		options.FindOne().SetSort(bson.D{{Key: "createdAt", Value: -1}}),
	)
	if err != nil {
		return nil, fmt.Errorf("get latest config snapshot for book %s: %w", bookType, mapMongoError(err))
	}
	return document, nil
}

func buildConfigSnapshotFilter(filter platformrepo.ConfigSnapshotFilter) bson.M {
	query := bson.M{}
	addObjectIDFilter(query, "_id", filter.IDs)
	if len(filter.BookTypes) > 0 {
		query["bookType"] = bson.M{"$in": filter.BookTypes}
	}
	if len(filter.Modes) > 0 {
		query["mode"] = bson.M{"$in": filter.Modes}
	}
	addTimeRangeFilter(query, "createdAt", filter.CreatedAt)
	return query
}

func buildConfigSnapshotSort(option platformrepo.ConfigSnapshotSortOption) bson.D {
	return bson.D{{Key: "createdAt", Value: sortDirection(option.Order, platformrepo.SortOrderDescending)}}
}
