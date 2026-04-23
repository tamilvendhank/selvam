package mongo

import (
	"context"

	"goserver/internal/platform/domain"
	"goserver/internal/platform/ports"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type WorkflowRunRepository struct {
	collection *mongo.Collection
}

func NewWorkflowRunRepository(collection *mongo.Collection) *WorkflowRunRepository {
	return &WorkflowRunRepository{collection: collection}
}

func (repository *WorkflowRunRepository) Create(ctx context.Context, run *domain.WorkflowRun) (*domain.WorkflowRun, error) {
	if err := run.Validate(); err != nil {
		return nil, err
	}

	document := toWorkflowRunDocument(run)
	document.ObjectID = newDocumentID()
	if _, err := repository.collection.InsertOne(ctx, document); err != nil {
		return nil, err
	}

	return fromWorkflowRunDocument(document), nil
}

func (repository *WorkflowRunRepository) Update(ctx context.Context, run *domain.WorkflowRun) (*domain.WorkflowRun, error) {
	if err := run.Validate(); err != nil {
		return nil, err
	}

	objectID, err := parseObjectID(run.ID)
	if err != nil {
		return nil, err
	}

	document := toWorkflowRunDocument(run)
	document.ObjectID = objectID
	if _, err := repository.collection.ReplaceOne(ctx, bson.M{"_id": objectID}, document); err != nil {
		return nil, err
	}

	return repository.GetByID(ctx, run.ID)
}

func (repository *WorkflowRunRepository) GetByID(ctx context.Context, id string) (*domain.WorkflowRun, error) {
	objectID, err := parseObjectID(id)
	if err != nil {
		return nil, nil
	}

	var document workflowRunDocument
	if err := repository.collection.FindOne(ctx, bson.M{"_id": objectID}).Decode(&document); err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, nil
		}
		return nil, err
	}

	return fromWorkflowRunDocument(&document), nil
}

func (repository *WorkflowRunRepository) GetByIdempotencyKey(ctx context.Context, key string) (*domain.WorkflowRun, error) {
	if key == "" {
		return nil, nil
	}

	var document workflowRunDocument
	if err := repository.collection.FindOne(
		ctx,
		bson.M{"idempotencyKey": key},
		options.FindOne().SetSort(bson.D{{Key: "createdAt", Value: -1}}),
	).Decode(&document); err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, nil
		}
		return nil, err
	}

	return fromWorkflowRunDocument(&document), nil
}

func (repository *WorkflowRunRepository) List(ctx context.Context, filter ports.WorkflowRunListFilter) ([]*domain.WorkflowRun, error) {
	query := bson.M{}
	if filter.BookType != "" {
		query["bookType"] = filter.BookType
	}
	if filter.Status != "" {
		query["status"] = filter.Status
	}

	documents, err := findAll[workflowRunDocument](
		ctx,
		repository.collection,
		query,
		options.Find().
			SetSort(bson.D{{Key: "startedAt", Value: -1}, {Key: "createdAt", Value: -1}}).
			SetLimit(normalizeLimit(filter.Limit, 100)).
			SetSkip(normalizeSkip(filter.Offset)),
	)
	if err != nil {
		return nil, err
	}

	result := make([]*domain.WorkflowRun, 0, len(documents))
	for index := range documents {
		result = append(result, fromWorkflowRunDocument(&documents[index]))
	}

	return result, nil
}
