package mongo

import (
	"context"
	"errors"
	"fmt"
	"time"

	companypkg "goserver/internal/domain/company"
	platformrepo "goserver/internal/platform/repository"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

type CompanyMongoRepository struct {
	collection *mongo.Collection
}

var _ platformrepo.CompanyRepository = (*CompanyMongoRepository)(nil)

func NewCompanyRepository(collection *mongo.Collection) *CompanyMongoRepository {
	return &CompanyMongoRepository{collection: collection}
}

func (repository *CompanyMongoRepository) Create(ctx context.Context, company *companypkg.Company) (*companypkg.Company, error) {
	if company == nil {
		return nil, fmt.Errorf("create company: company is required")
	}

	document := *company
	if document.ID.IsZero() {
		document.ID = newDocumentID()
	}
	document.Symbol = normalizeSymbol(document.Symbol)

	if err := document.Validate(); err != nil {
		return nil, fmt.Errorf("create company: validate company: %w", err)
	}

	if _, err := repository.collection.InsertOne(ctx, &document); err != nil {
		return nil, fmt.Errorf("create company %s: %w", document.Symbol, mapMongoError(err))
	}

	return &document, nil
}

func (repository *CompanyMongoRepository) GetByID(ctx context.Context, id primitive.ObjectID) (*companypkg.Company, error) {
	document, err := findOne[companypkg.Company](ctx, repository.collection, bson.M{"_id": id})
	if err != nil {
		return nil, fmt.Errorf("get company by id %s: %w", id.Hex(), mapMongoError(err))
	}
	return document, nil
}

func (repository *CompanyMongoRepository) GetBySymbol(ctx context.Context, symbol string) (*companypkg.Company, error) {
	normalizedSymbol := normalizeSymbol(symbol)
	document, err := findOne[companypkg.Company](ctx, repository.collection, bson.M{"symbol": normalizedSymbol})
	if err != nil {
		return nil, fmt.Errorf("get company by symbol %s: %w", normalizedSymbol, mapMongoError(err))
	}
	return document, nil
}

func (repository *CompanyMongoRepository) ExistsBySymbol(ctx context.Context, symbol string) (bool, error) {
	normalizedSymbol := normalizeSymbol(symbol)
	err := repository.collection.FindOne(
		ctx,
		bson.M{"symbol": normalizedSymbol},
	).Err()
	if err == nil {
		return true, nil
	}
	if errors.Is(err, mongo.ErrNoDocuments) {
		return false, nil
	}
	return false, fmt.Errorf("exists company by symbol %s: %w", normalizedSymbol, mapMongoError(err))
}

func (repository *CompanyMongoRepository) List(
	ctx context.Context,
	filter platformrepo.CompanyFilter,
	options platformrepo.CompanyListOptions,
) (*platformrepo.ListResult[*companypkg.Company], error) {
	result, err := findPage[companypkg.Company, *companypkg.Company](
		ctx,
		repository.collection,
		buildCompanyFilter(filter),
		options.Pagination,
		buildCompanySort(options.Sort),
		nil,
		func(document *companypkg.Company) *companypkg.Company {
			company := *document
			return &company
		},
	)
	if err != nil {
		return nil, fmt.Errorf("list companies: %w", mapMongoError(err))
	}
	return result, nil
}

func (repository *CompanyMongoRepository) UpdateMetadata(
	ctx context.Context,
	companyID primitive.ObjectID,
	patch platformrepo.CompanyUpdatePatch,
) (*companypkg.Company, error) {
	current, err := repository.GetByID(ctx, companyID)
	if err != nil {
		return nil, err
	}

	if patch.ExpectedUpdatedAt != nil && !current.UpdatedAt.Equal(patch.ExpectedUpdatedAt.UTC()) {
		return nil, preconditionFailed("update company metadata %s expected updatedAt %s", companyID.Hex(), patch.ExpectedUpdatedAt.UTC().Format(time.RFC3339Nano))
	}

	candidate := *current
	if patch.Exchange != nil {
		candidate.Exchange = *patch.Exchange
	}
	if patch.CompanyName != nil {
		candidate.CompanyName = *patch.CompanyName
	}
	if patch.Sector != nil {
		candidate.Sector = *patch.Sector
	}
	if patch.Industry != nil {
		candidate.Industry = *patch.Industry
	}
	if patch.SubIndustry != nil {
		candidate.SubIndustry = *patch.SubIndustry
	}
	if patch.BusinessSummary != nil {
		candidate.BusinessSummary = *patch.BusinessSummary
	}
	if patch.ListingDate != nil {
		candidate.ListingDate = patch.ListingDate.UTC()
	}
	if patch.MarketCapBucket != nil {
		candidate.MarketCapBucket = *patch.MarketCapBucket
	}
	if patch.StatusActive != nil {
		candidate.StatusActive = *patch.StatusActive
	}
	candidate.UpdatedAt = mutationTimestamp(patch.Mutation)

	if err := candidate.Validate(); err != nil {
		return nil, fmt.Errorf("update company metadata %s: validate company: %w", companyID.Hex(), err)
	}

	filter := bson.M{"_id": companyID, "updatedAt": current.UpdatedAt}
	update := bson.M{"$set": buildCompanyMetadataUpdate(&candidate, patch)}
	result, err := repository.collection.UpdateOne(ctx, filter, update)
	if err != nil {
		return nil, fmt.Errorf("update company metadata %s: %w", companyID.Hex(), mapMongoError(err))
	}
	if result.MatchedCount == 0 {
		return nil, preconditionFailed("update company metadata %s stale write rejected", companyID.Hex())
	}

	return &candidate, nil
}

func (repository *CompanyMongoRepository) UpdateUniverseFlags(
	ctx context.Context,
	companyID primitive.ObjectID,
	patch platformrepo.CompanyUniverseFlagPatch,
) (*companypkg.Company, error) {
	current, err := repository.GetByID(ctx, companyID)
	if err != nil {
		return nil, err
	}

	if patch.ExpectedUpdatedAt != nil && !current.UpdatedAt.Equal(patch.ExpectedUpdatedAt.UTC()) {
		return nil, preconditionFailed("update company universe flags %s expected updatedAt %s", companyID.Hex(), patch.ExpectedUpdatedAt.UTC().Format(time.RFC3339Nano))
	}

	candidate := *current
	if patch.InInvestingUniverse != nil {
		candidate.IsInInvestingUniverse = *patch.InInvestingUniverse
	}
	if patch.InTradingUniverse != nil {
		candidate.IsInTradingUniverse = *patch.InTradingUniverse
	}
	candidate.UpdatedAt = mutationTimestamp(patch.Mutation)

	if err := candidate.Validate(); err != nil {
		return nil, fmt.Errorf("update company universe flags %s: validate company: %w", companyID.Hex(), err)
	}

	filter := bson.M{"_id": companyID, "updatedAt": current.UpdatedAt}
	update := bson.M{"$set": buildCompanyUniverseFlagUpdate(&candidate, patch)}
	result, err := repository.collection.UpdateOne(ctx, filter, update)
	if err != nil {
		return nil, fmt.Errorf("update company universe flags %s: %w", companyID.Hex(), mapMongoError(err))
	}
	if result.MatchedCount == 0 {
		return nil, preconditionFailed("update company universe flags %s stale write rejected", companyID.Hex())
	}

	return &candidate, nil
}

func buildCompanyFilter(filter platformrepo.CompanyFilter) bson.M {
	query := bson.M{}

	addObjectIDFilter(query, "_id", filter.IDs)

	if len(filter.Symbols) > 0 {
		symbols := make([]string, 0, len(filter.Symbols))
		for _, symbol := range filter.Symbols {
			symbols = append(symbols, normalizeSymbol(symbol))
		}
		addStringFilter(query, "symbol", symbols)
	}

	if search := buildContainsRegex(filter.Search); len(search) > 0 {
		query["$or"] = bson.A{
			bson.M{"symbol": search},
			bson.M{"companyName": search},
			bson.M{"sector": search},
			bson.M{"industry": search},
			bson.M{"subIndustry": search},
		}
	}

	if filter.Exchange != "" {
		query["exchange"] = filter.Exchange
	}
	if filter.Sector != "" {
		query["sector"] = filter.Sector
	}
	if filter.Industry != "" {
		query["industry"] = filter.Industry
	}
	if filter.SubIndustry != "" {
		query["subIndustry"] = filter.SubIndustry
	}
	if filter.MarketCapBucket != "" {
		query["marketCapBucket"] = filter.MarketCapBucket
	}

	addBoolFilter(query, "isInInvestingUniverse", filter.InInvestingUniverse)
	addBoolFilter(query, "isInTradingUniverse", filter.InTradingUniverse)
	addBoolFilter(query, "statusActive", filter.StatusActive)
	addTimeRangeFilter(query, "createdAt", filter.CreatedAt)
	addTimeRangeFilter(query, "updatedAt", filter.UpdatedAt)

	return query
}

func buildCompanySort(option platformrepo.CompanySortOption) bson.D {
	switch option.By {
	case platformrepo.CompanySortBySymbol:
		return bson.D{{Key: "symbol", Value: sortDirection(option.Order, platformrepo.SortOrderAscending)}}
	case platformrepo.CompanySortByMarketCap:
		return bson.D{{Key: "marketCapBucket", Value: sortDirection(option.Order, platformrepo.SortOrderAscending)}, {Key: "symbol", Value: 1}}
	case platformrepo.CompanySortByCreatedAt:
		return bson.D{{Key: "createdAt", Value: sortDirection(option.Order, platformrepo.SortOrderDescending)}, {Key: "symbol", Value: 1}}
	case platformrepo.CompanySortByUpdatedAt:
		return bson.D{{Key: "updatedAt", Value: sortDirection(option.Order, platformrepo.SortOrderDescending)}, {Key: "symbol", Value: 1}}
	case platformrepo.CompanySortByListingDate:
		return bson.D{{Key: "listingDate", Value: sortDirection(option.Order, platformrepo.SortOrderDescending)}, {Key: "symbol", Value: 1}}
	case platformrepo.CompanySortByCompanyName, "":
		return bson.D{{Key: "companyName", Value: sortDirection(option.Order, platformrepo.SortOrderAscending)}, {Key: "symbol", Value: 1}}
	default:
		return bson.D{{Key: "companyName", Value: 1}, {Key: "symbol", Value: 1}}
	}
}

func buildCompanyMetadataUpdate(candidate *companypkg.Company, patch platformrepo.CompanyUpdatePatch) bson.M {
	update := bson.M{"updatedAt": candidate.UpdatedAt}
	if patch.Exchange != nil {
		update["exchange"] = candidate.Exchange
	}
	if patch.CompanyName != nil {
		update["companyName"] = candidate.CompanyName
	}
	if patch.Sector != nil {
		update["sector"] = candidate.Sector
	}
	if patch.Industry != nil {
		update["industry"] = candidate.Industry
	}
	if patch.SubIndustry != nil {
		update["subIndustry"] = candidate.SubIndustry
	}
	if patch.BusinessSummary != nil {
		update["businessSummary"] = candidate.BusinessSummary
	}
	if patch.ListingDate != nil {
		update["listingDate"] = candidate.ListingDate
	}
	if patch.MarketCapBucket != nil {
		update["marketCapBucket"] = candidate.MarketCapBucket
	}
	if patch.StatusActive != nil {
		update["statusActive"] = candidate.StatusActive
	}
	return update
}

func buildCompanyUniverseFlagUpdate(candidate *companypkg.Company, patch platformrepo.CompanyUniverseFlagPatch) bson.M {
	update := bson.M{"updatedAt": candidate.UpdatedAt}
	if patch.InInvestingUniverse != nil {
		update["isInInvestingUniverse"] = candidate.IsInInvestingUniverse
	}
	if patch.InTradingUniverse != nil {
		update["isInTradingUniverse"] = candidate.IsInTradingUniverse
	}
	return update
}
