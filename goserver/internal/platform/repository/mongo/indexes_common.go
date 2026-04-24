package mongo

import (
	"bytes"
	"context"
	"fmt"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type existingIndexDocument struct {
	Name                    string   `bson:"name"`
	Key                     bson.Raw `bson:"key"`
	Unique                  bool     `bson:"unique,omitempty"`
	Sparse                  bool     `bson:"sparse,omitempty"`
	PartialFilterExpression bson.Raw `bson:"partialFilterExpression,omitempty"`
}

func newIndex(name string, keys bson.D) mongo.IndexModel {
	return mongo.IndexModel{
		Keys:    keys,
		Options: options.Index().SetName(name),
	}
}

func newUniqueIndex(name string, keys bson.D) mongo.IndexModel {
	return mongo.IndexModel{
		Keys:    keys,
		Options: options.Index().SetName(name).SetUnique(true),
	}
}

func newPartialIndex(name string, keys bson.D, partialFilter bson.D) mongo.IndexModel {
	return mongo.IndexModel{
		Keys:    keys,
		Options: options.Index().SetName(name).SetPartialFilterExpression(partialFilter),
	}
}

func newPartialUniqueIndex(name string, keys bson.D, partialFilter bson.D) mongo.IndexModel {
	return mongo.IndexModel{
		Keys: keys,
		Options: options.Index().
			SetName(name).
			SetUnique(true).
			SetPartialFilterExpression(partialFilter),
	}
}

func fieldExistsPartialFilter(field string) bson.D {
	return bson.D{
		{
			Key: field,
			Value: bson.D{
				{Key: "$exists", Value: true},
			},
		},
	}
}

func createIndexes(ctx context.Context, collection *mongo.Collection, models []mongo.IndexModel) error {
	if len(models) == 0 {
		return nil
	}

	pending, err := missingIndexModels(ctx, collection, models)
	if err != nil {
		return fmt.Errorf("ensure indexes for collection %q: %w", collection.Name(), err)
	}
	if len(pending) == 0 {
		return nil
	}

	if _, err := collection.Indexes().CreateMany(ctx, pending); err != nil {
		return fmt.Errorf("ensure indexes for collection %q: %w", collection.Name(), err)
	}
	return nil
}

func missingIndexModels(ctx context.Context, collection *mongo.Collection, models []mongo.IndexModel) ([]mongo.IndexModel, error) {
	existing, err := listExistingIndexes(ctx, collection)
	if err != nil {
		return nil, err
	}

	pending := make([]mongo.IndexModel, 0, len(models))
	for _, model := range models {
		equivalent, err := hasEquivalentIndex(existing, model)
		if err != nil {
			return nil, err
		}
		if equivalent {
			continue
		}
		pending = append(pending, model)
	}

	return pending, nil
}

func listExistingIndexes(ctx context.Context, collection *mongo.Collection) ([]existingIndexDocument, error) {
	cursor, err := collection.Indexes().List(ctx)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var existing []existingIndexDocument
	if err := cursor.All(ctx, &existing); err != nil {
		return nil, err
	}
	return existing, nil
}

func hasEquivalentIndex(existing []existingIndexDocument, model mongo.IndexModel) (bool, error) {
	opts := model.Options
	if opts == nil {
		opts = options.Index()
	}

	requestedKeys, err := marshalIndexDocument(model.Keys)
	if err != nil {
		return false, fmt.Errorf("marshal index keys: %w", err)
	}
	requestedPartial, err := marshalIndexDocument(opts.PartialFilterExpression)
	if err != nil {
		return false, fmt.Errorf("marshal partial filter expression: %w", err)
	}

	requestedUnique := boolOptionValue(opts.Unique)
	requestedSparse := boolOptionValue(opts.Sparse)

	for _, current := range existing {
		if requestedUnique != current.Unique {
			continue
		}
		if requestedSparse != current.Sparse {
			continue
		}
		if !bytes.Equal(requestedKeys, current.Key) {
			continue
		}
		if !bytes.Equal(requestedPartial, current.PartialFilterExpression) {
			continue
		}
		return true, nil
	}

	return false, nil
}

func marshalIndexDocument(document any) (bson.Raw, error) {
	if document == nil {
		return nil, nil
	}

	if raw, ok := document.(bson.Raw); ok {
		return raw, nil
	}

	encoded, err := bson.Marshal(document)
	if err != nil {
		return nil, err
	}
	return bson.Raw(encoded), nil
}

func boolOptionValue(value *bool) bool {
	return value != nil && *value
}
