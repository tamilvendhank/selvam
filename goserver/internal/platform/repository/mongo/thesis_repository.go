package mongo

import (
	"context"

	"goserver/internal/platform/domain"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type ThesisRepository struct {
	collection *mongo.Collection
}

func NewThesisRepository(collection *mongo.Collection) *ThesisRepository {
	return &ThesisRepository{collection: collection}
}

func (repository *ThesisRepository) Create(ctx context.Context, thesis *domain.InvestmentThesis) (*domain.InvestmentThesis, error) {
	if err := thesis.Validate(); err != nil {
		return nil, err
	}

	document := toInvestmentThesisDocument(thesis)
	document.ObjectID = newDocumentID()
	if _, err := repository.collection.InsertOne(ctx, document); err != nil {
		return nil, err
	}

	return fromInvestmentThesisDocument(document), nil
}

func (repository *ThesisRepository) Update(ctx context.Context, thesis *domain.InvestmentThesis) (*domain.InvestmentThesis, error) {
	if err := thesis.Validate(); err != nil {
		return nil, err
	}

	objectID, err := parseObjectID(thesis.ID)
	if err != nil {
		return nil, err
	}

	document := toInvestmentThesisDocument(thesis)
	document.ObjectID = objectID
	if _, err := repository.collection.ReplaceOne(ctx, bson.M{"_id": objectID}, document); err != nil {
		return nil, err
	}

	return repository.GetByID(ctx, thesis.ID)
}

func (repository *ThesisRepository) GetByID(ctx context.Context, id string) (*domain.InvestmentThesis, error) {
	objectID, err := parseObjectID(id)
	if err != nil {
		return nil, nil
	}

	var document investmentThesisDocument
	if err := repository.collection.FindOne(ctx, bson.M{"_id": objectID}).Decode(&document); err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, nil
		}
		return nil, err
	}

	return fromInvestmentThesisDocument(&document), nil
}

func (repository *ThesisRepository) GetActiveByCompanyID(ctx context.Context, companyID string) (*domain.InvestmentThesis, error) {
	var document investmentThesisDocument
	if err := repository.collection.FindOne(
		ctx,
		bson.M{"companyId": companyID, "thesisStatus": domain.ThesisStatusActive},
		options.FindOne().SetSort(bson.D{{Key: "updatedAt", Value: -1}, {Key: "createdAt", Value: -1}}),
	).Decode(&document); err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, nil
		}
		return nil, err
	}

	return fromInvestmentThesisDocument(&document), nil
}

func (repository *ThesisRepository) ListByCompanyID(ctx context.Context, companyID string) ([]*domain.InvestmentThesis, error) {
	documents, err := findAll[investmentThesisDocument](
		ctx,
		repository.collection,
		bson.M{"companyId": companyID},
		options.Find().
			SetSort(bson.D{{Key: "updatedAt", Value: -1}, {Key: "createdAt", Value: -1}}).
			SetLimit(100),
	)
	if err != nil {
		return nil, err
	}

	result := make([]*domain.InvestmentThesis, 0, len(documents))
	for index := range documents {
		result = append(result, fromInvestmentThesisDocument(&documents[index]))
	}

	return result, nil
}
