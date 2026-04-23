package mongo

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"strings"
	"time"

	platformconfig "goserver/internal/platform/config"
	platformrepo "goserver/internal/platform/repository"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

const (
	defaultPageSize = 50
	maxPageSize     = 500
)

type Collections struct {
	Companies             *mongo.Collection
	CompanyReviews        *mongo.Collection
	InvestmentTheses      *mongo.Collection
	WorkflowRuns          *mongo.Collection
	WorkflowStepRuns      *mongo.Collection
	ConfigSnapshots       *mongo.Collection
	CapitalAllocationRuns *mongo.Collection
	ManualOverrides       *mongo.Collection
	CurrentPositions      *mongo.Collection
	AIBatchJobs           *mongo.Collection
	AIBatchItems          *mongo.Collection
	JobReconciliationLogs *mongo.Collection
}

func NewCollections(database *mongo.Database, names platformconfig.CollectionConfig) Collections {
	return Collections{
		Companies:             database.Collection(names.Companies),
		CompanyReviews:        database.Collection(names.CompanyReviews),
		InvestmentTheses:      database.Collection(names.InvestmentTheses),
		WorkflowRuns:          database.Collection(names.WorkflowRuns),
		WorkflowStepRuns:      database.Collection(names.WorkflowStepRuns),
		ConfigSnapshots:       database.Collection(names.ConfigSnapshots),
		CapitalAllocationRuns: database.Collection(names.CapitalAllocationRuns),
		ManualOverrides:       database.Collection(names.ManualOverrides),
		CurrentPositions:      database.Collection(names.CurrentPositions),
		AIBatchJobs:           database.Collection(names.AIBatchJobs),
		AIBatchItems:          database.Collection(names.AIBatchItems),
		JobReconciliationLogs: database.Collection(names.JobReconciliationLogs),
	}
}

func newDocumentID() primitive.ObjectID {
	return primitive.NewObjectID()
}

func currentTimeUTC() time.Time {
	return time.Now().UTC()
}

func mutationTimestamp(metadata platformrepo.MutationMetadata) time.Time {
	if metadata.OccurredAt.IsZero() {
		return currentTimeUTC()
	}
	return metadata.OccurredAt.UTC()
}

func normalizeSymbol(symbol string) string {
	return strings.ToUpper(strings.TrimSpace(symbol))
}

func buildContainsRegex(search string) bson.M {
	search = strings.TrimSpace(search)
	if search == "" {
		return bson.M{}
	}

	return bson.M{
		"$regex": primitive.Regex{
			Pattern: regexp.QuoteMeta(search),
			Options: "i",
		},
	}
}

func addObjectIDFilter(filter bson.M, field string, ids []primitive.ObjectID) {
	switch len(ids) {
	case 0:
		return
	case 1:
		filter[field] = ids[0]
	default:
		filter[field] = bson.M{"$in": ids}
	}
}

func addStringFilter(filter bson.M, field string, values []string) {
	switch len(values) {
	case 0:
		return
	case 1:
		filter[field] = values[0]
	default:
		filter[field] = bson.M{"$in": values}
	}
}

func addBoolFilter(filter bson.M, field string, value *bool) {
	if value == nil {
		return
	}
	filter[field] = *value
}

func addTimeRangeFilter(filter bson.M, field string, timeRange *platformrepo.TimeRange) {
	if timeRange == nil {
		return
	}

	rangeFilter := bson.M{}
	if timeRange.From != nil {
		rangeFilter["$gte"] = timeRange.From.UTC()
	}
	if timeRange.To != nil {
		rangeFilter["$lte"] = timeRange.To.UTC()
	}
	if len(rangeFilter) == 0 {
		return
	}

	filter[field] = rangeFilter
}

func normalizePageOptions(page platformrepo.PageOptions) (pageSize int, offset int64) {
	pageSize = page.PageSize
	if pageSize <= 0 {
		pageSize = defaultPageSize
	}
	if pageSize > maxPageSize {
		pageSize = maxPageSize
	}
	if page.Offset > 0 {
		offset = int64(page.Offset)
	}
	return pageSize, offset
}

func sortDirection(order platformrepo.SortOrder, defaultOrder platformrepo.SortOrder) int {
	if order == "" {
		order = defaultOrder
	}
	if order == platformrepo.SortOrderAscending {
		return 1
	}
	return -1
}

