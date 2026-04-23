package mongo

import (
	"context"
	"strings"

	"goserver/internal/platform/domain"
	"goserver/internal/platform/ports"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type CompanyRepository struct {
	collection *mongo.Collection
}

func NewCompanyRepository(collection *mongo.Collection) *CompanyRepository {
	return &CompanyRepository{collection: collection}
}

func (repository *CompanyRepository) Create(ctx context.Context, company *domain.Company) (*domain.Company, error) {
	if err := company.Validate(); err != nil {
		return nil, err
	}

	document := toCompanyDocument(company)
	document.ObjectID = newDocumentID()

	if _, err := repository.collection.InsertOne(ctx, document); err != nil {
		return nil, err
	}

	return fromCompanyDocument(document), nil
}

func (repository *CompanyRepository) Update(ctx context.Context, company *domain.Company) (*domain.Company, error) {
	if err := company.Validate(); err != nil {
		return nil, err
	}

	objectID, err := parseObjectID(company.ID)
	if err != nil {
		return nil, err
	}

	document := toCompanyDocument(company)
	document.ObjectID = objectID

	if _, err := repository.collection.ReplaceOne(ctx, bson.M{"_id": objectID}, document); err != nil {
		return nil, err
	}

	return repository.GetByID(ctx, company.ID)
}

func (repository *CompanyRepository) GetByID(ctx context.Context, id string) (*domain.Company, error) {
	objectID, err := parseObjectID(id)
	if err != nil {
		return nil, nil
	}

	var document companyDocument
	if err := repository.collection.FindOne(ctx, bson.M{"_id": objectID}).Decode(&document); err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, nil
		}
		return nil, err
	}

	return fromCompanyDocument(&document), nil
}

func (repository *CompanyRepository) GetBySymbol(ctx context.Context, symbol string) (*domain.Company, error) {
	var document companyDocument
	if err := repository.collection.FindOne(ctx, bson.M{"symbol": strings.ToUpper(strings.TrimSpace(symbol))}).Decode(&document); err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, nil
		}
		return nil, err
	}

	return fromCompanyDocument(&document), nil
}

func (repository *CompanyRepository) List(ctx context.Context, filter ports.CompanyListFilter) ([]*domain.Company, error) {
	query := bson.M{}
	if filter.BookType == domain.BookTypeInvesting {
		query["isInInvestingUniverse"] = true
	}
	if filter.BookType == domain.BookTypeTrading {
		query["isInTradingUniverse"] = true
	}
	if filter.ActiveOnly != nil {
		query["statusActive"] = *filter.ActiveOnly
	}
	if search := maybeCaseInsensitiveContains(filter.Search); len(search) != 0 {
		query["$or"] = bson.A{
			bson.M{"symbol": search},
			bson.M{"companyName": search},
			bson.M{"sector": search},
			bson.M{"industry": search},
		}
	}

	documents, err := findAll[companyDocument](
		ctx,
		repository.collection,
		query,
		options.Find().
			SetSort(bson.D{{Key: "companyName", Value: 1}}).
			SetLimit(normalizeLimit(filter.Limit, 100)).
			SetSkip(normalizeSkip(filter.Offset)),
	)
	if err != nil {
		return nil, err
	}

	result := make([]*domain.Company, 0, len(documents))
	for index := range documents {
		result = append(result, fromCompanyDocument(&documents[index]))
	}

	return result, nil
}
