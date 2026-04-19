package repository

import (
	"context"
	"time"

	"goserver/internal/domain"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type SubmissionIterationsRepository struct {
	collection *mongo.Collection
}

func NewSubmissionIterationsRepository(db *mongo.Database, collectionName string) *SubmissionIterationsRepository {
	return &SubmissionIterationsRepository{
		collection: db.Collection(collectionName),
	}
}

func (repository *SubmissionIterationsRepository) Create(ctx context.Context, iteration *domain.SubmissionIteration) (*domain.SubmissionIteration, error) {
	if iteration == nil {
		return nil, nil
	}

	if iteration.ObjectID.IsZero() {
		iteration.ObjectID = primitive.NewObjectID()
	}

	if _, err := repository.collection.InsertOne(ctx, iteration); err != nil {
		return nil, err
	}

	iteration.NormalizeID()
	return repository.GetByID(ctx, iteration.ID)
}

func (repository *SubmissionIterationsRepository) CreateMany(ctx context.Context, iterations []*domain.SubmissionIteration) ([]*domain.SubmissionIteration, error) {
	if len(iterations) == 0 {
		return []*domain.SubmissionIteration{}, nil
	}

	documents := make([]any, len(iterations))
	ids := make([]string, len(iterations))
	for index, iteration := range iterations {
		if iteration.ObjectID.IsZero() {
			iteration.ObjectID = primitive.NewObjectID()
		}

		documents[index] = iteration
		ids[index] = iteration.ObjectID.Hex()
	}

	if _, err := repository.collection.InsertMany(ctx, documents); err != nil {
		return nil, err
	}

	return repository.GetByIDs(ctx, ids)
}

func (repository *SubmissionIterationsRepository) GetByID(ctx context.Context, id string) (*domain.SubmissionIteration, error) {
	objectID, ok := parseObjectID(id)
	if !ok {
		return nil, nil
	}

	var iteration domain.SubmissionIteration
	err := repository.collection.FindOne(ctx, bson.M{"_id": objectID}).Decode(&iteration)
	if err == mongo.ErrNoDocuments {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	iteration.NormalizeID()
	return &iteration, nil
}

func (repository *SubmissionIterationsRepository) GetByIDs(ctx context.Context, ids []string) ([]*domain.SubmissionIteration, error) {
	objectIDs := make([]primitive.ObjectID, 0, len(ids))
	for _, id := range ids {
		objectID, ok := parseObjectID(id)
		if ok {
			objectIDs = append(objectIDs, objectID)
		}
	}

	if len(objectIDs) == 0 {
		return []*domain.SubmissionIteration{}, nil
	}

	cursor, err := repository.collection.Find(ctx, bson.M{"_id": bson.M{"$in": objectIDs}})
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var iterations []*domain.SubmissionIteration
	if err := cursor.All(ctx, &iterations); err != nil {
		return nil, err
	}

	iterationsByID := make(map[string]*domain.SubmissionIteration, len(iterations))
	for _, iteration := range iterations {
		iteration.NormalizeID()
		iterationsByID[iteration.ID] = iteration
	}

	ordered := make([]*domain.SubmissionIteration, 0, len(ids))
	for _, id := range ids {
		if iteration := iterationsByID[id]; iteration != nil {
			ordered = append(ordered, iteration)
		}
	}

	return ordered, nil
}

func (repository *SubmissionIterationsRepository) ListByJobID(ctx context.Context, jobID string) ([]*domain.SubmissionIteration, error) {
	cursor, err := repository.collection.Find(
		ctx,
		bson.M{"jobId": jobID},
		options.Find().SetSort(bson.D{{Key: "iterationNumber", Value: 1}, {Key: "createdAt", Value: 1}}),
	)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var iterations []*domain.SubmissionIteration
	if err := cursor.All(ctx, &iterations); err != nil {
		return nil, err
	}

	for _, iteration := range iterations {
		iteration.NormalizeID()
	}

	return iterations, nil
}

func (repository *SubmissionIterationsRepository) GetLatestByJobID(ctx context.Context, jobID string) (*domain.SubmissionIteration, error) {
	var iteration domain.SubmissionIteration
	err := repository.collection.FindOne(
		ctx,
		bson.M{"jobId": jobID},
		options.FindOne().SetSort(bson.D{{Key: "iterationNumber", Value: -1}, {Key: "createdAt", Value: -1}}),
	).Decode(&iteration)
	if err == mongo.ErrNoDocuments {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	iteration.NormalizeID()
	return &iteration, nil
}

func (repository *SubmissionIterationsRepository) ListByBatchID(ctx context.Context, batchID string) ([]*domain.SubmissionIteration, error) {
	cursor, err := repository.collection.Find(
		ctx,
		bson.M{"batchId": batchID},
		options.Find().SetSort(bson.D{{Key: "createdAt", Value: 1}}),
	)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var iterations []*domain.SubmissionIteration
	if err := cursor.All(ctx, &iterations); err != nil {
		return nil, err
	}

	for _, iteration := range iterations {
		iteration.NormalizeID()
	}

	return iterations, nil
}

func (repository *SubmissionIterationsRepository) ListByStatuses(ctx context.Context, statuses []string) ([]*domain.SubmissionIteration, error) {
	cursor, err := repository.collection.Find(
		ctx,
		bson.M{"status": bson.M{"$in": statuses}},
		options.Find().SetSort(bson.D{{Key: "createdAt", Value: -1}}),
	)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var iterations []*domain.SubmissionIteration
	if err := cursor.All(ctx, &iterations); err != nil {
		return nil, err
	}

	for _, iteration := range iterations {
		iteration.NormalizeID()
	}

	return iterations, nil
}

func (repository *SubmissionIterationsRepository) Update(ctx context.Context, id string, updates bson.M) (*domain.SubmissionIteration, error) {
	objectID, ok := parseObjectID(id)
	if !ok {
		return nil, nil
	}

	updateDocument := bson.M{
		"$set": mergeWithUpdatedAt(updates),
	}

	if _, err := repository.collection.UpdateOne(ctx, bson.M{"_id": objectID}, updateDocument); err != nil {
		return nil, err
	}

	return repository.GetByID(ctx, id)
}

func (repository *SubmissionIterationsRepository) TryClaimFollowUp(ctx context.Context, id string, staleBefore time.Time) (bool, error) {
	objectID, ok := parseObjectID(id)
	if !ok {
		return false, nil
	}

	now := time.Now()
	filter := bson.M{
		"_id": objectID,
		"$and": bson.A{
			bson.M{
				"$or": bson.A{
					bson.M{"nextIterationId": bson.M{"$exists": false}},
					bson.M{"nextIterationId": nil},
				},
			},
			bson.M{
				"$or": bson.A{
					bson.M{"followUpState": bson.M{"$exists": false}},
					bson.M{"followUpState": ""},
					bson.M{"followUpState": "pending"},
					bson.M{
						"$and": bson.A{
							bson.M{"followUpState": "advancing"},
							bson.M{"followUpClaimedAt": bson.M{"$lt": staleBefore}},
						},
					},
				},
			},
		},
	}

	result, err := repository.collection.UpdateOne(ctx, filter, bson.M{
		"$set": mergeWithUpdatedAt(bson.M{
			"followUpState":     "advancing",
			"followUpClaimedAt": now,
		}),
	})
	if err != nil {
		return false, err
	}

	return result.ModifiedCount > 0, nil
}
