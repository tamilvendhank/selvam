package mongo

import (
	"context"
	"errors"
	"fmt"

	"goserver/internal/domain/common"
	thesispkg "goserver/internal/domain/thesis"
	platformrepo "goserver/internal/platform/repository"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type InvestmentThesisMongoRepository struct {
	collection *mongo.Collection
}

var _ platformrepo.InvestmentThesisRepository = (*InvestmentThesisMongoRepository)(nil)

func NewInvestmentThesisRepository(collection *mongo.Collection) *InvestmentThesisMongoRepository {
	return &InvestmentThesisMongoRepository{collection: collection}
}

func NewThesisRepository(collection *mongo.Collection) *InvestmentThesisMongoRepository {
	return NewInvestmentThesisRepository(collection)
}

func (repository *InvestmentThesisMongoRepository) Create(ctx context.Context, thesis *thesispkg.InvestmentThesis) (*thesispkg.InvestmentThesis, error) {
	if thesis == nil {
		return nil, fmt.Errorf("create thesis: thesis is required")
	}

	document := *thesis
	if document.ID.IsZero() {
		document.ID = newDocumentID()
	}

	if err := document.Validate(); err != nil {
		return nil, fmt.Errorf("create thesis: validate thesis: %w", err)
	}

	if _, err := repository.collection.InsertOne(ctx, &document); err != nil {
		return nil, fmt.Errorf("create thesis for company %s: %w", document.CompanyID.Hex(), mapMongoError(err))
	}

	return &document, nil
}

func (repository *InvestmentThesisMongoRepository) SaveNewVersion(
	ctx context.Context,
	thesis *thesispkg.InvestmentThesis,
	options platformrepo.ThesisVersionCreateOptions,
) (*thesispkg.InvestmentThesis, error) {
	if thesis == nil {
		return nil, fmt.Errorf("save thesis version: thesis is required")
	}

	latest, err := repository.GetLatestByCompanyID(ctx, thesis.CompanyID)
	if err != nil && !errors.Is(err, platformrepo.ErrNotFound) {
		return nil, err
	}

	if options.ExpectedPreviousVersion != nil {
		switch {
		case latest == nil && *options.ExpectedPreviousVersion != 0:
			return nil, preconditionFailed("save thesis version for company %s expected previous version %d", thesis.CompanyID.Hex(), *options.ExpectedPreviousVersion)
		case latest != nil && latest.ThesisVersion != *options.ExpectedPreviousVersion:
			return nil, preconditionFailed("save thesis version for company %s expected previous version %d, got %d", thesis.CompanyID.Hex(), *options.ExpectedPreviousVersion, latest.ThesisVersion)
		}
	}
	if latest != nil && thesis.ThesisVersion <= latest.ThesisVersion {
		return nil, preconditionFailed("save thesis version for company %s requires version greater than %d", thesis.CompanyID.Hex(), latest.ThesisVersion)
	}

	return repository.Create(ctx, thesis)
}

func (repository *InvestmentThesisMongoRepository) GetByID(ctx context.Context, id primitive.ObjectID) (*thesispkg.InvestmentThesis, error) {
	document, err := findOne[thesispkg.InvestmentThesis](ctx, repository.collection, bson.M{"_id": id})
	if err != nil {
		return nil, fmt.Errorf("get thesis by id %s: %w", id.Hex(), mapMongoError(err))
	}
	return document, nil
}

func (repository *InvestmentThesisMongoRepository) GetActiveByCompanyID(ctx context.Context, companyID primitive.ObjectID) (*thesispkg.InvestmentThesis, error) {
	document, err := findOne[thesispkg.InvestmentThesis](
		ctx,
		repository.collection,
		bson.M{"companyId": companyID, "thesisStatus": common.ThesisStatusActive},
		options.FindOne().SetSort(bson.D{{Key: "thesisVersion", Value: -1}, {Key: "updatedAt", Value: -1}, {Key: "createdAt", Value: -1}}),
	)
	if err != nil {
		return nil, fmt.Errorf("get active thesis by company %s: %w", companyID.Hex(), mapMongoError(err))
	}
	return document, nil
}

func (repository *InvestmentThesisMongoRepository) GetLatestByCompanyID(ctx context.Context, companyID primitive.ObjectID) (*thesispkg.InvestmentThesis, error) {
	document, err := findOne[thesispkg.InvestmentThesis](
		ctx,
		repository.collection,
		bson.M{"companyId": companyID},
		options.FindOne().SetSort(bson.D{{Key: "thesisVersion", Value: -1}, {Key: "updatedAt", Value: -1}, {Key: "createdAt", Value: -1}}),
	)
	if err != nil {
		return nil, fmt.Errorf("get latest thesis by company %s: %w", companyID.Hex(), mapMongoError(err))
	}
	return document, nil
}

func (repository *InvestmentThesisMongoRepository) ListByCompanyID(
	ctx context.Context,
	companyID primitive.ObjectID,
	options platformrepo.InvestmentThesisListOptions,
) (*platformrepo.ListResult[*thesispkg.InvestmentThesis], error) {
	return repository.List(ctx, platformrepo.InvestmentThesisFilter{CompanyIDs: []primitive.ObjectID{companyID}}, options)
}

func (repository *InvestmentThesisMongoRepository) List(
	ctx context.Context,
	filter platformrepo.InvestmentThesisFilter,
	options platformrepo.InvestmentThesisListOptions,
) (*platformrepo.ListResult[*thesispkg.InvestmentThesis], error) {
	result, err := findPage[thesispkg.InvestmentThesis, *thesispkg.InvestmentThesis](
		ctx,
		repository.collection,
		buildInvestmentThesisFilter(filter),
		options.Pagination,
		buildInvestmentThesisSort(options.Sort),
		nil,
		func(document *thesispkg.InvestmentThesis) *thesispkg.InvestmentThesis {
			thesis := *document
			return &thesis
		},
	)
	if err != nil {
		return nil, fmt.Errorf("list theses: %w", mapMongoError(err))
	}
	return result, nil
}

func (repository *InvestmentThesisMongoRepository) UpdateStatus(
	ctx context.Context,
	thesisID primitive.ObjectID,
	patch platformrepo.ThesisStatusPatch,
) (*thesispkg.InvestmentThesis, error) {
	current, err := repository.GetByID(ctx, thesisID)
	if err != nil {
		return nil, err
	}

	if len(patch.ExpectedCurrentStatuses) > 0 && !containsThesisStatus(patch.ExpectedCurrentStatuses, current.ThesisStatus) {
		return nil, preconditionFailed("update thesis status %s expected current status %q", thesisID.Hex(), current.ThesisStatus)
	}
	if !current.CanTransitionTo(patch.NextStatus) {
		return nil, invalidTransition("update thesis status %s cannot transition from %q to %q", thesisID.Hex(), current.ThesisStatus, patch.NextStatus)
	}

	candidate := *current
	candidate.ThesisStatus = patch.NextStatus
	candidate.UpdatedAt = mutationTimestamp(patch.Mutation)
	if patch.LastUpdatedFromReviewID != nil {
		candidate.LastUpdatedFromReviewID = *patch.LastUpdatedFromReviewID
	}
	if patch.ThesisChangeSummary != nil {
		candidate.ThesisChangeSummary = *patch.ThesisChangeSummary
	}

	if err := candidate.Validate(); err != nil {
		return nil, fmt.Errorf("update thesis status %s: validate thesis: %w", thesisID.Hex(), err)
	}

	result, err := repository.collection.UpdateOne(
		ctx,
		bson.M{"_id": thesisID, "thesisStatus": current.ThesisStatus},
		bson.M{"$set": buildThesisStatusUpdate(&candidate, patch)},
	)
	if err != nil {
		return nil, fmt.Errorf("update thesis status %s: %w", thesisID.Hex(), mapMongoError(err))
	}
	if result.MatchedCount == 0 {
		return nil, preconditionFailed("update thesis status %s stale write rejected", thesisID.Hex())
	}

	return &candidate, nil
}

func buildInvestmentThesisFilter(filter platformrepo.InvestmentThesisFilter) bson.M {
	query := bson.M{}

	addObjectIDFilter(query, "_id", filter.IDs)
	addObjectIDFilter(query, "companyId", filter.CompanyIDs)
	addObjectIDFilter(query, "createdFromReviewId", filter.CreatedFromReviewIDs)
	addObjectIDFilter(query, "lastUpdatedFromReviewId", filter.LastUpdatedFromReviewIDs)

	if len(filter.ThesisStatuses) > 0 {
		statuses := make([]common.ThesisStatus, 0, len(filter.ThesisStatuses))
		statuses = append(statuses, filter.ThesisStatuses...)
		query["thesisStatus"] = bson.M{"$in": statuses}
	}
	if len(filter.CurrentPositionRoles) > 0 {
		query["currentPositionRole"] = bson.M{"$in": filter.CurrentPositionRoles}
	}
	if filter.MinVersion != nil || filter.MaxVersion != nil {
		versionRange := bson.M{}
		if filter.MinVersion != nil {
			versionRange["$gte"] = *filter.MinVersion
		}
		if filter.MaxVersion != nil {
			versionRange["$lte"] = *filter.MaxVersion
		}
		query["thesisVersion"] = versionRange
	}
	if filter.ActiveOnly {
		query["thesisStatus"] = common.ThesisStatusActive
	}

	addTimeRangeFilter(query, "createdAt", filter.CreatedAt)
	addTimeRangeFilter(query, "updatedAt", filter.UpdatedAt)

	return query
}

func buildInvestmentThesisSort(option platformrepo.InvestmentThesisSortOption) bson.D {
	switch option.By {
	case platformrepo.InvestmentThesisSortByCreatedAt:
		return bson.D{{Key: "createdAt", Value: sortDirection(option.Order, platformrepo.SortOrderDescending)}, {Key: "thesisVersion", Value: -1}}
	case platformrepo.InvestmentThesisSortByUpdatedAt:
		return bson.D{{Key: "updatedAt", Value: sortDirection(option.Order, platformrepo.SortOrderDescending)}, {Key: "thesisVersion", Value: -1}}
	case platformrepo.InvestmentThesisSortByVersion, "":
		return bson.D{{Key: "thesisVersion", Value: sortDirection(option.Order, platformrepo.SortOrderDescending)}, {Key: "updatedAt", Value: -1}}
	default:
		return bson.D{{Key: "thesisVersion", Value: -1}, {Key: "updatedAt", Value: -1}}
	}
}

func buildThesisStatusUpdate(candidate *thesispkg.InvestmentThesis, patch platformrepo.ThesisStatusPatch) bson.M {
	update := bson.M{
		"thesisStatus": candidate.ThesisStatus,
		"updatedAt":    candidate.UpdatedAt,
	}
	if patch.LastUpdatedFromReviewID != nil {
		update["lastUpdatedFromReviewId"] = candidate.LastUpdatedFromReviewID
	}
	if patch.ThesisChangeSummary != nil {
		update["thesisChangeSummary"] = candidate.ThesisChangeSummary
	}
	return update
}

func containsThesisStatus(expected []common.ThesisStatus, actual common.ThesisStatus) bool {
	for _, status := range expected {
		if status == actual {
			return true
		}
	}
	return false
}
