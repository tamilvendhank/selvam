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

type ProceduresRepository struct {
	collection *mongo.Collection
}

func NewProceduresRepository(db *mongo.Database, collectionName string) *ProceduresRepository {
	return &ProceduresRepository{
		collection: db.Collection(collectionName),
	}
}

func (repository *ProceduresRepository) List(ctx context.Context) ([]*domain.Procedure, error) {
	cursor, err := repository.collection.Find(ctx, bson.M{}, options.Find().SetSort(bson.D{
		{Key: "updatedAt", Value: -1},
		{Key: "createdAt", Value: -1},
	}))
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var procedures []*domain.Procedure
	if err := cursor.All(ctx, &procedures); err != nil {
		return nil, err
	}

	for _, procedure := range procedures {
		procedure.NormalizeID()
	}

	return procedures, nil
}

func (repository *ProceduresRepository) Create(ctx context.Context, procedure *domain.Procedure) (*domain.Procedure, error) {
	if procedure.ObjectID.IsZero() {
		procedure.ObjectID = primitive.NewObjectID()
	}

	if _, err := repository.collection.InsertOne(ctx, procedure); err != nil {
		return nil, err
	}

	return repository.GetByID(ctx, procedure.ObjectID.Hex())
}

func (repository *ProceduresRepository) GetByID(ctx context.Context, id string) (*domain.Procedure, error) {
	objectID, ok := parseObjectID(id)
	if !ok {
		return nil, nil
	}

	var procedure domain.Procedure
	err := repository.collection.FindOne(ctx, bson.M{"_id": objectID}).Decode(&procedure)
	if err == mongo.ErrNoDocuments {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	procedure.NormalizeID()
	return &procedure, nil
}

func (repository *ProceduresRepository) Update(ctx context.Context, id string, updates bson.M) (*domain.Procedure, error) {
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
