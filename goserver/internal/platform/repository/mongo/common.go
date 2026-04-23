package mongo

import (
	"context"
	"fmt"
	"strings"
	"time"

	platformconfig "goserver/internal/platform/config"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type Collections struct {
	Companies             *mongo.Collection
	CompanyReviews        *mongo.Collection
	InvestmentTheses      *mongo.Collection
	WorkflowRuns          *mongo.Collection
	ConfigSnapshots       *mongo.Collection
	CapitalAllocationRuns *mongo.Collection
	ManualOverrides       *mongo.Collection
	CurrentPositions      *mongo.Collection
}

func NewCollections(database *mongo.Database, names platformconfig.CollectionConfig) Collections {
	return Collections{
		Companies:             database.Collection(names.Companies),
		CompanyReviews:        database.Collection(names.CompanyReviews),
		InvestmentTheses:      database.Collection(names.InvestmentTheses),
		WorkflowRuns:          database.Collection(names.WorkflowRuns),
		ConfigSnapshots:       database.Collection(names.ConfigSnapshots),
		CapitalAllocationRuns: database.Collection(names.CapitalAllocationRuns),
		ManualOverrides:       database.Collection(names.ManualOverrides),
		CurrentPositions:      database.Collection(names.CurrentPositions),
	}
}

func parseObjectID(id string) (primitive.ObjectID, error) {
	if !primitive.IsValidObjectID(id) {
		return primitive.NilObjectID, fmt.Errorf("invalid object id %q", id)
	}

	objectID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return primitive.NilObjectID, err
	}

	return objectID, nil
}

func normalizeLimit(limit int, fallback int64) int64 {
	if limit <= 0 {
		return fallback
	}

	return int64(limit)
}

func normalizeSkip(offset int) int64 {
	if offset <= 0 {
		return 0
	}

	return int64(offset)
}

func findAll[T any](ctx context.Context, collection *mongo.Collection, filter any, opts *options.FindOptions) ([]T, error) {
	cursor, err := collection.Find(ctx, filter, opts)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var result []T
	if err := cursor.All(ctx, &result); err != nil {
		return nil, err
	}

	return result, nil
}

func mergeUpdatedAt(updates bson.M) bson.M {
	if updates == nil {
		updates = bson.M{}
	}
	updates["updatedAt"] = time.Now().UTC()
	return updates
}

func maybeCaseInsensitiveContains(search string) bson.M {
	search = strings.TrimSpace(search)
	if search == "" {
		return bson.M{}
	}

	return bson.M{
		"$regex": primitive.Regex{
			Pattern: search,
			Options: "i",
		},
	}
}
