package mongo

import (
	"context"
	"fmt"

	overridepkg "goserver/internal/domain/override"
	platformrepo "goserver/internal/platform/repository"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type ManualOverrideMongoRepository struct {
	collection *mongo.Collection
}

var _ platformrepo.ManualOverrideRepository = (*ManualOverrideMongoRepository)(nil)

func NewManualOverrideRepository(collection *mongo.Collection) *ManualOverrideMongoRepository {
	return &ManualOverrideMongoRepository{collection: collection}
}

func (repository *ManualOverrideMongoRepository) Create(ctx context.Context, override *overridepkg.ManualOverride) (*overridepkg.ManualOverride, error) {
	if override == nil {
		return nil, fmt.Errorf("create manual override: override is required")
	}

	document := *override
	if document.ID.IsZero() {
		document.ID = newDocumentID()
	}

	if err := document.Validate(); err != nil {
		return nil, fmt.Errorf("create manual override: validate override: %w", err)
	}

	if _, err := repository.collection.InsertOne(ctx, &document); err != nil {
		return nil, fmt.Errorf("create manual override: %w", mapMongoError(err))
	}

	return &document, nil
}

func (repository *ManualOverrideMongoRepository) GetByID(ctx context.Context, id primitive.ObjectID) (*overridepkg.ManualOverride, error) {
	document, err := findOne[overridepkg.ManualOverride](ctx, repository.collection, bson.M{"_id": id})
	if err != nil {
		return nil, fmt.Errorf("get manual override by id %s: %w", id.Hex(), mapMongoError(err))
	}
	return document, nil
}

func (repository *ManualOverrideMongoRepository) List(
	ctx context.Context,
	filter platformrepo.ManualOverrideFilter,
	options platformrepo.ManualOverrideListOptions,
) (*platformrepo.ListResult[*overridepkg.ManualOverride], error) {
	result, err := findPage[overridepkg.ManualOverride, *overridepkg.ManualOverride](
		ctx,
		repository.collection,
		buildManualOverrideFilter(filter),
		options.Pagination,
		buildManualOverrideSort(options.Sort),
		nil,
		func(document *overridepkg.ManualOverride) *overridepkg.ManualOverride {
			override := *document
			return &override
		},
	)
	if err != nil {
		return nil, fmt.Errorf("list manual overrides: %w", mapMongoError(err))
	}
	return result, nil
}

func (repository *ManualOverrideMongoRepository) GetLatestByReviewID(ctx context.Context, reviewID primitive.ObjectID) (*overridepkg.ManualOverride, error) {
	document, err := findOne[overridepkg.ManualOverride](
		ctx,
		repository.collection,
		bson.M{"reviewId": reviewID},
		options.FindOne().SetSort(bson.D{{Key: "overrideDate", Value: -1}, {Key: "createdAt", Value: -1}}),
	)
	if err != nil {
		return nil, fmt.Errorf("get latest manual override by review %s: %w", reviewID.Hex(), mapMongoError(err))
	}
	return document, nil
}

func buildManualOverrideFilter(filter platformrepo.ManualOverrideFilter) bson.M {
	query := bson.M{}
	addObjectIDFilter(query, "_id", filter.IDs)
	addObjectIDFilter(query, "companyId", filter.CompanyIDs)
	addObjectIDFilter(query, "reviewId", filter.ReviewIDs)
	if len(filter.BookTypes) > 0 {
		query["bookType"] = bson.M{"$in": filter.BookTypes}
	}
	if len(filter.OriginalActions) > 0 {
		query["originalAction"] = bson.M{"$in": filter.OriginalActions}
	}
	if len(filter.OverriddenActions) > 0 {
		query["overriddenAction"] = bson.M{"$in": filter.OverriddenActions}
	}
	if filter.OverrideBy != "" {
		query["overrideBy"] = filter.OverrideBy
	}
	addTimeRangeFilter(query, "overrideDate", filter.OverrideDate)
	addTimeRangeFilter(query, "createdAt", filter.CreatedAt)
	return query
}

func buildManualOverrideSort(option platformrepo.ManualOverrideSortOption) bson.D {
	switch option.By {
	case platformrepo.ManualOverrideSortByCreatedAt:
		return bson.D{{Key: "createdAt", Value: sortDirection(option.Order, platformrepo.SortOrderDescending)}, {Key: "overrideDate", Value: -1}}
	case platformrepo.ManualOverrideSortByOverrideDate, "":
		return bson.D{{Key: "overrideDate", Value: sortDirection(option.Order, platformrepo.SortOrderDescending)}, {Key: "createdAt", Value: -1}}
	default:
		return bson.D{{Key: "overrideDate", Value: -1}, {Key: "createdAt", Value: -1}}
	}
}
