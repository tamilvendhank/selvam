package mongo

import (
	"goserver/internal/domain/common"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
)

func investmentThesisIndexModels() []mongo.IndexModel {
	return []mongo.IndexModel{
		// The domain treats "active" as the current thesis, so at most one active thesis
		// should exist for a company at any point in time.
		newPartialUniqueIndex(
			"ux_investment_theses_company_active",
			bson.D{{Key: "companyId", Value: 1}},
			bson.D{{Key: "thesisStatus", Value: common.ThesisStatusActive}},
		),
		newIndex(
			"ix_investment_theses_company_version_desc",
			bson.D{
				{Key: "companyId", Value: 1},
				{Key: "thesisVersion", Value: -1},
			},
		),
		newIndex(
			"ix_investment_theses_status_updated_at_desc",
			bson.D{
				{Key: "thesisStatus", Value: 1},
				{Key: "updatedAt", Value: -1},
			},
		),
		newIndex(
			"ix_investment_theses_created_from_review_id",
			bson.D{{Key: "createdFromReviewId", Value: 1}},
		),
	}
}

func workflowRunIndexModels() []mongo.IndexModel {
	return []mongo.IndexModel{
		newIndex(
			"ix_workflow_runs_started_at_desc",
			bson.D{{Key: "startedAt", Value: -1}},
		),
		newIndex(
			"ix_workflow_runs_book_type_started_at_desc",
			bson.D{
				{Key: "bookType", Value: 1},
				{Key: "startedAt", Value: -1},
			},
		),
		newIndex(
			"ix_workflow_runs_status_started_at_desc",
			bson.D{
				{Key: "status", Value: 1},
				{Key: "startedAt", Value: -1},
			},
		),
		newIndex(
			"ix_workflow_runs_book_type_status_started_at_desc",
			bson.D{
				{Key: "bookType", Value: 1},
				{Key: "status", Value: 1},
				{Key: "startedAt", Value: -1},
			},
		),
		newIndex(
			"ix_workflow_runs_run_type_started_at_desc",
			bson.D{
				{Key: "runType", Value: 1},
				{Key: "startedAt", Value: -1},
			},
		),
	}
}

func workflowStepRunIndexModels() []mongo.IndexModel {
	return []mongo.IndexModel{
		newUniqueIndex(
			"ux_workflow_step_runs_workflow_run_step_name",
			bson.D{
				{Key: "workflowRunId", Value: 1},
				{Key: "stepName", Value: 1},
			},
		),
		newIndex(
			"ix_workflow_step_runs_workflow_run_status",
			bson.D{
				{Key: "workflowRunId", Value: 1},
				{Key: "status", Value: 1},
			},
		),
		newIndex(
			"ix_workflow_step_runs_status_updated_at_desc",
			bson.D{
				{Key: "status", Value: 1},
				{Key: "updatedAt", Value: -1},
			},
		),
	}
}
