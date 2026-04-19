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

type JobsRepository struct {
	collection *mongo.Collection
}

func NewJobsRepository(db *mongo.Database, collectionName string) *JobsRepository {
	return &JobsRepository{
		collection: db.Collection(collectionName),
	}
}

func (repository *JobsRepository) List(ctx context.Context) ([]*domain.Job, error) {
	cursor, err := repository.collection.Find(ctx, bson.M{}, options.Find().SetSort(bson.D{{Key: "createdAt", Value: -1}}))
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var jobs []*domain.Job
	if err := cursor.All(ctx, &jobs); err != nil {
		return nil, err
	}

	for _, job := range jobs {
		job.NormalizeID()
	}

	return jobs, nil
}

func (repository *JobsRepository) CreateMany(ctx context.Context, jobs []*domain.Job) ([]*domain.Job, error) {
	if len(jobs) == 0 {
		return []*domain.Job{}, nil
	}

	documents := make([]any, len(jobs))
	ids := make([]string, len(jobs))
	for index, job := range jobs {
		if job.ObjectID.IsZero() {
			job.ObjectID = primitive.NewObjectID()
		}

		documents[index] = job
		ids[index] = job.ObjectID.Hex()
	}

	if _, err := repository.collection.InsertMany(ctx, documents); err != nil {
		return nil, err
	}

	return repository.GetByIDs(ctx, ids)
}

func (repository *JobsRepository) GetByID(ctx context.Context, id string) (*domain.Job, error) {
	objectID, ok := parseObjectID(id)
	if !ok {
		return nil, nil
	}

	var job domain.Job
	err := repository.collection.FindOne(ctx, bson.M{"_id": objectID}).Decode(&job)
	if err == mongo.ErrNoDocuments {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	job.NormalizeID()
	return &job, nil
}

func (repository *JobsRepository) GetByIDs(ctx context.Context, ids []string) ([]*domain.Job, error) {
	objectIDs := make([]primitive.ObjectID, 0, len(ids))
	for _, id := range ids {
		objectID, ok := parseObjectID(id)
		if ok {
			objectIDs = append(objectIDs, objectID)
		}
	}

	if len(objectIDs) == 0 {
		return []*domain.Job{}, nil
	}

	cursor, err := repository.collection.Find(ctx, bson.M{"_id": bson.M{"$in": objectIDs}})
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var jobs []*domain.Job
	if err := cursor.All(ctx, &jobs); err != nil {
		return nil, err
	}

	jobsByID := make(map[string]*domain.Job, len(jobs))
	for _, job := range jobs {
		job.NormalizeID()
		jobsByID[job.ID] = job
	}

	ordered := make([]*domain.Job, 0, len(ids))
	for _, id := range ids {
		if job := jobsByID[id]; job != nil {
			ordered = append(ordered, job)
		}
	}

	return ordered, nil
}

func (repository *JobsRepository) ListByBatchID(ctx context.Context, batchID string) ([]*domain.Job, error) {
	cursor, err := repository.collection.Find(
		ctx,
		bson.M{"batchId": batchID},
		options.Find().SetSort(bson.D{{Key: "createdAt", Value: -1}}),
	)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var jobs []*domain.Job
	if err := cursor.All(ctx, &jobs); err != nil {
		return nil, err
	}

	for _, job := range jobs {
		job.NormalizeID()
	}

	return jobs, nil
}

func (repository *JobsRepository) ListByStatuses(ctx context.Context, statuses []string) ([]*domain.Job, error) {
	cursor, err := repository.collection.Find(
		ctx,
		bson.M{"status": bson.M{"$in": statuses}},
		options.Find().SetSort(bson.D{{Key: "createdAt", Value: -1}}),
	)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var jobs []*domain.Job
	if err := cursor.All(ctx, &jobs); err != nil {
		return nil, err
	}

	for _, job := range jobs {
		job.NormalizeID()
	}

	return jobs, nil
}

func (repository *JobsRepository) Update(ctx context.Context, id string, updates bson.M) (*domain.Job, error) {
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

func mergeWithUpdatedAt(updates bson.M) bson.M {
	merged := bson.M{
		"updatedAt": time.Now(),
	}

	for key, value := range updates {
		merged[key] = value
	}

	return merged
}

func parseObjectID(id string) (primitive.ObjectID, bool) {
	if !primitive.IsValidObjectID(id) {
		return primitive.NilObjectID, false
	}

	objectID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return primitive.NilObjectID, false
	}

	return objectID, true
}
