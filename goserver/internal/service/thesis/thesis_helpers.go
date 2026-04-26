package thesis

import (
	"context"
	"errors"
	"fmt"
	"strings"

	domaincommon "goserver/internal/domain/common"
	domainreview "goserver/internal/domain/review"
	domainthesis "goserver/internal/domain/thesis"
	platformrepo "goserver/internal/platform/repository"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

var errThesisSkipped = errors.New("thesis evaluation skipped")

type thesisEvaluationOptions struct {
	ReviewID        primitive.ObjectID
	CompanyID       primitive.ObjectID
	WorkflowRunID   primitive.ObjectID
	BookType        domaincommon.BookType
	DryRun          bool
	Force           bool
	InitiatedBy     string
	CorrelationID   string
	PreloadedReview *domainreview.CompanyReview
}

type thesisDecision struct {
	Status                   domaincommon.ThesisStatus
	HealthScore              float64
	PositionRole             domaincommon.PositionRole
	ThesisChangeSummary      string
	NewSupportingEvidence    []string
	NewContradictingEvidence []string
	BreakSignals             []string
	Summary                  string
}

type thesisOneOutcome struct {
	ReviewID     primitive.ObjectID
	CompanyID    primitive.ObjectID
	ThesisID     primitive.ObjectID
	Created      bool
	Updated      bool
	Broken       bool
	UnderReview  bool
	Skipped      bool
	DryRun       bool
	StatusBefore domaincommon.ThesisStatus
	StatusAfter  domaincommon.ThesisStatus
	HealthScore  float64
	Summary      string
}

func (service *thesisEvaluationService) loadThesisContext(
	ctx context.Context,
	options thesisEvaluationOptions,
) (*domainreview.CompanyReview, *domainthesis.InvestmentThesis, *domainthesis.InvestmentThesis, error) {
	review, err := service.loadReview(ctx, options)
	if err != nil {
		return nil, nil, nil, err
	}
	if service.theses == nil {
		return nil, nil, nil, fmt.Errorf("evaluate thesis review %s: thesis repository is required", review.ID.Hex())
	}

	active, err := service.theses.GetActiveByCompanyID(ctx, review.CompanyID)
	if err != nil && !isRepositoryNotFound(err) {
		return nil, nil, nil, fmt.Errorf("evaluate thesis review %s company %s: load active thesis: %w", review.ID.Hex(), review.CompanyID.Hex(), err)
	}
	if isRepositoryNotFound(err) {
		active = nil
	}

	existing := active
	if existing == nil {
		latest, latestErr := service.theses.GetLatestByCompanyID(ctx, review.CompanyID)
		if latestErr != nil && !isRepositoryNotFound(latestErr) {
			return nil, nil, nil, fmt.Errorf("evaluate thesis review %s company %s: load latest thesis: %w", review.ID.Hex(), review.CompanyID.Hex(), latestErr)
		}
		if !isRepositoryNotFound(latestErr) {
			existing = latest
		}
	}
	return review, existing, active, nil
}

func (service *thesisEvaluationService) loadReview(
	ctx context.Context,
	options thesisEvaluationOptions,
) (*domainreview.CompanyReview, error) {
	if options.PreloadedReview != nil {
		return options.PreloadedReview, nil
	}
	if service.reviews == nil {
		return nil, fmt.Errorf("review repository is required")
	}
	if !options.ReviewID.IsZero() {
		review, err := service.reviews.GetByID(ctx, options.ReviewID)
		if err != nil {
			return nil, fmt.Errorf("load review %s: %w", options.ReviewID.Hex(), err)
		}
		if review == nil {
			return nil, fmt.Errorf("load review %s: %w", options.ReviewID.Hex(), platformrepo.ErrNotFound)
		}
		return review, nil
	}

	bookType := options.BookType
	if bookType == "" {
		bookType = domaincommon.BookTypeInvesting
	}
	review, err := service.reviews.GetLatestByCompanyAndBook(ctx, options.CompanyID, bookType, platformrepo.LatestCompanyReviewOptions{
		FinalizedOnly:     true,
		IncludeSuperseded: false,
	})
	if err != nil {
		return nil, fmt.Errorf("load latest review for company %s book %q: %w", options.CompanyID.Hex(), bookType, err)
	}
	if review == nil {
		return nil, fmt.Errorf("load latest review for company %s book %q: %w", options.CompanyID.Hex(), bookType, platformrepo.ErrNotFound)
	}
	return review, nil
}

func validateReviewForThesis(review *domainreview.CompanyReview, options thesisEvaluationOptions) error {
	if review == nil {
		return fmt.Errorf("review is required")
	}
	if review.ID.IsZero() {
		return fmt.Errorf("review id is required for thesis evaluation")
	}
	if review.CompanyID.IsZero() {
		return fmt.Errorf("review %s companyId is required for thesis evaluation", review.ID.Hex())
	}
	if !options.CompanyID.IsZero() && review.CompanyID != options.CompanyID {
		return fmt.Errorf("%w: companyId filter does not match", errThesisSkipped)
	}
	if !options.WorkflowRunID.IsZero() && !review.WorkflowRunID.IsZero() && review.WorkflowRunID != options.WorkflowRunID {
		return fmt.Errorf("%w: workflowRunId filter does not match", errThesisSkipped)
	}
	if options.BookType != "" && review.BookType != options.BookType {
		return fmt.Errorf("%w: bookType filter does not match", errThesisSkipped)
	}
	if review.BookType != domaincommon.BookTypeInvesting {
		return fmt.Errorf("%w: thesis evaluation only applies to investing reviews", errThesisSkipped)
	}
	if review.ReviewLifecycleState == domaincommon.ReviewLifecycleStateSuperseded ||
		review.ReviewStatus == domaincommon.ReviewStatusSuperseded {
		return fmt.Errorf("%w: superseded review cannot update thesis", errThesisSkipped)
	}
	if !isReviewMaterializedForThesis(review) {
		return fmt.Errorf("%w: review is not finalized or AI-validated", errThesisSkipped)
	}
	if len(review.Sections) == 0 {
		return fmt.Errorf("review %s has no scorecard sections", review.ID.Hex())
	}
	if review.WeightedTotalScore <= 0 {
		return fmt.Errorf("review %s weightedTotalScore is required for thesis evaluation", review.ID.Hex())
	}
	if finalAction(review) == "" {
		return fmt.Errorf("review %s final action is required for thesis evaluation", review.ID.Hex())
	}
	return nil
}

func isReviewMaterializedForThesis(review *domainreview.CompanyReview) bool {
	if review == nil {
		return false
	}
	switch review.ReviewLifecycleState {
	case domaincommon.ReviewLifecycleStateFinalized:
		return review.ReviewStatus == domaincommon.ReviewStatusFinal
	case domaincommon.ReviewLifecycleStateAIValidated:
		return true
	default:
		return false
	}
}

func shouldCreateThesis(
	review *domainreview.CompanyReview,
	existing *domainthesis.InvestmentThesis,
	config ThesisEvaluationConfig,
) bool {
	return config.RequireWrittenThesisForBuy &&
		existing == nil &&
		finalAction(review) == domaincommon.InvestingActionTypeBuy
}

func shouldUpdateThesis(review *domainreview.CompanyReview, existing *domainthesis.InvestmentThesis) bool {
	if existing == nil {
		return false
	}
	switch finalAction(review) {
	case domaincommon.InvestingActionTypeBuy,
		domaincommon.InvestingActionTypeHold,
		domaincommon.InvestingActionTypeTrim,
		domaincommon.InvestingActionTypeSell,
		domaincommon.InvestingActionTypeWatch:
		return true
	default:
		return false
	}
}

func finalAction(review *domainreview.CompanyReview) domaincommon.InvestingActionType {
	if review == nil {
		return ""
	}
	if review.FinalActionAfterReview != "" {
		return review.FinalActionAfterReview
	}
	if review.DecisionAction != nil {
		return review.DecisionAction.ActionType
	}
	return ""
}

func finalBucket(review *domainreview.CompanyReview) domaincommon.WatchlistBucket {
	if review == nil {
		return ""
	}
	if review.FinalBucketAfterReview != "" {
		return review.FinalBucketAfterReview
	}
	if review.DecisionAction != nil {
		return review.DecisionAction.BucketAfterAction
	}
	return ""
}

func sectionByName(review *domainreview.CompanyReview, name domaincommon.SectionName) *domainreview.SectionScore {
	if review == nil {
		return nil
	}
	for index := range review.Sections {
		if review.Sections[index].SectionName == name {
			return &review.Sections[index]
		}
	}
	return nil
}

func sectionScore(review *domainreview.CompanyReview, name domaincommon.SectionName) float64 {
	section := sectionByName(review, name)
	if section == nil {
		return 0
	}
	return section.SectionScoreRaw
}

func coreWeakSectionCount(review *domainreview.CompanyReview, threshold float64) int {
	count := 0
	for _, name := range domaincommon.InvestingCoreSections {
		if score := sectionScore(review, name); score > 0 && score < threshold {
			count++
		}
	}
	return count
}

func coreStrongSectionCount(review *domainreview.CompanyReview, threshold float64) int {
	count := 0
	for _, name := range domaincommon.InvestingCoreSections {
		if score := sectionScore(review, name); score >= threshold {
			count++
		}
	}
	return count
}

func totalScoreDropped(review *domainreview.CompanyReview, threshold float64) bool {
	return review != nil &&
		review.ChangeLog != nil &&
		review.ChangeLog.WeightedTotalScoreChange <= -threshold
}

func coreSectionDropped(review *domainreview.CompanyReview, threshold float64) bool {
	if review == nil || review.ChangeLog == nil {
		return false
	}
	for key, change := range review.ChangeLog.SectionScoreChanges {
		if change > -threshold {
			continue
		}
		if isCoreSectionKey(key) {
			return true
		}
	}
	return false
}

func managementDropped(review *domainreview.CompanyReview, threshold float64) bool {
	return sectionDropped(review, domaincommon.SectionNameManagementGovernance, threshold)
}

func sectionDropped(review *domainreview.CompanyReview, section domaincommon.SectionName, threshold float64) bool {
	if review == nil || review.ChangeLog == nil {
		return false
	}
	target := normalizeSectionKey(string(section))
	for key, change := range review.ChangeLog.SectionScoreChanges {
		if normalizeSectionKey(key) == target && change <= -threshold {
			return true
		}
	}
	return false
}

func isCoreSectionKey(key string) bool {
	normalized := normalizeSectionKey(key)
	for _, section := range domaincommon.InvestingCoreSections {
		if normalized == normalizeSectionKey(string(section)) {
			return true
		}
	}
	return false
}

func normalizeSectionKey(value string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	var builder strings.Builder
	lastUnderscore := false
	for _, r := range value {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') {
			builder.WriteRune(r)
			lastUnderscore = false
			continue
		}
		if !lastUnderscore {
			builder.WriteByte('_')
			lastUnderscore = true
		}
	}
	return strings.Trim(builder.String(), "_")
}

