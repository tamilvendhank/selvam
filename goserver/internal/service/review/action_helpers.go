package review

import (
	"errors"
	"fmt"
	"math"
	"strings"
	"time"

	domaincommon "goserver/internal/domain/common"
	domainposition "goserver/internal/domain/position"
	domainreview "goserver/internal/domain/review"
	domainthesis "goserver/internal/domain/thesis"
	platformrepo "goserver/internal/platform/repository"
	servicecommon "goserver/internal/service/common"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

var errReviewServiceSkipped = errors.New("review service skipped")

type mapActionOptions struct {
	ReviewID              primitive.ObjectID
	WorkflowRunID         primitive.ObjectID
	BookType              domaincommon.BookType
	Mode                  ActionMappingMode
	DryRun                bool
	Force                 bool
	InitiatedBy           string
	CorrelationID         string
	TreatIneligibleAsSkip bool
	PreloadedReview       *domainreview.CompanyReview
}

type actionMappingOutcome struct {
	ReviewID          primitive.ObjectID
	CompanyID         primitive.ObjectID
	WorkflowRunID     primitive.ObjectID
	ActionType        domaincommon.InvestingActionType
	BucketAfterAction domaincommon.WatchlistBucket
	CapitalEligible   bool
	PriorityScore     float64
	Constraints       []ActionConstraint
	Mapped            bool
	Persisted         bool
	AlreadyPresent    bool
	Skipped           bool
	DryRun            bool
	Message           string
}

type scoreContext struct {
	WeightedTotal        float64
	ManagementGovernance float64
	CapitalEfficiency    float64
	Valuation            float64
	Investability        float64

	HasManagement        bool
	HasCapitalEfficiency bool
	HasValuation         bool
	HasInvestability     bool

	CoreAtOrAboveStrong int
	CoreBelowFloor      int
	CoreBelowWeak       int
	MinCoreScore        float64
	MissingCoreSections []string
}

type positionContext struct {
	Owned        bool
	PositionPct  float64
	TargetPct    float64
	MaxPct       float64
	UnderTarget  bool
	OverTarget   bool
	AtOrAboveMax bool
}

type actionReasonBuilder struct {
	reasons     []string
	constraints []ActionConstraint
}

func defaultActionMappingConfig() ActionMappingConfig {
	return ActionMappingConfig{
		DefaultMaxReviews:           defaultActionMappingMaxReviews,
		MaxPageSize:                 maxActionMappingPageSize,
		ExceptionalMin:              8.5,
		StrongMin:                   7.5,
		AcceptableMin:               6.5,
		WeakMin:                     5.5,
		BuyMinOverall:               7.5,
		BuyMinManagementGovernance:  7.0,
		BuyMinCapitalEfficiency:     7.0,
		BuyMinValuation:             6.0,
		CoreStrongThreshold:         7.0,
		CoreFloorThreshold:          6.5,
		CoreWeakThreshold:           5.5,
		MinStrongCoreSectionsForBuy: 3,
		MaxWeakCoreSectionsForBuy:   1,
		HoldMinOverall:              7.0,
		RejectBelowOverall:          6.0,
		SellBelowOverall:            5.5,
		ExitReviewTotalDrop:         1.0,
		ExitReviewCoreDrop:          1.5,
		ExitReviewManagementDrop:    1.0,
		RequireWrittenThesisForBuy:  true,
		SellOnThesisBreak:           true,
		DefaultTargetPositionPct:    5.0,
		DefaultMaxPositionPct:       10.0,
	}
}

func normalizeActionMappingConfig(config ActionMappingConfig) ActionMappingConfig {
	defaults := defaultActionMappingConfig()
	if config.DefaultMaxReviews <= 0 {
		config.DefaultMaxReviews = defaults.DefaultMaxReviews
	}
	if config.MaxPageSize <= 0 {
		config.MaxPageSize = defaults.MaxPageSize
	}
	applyDefaultFloat(&config.ExceptionalMin, defaults.ExceptionalMin)
	applyDefaultFloat(&config.StrongMin, defaults.StrongMin)
	applyDefaultFloat(&config.AcceptableMin, defaults.AcceptableMin)
	applyDefaultFloat(&config.WeakMin, defaults.WeakMin)
	applyDefaultFloat(&config.BuyMinOverall, defaults.BuyMinOverall)
	applyDefaultFloat(&config.BuyMinManagementGovernance, defaults.BuyMinManagementGovernance)
	applyDefaultFloat(&config.BuyMinCapitalEfficiency, defaults.BuyMinCapitalEfficiency)
	applyDefaultFloat(&config.BuyMinValuation, defaults.BuyMinValuation)
	applyDefaultFloat(&config.CoreStrongThreshold, defaults.CoreStrongThreshold)
	applyDefaultFloat(&config.CoreFloorThreshold, defaults.CoreFloorThreshold)
	applyDefaultFloat(&config.CoreWeakThreshold, defaults.CoreWeakThreshold)
	applyDefaultInt(&config.MinStrongCoreSectionsForBuy, defaults.MinStrongCoreSectionsForBuy)
	applyDefaultInt(&config.MaxWeakCoreSectionsForBuy, defaults.MaxWeakCoreSectionsForBuy)
	applyDefaultFloat(&config.HoldMinOverall, defaults.HoldMinOverall)
	applyDefaultFloat(&config.RejectBelowOverall, defaults.RejectBelowOverall)
	applyDefaultFloat(&config.SellBelowOverall, defaults.SellBelowOverall)
	applyDefaultFloat(&config.ExitReviewTotalDrop, defaults.ExitReviewTotalDrop)
	applyDefaultFloat(&config.ExitReviewCoreDrop, defaults.ExitReviewCoreDrop)
	applyDefaultFloat(&config.ExitReviewManagementDrop, defaults.ExitReviewManagementDrop)
	applyDefaultFloat(&config.DefaultTargetPositionPct, defaults.DefaultTargetPositionPct)
	applyDefaultFloat(&config.DefaultMaxPositionPct, defaults.DefaultMaxPositionPct)
	return config
}

func validateActionEligibleReview(review *domainreview.CompanyReview, options mapActionOptions) error {
	if review == nil {
		return fmt.Errorf("review is required")
	}
	if review.ID.IsZero() {
		return fmt.Errorf("review id is required")
	}
	if review.CompanyID.IsZero() {
		return fmt.Errorf("review %s companyId is required", review.ID.Hex())
	}
	if !options.WorkflowRunID.IsZero() && !review.WorkflowRunID.IsZero() && review.WorkflowRunID != options.WorkflowRunID {
		return fmt.Errorf("%w: workflowRunId filter does not match", errReviewServiceSkipped)
	}
	if options.BookType != "" && review.BookType != options.BookType {
		return fmt.Errorf("%w: bookType filter does not match", errReviewServiceSkipped)
	}
	if review.BookType != domaincommon.BookTypeInvesting {
		return fmt.Errorf("%w: action mapping only applies to investing reviews", errReviewServiceSkipped)
	}
	if review.ReviewLifecycleState == domaincommon.ReviewLifecycleStateSuperseded ||
		review.ReviewStatus == domaincommon.ReviewStatusSuperseded {
		return fmt.Errorf("%w: superseded review cannot be action mapped", errReviewServiceSkipped)
	}
	if !isReviewMaterializedForAction(review) {
		return fmt.Errorf("%w: review is not finalized or AI-validated", errReviewServiceSkipped)
	}
	if len(review.Sections) == 0 {
		return fmt.Errorf("review %s has no scorecard sections", review.ID.Hex())
	}
	if review.WeightedTotalScore <= 0 {
		return fmt.Errorf("review %s weightedTotalScore is required", review.ID.Hex())
	}
	return nil
}

func isReviewMaterializedForAction(review *domainreview.CompanyReview) bool {
	if review == nil {
		return false
	}
	switch review.ReviewLifecycleState {
	case domaincommon.ReviewLifecycleStateAIValidated:
		return true
	case domaincommon.ReviewLifecycleStateFinalized:
		return review.ReviewStatus == domaincommon.ReviewStatusFinal
	default:
		return false
	}
}

func isMutableValidatedReview(review *domainreview.CompanyReview) bool {
	return review != nil &&
		review.ReviewLifecycleState == domaincommon.ReviewLifecycleStateAIValidated &&
		review.ReviewStatus == domaincommon.ReviewStatusDraft
}

func extractScoreContext(review *domainreview.CompanyReview, config ActionMappingConfig) scoreContext {
	ctx := scoreContext{
		WeightedTotal: review.WeightedTotalScore,
		MinCoreScore:  math.MaxFloat64,
	}
	ctx.ManagementGovernance, ctx.HasManagement = getSectionScore(review, domaincommon.SectionNameManagementGovernance)
	ctx.CapitalEfficiency, ctx.HasCapitalEfficiency = getSectionScore(review, domaincommon.SectionNameCapitalEfficiencyFinancialStrength)
	ctx.Valuation, ctx.HasValuation = getSectionScore(review, domaincommon.SectionNameValuationEntryAttractiveness)
	ctx.Investability, ctx.HasInvestability = getSectionScore(review, domaincommon.SectionNameInvestability)
	for _, sectionName := range domaincommon.InvestingCoreSections {
		score, ok := getSectionScore(review, sectionName)
		if !ok {
			ctx.MissingCoreSections = append(ctx.MissingCoreSections, string(sectionName))
			continue
		}
		if score >= config.CoreStrongThreshold {
			ctx.CoreAtOrAboveStrong++
		}
		if score < config.CoreFloorThreshold {
			ctx.CoreBelowFloor++
		}
		if score < config.CoreWeakThreshold {
			ctx.CoreBelowWeak++
		}
		if score < ctx.MinCoreScore {
			ctx.MinCoreScore = score
		}
	}
	if ctx.MinCoreScore == math.MaxFloat64 {
		ctx.MinCoreScore = 0
	}
	return ctx
}

func extractPositionContext(
	review *domainreview.CompanyReview,
	position *domainposition.CurrentPosition,
	config ActionMappingConfig,
) positionContext {
	ctx := positionContext{
		Owned:     review.OwnedBeforeReview,
		TargetPct: config.DefaultTargetPositionPct,
		MaxPct:    config.DefaultMaxPositionPct,
	}
	if review.PositionSnapshot != nil {
		ctx.Owned = ctx.Owned || review.PositionSnapshot.IsOwned
		ctx.PositionPct = review.PositionSnapshot.PositionPctOfBook
		if review.PositionSnapshot.TargetPositionPct > 0 {
			ctx.TargetPct = review.PositionSnapshot.TargetPositionPct
		}
		if review.PositionSnapshot.MaxPositionPct > 0 {
			ctx.MaxPct = review.PositionSnapshot.MaxPositionPct
		}
		ctx.UnderTarget = review.PositionSnapshot.UnderweightVsTargetPct > 0
		ctx.OverTarget = review.PositionSnapshot.OverweightVsTargetPct > 0
	}
	if position != nil && position.IsOpen {
		ctx.Owned = true
		if ctx.PositionPct <= 0 {
			ctx.PositionPct = position.CurrentPositionPctOfBook
		}
	}
	if ctx.TargetPct > 0 && ctx.PositionPct > 0 {
		ctx.UnderTarget = ctx.UnderTarget || ctx.PositionPct < ctx.TargetPct*0.8
		ctx.OverTarget = ctx.OverTarget || ctx.PositionPct > ctx.TargetPct*1.15
	}
	if ctx.MaxPct > 0 && ctx.PositionPct > 0 {
		ctx.AtOrAboveMax = ctx.PositionPct >= ctx.MaxPct*0.98
	}
	return ctx
}

func getSectionScore(review *domainreview.CompanyReview, sectionName domaincommon.SectionName) (float64, bool) {
	section := findSection(review, sectionName)
	if section == nil {
		return 0, false
	}
	return section.SectionScoreRaw, true
}

func findSection(review *domainreview.CompanyReview, sectionName domaincommon.SectionName) *domainreview.SectionScore {
	if review == nil {
		return nil
	}
	for index := range review.Sections {
		if review.Sections[index].SectionName == sectionName {
			return &review.Sections[index]
		}
	}
	return nil
}

func hasActionCap(review *domainreview.CompanyReview, cap domaincommon.SectionActionCap) bool {
	if review == nil {
		return false
	}
	for _, section := range review.Sections {
		if section.SectionActionCap == cap {
			return true
		}
	}
	return false
}

func investabilityTooWeak(review *domainreview.CompanyReview, config ActionMappingConfig) bool {
	score, ok := getSectionScore(review, domaincommon.SectionNameInvestability)
	return ok && score < config.WeakMin
}

func isThesisBroken(thesis *domainthesis.InvestmentThesis, review *domainreview.CompanyReview) bool {
	if thesis != nil && thesis.ThesisStatus == domaincommon.ThesisStatusBroken {
		return true
	}
	return changeLogIndicatesThesisBreak(review)
}

func isThesisUnderReview(thesis *domainthesis.InvestmentThesis, review *domainreview.CompanyReview) bool {
	if thesis != nil && thesis.ThesisStatus == domaincommon.ThesisStatusUnderReview {
		return true
	}
	if review != nil && review.ChangeLog != nil {
		return strings.EqualFold(strings.TrimSpace(review.ChangeLog.ThesisStatusChange), string(domaincommon.ThesisStatusUnderReview))
	}
	return false
}

func hasActiveThesis(thesis *domainthesis.InvestmentThesis) bool {
	return thesis != nil &&
		thesis.ThesisStatus == domaincommon.ThesisStatusActive &&
		strings.TrimSpace(thesis.ThesisSummary) != ""
}

func requiresExitReview(review *domainreview.CompanyReview) bool {
	return review != nil && review.ChangeLog != nil && review.ChangeLog.RequiresExitReview
}

func totalScoreDropped(review *domainreview.CompanyReview, threshold float64) bool {
	return review != nil &&
		review.ChangeLog != nil &&
		review.ChangeLog.WeightedTotalScoreChange <= -threshold
}

func anyCoreSectionDropped(review *domainreview.CompanyReview, threshold float64) bool {
	if review == nil || review.ChangeLog == nil {
		return false
	}
	for key, change := range review.ChangeLog.SectionScoreChanges {
		if change <= -threshold && isCoreSectionKey(key) {
			return true
		}
	}
	return false
}

func managementGovernanceDropped(review *domainreview.CompanyReview, threshold float64) bool {
	if review == nil || review.ChangeLog == nil {
		return false
	}
	target := normalizeKey(string(domaincommon.SectionNameManagementGovernance))
	for key, change := range review.ChangeLog.SectionScoreChanges {
		if normalizeKey(key) == target && change <= -threshold {
			return true
		}
	}
	return false
}

func hasMajorNegativeChanges(review *domainreview.CompanyReview) bool {
	return review != nil &&
		review.ChangeLog != nil &&
		len(nonBlankStrings(review.ChangeLog.MajorNegativeChanges)) > 0
}

func valuationExtremeWithBusinessSoftening(review *domainreview.CompanyReview) bool {
	if review == nil || review.ChangeLog == nil {
		return false
	}
	valuation := strings.ToLower(review.ChangeLog.ValuationStateChange)
	if !strings.Contains(valuation, "extreme") && !strings.Contains(valuation, "overvalu") && !strings.Contains(valuation, "expensive") {
		return false
	}
	return totalScoreDropped(review, 0.5) || anyCoreSectionDropped(review, 0.5) || hasMajorNegativeChanges(review)
}

func weakeningDetected(review *domainreview.CompanyReview, config ActionMappingConfig) bool {
	return totalScoreDropped(review, config.ExitReviewTotalDrop) ||
		anyCoreSectionDropped(review, config.ExitReviewCoreDrop) ||
		managementGovernanceDropped(review, config.ExitReviewManagementDrop) ||
		hasMajorNegativeChanges(review) ||
		requiresExitReview(review)
}

func hasNegativeEvidenceForSection(review *domainreview.CompanyReview, sectionName domaincommon.SectionName) bool {
	section := findSection(review, sectionName)
	if section != nil && (len(nonBlankStrings(section.SectionWeaknesses)) > 0 || len(nonBlankStrings(section.SectionRisks)) > 0) {
		return true
	}
	if review == nil || review.ChangeLog == nil {
		return false
	}
	sectionKey := normalizeKey(string(sectionName))
	for _, change := range review.ChangeLog.MajorNegativeChanges {
		if strings.Contains(normalizeKey(change), sectionKey) {
			return true
		}
	}
	return false
}

func hasWorseningRiskForSection(review *domainreview.CompanyReview, sectionName domaincommon.SectionName) bool {
	return hasNegativeEvidenceForSection(review, sectionName) || sectionDropped(review, sectionName, 0.5)
}

func structuralBusinessDeteriorationVisible(review *domainreview.CompanyReview) bool {
	if review == nil || review.ChangeLog == nil {
		return false
	}
	for _, change := range review.ChangeLog.MajorNegativeChanges {
		normalized := normalizeKey(change)
		if strings.Contains(normalized, "structural") ||
			strings.Contains(normalized, "deterioration") ||
			strings.Contains(normalized, "demand") ||
			strings.Contains(normalized, "competitive") {
			return true
		}
	}
	return false
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
	for _, phrase := range []string{
		"without thesis break",
		"no thesis break",
		"not a thesis break",
		"thesis stays intact",
		"thesis remains intact",
		"thesis intact",
	} {
		if strings.Contains(lower, phrase) {
			return false
		}
	}
	for _, phrase := range []string{
		"thesis break",
		"thesis broken",
		"breaks the thesis",
		"broken thesis",
		"invalidates thesis",
		"invalidates the thesis",
		"thesis no longer",
		"thesis impaired",
	} {
		if strings.Contains(lower, phrase) {
			return true
		}
	}
	return false
}

func sectionDropped(review *domainreview.CompanyReview, sectionName domaincommon.SectionName, threshold float64) bool {
	if review == nil || review.ChangeLog == nil {
		return false
	}
	target := normalizeKey(string(sectionName))
	for key, change := range review.ChangeLog.SectionScoreChanges {
		if normalizeKey(key) == target && change <= -threshold {
			return true
		}
	}
	return false
}

func isCoreSectionKey(key string) bool {
	normalized := normalizeKey(key)
	for _, section := range domaincommon.InvestingCoreSections {
		if normalized == normalizeKey(string(section)) {
			return true
		}
	}
	return false
}

func recommendedPositionBounds(review *domainreview.CompanyReview, config ActionMappingConfig) (float64, float64) {
	target := config.DefaultTargetPositionPct
	capPct := config.DefaultMaxPositionPct
	if review != nil && review.PositionSnapshot != nil {
		if review.PositionSnapshot.TargetPositionPct > 0 {
			target = review.PositionSnapshot.TargetPositionPct
		}
		if review.PositionSnapshot.MaxPositionPct > 0 {
			capPct = review.PositionSnapshot.MaxPositionPct
		}
	}
	return target, capPct
}

func computeCapitalPriorityScore(
	review *domainreview.CompanyReview,
	action domaincommon.InvestingActionType,
	capitalEligible bool,
	builder *actionReasonBuilder,
	config ActionMappingConfig,
) float64 {
	if action != domaincommon.InvestingActionTypeBuy {
		return 0
	}
	score := review.WeightedTotalScore
	if valuation, ok := getSectionScore(review, domaincommon.SectionNameValuationEntryAttractiveness); ok {
		if valuation >= 7 {
			score += 0.3
		}
		if valuation < config.BuyMinValuation {
			score -= 0.5
		}
	}
	strongCore := extractScoreContext(review, config).CoreAtOrAboveStrong
	score += math.Min(float64(strongCore)*0.1, 0.4)
	if builder.hasConstraint("requires_written_thesis") {
		score -= 0.5
	}
	if weakeningDetected(review, config) {
		score -= 0.4
	}
	if !capitalEligible {
		score -= 1.0
	}
	return roundToTenth(clampScore(score))
}

func priorityRankForAction(action domaincommon.InvestingActionType) int {
	switch action {
	case domaincommon.InvestingActionTypeBuy:
		return 1
	case domaincommon.InvestingActionTypeHold:
		return 2
	case domaincommon.InvestingActionTypeWatch:
		return 3
	case domaincommon.InvestingActionTypeTrim:
		return 4
	case domaincommon.InvestingActionTypeSell:
		return 5
	default:
		return 6
	}
}

func trancheStyleForAction(action domaincommon.InvestingActionType) domaincommon.RecommendedTrancheStyle {
	switch action {
	case domaincommon.InvestingActionTypeBuy:
		return domaincommon.RecommendedTrancheStyleStart
	case domaincommon.InvestingActionTypeTrim:
		return domaincommon.RecommendedTrancheStyleReduce
	case domaincommon.InvestingActionTypeSell:
		return domaincommon.RecommendedTrancheStyleExit
	default:
		return domaincommon.RecommendedTrancheStylePause
	}
}

func newActionReasonBuilder() *actionReasonBuilder {
	return &actionReasonBuilder{}
}

func (builder *actionReasonBuilder) addReason(reason string) {
	reason = strings.TrimSpace(reason)
	if reason == "" {
		return
	}
	for _, existing := range builder.reasons {
		if existing == reason {
			return
		}
	}
	builder.reasons = append(builder.reasons, reason)
}

func (builder *actionReasonBuilder) addReasons(reasons ...string) {
	for _, reason := range reasons {
		builder.addReason(reason)
	}
}

func (builder *actionReasonBuilder) addConstraint(code string, message string, blocking bool) {
	builder.addConstraints(actionConstraint(code, message, blocking))
}

func (builder *actionReasonBuilder) addConstraints(constraints ...ActionConstraint) {
	for _, constraint := range constraints {
		if strings.TrimSpace(constraint.Code) == "" {
			continue
		}
		already := false
		for _, existing := range builder.constraints {
			if existing.Code == constraint.Code {
				already = true
				break
			}
		}
		if !already {
			builder.constraints = append(builder.constraints, constraint)
		}
	}
}

func (builder *actionReasonBuilder) hasBlockingConstraint() bool {
	for _, constraint := range builder.constraints {
		if constraint.Blocking {
			return true
		}
	}
	return false
}

func (builder *actionReasonBuilder) hasConstraint(code string) bool {
	for _, constraint := range builder.constraints {
		if constraint.Code == code {
			return true
		}
	}
	return false
}

func actionConstraint(code string, message string, blocking bool) ActionConstraint {
	message = strings.TrimSpace(message)
	if message == "" {
		message = code
	}
	return ActionConstraint{Code: code, Message: message, Blocking: blocking}
}

func constraintCodes(constraints []ActionConstraint) []string {
	codes := make([]string, 0, len(constraints))
	for _, constraint := range constraints {
		if strings.TrimSpace(constraint.Code) != "" {
			codes = append(codes, constraint.Code)
		}
	}
	return codes
}

func isReviewServiceSkip(err error) bool {
	return errors.Is(err, errReviewServiceSkipped)
}

func trimSkipMessage(err error) string {
	message := strings.TrimSpace(err.Error())
	return strings.TrimPrefix(message, errReviewServiceSkipped.Error()+": ")
}

func isRepositoryNotFound(err error) bool {
	return errors.Is(err, platformrepo.ErrNotFound)
}

func mutationMetadata(at time.Time, actor string, reason string) platformrepo.MutationMetadata {
	return platformrepo.MutationMetadata{OccurredAt: at.UTC(), Actor: actor, Reason: reason}
}

func invalidReviewServiceRequestf(format string, args ...any) error {
	return fmt.Errorf("%w: %s", servicecommon.ErrInvalidServiceRequest, fmt.Sprintf(format, args...))
}
