package allocation

import (
	"context"
	"errors"
	"fmt"
	"math"
	"sort"
	"strings"
	"time"

	domaincommon "goserver/internal/domain/common"
	domainposition "goserver/internal/domain/position"
	domainreview "goserver/internal/domain/review"
	domainthesis "goserver/internal/domain/thesis"
	platformrepo "goserver/internal/platform/repository"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

type candidateEligibility struct {
	Eligible bool
	Code     string
	Reasons  []string
}

type candidateScoreContext struct {
	WeightedTotal        float64
	ManagementGovernance float64
	CapitalEfficiency    float64
	Valuation            float64
	MarketConfirmation   float64

	ValidWeightedTotal      bool
	HasManagementGovernance bool
	HasCapitalEfficiency    bool
	HasValuation            bool
	HasMarketConfirmation   bool

	CoreAtOrAboveStrong int
	CoreBelowFloor      int
	MinCoreScore        float64
	MissingCoreSections []string
}

type candidatePositionContext struct {
	Owned              bool
	CurrentPct         float64
	TargetPct          float64
	MaxPct             float64
	GapToTargetPct     float64
	GapToMaxPct        float64
	CurrentMarketValue float64
	HasMarketValue     bool
}

type capitalCandidate struct {
	Ref           CapitalCandidateRef
	Review        *domainreview.CompanyReview
	Score         candidateScoreContext
	Position      candidatePositionContext
	ReviewDate    time.Time
	AsOfDate      time.Time
	RankingReason string
}

func baseCandidateRef(review *domainreview.CompanyReview) CapitalCandidateRef {
	if review == nil {
		return CapitalCandidateRef{}
	}
	return CapitalCandidateRef{
		CompanyID:            review.CompanyID,
		ReviewID:             review.ID,
		WorkflowRunID:        review.WorkflowRunID,
		Symbol:               review.Symbol,
		ActionType:           candidateAction(review),
		CurrentBucket:        candidateBucket(review),
		RecommendedTargetPct: 0,
	}
}

func extractCandidateScoreContext(review *domainreview.CompanyReview, config CapitalAllocationConfig) candidateScoreContext {
	score := candidateScoreContext{MinCoreScore: math.MaxFloat64}
	if review == nil {
		return score
	}
	score.WeightedTotal = review.WeightedTotalScore
	score.ValidWeightedTotal = review.WeightedTotalScore > 0 && review.WeightedTotalScore <= 10

	score.ManagementGovernance, score.HasManagementGovernance = getSectionScore(review, domaincommon.SectionNameManagementGovernance)
	score.CapitalEfficiency, score.HasCapitalEfficiency = getSectionScore(review, domaincommon.SectionNameCapitalEfficiencyFinancialStrength)
	score.Valuation, score.HasValuation = getSectionScore(review, domaincommon.SectionNameValuationEntryAttractiveness)
	score.MarketConfirmation, score.HasMarketConfirmation = getSectionScore(review, domaincommon.SectionNameMarketConfirmation)

	for _, sectionName := range domaincommon.InvestingCoreSections {
		value, ok := getSectionScore(review, sectionName)
		if !ok {
			score.MissingCoreSections = append(score.MissingCoreSections, string(sectionName))
			continue
		}
		if value >= config.CoreStrongThreshold {
			score.CoreAtOrAboveStrong++
		}
		if value < config.CoreFloorThreshold {
			score.CoreBelowFloor++
		}
		if value < score.MinCoreScore {
			score.MinCoreScore = value
		}
	}
	if score.MinCoreScore == math.MaxFloat64 {
		score.MinCoreScore = 0
	}
	sort.Strings(score.MissingCoreSections)
	return score
}

func loadPositionContext(
	ctx context.Context,
	positions platformrepo.CurrentPositionRepository,
	review *domainreview.CompanyReview,
	config CapitalAllocationConfig,
) (candidatePositionContext, error) {
	if review == nil {
		return candidatePositionContext{}, nil
	}

	var projected *domainposition.CurrentPosition
	var loadErr error
	if positions != nil && !review.CompanyID.IsZero() {
		projected, loadErr = positions.GetByCompanyAndBook(ctx, review.CompanyID, domaincommon.BookTypeInvesting)
		if errors.Is(loadErr, platformrepo.ErrNotFound) {
			loadErr = nil
		}
	}

	currentPct, marketValue, hasMarketValue := positionSnapshotValues(review)
	if projected != nil {
		if projected.IsOpen {
			currentPct = projected.CurrentPositionPctOfBook
			marketValue = projected.CurrentMarketValue
			hasMarketValue = projected.CurrentMarketValue > 0
		} else {
			currentPct = 0
			marketValue = 0
			hasMarketValue = false
		}
	}

	score := extractCandidateScoreContext(review, config)
	targetPct := computeTargetPositionPct(review, score, config)
	maxPct := computeMaxPositionPct(review, targetPct, config)
	position := computePositionGap(currentPct, targetPct, maxPct)
	position.CurrentMarketValue = marketValue
	position.HasMarketValue = hasMarketValue
	position.Owned = currentPct > scoreEpsilon
	return position, loadErr
}

func loadThesisContext(
	ctx context.Context,
	theses platformrepo.InvestmentThesisRepository,
	companyID primitive.ObjectID,
) (*domainthesis.InvestmentThesis, bool, error) {
	if theses == nil || companyID.IsZero() {
		return nil, false, nil
	}

	active, err := theses.GetActiveByCompanyID(ctx, companyID)
	if err == nil {
		return active, active != nil, nil
	}
	if !errors.Is(err, platformrepo.ErrNotFound) {
		return nil, false, err
	}

	latest, latestErr := theses.GetLatestByCompanyID(ctx, companyID)
	if latestErr == nil {
		return latest, latest != nil && latest.ThesisStatus == domaincommon.ThesisStatusActive, nil
	}
	if errors.Is(latestErr, platformrepo.ErrNotFound) {
		return nil, false, nil
	}
	return nil, false, latestErr
}

func getSectionScore(review *domainreview.CompanyReview, sectionName domaincommon.SectionName) (float64, bool) {
	if review == nil {
		return 0, false
	}
	for _, section := range review.Sections {
		if section.SectionName == sectionName {
			return section.SectionScoreRaw, true
		}
	}
	return 0, false
}

func positionSnapshotValues(review *domainreview.CompanyReview) (float64, float64, bool) {
	if review == nil || review.PositionSnapshot == nil {
		return 0, 0, false
	}
	snapshot := review.PositionSnapshot
	return snapshot.PositionPctOfBook, snapshot.MarketValue, snapshot.MarketValue > 0
}

func candidateAction(review *domainreview.CompanyReview) domaincommon.InvestingActionType {
	if review == nil {
		return ""
	}
	if review.FinalActionAfterReview == domaincommon.InvestingActionTypeBuy {
		return domaincommon.InvestingActionTypeBuy
	}
	if review.DecisionAction != nil && review.DecisionAction.ActionType == domaincommon.InvestingActionTypeBuy {
		return domaincommon.InvestingActionTypeBuy
	}
	if review.FinalActionAfterReview != "" {
		return review.FinalActionAfterReview
	}
	if review.DecisionAction != nil {
		return review.DecisionAction.ActionType
	}
	return ""
}

func candidateBucket(review *domainreview.CompanyReview) domaincommon.WatchlistBucket {
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

func rankCandidates(candidates []capitalCandidate) {
	sort.SliceStable(candidates, func(i, j int) bool {
		left := candidates[i]
		right := candidates[j]
		if !nearlyEqual(left.Ref.PriorityScore, right.Ref.PriorityScore) {
			return left.Ref.PriorityScore > right.Ref.PriorityScore
		}
		if !nearlyEqual(left.Score.WeightedTotal, right.Score.WeightedTotal) {
			return left.Score.WeightedTotal > right.Score.WeightedTotal
		}
		if !nearlyEqual(left.Score.Valuation, right.Score.Valuation) {
			return left.Score.Valuation > right.Score.Valuation
		}
		if !nearlyEqual(left.Position.GapToTargetPct, right.Position.GapToTargetPct) {
			return left.Position.GapToTargetPct > right.Position.GapToTargetPct
		}
		if !left.ReviewDate.Equal(right.ReviewDate) {
			return left.ReviewDate.After(right.ReviewDate)
		}
		return left.Ref.CompanyID.Hex() < right.Ref.CompanyID.Hex()
	})
	for index := range candidates {
		candidates[index].Ref.PriorityRank = index + 1
	}
}

func buildCandidateReason(
	review *domainreview.CompanyReview,
	score candidateScoreContext,
	position candidatePositionContext,
	priorityScore float64,
) string {
	symbol := ""
	if review != nil {
		symbol = review.Symbol
	}
	return fmt.Sprintf(
		"%s priority %.1f from score %.1f, valuation %.1f, target gap %.2f%%",
		strings.TrimSpace(symbol),
		priorityScore,
		score.WeightedTotal,
		score.Valuation,
		position.GapToTargetPct,
	)
}

func inferBookValue(candidate capitalCandidate) (float64, bool) {
	if candidate.Position.CurrentPct <= scoreEpsilon || !candidate.Position.HasMarketValue || candidate.Position.CurrentMarketValue <= 0 {
		return 0, false
	}
	bookValue := candidate.Position.CurrentMarketValue / (candidate.Position.CurrentPct / 100)
	if bookValue <= 0 || math.IsNaN(bookValue) || math.IsInf(bookValue, 0) {
		return 0, false
	}
	return bookValue, true
}

func sanePercentage(value float64) bool {
	return value > 0 && value <= 100
}

func normalizeConstraintCode(value string) string {
	normalized := strings.ToLower(strings.TrimSpace(value))
	normalized = strings.ReplaceAll(normalized, "-", "_")
	normalized = strings.ReplaceAll(normalized, " ", "_")
	for strings.Contains(normalized, "__") {
		normalized = strings.ReplaceAll(normalized, "__", "_")
	}
	return normalized
}

func joinReasons(reasons []string) string {
	cleaned := make([]string, 0, len(reasons))
	for _, reason := range reasons {
		if trimmed := strings.TrimSpace(reason); trimmed != "" {
			cleaned = append(cleaned, trimmed)
		}
	}
	return strings.Join(cleaned, "; ")
}

func applyPositiveFloat(target *float64, value float64) {
	if value > 0 {
		*target = value
	}
}