func hasMajorNegativeChanges(review *domainreview.CompanyReview) bool {
	return review != nil &&
		review.ChangeLog != nil &&
		len(nonBlankStrings(review.ChangeLog.MajorNegativeChanges)) > 0
}

func hardGateIsThesisBreaking(review *domainreview.CompanyReview) bool {
	if review == nil || !review.HardGateFailed {
		return false
	}
	reasons := nonBlankStrings(review.HardGateFailureReasons)
	if len(reasons) == 0 {
		return true
	}
	for _, reason := range reasons {
		if !isValuationOnlyText(reason) {
			return true
		}
	}
	return false
}

func isValuationOnlyText(value string) bool {
	lower := strings.ToLower(value)
	return strings.Contains(lower, "valuation") ||
		strings.Contains(lower, "overvalu") ||
		strings.Contains(lower, "expensive") ||
		strings.Contains(lower, "entry")
}

func hasNegativeEvidenceForSection(review *domainreview.CompanyReview, sectionName domaincommon.SectionName) bool {
	section := sectionByName(review, sectionName)
	if section != nil && (len(nonBlankStrings(section.SectionWeaknesses)) > 0 || len(nonBlankStrings(section.SectionRisks)) > 0) {
		return true
	}
	if review == nil || review.ChangeLog == nil {
		return false
	}
	sectionKey := normalizeSectionKey(string(sectionName))
	for _, change := range review.ChangeLog.MajorNegativeChanges {
		if strings.Contains(normalizeSectionKey(change), sectionKey) {
			return true
		}
	}
	return false
}

