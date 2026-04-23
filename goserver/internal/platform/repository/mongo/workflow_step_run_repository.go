package mongo

import (
	"context"

	"goserver/internal/platform/domain"
	"goserver/internal/platform/ports"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type WorkflowStepRunRepository struct {
	collection *mongo.Collection
}

func NewWorkflowStepRunRepository(collection *mongo.Collection) *WorkflowStepRunRepository {
	return &WorkflowStepRunRepository{collection: collection}
}

func (repository *WorkflowStepRunRepository) Create(ctx context.Context, run *domain.WorkflowStepRun) (*domain.WorkflowStepRun, error) {
	if err := run.Validate(); err != nil {
		return nil, err
	}

	document := toWorkflowStepRunDocument(run)
	document.ObjectID = newDocumentID()
	if _, err := repository.collection.InsertOne(ctx, document); err != nil {
		return nil, err
	}

	return fromWorkflowStepRunDocument(document), nil
}

func (repository *WorkflowStepRunRepository) Upsert(ctx context.Context, run *domain.WorkflowStepRun) (*domain.WorkflowStepRun, error) {
	if err := run.Validate(); err != nil {
		return nil, err
	}

	filter := bson.M{
		"workflowRunId": run.WorkflowRunID,
		"stepName":      run.StepName,
	}
	update := bson.M{
		"$set": toWorkflowStepRunDocument(run),
		"$setOnInsert": bson.M{
			"createdAt": run.CreatedAt,
		},
	}
	if _, err := repository.collection.UpdateOne(ctx, filter, update, options.Update().SetUpsert(true)); err != nil {
		return nil, err
	}

	return repository.GetByWorkflowRunAndStep(ctx, run.WorkflowRunID, run.StepName)
}

func (repository *WorkflowStepRunRepository) GetByID(ctx context.Context, id string) (*domain.WorkflowStepRun, error) {
	objectID, err := parseObjectID(id)
	if err != nil {
		return nil, nil
	}

	var document workflowStepRunDocument
	if err := repository.collection.FindOne(ctx, bson.M{"_id": objectID}).Decode(&document); err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, nil
		}
		return nil, err
	}

	return fromWorkflowStepRunDocument(&document), nil
}

func (repository *WorkflowStepRunRepository) GetByWorkflowRunAndStep(ctx context.Context, workflowRunID string, stepName string) (*domain.WorkflowStepRun, error) {
	var document workflowStepRunDocument
	if err := repository.collection.FindOne(ctx, bson.M{
		"workflowRunId": workflowRunID,
		"stepName":      stepName,
	}).Decode(&document); err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, nil
		}
		return nil, err
	}

	return fromWorkflowStepRunDocument(&document), nil
}

func (repository *WorkflowStepRunRepository) List(ctx context.Context, filter ports.WorkflowStepRunListFilter) ([]*domain.WorkflowStepRun, error) {
	query := bson.M{}
	if filter.WorkflowRunID != "" {
		query["workflowRunId"] = filter.WorkflowRunID
	}
	if filter.Status != "" {
		query["status"] = filter.Status
	}

	documents, err := findAll[workflowStepRunDocument](
		ctx,
		repository.collection,
		query,
		options.Find().
			SetSort(bson.D{{Key: "createdAt", Value: 1}}).
			SetLimit(normalizeLimit(filter.Limit, 200)).
			SetSkip(normalizeSkip(filter.Offset)),
	)
	if err != nil {
		return nil, err
	}

	result := make([]*domain.WorkflowStepRun, 0, len(documents))
	for index := range documents {
		result = append(result, fromWorkflowStepRunDocument(&documents[index]))
	}

	return result, nil
}
