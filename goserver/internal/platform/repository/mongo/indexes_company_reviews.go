package mongo

import (
	"goserver/internal/domain/common"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
)

func companyReviewIndexModels() []mongo.IndexModel {
	finalizedHistoryFilter := finalizedCompanyReviewPartialFilter()

	return []mongo.IndexModel{
		newIndex(
			"ix_company_reviews_company_book_review_date_desc",
			bson.D{
				{Key: "companyId", Value: 1},
				{Key: "bookType", Value: 1},
				{Key: "reviewDate", Value: -1},
				{Key: "createdAt", Value: -1},
			},
		),
		// Finalized-review lookups are hot in change detection and historical comparisons,
		// so we keep a smaller partial index just for immutable history.
		newPartialIndex(
			"ix_company_reviews_company_book_final_status_review_date_desc",
			bson.D{
				{Key: "companyId", Value: 1},
				{Key: "bookType", Value: 1},
				{Key: "reviewStatus", Value: 1},
				{Key: "reviewDate", Value: -1},
				{Key: "finalizedAt", Value: -1},
			},
			finalizedHistoryFilter,
		),
		newIndex(
			"ix_company_reviews_workflow_run_review_date_desc",
			bson.D{
				{Key: "workflowRunId", Value: 1},
				{Key: "reviewDate", Value: -1},
				{Key: "createdAt", Value: -1},
			},
		),
		// Pending-state workers typically want the stalest mutable review first.
		newIndex(
			"ix_company_reviews_lifecycle_updated_at",
			bson.D{
				{Key: "reviewLifecycleState", Value: 1},
				{Key: "updatedAt", Value: 1},
			},
		),
		newIndex(
			"ix_company_reviews_book_review_date_desc",
			bson.D{
				{Key: "bookType", Value: 1},
				{Key: "reviewDate", Value: -1},
				{Key: "createdAt", Value: -1},
			},
		),
		newIndex(
			"ix_company_reviews_status_book_review_date_desc",
			bson.D{
				{Key: "reviewStatus", Value: 1},
				{Key: "bookType", Value: 1},
				{Key: "reviewDate", Value: -1},
			},
		),
		newPartialIndex(
			"ix_company_reviews_final_action_book_review_date_desc",
			bson.D{
				{Key: "finalActionAfterReview", Value: 1},
				{Key: "bookType", Value: 1},
				{Key: "reviewDate", Value: -1},
			},
			finalizedHistoryFilter,
		),
		newPartialIndex(
			"ix_company_reviews_final_bucket_book_review_date_desc",
			bson.D{
				{Key: "finalBucketAfterReview", Value: 1},
				{Key: "bookType", Value: 1},
				{Key: "reviewDate", Value: -1},
			},
			finalizedHistoryFilter,
		),
	}
}

func finalizedCompanyReviewPartialFilter() bson.D {
	return bson.D{
		{
			Key: "reviewStatus",
			Value: bson.D{
				{
					Key: "$in",
					Value: bson.A{
						common.ReviewStatusFinal,
						common.ReviewStatusSuperseded,
					},
				},
			},
		},
	}
}
