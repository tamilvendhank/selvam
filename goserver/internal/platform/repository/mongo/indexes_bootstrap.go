package mongo

import (
	"context"
	"fmt"

	platformconfig "goserver/internal/platform/config"

	"go.mongodb.org/mongo-driver/mongo"
)

type collectionIndexSpec struct {
	name   string
	models []mongo.IndexModel
}

// EnsureIndexes is safe to call repeatedly from startup or an admin/bootstrap path.
// When no collection-name override is provided, the repository-local default names are used.
func EnsureIndexes(ctx context.Context, database *mongo.Database, collections ...platformconfig.CollectionConfig) error {
	resolved := resolveIndexCollections(collections...)
	for _, spec := range collectionIndexSpecs(resolved) {
		if err := createIndexes(ctx, database.Collection(spec.name), spec.models); err != nil {
			return err
		}
	}
	return nil
}

// EnsureCollectionIndexes allows an admin/bootstrap path to target a single logical collection.
func EnsureCollectionIndexes(ctx context.Context, database *mongo.Database, collectionName string, collections ...platformconfig.CollectionConfig) error {
	resolved := resolveIndexCollections(collections...)
	for _, spec := range collectionIndexSpecs(resolved) {
		if spec.name == collectionName {
			return createIndexes(ctx, database.Collection(spec.name), spec.models)
		}
	}

	return fmt.Errorf("ensure indexes: unknown collection %q", collectionName)
}

func collectionIndexSpecs(collections platformconfig.CollectionConfig) []collectionIndexSpec {
	return []collectionIndexSpec{
		{name: collections.Companies, models: companyIndexModels()},
		{name: collections.CompanyReviews, models: companyReviewIndexModels()},
		{name: collections.InvestmentTheses, models: investmentThesisIndexModels()},
		{name: collections.WorkflowRuns, models: workflowRunIndexModels()},
		{name: collections.WorkflowStepRuns, models: workflowStepRunIndexModels()},
		{name: collections.AIBatchJobs, models: aiBatchJobIndexModels()},
		{name: collections.AIBatchItems, models: aiBatchItemIndexModels()},
		{name: collections.ConfigSnapshots, models: configSnapshotIndexModels()},
		{name: collections.CapitalAllocationRuns, models: capitalAllocationRunIndexModels()},
		{name: collections.ManualOverrides, models: manualOverrideIndexModels()},
		{name: collections.CurrentPositions, models: currentPositionIndexModels()},
		{name: collections.JobReconciliationLogs, models: jobReconciliationLogIndexModels()},
	}
}
