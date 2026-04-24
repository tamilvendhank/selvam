package mongo

import (
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
)

func configSnapshotIndexModels() []mongo.IndexModel {
	return []mongo.IndexModel{
		newIndex(
			"ix_config_snapshots_created_at_desc",
			bson.D{{Key: "createdAt", Value: -1}},
		),
		newIndex(
			"ix_config_snapshots_book_type_created_at_desc",
			bson.D{
				{Key: "bookType", Value: 1},
				{Key: "createdAt", Value: -1},
			},
		),
		newIndex(
			"ix_config_snapshots_book_type_mode_created_at_desc",
			bson.D{
				{Key: "bookType", Value: 1},
				{Key: "mode", Value: 1},
				{Key: "createdAt", Value: -1},
			},
		),
	}
}

func capitalAllocationRunIndexModels() []mongo.IndexModel {
	return []mongo.IndexModel{
		newIndex(
			"ix_capital_allocation_runs_allocation_date_desc",
			bson.D{{Key: "allocationDate", Value: -1}},
		),
		newIndex(
			"ix_capital_allocation_runs_workflow_run_id",
			bson.D{{Key: "workflowRunId", Value: 1}},
		),
		newIndex(
			"ix_capital_allocation_runs_book_type_allocation_date_desc",
			bson.D{
				{Key: "bookType", Value: 1},
				{Key: "allocationDate", Value: -1},
			},
		),
	}
}

func manualOverrideIndexModels() []mongo.IndexModel {
	return []mongo.IndexModel{
		newIndex(
			"ix_manual_overrides_review_id_override_date_desc",
			bson.D{
				{Key: "reviewId", Value: 1},
				{Key: "overrideDate", Value: -1},
				{Key: "createdAt", Value: -1},
			},
		),
		newIndex(
			"ix_manual_overrides_company_id_created_at_desc",
			bson.D{
				{Key: "companyId", Value: 1},
				{Key: "createdAt", Value: -1},
			},
		),
		newIndex(
			"ix_manual_overrides_book_type_created_at_desc",
			bson.D{
				{Key: "bookType", Value: 1},
				{Key: "createdAt", Value: -1},
			},
		),
		newIndex(
			"ix_manual_overrides_override_date_desc",
			bson.D{{Key: "overrideDate", Value: -1}},
		),
	}
}

func currentPositionIndexModels() []mongo.IndexModel {
	return []mongo.IndexModel{
		newUniqueIndex(
			"ux_current_positions_company_book",
			bson.D{
				{Key: "companyId", Value: 1},
				{Key: "bookType", Value: 1},
			},
		),
		newIndex(
			"ix_current_positions_book_open_last_updated_at_desc",
			bson.D{
				{Key: "bookType", Value: 1},
				{Key: "isOpen", Value: 1},
				{Key: "lastUpdatedAt", Value: -1},
			},
		),
		newIndex(
			"ix_current_positions_open_last_updated_at_desc",
			bson.D{
				{Key: "isOpen", Value: 1},
				{Key: "lastUpdatedAt", Value: -1},
			},
		),
	}
}
