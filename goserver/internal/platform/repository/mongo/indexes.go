package mongo

import (
	"context"

	platformconfig "goserver/internal/platform/config"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

func EnsureIndexes(ctx context.Context, database *mongo.Database, collections platformconfig.CollectionConfig) error {
	type collectionIndexes struct {
		name   string
		models []mongo.IndexModel
	}

	indexes := []collectionIndexes{
		{
			name: collections.Companies,
			models: []mongo.IndexModel{
				{Keys: bson.D{{Key: "symbol", Value: 1}}, Options: options.Index().SetUnique(true).SetName("uniq_symbol")},
			},
		},
		{
			name: collections.CompanyReviews,
			models: []mongo.IndexModel{
				{Keys: bson.D{{Key: "companyId", Value: 1}, {Key: "reviewDate", Value: -1}}, Options: options.Index().SetName("company_review_date_desc")},
				{Keys: bson.D{{Key: "workflowRunId", Value: 1}}, Options: options.Index().SetName("workflow_run_id")},
				{Keys: bson.D{{Key: "bookType", Value: 1}, {Key: "reviewDate", Value: -1}}, Options: options.Index().SetName("book_review_date_desc")},
			},
		},
		{
			name: collections.InvestmentTheses,
			models: []mongo.IndexModel{
				{Keys: bson.D{{Key: "companyId", Value: 1}, {Key: "thesisStatus", Value: 1}}, Options: options.Index().SetName("company_thesis_status")},
			},
		},
		{
			name: collections.WorkflowRuns,
			models: []mongo.IndexModel{
				{Keys: bson.D{{Key: "startedAt", Value: -1}}, Options: options.Index().SetName("started_at_desc")},
				{Keys: bson.D{{Key: "idempotencyKey", Value: 1}}, Options: options.Index().SetName("idempotency_key")},
			},
		},
		{
			name: collections.WorkflowStepRuns,
			models: []mongo.IndexModel{
				{Keys: bson.D{{Key: "workflowRunId", Value: 1}, {Key: "stepName", Value: 1}}, Options: options.Index().SetUnique(true).SetName("uniq_workflow_step")},
				{Keys: bson.D{{Key: "workflowRunId", Value: 1}, {Key: "status", Value: 1}}, Options: options.Index().SetName("workflow_run_status")},
			},
		},
		{
			name: collections.CapitalAllocationRuns,
			models: []mongo.IndexModel{
				{Keys: bson.D{{Key: "allocationDate", Value: -1}}, Options: options.Index().SetName("allocation_date_desc")},
			},
		},
		{
			name: collections.ManualOverrides,
			models: []mongo.IndexModel{
				{Keys: bson.D{{Key: "reviewId", Value: 1}}, Options: options.Index().SetName("review_id")},
			},
		},
		{
			name: collections.CurrentPositions,
			models: []mongo.IndexModel{
				{Keys: bson.D{{Key: "companyId", Value: 1}, {Key: "bookType", Value: 1}}, Options: options.Index().SetUnique(true).SetName("uniq_company_book")},
			},
		},
		{
			name: collections.AIBatchJobs,
			models: []mongo.IndexModel{
				{Keys: bson.D{{Key: "workflowRunId", Value: 1}}, Options: options.Index().SetName("workflow_run_id")},
				{Keys: bson.D{{Key: "status", Value: 1}}, Options: options.Index().SetName("status")},
				{Keys: bson.D{{Key: "providerJobHandle", Value: 1}}, Options: options.Index().SetUnique(true).SetSparse(true).SetName("uniq_provider_job_handle")},
				{Keys: bson.D{{Key: "submittedAt", Value: -1}}, Options: options.Index().SetName("submitted_at_desc")},
				{Keys: bson.D{{Key: "idempotencyKey", Value: 1}}, Options: options.Index().SetName("idempotency_key")},
			},
		},
		{
			name: collections.AIBatchItems,
			models: []mongo.IndexModel{
				{Keys: bson.D{{Key: "aiBatchJobId", Value: 1}}, Options: options.Index().SetName("ai_batch_job_id")},
				{Keys: bson.D{{Key: "workflowRunId", Value: 1}, {Key: "companyId", Value: 1}}, Options: options.Index().SetName("workflow_company")},
				{Keys: bson.D{{Key: "status", Value: 1}}, Options: options.Index().SetName("status")},
				{Keys: bson.D{{Key: "companyId", Value: 1}, {Key: "itemType", Value: 1}, {Key: "createdAt", Value: -1}}, Options: options.Index().SetName("company_item_created_desc")},
			},
		},
		{
			name: collections.JobReconciliationLogs,
			models: []mongo.IndexModel{
				{Keys: bson.D{{Key: "aiBatchJobId", Value: 1}, {Key: "polledAt", Value: -1}}, Options: options.Index().SetName("ai_batch_job_polled_desc")},
			},
		},
	}

	for _, entry := range indexes {
		if len(entry.models) == 0 {
			continue
		}
		if _, err := database.Collection(entry.name).Indexes().CreateMany(ctx, entry.models); err != nil {
			return err
		}
	}

	return nil
}
