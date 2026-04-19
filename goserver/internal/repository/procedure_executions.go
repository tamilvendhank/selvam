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

type ProcedureExecutionsRepository struct {
	collection *mongo.Collection
}

func NewProcedureExecutionsRepository(db *mongo.Database, collectionName string) *ProcedureExecutionsRepository {
	return &ProcedureExecutionsRepository{
		collection: db.Collection(collectionName),
	}
}

func (repository *ProcedureExecutionsRepository) List(ctx context.Context) ([]*domain.ProcedureExecution, error) {
	cursor, err := repository.collection.Find(ctx, bson.M{}, options.Find().SetSort(bson.D{
		{Key: "updatedAt", Value: -1},
		{Key: "createdAt", Value: -1},
	}))
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var executions []*domain.ProcedureExecution
	if err := cursor.All(ctx, &executions); err != nil {
		return nil, err
	}

	for _, execution := range executions {
		execution.NormalizeID()
	}

	return executions, nil
}

func (repository *ProcedureExecutionsRepository) ListRunningByJobID(ctx context.Context, jobID string) ([]*domain.ProcedureExecution, error) {
	if jobID == "" {
		return []*domain.ProcedureExecution{}, nil
	}

	cursor, err := repository.collection.Find(
		ctx,
		bson.M{
			"status":      "running",
			"steps.jobId": jobID,
		},
		options.Find().SetSort(bson.D{
			{Key: "updatedAt", Value: -1},
			{Key: "createdAt", Value: -1},
		}),
	)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var executions []*domain.ProcedureExecution
	if err := cursor.All(ctx, &executions); err != nil {
		return nil, err
	}

	for _, execution := range executions {
		execution.NormalizeID()
	}

	return executions, nil
}

func (repository *ProcedureExecutionsRepository) ListRunning(ctx context.Context) ([]*domain.ProcedureExecution, error) {
	cursor, err := repository.collection.Find(
		ctx,
		bson.M{"status": "running"},
		options.Find().SetSort(bson.D{
			{Key: "updatedAt", Value: -1},
			{Key: "createdAt", Value: -1},
		}),
	)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var executions []*domain.ProcedureExecution
	if err := cursor.All(ctx, &executions); err != nil {
		return nil, err
	}

	for _, execution := range executions {
		execution.NormalizeID()
	}

	return executions, nil
}

func (repository *ProcedureExecutionsRepository) Create(ctx context.Context, execution *domain.ProcedureExecution) (*domain.ProcedureExecution, error) {
	if execution.ObjectID.IsZero() {
		execution.ObjectID = primitive.NewObjectID()
	}

	if _, err := repository.collection.InsertOne(ctx, execution); err != nil {
		return nil, err
	}

	return repository.GetByID(ctx, execution.ObjectID.Hex())
}

func (repository *ProcedureExecutionsRepository) GetByID(ctx context.Context, id string) (*domain.ProcedureExecution, error) {
	objectID, ok := parseObjectID(id)
	if !ok {
		return nil, nil
	}

	var execution domain.ProcedureExecution
	err := repository.collection.FindOne(ctx, bson.M{"_id": objectID}).Decode(&execution)
	if err == mongo.ErrNoDocuments {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	execution.NormalizeID()
	return &execution, nil
}

func (repository *ProcedureExecutionsRepository) Update(ctx context.Context, id string, updates bson.M) (*domain.ProcedureExecution, error) {
	objectID, ok := parseObjectID(id)
	if !ok {
		return nil, nil
	}

	setDoc := bson.M{
		"updatedAt": time.Now(),
	}
	for key, value := range updates {
		setDoc[key] = value
	}

	if _, err := repository.collection.UpdateOne(ctx, bson.M{"_id": objectID}, bson.M{"$set": setDoc}); err != nil {
		return nil, err
	}

	return repository.GetByID(ctx, id)
}