func findPage[D any, T any](
	ctx context.Context,
	collection *mongo.Collection,
	filter any,
	page platformrepo.PageOptions,
	sort bson.D,
	projection any,
	mapper func(*D) T,
) (*platformrepo.ListResult[T], error) {
	pageSize, offset := normalizePageOptions(page)

	findOptions := options.Find().
		SetSort(sort).
		SetSkip(offset).
		SetLimit(int64(pageSize + 1))
	if projection != nil {
		findOptions.SetProjection(projection)
	}

	cursor, err := collection.Find(ctx, filter, findOptions)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var documents []D
	if err := cursor.All(ctx, &documents); err != nil {
		return nil, err
	}

	hasMore := len(documents) > pageSize
	if hasMore {
		documents = documents[:pageSize]
	}

	items := make([]T, 0, len(documents))
	for index := range documents {
		items = append(items, mapper(&documents[index]))
	}

	return &platformrepo.ListResult[T]{
		Items: items,
		Page: platformrepo.PageInfo{
			PageSize: len(items),
			Offset:   int(offset),
			HasMore:  hasMore,
		},
	}, nil
}

func findOne[D any](ctx context.Context, collection *mongo.Collection, filter any, opts ...*options.FindOneOptions) (*D, error) {
	var document D
	if err := collection.FindOne(ctx, filter, opts...).Decode(&document); err != nil {
		return nil, err
	}
	return &document, nil
}

func cloneMap(input map[string]any) map[string]any {
	if len(input) == 0 {
		return nil
	}

	cloned := make(map[string]any, len(input))
	for key, value := range input {
		cloned[key] = value
	}
	return cloned
}

// applyMetadataPatch merges into the existing map by default. When Replace is true the
// entire blob is replaced, which keeps repository semantics explicit at call sites.
func applyMetadataPatch(field string, patch *platformrepo.MetadataPatch, current map[string]any, set bson.M, unset bson.M) map[string]any {
	if patch == nil {
		return cloneMap(current)
	}

	if patch.Replace {
		if len(patch.Values) == 0 {
			unset[field] = ""
			return nil
		}

		replacement := cloneMap(patch.Values)
		set[field] = replacement
		return replacement
	}

	next := cloneMap(current)
	if next == nil {
		next = map[string]any{}
	}
	for key, value := range patch.Values {
		normalizedKey := strings.TrimSpace(key)
		if normalizedKey == "" {
			continue
		}
		next[normalizedKey] = value
		set[field+"."+normalizedKey] = value
	}

	if len(next) == 0 {
		return nil
	}
	return next
}

func mapMongoError(err error) error {
	if err == nil {
		return nil
	}
	if errors.Is(err, mongo.ErrNoDocuments) {
		return errors.Join(platformrepo.ErrNotFound, err)
	}
	if mongo.IsDuplicateKeyError(err) {
		return errors.Join(platformrepo.ErrAlreadyExists, err)
	}
	if isMongoWriteConflict(err) {
		return errors.Join(platformrepo.ErrConflict, err)
	}
	return err
}

func isMongoWriteConflict(err error) bool {
	var commandError mongo.CommandError
	if errors.As(err, &commandError) && commandError.Code == 112 {
		return true
	}

	var writeException mongo.WriteException
	if errors.As(err, &writeException) {
		for _, writeError := range writeException.WriteErrors {
			if writeError.Code == 112 {
				return true
			}
		}
	}

	var bulkWriteException mongo.BulkWriteException
	if errors.As(err, &bulkWriteException) {
		for _, writeError := range bulkWriteException.WriteErrors {
			if writeError.Code == 112 {
				return true
			}
		}
	}

	return false
}

func preconditionFailed(format string, arguments ...any) error {
	return fmt.Errorf(format+": %w", append(arguments, platformrepo.ErrPreconditionFailed)...)
}

func invalidTransition(format string, arguments ...any) error {
	return fmt.Errorf(format+": %w", append(arguments, platformrepo.ErrInvalidTransition)...)
}

func immutableState(format string, arguments ...any) error {
	return fmt.Errorf(format+": %w", append(arguments, platformrepo.ErrImmutableState)...)
}
