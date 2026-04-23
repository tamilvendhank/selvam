package mongo

import (
	"context"
	"fmt"
	"time"

	"goserver/internal/domain/common"
	reviewpkg "goserver/internal/domain/review"
	platformrepo "goserver/internal/platform/repository"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type companyReviewSummaryDocument struct {
	ID                     primitive.ObjectID          `bson:"_id"`
	CompanyID              primitive.ObjectID          `bson:"companyId"`
	Symbol                 string                      `bson:"symbol"`
	BookType               common.BookType             `bson:"bookType"`
	WorkflowRunID          primitive.ObjectID          `bson:"workflowRunId,omitempty"`
	ReviewDate             time.Time                   `bson:"reviewDate"`
	ReviewStatus           common.ReviewStatus         `bson:"reviewStatus"`
	ReviewLifecycleState   common.ReviewLifecycleState `bson:"reviewLifecycleState"`
	WeightedTotalScore     float64                     `bson:"weightedTotalScore,omitempty"`
	FinalActionAfterReview common.InvestingActionType  `bson:"finalActionAfterReview,omitempty"`
	FinalBucketAfterReview common.WatchlistBucket      `bson:"finalBucketAfterReview,omitempty"`
	ReviewerType           common.ReviewerType         `bson:"reviewerType"`
	UpdatedAt              time.Time                   `bson:"updatedAt"`
	FinalizedAt            *time.Time                  `bson:"finalizedAt,omitempty"`
}

func (document *companyReviewSummaryDocument) toSummary() *platformrepo.CompanyReviewSummary {
	if document == nil {
		return nil
	}
	return &platformrepo.CompanyReviewSummary{
		ID:                     document.ID,
		CompanyID:              document.CompanyID,
		Symbol:                 document.Symbol,
		BookType:               document.BookType,
		WorkflowRunID:          document.WorkflowRunID,
		ReviewDate:             document.ReviewDate,
		ReviewStatus:           document.ReviewStatus,
		ReviewLifecycleState:   document.ReviewLifecycleState,
		WeightedTotalScore:     document.WeightedTotalScore,
		FinalActionAfterReview: document.FinalActionAfterReview,
		FinalBucketAfterReview: document.FinalBucketAfterReview,
		ReviewerType:           document.ReviewerType,
		UpdatedAt:              document.UpdatedAt,
		FinalizedAt:            document.FinalizedAt,
	}
}

type CompanyReviewMongoRepository struct {
	collection *mongo.Collection
}

var _ platformrepo.CompanyReviewRepository = (*CompanyReviewMongoRepository)(nil)

func NewCompanyReviewRepository(collection *mongo.Collection) *CompanyReviewMongoRepository {
	return &CompanyReviewMongoRepository{collection: collection}
}

func (repository *CompanyReviewMongoRepository) CreateShell(ctx context.Context, review *reviewpkg.CompanyReview) (*reviewpkg.CompanyReview, error) {
	if review == nil {
		return nil, fmt.Errorf("create review shell: review is required")
	}

	document := *review
	if document.ID.IsZero() {
		document.ID = newDocumentID()
	}
	document.Symbol = normalizeSymbol(document.Symbol)

	if err := document.Validate(); err != nil {
		return nil, fmt.Errorf("create review shell: validate review: %w", err)
	}

	if _, err := repository.collection.InsertOne(ctx, &document); err != nil {
		return nil, fmt.Errorf("create review shell for company %s: %w", document.CompanyID.Hex(), mapMongoError(err))
	}

	return &document, nil
}

func (repository *CompanyReviewMongoRepository) GetByID(ctx context.Context, id primitive.ObjectID) (*reviewpkg.CompanyReview, error) {
	document, err := findOne[reviewpkg.CompanyReview](ctx, repository.collection, bson.M{"_id": id})
	if err != nil {
		return nil, fmt.Errorf("get company review by id %s: %w", id.Hex(), mapMongoError(err))
	}
	return document, nil
}

func (repository *CompanyReviewMongoRepository) GetLatestByCompanyAndBook(
	ctx context.Context,
	companyID primitive.ObjectID,
	bookType common.BookType,
	lookup platformrepo.LatestCompanyReviewOptions,
) (*reviewpkg.CompanyReview, error) {
	filter := bson.M{
		"companyId": companyID,
		"bookType":  bookType,
	}
	applyLatestCompanyReviewOptions(filter, lookup)

	document, err := findOne[reviewpkg.CompanyReview](
		ctx,
		repository.collection,
		filter,
		options.FindOne().SetSort(bson.D{{Key: "reviewDate", Value: -1}, {Key: "createdAt", Value: -1}}),
	)
	if err != nil {
		return nil, fmt.Errorf("get latest company review for company %s book %s: %w", companyID.Hex(), bookType, mapMongoError(err))
	}
	return document, nil
}

func (repository *CompanyReviewMongoRepository) GetPreviousFinalizedByCompanyAndBook(
	ctx context.Context,
	companyID primitive.ObjectID,
	bookType common.BookType,
	lookup platformrepo.PreviousFinalizedReviewLookup,
) (*reviewpkg.CompanyReview, error) {
	filter := bson.M{
		"companyId": companyID,
		"bookType":  bookType,
	}
	if lookup.IncludeSuperseded {
		filter["reviewStatus"] = bson.M{"$in": []common.ReviewStatus{common.ReviewStatusFinal, common.ReviewStatusSuperseded}}
	} else {
		filter["reviewStatus"] = common.ReviewStatusFinal
	}
	if !lookup.ExcludeReviewID.IsZero() {
		filter["_id"] = bson.M{"$ne": lookup.ExcludeReviewID}
	}
	if lookup.BeforeReviewDate != nil {
		filter["reviewDate"] = bson.M{"$lt": lookup.BeforeReviewDate.UTC()}
	}
	if lookup.BeforeFinalizedAt != nil {
		filter["finalizedAt"] = bson.M{"$lt": lookup.BeforeFinalizedAt.UTC()}
	}

	document, err := findOne[reviewpkg.CompanyReview](
		ctx,
		repository.collection,
		filter,
		options.FindOne().SetSort(bson.D{
			{Key: "reviewDate", Value: -1},
			{Key: "finalizedAt", Value: -1},
			{Key: "createdAt", Value: -1},
		}),
	)
	if err != nil {
		return nil, fmt.Errorf("get previous finalized review for company %s book %s: %w", companyID.Hex(), bookType, mapMongoError(err))
	}
	return document, nil
}

func (repository *CompanyReviewMongoRepository) ListByCompany(
	ctx context.Context,
	companyID primitive.ObjectID,
	options platformrepo.CompanyReviewListOptions,
) (*platformrepo.ListResult[*reviewpkg.CompanyReview], error) {
	return repository.List(ctx, platformrepo.CompanyReviewFilter{CompanyIDs: []primitive.ObjectID{companyID}}, options)
}

func (repository *CompanyReviewMongoRepository) ListByWorkflowRun(
	ctx context.Context,
	workflowRunID primitive.ObjectID,
	options platformrepo.CompanyReviewListOptions,
) (*platformrepo.ListResult[*reviewpkg.CompanyReview], error) {
	return repository.List(ctx, platformrepo.CompanyReviewFilter{WorkflowRunIDs: []primitive.ObjectID{workflowRunID}}, options)
}

func (repository *CompanyReviewMongoRepository) ListPendingByLifecycleState(
	ctx context.Context,
	states []common.ReviewLifecycleState,
	options platformrepo.CompanyReviewListOptions,
) (*platformrepo.ListResult[*reviewpkg.CompanyReview], error) {
	return repository.List(ctx, platformrepo.CompanyReviewFilter{
		LifecycleStates: states,
		PendingOnly:     true,
	}, options)
}

func (repository *CompanyReviewMongoRepository) List(
	ctx context.Context,
	filter platformrepo.CompanyReviewFilter,
	options platformrepo.CompanyReviewListOptions,
) (*platformrepo.ListResult[*reviewpkg.CompanyReview], error) {
	result, err := findPage[reviewpkg.CompanyReview, *reviewpkg.CompanyReview](
		ctx,
		repository.collection,
		buildCompanyReviewFilter(filter),
		options.Pagination,
		buildCompanyReviewSort(options.Sort),
		nil,
		func(document *reviewpkg.CompanyReview) *reviewpkg.CompanyReview {
			review := *document
			return &review
		},
	)
	if err != nil {
		return nil, fmt.Errorf("list company reviews: %w", mapMongoError(err))
	}
	return result, nil
}

func (repository *CompanyReviewMongoRepository) ListSummaries(
	ctx context.Context,
	filter platformrepo.CompanyReviewFilter,
	options platformrepo.CompanyReviewListOptions,
) (*platformrepo.ListResult[*platformrepo.CompanyReviewSummary], error) {
	result, err := findPage[companyReviewSummaryDocument, *platformrepo.CompanyReviewSummary](
		ctx,
		repository.collection,
		buildCompanyReviewFilter(filter),
		options.Pagination,
		buildCompanyReviewSort(options.Sort),
		bson.M{
			"_id":                    1,
			"companyId":              1,
			"symbol":                 1,
			"bookType":               1,
			"workflowRunId":          1,
			"reviewDate":             1,
			"reviewStatus":           1,
			"reviewLifecycleState":   1,
			"weightedTotalScore":     1,
			"finalActionAfterReview": 1,
			"finalBucketAfterReview": 1,
			"reviewerType":           1,
			"updatedAt":              1,
			"finalizedAt":            1,
		},
		func(document *companyReviewSummaryDocument) *platformrepo.CompanyReviewSummary {
			return document.toSummary()
		},
	)
	if err != nil {
		return nil, fmt.Errorf("list company review summaries: %w", mapMongoError(err))
	}
	return result, nil
}

func (repository *CompanyReviewMongoRepository) CountByWorkflowRun(ctx context.Context, workflowRunID primitive.ObjectID) (int64, error) {
	count, err := repository.collection.CountDocuments(ctx, bson.M{"workflowRunId": workflowRunID})
	if err != nil {
		return 0, fmt.Errorf("count company reviews by workflow run %s: %w", workflowRunID.Hex(), mapMongoError(err))
	}
	return count, nil
}

func (repository *CompanyReviewMongoRepository) UpdateLifecycleState(
	ctx context.Context,
	reviewID primitive.ObjectID,
	patch platformrepo.ReviewLifecycleUpdatePatch,
) (*reviewpkg.CompanyReview, error) {
	current, err := repository.GetByID(ctx, reviewID)
	if err != nil {
		return nil, err
	}
	if err := ensureReviewPatchPreconditions(current, patch.ExpectedCurrentLifecycleStates, patch.ExpectedCurrentStatuses, "update review lifecycle"); err != nil {
		return nil, err
	}
	if current.IsFinalized() && patch.NextLifecycleState != common.ReviewLifecycleStateSuperseded {
		return nil, immutableState("update review lifecycle %s rejected for immutable review", reviewID.Hex())
	}

	candidate := *current
	if err := candidate.TransitionLifecycleTo(patch.NextLifecycleState, mutationTimestamp(patch.Mutation)); err != nil {
		return nil, invalidTransition("update review lifecycle %s %v", reviewID.Hex(), err)
	}
	if err := candidate.Validate(); err != nil {
		return nil, fmt.Errorf("update review lifecycle %s: validate review: %w", reviewID.Hex(), err)
	}

	update := bson.M{"$set": bson.M{
		"reviewLifecycleState": candidate.ReviewLifecycleState,
		"reviewStatus":         candidate.ReviewStatus,
		"updatedAt":            candidate.UpdatedAt,
	}}
	if candidate.FinalizedAt != nil {
		update["$set"].(bson.M)["finalizedAt"] = candidate.FinalizedAt.UTC()
	}
	result, err := repository.collection.UpdateOne(ctx, reviewIdentityGuard(current), update)
	if err != nil {
		return nil, fmt.Errorf("update review lifecycle %s: %w", reviewID.Hex(), mapMongoError(err))
	}
	if result.MatchedCount == 0 {
		return nil, preconditionFailed("update review lifecycle %s stale write rejected", reviewID.Hex())
	}
	return &candidate, nil
}

func (repository *CompanyReviewMongoRepository) AttachAIResultReference(
	ctx context.Context,
	reviewID primitive.ObjectID,
	patch platformrepo.ReviewAIResultPatch,
) (*reviewpkg.CompanyReview, error) {
	current, err := repository.GetByID(ctx, reviewID)
	if err != nil {
		return nil, err
	}
	if current.IsFinalized() {
		return nil, immutableState("attach ai result reference %s rejected for immutable review", reviewID.Hex())
	}
	if err := ensureReviewPatchPreconditions(current, patch.ExpectedCurrentLifecycleStates, patch.ExpectedCurrentStatuses, "attach ai result reference"); err != nil {
		return nil, err
	}

	candidate := *current
	set := bson.M{
		"updatedAt": mutationTimestamp(patch.Mutation),
	}
	unset := bson.M{}
	candidate.UpdatedAt = set["updatedAt"].(time.Time)
	if patch.RawAIResultRef != nil {
		candidate.RawAIResultRef = patch.RawAIResultRef
		set["rawAIResultRef"] = candidate.RawAIResultRef
	}
	if patch.AIModelName != nil {
		candidate.AIModelName = *patch.AIModelName
		set["aiModelName"] = candidate.AIModelName
	}
	if patch.AIPromptVersion != nil {
		candidate.AIPromptVersion = *patch.AIPromptVersion
		set["aiPromptVersion"] = candidate.AIPromptVersion
	}
	candidate.ReviewMetadata = applyMetadataPatch("reviewMetadata", patch.ReviewMetadata, candidate.ReviewMetadata, set, unset)

	if err := candidate.Validate(); err != nil {
		return nil, fmt.Errorf("attach ai result reference %s: validate review: %w", reviewID.Hex(), err)
	}

	update := bson.M{"$set": set}
	if len(unset) > 0 {
		update["$unset"] = unset
	}
	result, err := repository.collection.UpdateOne(ctx, reviewIdentityGuard(current), update)
	if err != nil {
		return nil, fmt.Errorf("attach ai result reference %s: %w", reviewID.Hex(), mapMongoError(err))
	}
	if result.MatchedCount == 0 {
		return nil, preconditionFailed("attach ai result reference %s stale write rejected", reviewID.Hex())
	}
	return &candidate, nil
}

func (repository *CompanyReviewMongoRepository) SaveValidatedReviewContent(
	ctx context.Context,
	reviewID primitive.ObjectID,
	patch platformrepo.ReviewValidatedContentPatch,
) (*reviewpkg.CompanyReview, error) {
	current, err := repository.GetByID(ctx, reviewID)
	if err != nil {
		return nil, err
	}
	if current.IsFinalized() {
		return nil, immutableState("save validated review content %s rejected for immutable review", reviewID.Hex())
	}
	if err := ensureReviewPatchPreconditions(current, patch.ExpectedCurrentLifecycleStates, patch.ExpectedCurrentStatuses, "save validated review content"); err != nil {
		return nil, err
	}

	candidate := *current
	candidate.Sections = append([]reviewpkg.SectionScore(nil), patch.Sections...)
	candidate.DecisionAction = patch.DecisionAction
	candidate.PositionSnapshot = patch.PositionSnapshot
	candidate.ChangeLog = patch.ChangeLog
	candidate.WeightedTotalScore = patch.WeightedTotalScore
	candidate.HardGateFailed = patch.HardGateFailed
	candidate.HardGateFailureReasons = append([]string(nil), patch.HardGateFailureReasons...)
	candidate.ConfidenceScore = patch.ConfidenceScore
	candidate.FinalBucketAfterReview = patch.FinalBucketAfterReview
	candidate.FinalActionAfterReview = patch.FinalActionAfterReview
	candidate.ActionRationaleSummary = patch.ActionRationaleSummary
	candidate.WhatChangedSummary = patch.WhatChangedSummary
	if patch.ReviewerType != nil {
		candidate.ReviewerType = *patch.ReviewerType
	}

	set := bson.M{
		"sections":               candidate.Sections,
		"decisionAction":         candidate.DecisionAction,
		"positionSnapshot":       candidate.PositionSnapshot,
		"changeLog":              candidate.ChangeLog,
		"weightedTotalScore":     candidate.WeightedTotalScore,
		"hardGateFailed":         candidate.HardGateFailed,
		"hardGateFailureReasons": candidate.HardGateFailureReasons,
		"confidenceScore":        candidate.ConfidenceScore,
		"finalBucketAfterReview": candidate.FinalBucketAfterReview,
		"finalActionAfterReview": candidate.FinalActionAfterReview,
		"actionRationaleSummary": candidate.ActionRationaleSummary,
		"whatChangedSummary":     candidate.WhatChangedSummary,
		"updatedAt":              mutationTimestamp(patch.Mutation),
	}
	if patch.ReviewerType != nil {
		set["reviewerType"] = candidate.ReviewerType
	}
	unset := bson.M{}
	candidate.UpdatedAt = set["updatedAt"].(time.Time)
	candidate.ReviewMetadata = applyMetadataPatch("reviewMetadata", patch.ReviewMetadata, candidate.ReviewMetadata, set, unset)

	if err := candidate.Validate(); err != nil {
		return nil, fmt.Errorf("save validated review content %s: validate review: %w", reviewID.Hex(), err)
	}

	update := bson.M{"$set": set}
	if len(unset) > 0 {
		update["$unset"] = unset
	}
	result, err := repository.collection.UpdateOne(ctx, reviewIdentityGuard(current), update)
	if err != nil {
		return nil, fmt.Errorf("save validated review content %s: %w", reviewID.Hex(), mapMongoError(err))
	}
	if result.MatchedCount == 0 {
		return nil, preconditionFailed("save validated review content %s stale write rejected", reviewID.Hex())
	}
	return &candidate, nil
}

func (repository *CompanyReviewMongoRepository) FinalizeReview(
	ctx context.Context,
	reviewID primitive.ObjectID,
	patch platformrepo.ReviewFinalizationPatch,
) (*reviewpkg.CompanyReview, error) {
	current, err := repository.GetByID(ctx, reviewID)
	if err != nil {
		return nil, err
	}
	if current.IsFinalized() {
		return nil, immutableState("finalize review %s rejected for immutable review", reviewID.Hex())
	}
	if err := ensureReviewPatchPreconditions(current, patch.ExpectedCurrentLifecycleStates, patch.ExpectedCurrentStatuses, "finalize review"); err != nil {
		return nil, err
	}
	if !current.CanFinalize() {
		return nil, invalidTransition("finalize review %s cannot finalize from lifecycle %q", reviewID.Hex(), current.ReviewLifecycleState)
	}

	candidate := *current
	if err := candidate.Finalize(patch.FinalizedAt.UTC()); err != nil {
		return nil, invalidTransition("finalize review %s %v", reviewID.Hex(), err)
	}
	if err := candidate.Validate(); err != nil {
		return nil, fmt.Errorf("finalize review %s: validate review: %w", reviewID.Hex(), err)
	}

	update := bson.M{"$set": bson.M{
		"reviewLifecycleState": candidate.ReviewLifecycleState,
		"reviewStatus":         candidate.ReviewStatus,
		"updatedAt":            candidate.UpdatedAt,
		"finalizedAt":          candidate.FinalizedAt.UTC(),
		"finalizedBy":          patch.FinalizedBy,
		"finalizationReason":   patch.Reason,
	}}
	result, err := repository.collection.UpdateOne(ctx, reviewIdentityGuard(current), update)
	if err != nil {
		return nil, fmt.Errorf("finalize review %s: %w", reviewID.Hex(), mapMongoError(err))
	}
	if result.MatchedCount == 0 {
		return nil, preconditionFailed("finalize review %s stale write rejected", reviewID.Hex())
	}
	return &candidate, nil
}

func (repository *CompanyReviewMongoRepository) MarkSuperseded(
	ctx context.Context,
	reviewID primitive.ObjectID,
	patch platformrepo.ReviewSupersedePatch,
) (*reviewpkg.CompanyReview, error) {
	current, err := repository.GetByID(ctx, reviewID)
	if err != nil {
		return nil, err
	}
	if err := ensureReviewPatchPreconditions(current, patch.ExpectedCurrentLifecycleStates, patch.ExpectedCurrentStatuses, "mark review superseded"); err != nil {
		return nil, err
	}
	if !current.CanSupersede() {
		return nil, invalidTransition("mark review superseded %s cannot supersede lifecycle %q", reviewID.Hex(), current.ReviewLifecycleState)
	}
	if patch.ReplacementReview.IsZero() {
		return nil, preconditionFailed("mark review superseded %s requires replacement review id", reviewID.Hex())
	}

	candidate := *current
	if err := candidate.Supersede(patch.SupersededAt.UTC()); err != nil {
		return nil, invalidTransition("mark review superseded %s %v", reviewID.Hex(), err)
	}
	if err := candidate.Validate(); err != nil {
		return nil, fmt.Errorf("mark review superseded %s: validate review: %w", reviewID.Hex(), err)
	}

	update := bson.M{"$set": bson.M{
		"reviewLifecycleState": candidate.ReviewLifecycleState,
		"reviewStatus":         candidate.ReviewStatus,
		"updatedAt":            candidate.UpdatedAt,
		"replacementReviewId":  patch.ReplacementReview,
		"supersededAt":         patch.SupersededAt.UTC(),
		"supersededBy":         patch.SupersededBy,
		"supersedeReason":      patch.Reason,
	}}
	result, err := repository.collection.UpdateOne(ctx, reviewIdentityGuard(current), update)
	if err != nil {
		return nil, fmt.Errorf("mark review superseded %s: %w", reviewID.Hex(), mapMongoError(err))
	}
	if result.MatchedCount == 0 {
		return nil, preconditionFailed("mark review superseded %s stale write rejected", reviewID.Hex())
	}
	return &candidate, nil
}

func buildCompanyReviewFilter(filter platformrepo.CompanyReviewFilter) bson.M {
	query := bson.M{}

	addObjectIDFilter(query, "_id", filter.IDs)
	addObjectIDFilter(query, "companyId", filter.CompanyIDs)
	addObjectIDFilter(query, "workflowRunId", filter.WorkflowRunIDs)

	if len(filter.Symbols) > 0 {
		symbols := make([]string, 0, len(filter.Symbols))
		for _, symbol := range filter.Symbols {
			symbols = append(symbols, normalizeSymbol(symbol))
		}
		addStringFilter(query, "symbol", symbols)
	}
	if len(filter.BookTypes) > 0 {
		query["bookType"] = bson.M{"$in": filter.BookTypes}
	}

	reviewStatuses := filter.ReviewStatuses
	if filter.FinalizedOnly {
		if filter.IncludeSuperseded {
			reviewStatuses = []common.ReviewStatus{common.ReviewStatusFinal, common.ReviewStatusSuperseded}
		} else {
			reviewStatuses = []common.ReviewStatus{common.ReviewStatusFinal}
		}
	} else if filter.PendingOnly {
		reviewStatuses = []common.ReviewStatus{common.ReviewStatusDraft}
	}
	if len(reviewStatuses) > 0 {
		if !filter.IncludeSuperseded {
			reviewStatuses = removeSupersededStatuses(reviewStatuses)
		}
		if len(reviewStatuses) == 1 {
			query["reviewStatus"] = reviewStatuses[0]
		} else if len(reviewStatuses) > 1 {
			query["reviewStatus"] = bson.M{"$in": reviewStatuses}
		}
	}

	lifecycleStates := filter.LifecycleStates
	if filter.PendingOnly && len(lifecycleStates) == 0 {
		lifecycleStates = []common.ReviewLifecycleState{
			common.ReviewLifecycleStatePendingInput,
			common.ReviewLifecycleStatePendingAI,
			common.ReviewLifecycleStateAICompletedUnvalidated,
			common.ReviewLifecycleStateValidationFailed,
			common.ReviewLifecycleStateAIValidated,
		}
	}
	if len(lifecycleStates) > 0 {
		if !filter.IncludeSuperseded {
			lifecycleStates = removeSupersededLifecycleStates(lifecycleStates)
		}
		if len(lifecycleStates) == 1 {
			query["reviewLifecycleState"] = lifecycleStates[0]
		} else if len(lifecycleStates) > 1 {
			query["reviewLifecycleState"] = bson.M{"$in": lifecycleStates}
		}
	} else if !filter.IncludeSuperseded {
		query["reviewLifecycleState"] = bson.M{"$ne": common.ReviewLifecycleStateSuperseded}
	}

	if len(filter.FinalActions) > 0 {
		query["finalActionAfterReview"] = bson.M{"$in": filter.FinalActions}
	}
	if len(filter.FinalBuckets) > 0 {
		query["finalBucketAfterReview"] = bson.M{"$in": filter.FinalBuckets}
	}
	addBoolFilter(query, "ownedBeforeReview", filter.OwnedBeforeReview)
	if len(filter.ReviewerTypes) > 0 {
		query["reviewerType"] = bson.M{"$in": filter.ReviewerTypes}
	}

	addTimeRangeFilter(query, "reviewDate", filter.ReviewDate)
	addTimeRangeFilter(query, "createdAt", filter.CreatedAt)
	addTimeRangeFilter(query, "updatedAt", filter.UpdatedAt)
	addTimeRangeFilter(query, "finalizedAt", filter.FinalizedAt)

	return query
}

func buildCompanyReviewSort(option platformrepo.CompanyReviewSortOption) bson.D {
	switch option.By {
	case platformrepo.CompanyReviewSortByCreatedAt:
		return bson.D{{Key: "createdAt", Value: sortDirection(option.Order, platformrepo.SortOrderDescending)}, {Key: "reviewDate", Value: -1}}
	case platformrepo.CompanyReviewSortByUpdatedAt:
		return bson.D{{Key: "updatedAt", Value: sortDirection(option.Order, platformrepo.SortOrderDescending)}, {Key: "reviewDate", Value: -1}}
	case platformrepo.CompanyReviewSortByFinalizedAt:
		return bson.D{{Key: "finalizedAt", Value: sortDirection(option.Order, platformrepo.SortOrderDescending)}, {Key: "reviewDate", Value: -1}}
	case platformrepo.CompanyReviewSortByScore:
		return bson.D{{Key: "weightedTotalScore", Value: sortDirection(option.Order, platformrepo.SortOrderDescending)}, {Key: "reviewDate", Value: -1}}
	case platformrepo.CompanyReviewSortBySymbol:
		return bson.D{{Key: "symbol", Value: sortDirection(option.Order, platformrepo.SortOrderAscending)}, {Key: "reviewDate", Value: -1}}
	case platformrepo.CompanyReviewSortByReviewDate, "":
		return bson.D{{Key: "reviewDate", Value: sortDirection(option.Order, platformrepo.SortOrderDescending)}, {Key: "createdAt", Value: -1}}
	default:
		return bson.D{{Key: "reviewDate", Value: -1}, {Key: "createdAt", Value: -1}}
	}
}

func applyLatestCompanyReviewOptions(filter bson.M, options platformrepo.LatestCompanyReviewOptions) {
	if options.FinalizedOnly {
		if options.IncludeSuperseded {
			filter["reviewStatus"] = bson.M{"$in": []common.ReviewStatus{common.ReviewStatusFinal, common.ReviewStatusSuperseded}}
		} else {
			filter["reviewStatus"] = common.ReviewStatusFinal
			filter["reviewLifecycleState"] = bson.M{"$ne": common.ReviewLifecycleStateSuperseded}
		}
		return
	}
	if !options.IncludeSuperseded {
		filter["reviewLifecycleState"] = bson.M{"$ne": common.ReviewLifecycleStateSuperseded}
	}
}

func reviewIdentityGuard(review *reviewpkg.CompanyReview) bson.M {
	return bson.M{
		"_id":                  review.ID,
		"reviewStatus":         review.ReviewStatus,
		"reviewLifecycleState": review.ReviewLifecycleState,
	}
}

func ensureReviewPatchPreconditions(
	current *reviewpkg.CompanyReview,
	expectedLifecycleStates []common.ReviewLifecycleState,
	expectedStatuses []common.ReviewStatus,
	operation string,
) error {
	if len(expectedLifecycleStates) > 0 && !containsReviewLifecycleState(expectedLifecycleStates, current.ReviewLifecycleState) {
		return preconditionFailed("%s %s expected lifecycle state %q", operation, current.ID.Hex(), current.ReviewLifecycleState)
	}
	if len(expectedStatuses) > 0 && !containsReviewStatus(expectedStatuses, current.ReviewStatus) {
		return preconditionFailed("%s %s expected review status %q", operation, current.ID.Hex(), current.ReviewStatus)
	}
	return nil
}

func containsReviewLifecycleState(expected []common.ReviewLifecycleState, actual common.ReviewLifecycleState) bool {
	for _, state := range expected {
		if state == actual {
			return true
		}
	}
	return false
}

func containsReviewStatus(expected []common.ReviewStatus, actual common.ReviewStatus) bool {
	for _, status := range expected {
		if status == actual {
			return true
		}
	}
	return false
}

func removeSupersededStatuses(statuses []common.ReviewStatus) []common.ReviewStatus {
	filtered := make([]common.ReviewStatus, 0, len(statuses))
	for _, status := range statuses {
		if status != common.ReviewStatusSuperseded {
			filtered = append(filtered, status)
		}
	}
	return filtered
}

func removeSupersededLifecycleStates(states []common.ReviewLifecycleState) []common.ReviewLifecycleState {
	filtered := make([]common.ReviewLifecycleState, 0, len(states))
	for _, state := range states {
		if state != common.ReviewLifecycleStateSuperseded {
			filtered = append(filtered, state)
		}
	}
	return filtered
}
