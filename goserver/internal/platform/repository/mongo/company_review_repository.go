package mongo

import (
	"context"
	"errors"

	"goserver/internal/platform/domain"
	"goserver/internal/platform/ports"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

var ErrImmutableReview = errors.New("finalized reviews are immutable")

type CompanyReviewRepository struct {
	collection *mongo.Collection
}

func NewCompanyReviewRepository(collection *mongo.Collection) *CompanyReviewRepository {
	return &CompanyReviewRepository{collection: collection}
}

func (repository *CompanyReviewRepository) Create(ctx context.Context, review *domain.CompanyReview) (*domain.CompanyReview, error) {
	if err := review.Validate(); err != nil {
		return nil, err
	}

	document := toCompanyReviewDocument(review)
	document.ObjectID = newDocumentID()
	if _, err := repository.collection.InsertOne(ctx, document); err != nil {
		return nil, err
	}

	return fromCompanyReviewDocument(document), nil
}

func (repository *CompanyReviewRepository) UpdateDraft(ctx context.Context, review *domain.CompanyReview) (*domain.CompanyReview, error) {
	return repository.UpdateMutable(ctx, review)
}

func (repository *CompanyReviewRepository) UpdateMutable(ctx context.Context, review *domain.CompanyReview) (*domain.CompanyReview, error) {
	if err := review.Validate(); err != nil {
		return nil, err
	}

	objectID, err := parseObjectID(review.ID)
	if err != nil {
		return nil, err
	}

	document := toCompanyReviewDocument(review)
	document.ObjectID = objectID

	result, err := repository.collection.ReplaceOne(ctx, bson.M{
		"_id": objectID,
		"reviewStatus": bson.M{
			"$nin": []domain.ReviewStatus{domain.ReviewStatusFinalized, domain.ReviewStatusSuperseded},
		},
	}, document)
	if err != nil {
		return nil, err
	}
	if result.MatchedCount == 0 {
		return nil, ErrImmutableReview
	}

	return repository.GetByID(ctx, review.ID)
}

func (repository *CompanyReviewRepository) Finalize(ctx context.Context, reviewID string) (*domain.CompanyReview, error) {
	objectID, err := parseObjectID(reviewID)
	if err != nil {
		return nil, err
	}

	result, err := repository.collection.UpdateOne(ctx, bson.M{
		"_id": objectID,
		"reviewStatus": bson.M{
			"$in": []domain.ReviewStatus{domain.ReviewStatusAIValidated, domain.ReviewStatusDraft},
		},
	}, bson.M{
		"$set": mergeUpdatedAt(bson.M{
			"reviewStatus": domain.ReviewStatusFinalized,
		}),
	})
	if err != nil {
		return nil, err
	}
	if result.MatchedCount == 0 {
		return nil, ErrImmutableReview
	}

	return repository.GetByID(ctx, reviewID)
}

func (repository *CompanyReviewRepository) MarkSuperseded(ctx context.Context, reviewID string) (*domain.CompanyReview, error) {
	objectID, err := parseObjectID(reviewID)
	if err != nil {
		return nil, err
	}

	if _, err := repository.collection.UpdateOne(ctx, bson.M{"_id": objectID}, bson.M{
		"$set": mergeUpdatedAt(bson.M{
			"reviewStatus": domain.ReviewStatusSuperseded,
		}),
	}); err != nil {
		return nil, err
	}

	return repository.GetByID(ctx, reviewID)
}

func (repository *CompanyReviewRepository) GetByID(ctx context.Context, id string) (*domain.CompanyReview, error) {
	objectID, err := parseObjectID(id)
	if err != nil {
		return nil, nil
	}

	var document companyReviewDocument
	if err := repository.collection.FindOne(ctx, bson.M{"_id": objectID}).Decode(&document); err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, nil
		}
		return nil, err
	}

	return fromCompanyReviewDocument(&document), nil
}

func (repository *CompanyReviewRepository) GetLatestByCompany(ctx context.Context, companyID string, bookType domain.BookType) (*domain.CompanyReview, error) {
	var document companyReviewDocument
	if err := repository.collection.FindOne(
		ctx,
		bson.M{
			"companyId": companyID,
			"bookType":  bookType,
		},
		options.FindOne().SetSort(bson.D{{Key: "reviewDate", Value: -1}, {Key: "createdAt", Value: -1}}),
	).Decode(&document); err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, nil
		}
		return nil, err
	}

	return fromCompanyReviewDocument(&document), nil
}

func (repository *CompanyReviewRepository) GetLatestComparableByCompany(ctx context.Context, companyID string, bookType domain.BookType, excludeReviewID string) (*domain.CompanyReview, error) {
	query := bson.M{
		"companyId": companyID,
		"bookType":  bookType,
		"reviewStatus": bson.M{
			"$in": []domain.ReviewStatus{domain.ReviewStatusAIValidated, domain.ReviewStatusFinalized, domain.ReviewStatusSuperseded},
		},
	}

	if excludeReviewID != "" {
		if objectID, err := parseObjectID(excludeReviewID); err == nil {
			query["_id"] = bson.M{"$ne": objectID}
		}
	}

	var document companyReviewDocument
	if err := repository.collection.FindOne(
		ctx,
		query,
		options.FindOne().SetSort(bson.D{{Key: "reviewDate", Value: -1}, {Key: "createdAt", Value: -1}}),
	).Decode(&document); err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, nil
		}
		return nil, err
	}

	return fromCompanyReviewDocument(&document), nil
}

func (repository *CompanyReviewRepository) List(ctx context.Context, filter ports.CompanyReviewListFilter) ([]*domain.CompanyReview, error) {
	query := bson.M{}
	if filter.CompanyID != "" {
		query["companyId"] = filter.CompanyID
	}
	if filter.Symbol != "" {
		query["symbol"] = filter.Symbol
	}
	if filter.BookType != "" {
		query["bookType"] = filter.BookType
	}
	if filter.ReviewStatus != "" {
		query["reviewStatus"] = filter.ReviewStatus
	}

	documents, err := findAll[companyReviewDocument](
		ctx,
		repository.collection,
		query,
		options.Find().
			SetSort(bson.D{{Key: "reviewDate", Value: -1}, {Key: "createdAt", Value: -1}}).
			SetLimit(normalizeLimit(filter.Limit, 100)).
			SetSkip(normalizeSkip(filter.Offset)),
	)
	if err != nil {
		return nil, err
	}

	result := make([]*domain.CompanyReview, 0, len(documents))
	for index := range documents {
		result = append(result, fromCompanyReviewDocument(&documents[index]))
	}

	return result, nil
}
