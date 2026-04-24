package mongo

import (
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
)

func aiBatchJobIndexModels() []mongo.IndexModel {
	return []mongo.IndexModel{
		newIndex(
			"ix_ai_batch_jobs_workflow_run_created_at_desc",
			bson.D{
				{Key: "workflowRunId", Value: 1},
				{Key: "createdAt", Value: -1},
			},
		),
		newIndex(
			"ix_ai_batch_jobs_status_updated_at_desc",
			bson.D{
				{Key: "status", Value: 1},
				{Key: "updatedAt", Value: -1},
			},
		),
		// Pollers care about the oldest not-recently-polled jobs within pollable states.
		newIndex(
			"ix_ai_batch_jobs_status_last_polled_at",
			bson.D{
				{Key: "status", Value: 1},
				{Key: "lastPolledAt", Value: 1},
				{Key: "updatedAt", Value: 1},
			},
		),
		// Provider handles are only assigned after submission, and the uniqueness scope is
		// provider-specific rather than global across every AI backend.
		newPartialUniqueIndex(
			"ux_ai_batch_jobs_provider_name_provider_job_handle",
			bson.D{
				{Key: "providerName", Value: 1},
				{Key: "providerJobHandle", Value: 1},
			},
			fieldExistsPartialFilter("providerJobHandle"),
		),
		newPartialUniqueIndex(
			"ux_ai_batch_jobs_local_job_handle",
			bson.D{{Key: "localJobHandle", Value: 1}},
			fieldExistsPartialFilter("localJobHandle"),
		),
		newIndex(
			"ix_ai_batch_jobs_job_type_status_created_at_desc",
			bson.D{
				{Key: "jobType", Value: 1},
				{Key: "status", Value: 1},
				{Key: "createdAt", Value: -1},
			},
		),
		newPartialIndex(
			"ix_ai_batch_jobs_idempotency_key_created_at_desc",
			bson.D{
				{Key: "idempotencyKey", Value: 1},
				{Key: "createdAt", Value: -1},
			},
			fieldExistsPartialFilter("idempotencyKey"),
		),
	}
}

func aiBatchItemIndexModels() []mongo.IndexModel {
	return []mongo.IndexModel{
		newIndex(
			"ix_ai_batch_items_batch_job_created_at_desc",
			bson.D{
				{Key: "aiBatchJobId", Value: 1},
				{Key: "createdAt", Value: -1},
			},
		),
		newIndex(
			"ix_ai_batch_items_workflow_run_company",
			bson.D{
				{Key: "workflowRunId", Value: 1},
				{Key: "companyId", Value: 1},
			},
		),
		newIndex(
			"ix_ai_batch_items_status_updated_at_desc",
			bson.D{
				{Key: "status", Value: 1},
				{Key: "updatedAt", Value: -1},
			},
		),
		newIndex(
			"ix_ai_batch_items_validation_status_updated_at_desc",
			bson.D{
				{Key: "validationStatus", Value: 1},
				{Key: "updatedAt", Value: -1},
			},
		),
		newIndex(
			"ix_ai_batch_items_company_item_type_created_at_desc",
			bson.D{
				{Key: "companyId", Value: 1},
				{Key: "itemType", Value: 1},
				{Key: "createdAt", Value: -1},
			},
		),
		newIndex(
			"ix_ai_batch_items_target_review_id",
			bson.D{{Key: "targetReviewId", Value: 1}},
		),
		newIndex(
			"ix_ai_batch_items_target_thesis_id",
			bson.D{{Key: "targetThesisId", Value: 1}},
		),
	}
}

func jobReconciliationLogIndexModels() []mongo.IndexModel {
	return []mongo.IndexModel{
		newIndex(
			"ix_job_reconciliation_logs_batch_job_polled_at_desc",
			bson.D{
				{Key: "aiBatchJobId", Value: 1},
				{Key: "polledAt", Value: -1},
			},
		),
	}
}
