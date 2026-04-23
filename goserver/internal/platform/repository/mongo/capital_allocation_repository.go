package mongo

import (
	"context"
	"fmt"
	"time"

	allocationpkg "goserver/internal/domain/allocation"
	"goserver/internal/domain/common"
	platformrepo "goserver/internal/platform/repository"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type capitalAllocationRunRecord struct {
	allocationpkg.CapitalAllocationRun `bson:",inline"`
	FinalizedAt                        *time.Time `bson:"finalizedAt,omitempty"`
	FinalizedBy                        string     `bson:"finalizedBy,omitempty"`
	FinalizationReason                 string     `bson:"finalizationReason,omitempty"`
	DraftUpdatedAt                     *time.Time `bson:"draftUpdatedAt,omitempty"`
	DraftUpdatedBy                     string     `bson:"draftUpdatedBy,omitempty"`
	DraftUpdateReason                  string     `bson:"draftUpdateReason,omitempty"`
}

func (record *capitalAllocationRunRecord) toDomain() *allocationpkg.CapitalAllocationRun {
	if record == nil {
		return nil
	}
	run := record.CapitalAllocationRun
	return &run
}

type CapitalAllocationRunMongoRepository struct {
	collection *mongo.Collection
}

var _ platformrepo.CapitalAllocationRunRepository = (*CapitalAllocationRunMongoRepository)(nil)

func NewCapitalAllocationRunRepository(collection *mongo.Collection) *CapitalAllocationRunMongoRepository {
	return &CapitalAllocationRunMongoRepository{collection: collection}
}

func NewCapitalAllocationRepository(collection *mongo.Collection) *CapitalAllocationRunMongoRepository {
	return NewCapitalAllocationRunRepository(collection)
}

func (repository *CapitalAllocationRunMongoRepository) Create(ctx context.Context, run *allocationpkg.CapitalAllocationRun) (*allocationpkg.CapitalAllocationRun, error) {
	if run == nil {
		return nil, fmt.Errorf("create capital allocation run: run is required")
	}

	document := *run
	if document.ID.IsZero() {
		document.ID = newDocumentID()
	}

	if err := document.Validate(); err != nil {
		return nil, fmt.Errorf("create capital allocation run: validate run: %w", err)
	}

	if _, err := repository.collection.InsertOne(ctx, capitalAllocationRunRecord{CapitalAllocationRun: document}); err != nil {
		return nil, fmt.Errorf("create capital allocation run: %w", mapMongoError(err))
	}

	return &document, nil
}

func (repository *CapitalAllocationRunMongoRepository) GetByID(ctx context.Context, id primitive.ObjectID) (*allocationpkg.CapitalAllocationRun, error) {
	document, err := findOne[capitalAllocationRunRecord](ctx, repository.collection, bson.M{"_id": id})
	if err != nil {
		return nil, fmt.Errorf("get capital allocation run by id %s: %w", id.Hex(), mapMongoError(err))
	}
	return document.toDomain(), nil
}

func (repository *CapitalAllocationRunMongoRepository) List(
	ctx context.Context,
	filter platformrepo.CapitalAllocationRunFilter,
	options platformrepo.CapitalAllocationRunListOptions,
) (*platformrepo.ListResult[*allocationpkg.CapitalAllocationRun], error) {
	result, err := findPage[capitalAllocationRunRecord, *allocationpkg.CapitalAllocationRun](
		ctx,
		repository.collection,
		buildCapitalAllocationRunFilter(filter),
		options.Pagination,
		buildCapitalAllocationRunSort(options.Sort),
		nil,
		func(document *capitalAllocationRunRecord) *allocationpkg.CapitalAllocationRun {
			return document.toDomain()
		},
	)
	if err != nil {
		return nil, fmt.Errorf("list capital allocation runs: %w", mapMongoError(err))
	}
	return result, nil
}

func (repository *CapitalAllocationRunMongoRepository) UpdateDraftAllocation(
	ctx context.Context,
	runID primitive.ObjectID,
	patch platformrepo.CapitalAllocationRunDraftPatch,
) (*allocationpkg.CapitalAllocationRun, error) {
	current, err := repository.getRecordByID(ctx, runID)
	if err != nil {
		return nil, err
	}
	if current.FinalizedAt != nil {
		return nil, immutableState("update capital allocation draft %s is already finalized", runID.Hex())
	}

	candidate := current.CapitalAllocationRun
	if patch.AvailableCashStart != nil {
		candidate.AvailableCashStart = *patch.AvailableCashStart
	}
	if patch.FreshMonthlyCash != nil {
		candidate.FreshMonthlyCash = *patch.FreshMonthlyCash
	}
	if patch.SellProceedsAvailable != nil {
		candidate.SellProceedsAvailable = *patch.SellProceedsAvailable
	}
	if patch.CarryForwardCash != nil {
		candidate.CarryForwardCash = *patch.CarryForwardCash
	}
	if patch.TargetDeployableCash != nil {
		candidate.TargetDeployableCash = *patch.TargetDeployableCash
	}
	if patch.AllocatedCashTotal != nil {
		candidate.AllocatedCashTotal = *patch.AllocatedCashTotal
	}
	if patch.CashLeftUnallocated != nil {
		candidate.CashLeftUnallocated = *patch.CashLeftUnallocated
	}
	if patch.AllocationNotes != nil {
		candidate.AllocationNotes = *patch.AllocationNotes
	}
	if len(patch.Items) > 0 {
		if patch.ReplaceItems {
			candidate.Items = append([]allocationpkg.CapitalAllocationItem(nil), patch.Items...)
		} else {
			candidate.Items = append(append([]allocationpkg.CapitalAllocationItem(nil), candidate.Items...), patch.Items...)
		}
	}

	if err := candidate.Validate(); err != nil {
		return nil, fmt.Errorf("update capital allocation draft %s: validate run: %w", runID.Hex(), err)
	}

	updateTime := mutationTimestamp(patch.Mutation)
	update := bson.M{"$set": buildCapitalAllocationDraftUpdate(&candidate, patch, updateTime)}
	result, err := repository.collection.UpdateOne(
		ctx,
		bson.M{"_id": runID, "finalizedAt": bson.M{"$exists": false}},
		update,
	)
	if err != nil {
		return nil, fmt.Errorf("update capital allocation draft %s: %w", runID.Hex(), mapMongoError(err))
	}
	if result.MatchedCount == 0 {
		return nil, preconditionFailed("update capital allocation draft %s stale write rejected", runID.Hex())
	}

	return &candidate, nil
}

func (repository *CapitalAllocationRunMongoRepository) FinalizeAllocationRun(
	ctx context.Context,
	runID primitive.ObjectID,
	patch platformrepo.CapitalAllocationRunFinalizationPatch,
) (*allocationpkg.CapitalAllocationRun, error) {
	current, err := repository.getRecordByID(ctx, runID)
	if err != nil {
		return nil, err
	}
	if current.FinalizedAt != nil {
		return nil, immutableState("finalize capital allocation run %s is already finalized", runID.Hex())
	}

	update := bson.M{"$set": bson.M{
		"finalizedAt":        patch.FinalizedAt.UTC(),
		"finalizedBy":        patch.FinalizedBy,
		"finalizationReason": patch.Reason,
	}}
	result, err := repository.collection.UpdateOne(
		ctx,
		bson.M{"_id": runID, "finalizedAt": bson.M{"$exists": false}},
		update,
	)
	if err != nil {
		return nil, fmt.Errorf("finalize capital allocation run %s: %w", runID.Hex(), mapMongoError(err))
	}
	if result.MatchedCount == 0 {
		return nil, preconditionFailed("finalize capital allocation run %s stale write rejected", runID.Hex())
	}

	return current.toDomain(), nil
}

func (repository *CapitalAllocationRunMongoRepository) GetLatestByBookType(ctx context.Context, bookType common.BookType) (*allocationpkg.CapitalAllocationRun, error) {
	document, err := findOne[capitalAllocationRunRecord](
		ctx,
		repository.collection,
		bson.M{"bookType": bookType},
		options.FindOne().SetSort(bson.D{{Key: "allocationDate", Value: -1}, {Key: "createdAt", Value: -1}}),
	)
	if err != nil {
		return nil, fmt.Errorf("get latest capital allocation run for book %s: %w", bookType, mapMongoError(err))
	}
	return document.toDomain(), nil
}

func (repository *CapitalAllocationRunMongoRepository) getRecordByID(ctx context.Context, id primitive.ObjectID) (*capitalAllocationRunRecord, error) {
	document, err := findOne[capitalAllocationRunRecord](ctx, repository.collection, bson.M{"_id": id})
	if err != nil {
		return nil, fmt.Errorf("get capital allocation record by id %s: %w", id.Hex(), mapMongoError(err))
	}
	return document, nil
}

func buildCapitalAllocationRunFilter(filter platformrepo.CapitalAllocationRunFilter) bson.M {
	query := bson.M{}
	addObjectIDFilter(query, "_id", filter.IDs)
	addObjectIDFilter(query, "workflowRunId", filter.WorkflowRunIDs)
	if len(filter.BookTypes) > 0 {
		query["bookType"] = bson.M{"$in": filter.BookTypes}
	}
	if filter.ContainsCompanyID != nil {
		query["items.companyId"] = *filter.ContainsCompanyID
	}
	if filter.ContainsDecisionReviewID != nil {
		query["items.decisionReviewId"] = *filter.ContainsDecisionReviewID
	}
	addTimeRangeFilter(query, "allocationDate", filter.AllocationDate)
	addTimeRangeFilter(query, "createdAt", filter.CreatedAt)
	return query
}

func buildCapitalAllocationRunSort(option platformrepo.CapitalAllocationRunSortOption) bson.D {
	switch option.By {
	case platformrepo.CapitalAllocationRunSortByCreatedAt:
		return bson.D{{Key: "createdAt", Value: sortDirection(option.Order, platformrepo.SortOrderDescending)}, {Key: "allocationDate", Value: -1}}
	case platformrepo.CapitalAllocationRunSortByAllocationDate, "":
		return bson.D{{Key: "allocationDate", Value: sortDirection(option.Order, platformrepo.SortOrderDescending)}, {Key: "createdAt", Value: -1}}
	default:
		return bson.D{{Key: "allocationDate", Value: -1}, {Key: "createdAt", Value: -1}}
	}
}

func buildCapitalAllocationDraftUpdate(
	candidate *allocationpkg.CapitalAllocationRun,
	patch platformrepo.CapitalAllocationRunDraftPatch,
	updateTime time.Time,
) bson.M {
	update := bson.M{
		"draftUpdatedAt": updateTime,
	}
	if patch.AvailableCashStart != nil {
		update["availableCashStart"] = candidate.AvailableCashStart
	}
	if patch.FreshMonthlyCash != nil {
		update["freshMonthlyCash"] = candidate.FreshMonthlyCash
	}
	if patch.SellProceedsAvailable != nil {
		update["sellProceedsAvailable"] = candidate.SellProceedsAvailable
	}
	if patch.CarryForwardCash != nil {
		update["carryForwardCash"] = candidate.CarryForwardCash
	}
	if patch.TargetDeployableCash != nil {
		update["targetDeployableCash"] = candidate.TargetDeployableCash
	}
	if patch.AllocatedCashTotal != nil {
		update["allocatedCashTotal"] = candidate.AllocatedCashTotal
	}
	if patch.CashLeftUnallocated != nil {
		update["cashLeftUnallocated"] = candidate.CashLeftUnallocated
	}
	if patch.AllocationNotes != nil {
		update["allocationNotes"] = candidate.AllocationNotes
	}
	if len(patch.Items) > 0 {
		update["items"] = candidate.Items
	}
	if patch.Mutation.Actor != "" {
		update["draftUpdatedBy"] = patch.Mutation.Actor
	}
	if patch.Mutation.Reason != "" {
		update["draftUpdateReason"] = patch.Mutation.Reason
	}
	return update
}