func hasWorseningRiskForSection(review *domainreview.CompanyReview, sectionName domaincommon.SectionName) bool {
	return hasNegativeEvidenceForSection(review, sectionName) || sectionDropped(review, sectionName, 0.5)
}

func changeLogIndicatesThesisBreak(review *domainreview.CompanyReview) bool {
	if review == nil || review.ChangeLog == nil {
		return false
	}
	if strings.EqualFold(strings.TrimSpace(review.ChangeLog.ThesisStatusChange), string(domaincommon.ThesisStatusBroken)) {
		return true
	}
	if containsThesisBreakLanguage(review.ChangeLog.ChangeSummary) {
		return true
	}
	for _, change := range review.ChangeLog.MajorNegativeChanges {
		if containsThesisBreakLanguage(change) {
			return true
		}
	}
	return false
}

func containsThesisBreakLanguage(value string) bool {
	lower := strings.ToLower(strings.TrimSpace(value))
	if lower == "" {
		return false
	}
	intactPhrases := []string{
		"without thesis break",
		"no thesis break",
		"not a thesis break",
		"thesis stays intact",
		"thesis remains intact",
		"thesis intact",
	}
	for _, phrase := range intactPhrases {
		if strings.Contains(lower, phrase) {
			return false
		}
	}
	breakPhrases := []string{
		"thesis break",
		"thesis broken",
		"breaks the thesis",
		"broken thesis",
		"invalidates the thesis",
		"invalidates thesis",
		"thesis no longer",
		"thesis impaired",
	}
	for _, phrase := range breakPhrases {
		if strings.Contains(lower, phrase) {
			return true
		}
	}
	return false
}

func valuationExtremeWithBusinessSoftening(review *domainreview.CompanyReview) bool {
	if review == nil || review.ChangeLog == nil {
		return false
	}
	valuation := strings.ToLower(review.ChangeLog.ValuationStateChange)
	if !strings.Contains(valuation, "extreme") && !strings.Contains(valuation, "overvalu") && !strings.Contains(valuation, "expensive") {
		return false
	}
	return coreSectionDropped(review, 0.5) || hasMajorNegativeChanges(review)
}

func isThesisSkip(err error) bool {
	return errors.Is(err, errThesisSkipped)
}
