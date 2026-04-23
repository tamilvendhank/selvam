package review

import (
	"fmt"
	"strings"
	"time"

	"goserver/internal/domain/common"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

type EvidenceReference struct {
	ID                   primitive.ObjectID        `bson:"id,omitempty" json:"id,omitempty"`
	SourceType           common.EvidenceSourceType `bson:"sourceType" json:"sourceType"`
	SourceDate           *time.Time                `bson:"sourceDate,omitempty" json:"sourceDate,omitempty"`
	SourceTitle          string                    `bson:"sourceTitle,omitempty" json:"sourceTitle,omitempty"`
	SourcePeriod         string                    `bson:"sourcePeriod,omitempty" json:"sourcePeriod,omitempty"`
	SourceURLOrPath      string                    `bson:"sourceURLOrPath,omitempty" json:"sourceURLOrPath,omitempty"`
	ExcerptOrMetricName  string                    `bson:"excerptOrMetricName,omitempty" json:"excerptOrMetricName,omitempty"`
	ExcerptOrMetricValue string                    `bson:"excerptOrMetricValue,omitempty" json:"excerptOrMetricValue,omitempty"`
	EvidenceSummary      string                    `bson:"evidenceSummary,omitempty" json:"evidenceSummary,omitempty"`
	EvidenceDirection    common.EvidenceDirection  `bson:"evidenceDirection,omitempty" json:"evidenceDirection,omitempty"`
}

func (reference EvidenceReference) Validate() error {
	if err := common.RequireObjectID("evidenceRef.id", reference.ID); err != nil {
		return err
	}
	if !reference.SourceType.IsValid() {
		return fmt.Errorf("invalid evidence source type %q", reference.SourceType)
	}
	if !reference.EvidenceDirection.IsValid() {
		return fmt.Errorf("invalid evidence direction %q", reference.EvidenceDirection)
	}
	if reference.SourceTitle == "" && reference.SourceURLOrPath == "" && reference.ExcerptOrMetricName == "" {
		return fmt.Errorf("evidence reference must include title, url/path, or excerpt/metric name")
	}
	if reference.SourceDate != nil && reference.SourceDate.IsZero() {
		return fmt.Errorf("evidence sourceDate cannot be zero")
	}
	return nil
}

type SubScore struct {
	SubScoreName     common.SubScoreName     `bson:"subScoreName" json:"subScoreName"`
	SubScoreWeight   float64                 `bson:"subScoreWeight" json:"subScoreWeight"`
	SubScoreValue    float64                 `bson:"subScoreValue" json:"subScoreValue"`
	SubScoreSummary  string                  `bson:"subScoreSummary,omitempty" json:"subScoreSummary,omitempty"`
	TrendDirection   common.TrendDirection   `bson:"trendDirection,omitempty" json:"trendDirection,omitempty"`
	EvidenceStrength common.EvidenceStrength `bson:"evidenceStrength,omitempty" json:"evidenceStrength,omitempty"`
	MetricBasis      common.MetricBasis      `bson:"metricBasis,omitempty" json:"metricBasis,omitempty"`
	Notes            string                  `bson:"notes,omitempty" json:"notes,omitempty"`
	EvidenceRefIDs   []primitive.ObjectID    `bson:"evidenceRefIDs,omitempty" json:"evidenceRefIDs,omitempty"`
}

func (subScore SubScore) Validate() error {
	if !subScore.SubScoreName.IsValid() {
		return fmt.Errorf("invalid sub-score name %q", subScore.SubScoreName)
	}
	if err := common.ValidatePercentage("subScoreWeight", subScore.SubScoreWeight); err != nil {
		return err
	}
	if err := common.ValidateScore("subScoreValue", subScore.SubScoreValue); err != nil {
		return err
	}
	if !subScore.TrendDirection.IsValid() {
		return fmt.Errorf("invalid trendDirection %q", subScore.TrendDirection)
	}
	if !subScore.EvidenceStrength.IsValid() {
		return fmt.Errorf("invalid evidenceStrength %q", subScore.EvidenceStrength)
	}
	if !subScore.MetricBasis.IsValid() {
		return fmt.Errorf("invalid metricBasis %q", subScore.MetricBasis)
	}
	for _, evidenceRefID := range subScore.EvidenceRefIDs {
		if evidenceRefID.IsZero() {
			return fmt.Errorf("evidenceRefIDs cannot contain zero object ids")
		}
	}
	return nil
}

type SectionScore struct {
	SectionName               common.SectionName      `bson:"sectionName" json:"sectionName"`
	SectionWeight             float64                 `bson:"sectionWeight" json:"sectionWeight"`
	SectionScoreRaw           float64                 `bson:"sectionScoreRaw" json:"sectionScoreRaw"`
	SectionScoreWeighted      float64                 `bson:"sectionScoreWeighted" json:"sectionScoreWeighted"`
	SectionPassedMinimumCheck bool                    `bson:"sectionPassedMinimumCheck" json:"sectionPassedMinimumCheck"`
	SectionActionCap          common.SectionActionCap `bson:"sectionActionCap,omitempty" json:"sectionActionCap,omitempty"`
	SectionSummary            string                  `bson:"sectionSummary,omitempty" json:"sectionSummary,omitempty"`
	SectionStrengths          []string                `bson:"sectionStrengths,omitempty" json:"sectionStrengths,omitempty"`
	SectionWeaknesses         []string                `bson:"sectionWeaknesses,omitempty" json:"sectionWeaknesses,omitempty"`
	SectionRisks              []string                `bson:"sectionRisks,omitempty" json:"sectionRisks,omitempty"`
	SectionConfidenceScore    float64                 `bson:"sectionConfidenceScore" json:"sectionConfidenceScore"`
	SubScores                 []SubScore              `bson:"subScores" json:"subScores"`
	EvidenceRefs              []EvidenceReference     `bson:"evidenceRefs,omitempty" json:"evidenceRefs,omitempty"`
}

func (section SectionScore) CalculatedWeightedScore() float64 {
	return common.NormalizeWeightedScore(section.SectionScoreRaw, section.SectionWeight)
}

func (section SectionScore) Validate() error {
	if !section.SectionName.IsValid() {
		return fmt.Errorf("invalid sectionName %q", section.SectionName)
	}
	if err := common.ValidatePercentage("sectionWeight", section.SectionWeight); err != nil {
		return err
	}
	if err := common.ValidateScore("sectionScoreRaw", section.SectionScoreRaw); err != nil {
		return err
	}
	if err := common.ValidateComputedScore("sectionScoreWeighted", section.SectionScoreWeighted); err != nil {
		return err
	}
	if !common.NearlyEqual(section.CalculatedWeightedScore(), section.SectionScoreWeighted) {
		return fmt.Errorf("section weighted score does not match raw score and weight for %q", section.SectionName)
	}
	if err := common.ValidateUnitInterval("sectionConfidenceScore", section.SectionConfidenceScore); err != nil {
		return err
	}
	if !section.SectionActionCap.IsValid() {
		return fmt.Errorf("invalid sectionActionCap %q", section.SectionActionCap)
	}
	if err := common.ValidateStringSlice("sectionStrengths", section.SectionStrengths); err != nil {
		return err
	}
	if err := common.ValidateStringSlice("sectionWeaknesses", section.SectionWeaknesses); err != nil {
		return err
	}
	if err := common.ValidateStringSlice("sectionRisks", section.SectionRisks); err != nil {
		return err
	}
	if len(section.SubScores) == 0 {
		return fmt.Errorf("section %q must include sub-scores", section.SectionName)
	}

	subScoreNames := make(map[common.SubScoreName]struct{}, len(section.SubScores))
	var weightTotal float64
	for _, subScore := range section.SubScores {
		if err := subScore.Validate(); err != nil {
			return err
		}
		if _, exists := subScoreNames[subScore.SubScoreName]; exists {
			return fmt.Errorf("duplicate sub-score %q", subScore.SubScoreName)
		}
		subScoreNames[subScore.SubScoreName] = struct{}{}
		weightTotal += subScore.SubScoreWeight
	}
	if !common.NearlyEqual(weightTotal, 100) {
		return fmt.Errorf("sub-score weights must total 100 for section %q", section.SectionName)
	}

	evidenceRefIDs := make(map[primitive.ObjectID]struct{}, len(section.EvidenceRefs))
	for _, evidenceRef := range section.EvidenceRefs {
		if err := evidenceRef.Validate(); err != nil {
			return err
		}
		if _, exists := evidenceRefIDs[evidenceRef.ID]; exists {
			return fmt.Errorf("duplicate evidence reference id %q", evidenceRef.ID.Hex())
		}
		evidenceRefIDs[evidenceRef.ID] = struct{}{}
	}
	return nil
}

type DecisionAction struct {
	ActionType                   common.InvestingActionType     `bson:"actionType" json:"actionType"`
	BucketAfterAction            common.WatchlistBucket         `bson:"bucketAfterAction,omitempty" json:"bucketAfterAction,omitempty"`
	ActionPriorityRank           int                            `bson:"actionPriorityRank,omitempty" json:"actionPriorityRank,omitempty"`
	ActionReasonPrimary          string                         `bson:"actionReasonPrimary" json:"actionReasonPrimary"`
	ActionReasonSecondary        string                         `bson:"actionReasonSecondary,omitempty" json:"actionReasonSecondary,omitempty"`
	ActionConstraints            []string                       `bson:"actionConstraints,omitempty" json:"actionConstraints,omitempty"`
	CapitalEligible              bool                           `bson:"capitalEligible" json:"capitalEligible"`
	CapitalPriorityScore         float64                        `bson:"capitalPriorityScore,omitempty" json:"capitalPriorityScore,omitempty"`
	RecommendedPositionTargetPct float64                        `bson:"recommendedPositionTargetPct,omitempty" json:"recommendedPositionTargetPct,omitempty"`
	RecommendedPositionCapPct    float64                        `bson:"recommendedPositionCapPct,omitempty" json:"recommendedPositionCapPct,omitempty"`
	RecommendedTrancheStyle      common.RecommendedTrancheStyle `bson:"recommendedTrancheStyle,omitempty" json:"recommendedTrancheStyle,omitempty"`
	Notes                        string                         `bson:"notes,omitempty" json:"notes,omitempty"`
}

func (decision DecisionAction) Validate() error {
	if !decision.ActionType.IsValid() {
		return fmt.Errorf("invalid actionType %q", decision.ActionType)
	}
	if decision.BucketAfterAction != "" && !decision.BucketAfterAction.IsValid() {
		return fmt.Errorf("invalid bucketAfterAction %q", decision.BucketAfterAction)
	}
	if err := common.ValidateNonNegativeInt("actionPriorityRank", decision.ActionPriorityRank); err != nil {
		return err
	}
	if err := common.RequireString("actionReasonPrimary", decision.ActionReasonPrimary); err != nil {
		return err
	}
	if err := common.ValidateStringSlice("actionConstraints", decision.ActionConstraints); err != nil {
		return err
	}
	if err := common.ValidateNonNegativeFloat("capitalPriorityScore", decision.CapitalPriorityScore); err != nil {
		return err
	}
	if err := common.ValidatePercentage("recommendedPositionTargetPct", decision.RecommendedPositionTargetPct); err != nil {
		return err
	}
	if err := common.ValidatePercentage("recommendedPositionCapPct", decision.RecommendedPositionCapPct); err != nil {
		return err
	}
	if decision.RecommendedPositionCapPct > 0 && decision.RecommendedPositionCapPct < decision.RecommendedPositionTargetPct {
		return fmt.Errorf("recommendedPositionCapPct cannot be lower than recommendedPositionTargetPct")
	}
	if !decision.RecommendedTrancheStyle.IsValid() {
		return fmt.Errorf("invalid recommendedTrancheStyle %q", decision.RecommendedTrancheStyle)
	}
	return nil
}

type PositionSnapshot struct {
	IsOwned                     bool       `bson:"isOwned" json:"isOwned"`
	Quantity                    float64    `bson:"quantity" json:"quantity"`
	AverageCost                 float64    `bson:"averageCost" json:"averageCost"`
	MarketPriceAtReview         float64    `bson:"marketPriceAtReview" json:"marketPriceAtReview"`
	MarketValue                 float64    `bson:"marketValue" json:"marketValue"`
	PositionPctOfBook           float64    `bson:"positionPctOfBook" json:"positionPctOfBook"`
	PositionPctOfTotalPortfolio float64    `bson:"positionPctOfTotalPortfolio" json:"positionPctOfTotalPortfolio"`
	UnrealizedPnLAbs            float64    `bson:"unrealizedPnLAbs" json:"unrealizedPnLAbs"`
	UnrealizedPnLPct            float64    `bson:"unrealizedPnLPct" json:"unrealizedPnLPct"`
	TargetPositionPct           float64    `bson:"targetPositionPct" json:"targetPositionPct"`
	MaxPositionPct              float64    `bson:"maxPositionPct" json:"maxPositionPct"`
	UnderweightVsTargetPct      float64    `bson:"underweightVsTargetPct,omitempty" json:"underweightVsTargetPct,omitempty"`
	OverweightVsTargetPct       float64    `bson:"overweightVsTargetPct,omitempty" json:"overweightVsTargetPct,omitempty"`
	OwnedSinceDate              *time.Time `bson:"ownedSinceDate,omitempty" json:"ownedSinceDate,omitempty"`
}

func (snapshot PositionSnapshot) Validate() error {
	if err := common.ValidateNonNegativeFloat("quantity", snapshot.Quantity); err != nil {
		return err
	}
	if err := common.ValidateNonNegativeFloat("averageCost", snapshot.AverageCost); err != nil {
		return err
	}
	if err := common.ValidateNonNegativeFloat("marketPriceAtReview", snapshot.MarketPriceAtReview); err != nil {
		return err
	}
	if err := common.ValidateNonNegativeFloat("marketValue", snapshot.MarketValue); err != nil {
		return err
	}
	if err := common.ValidatePercentage("positionPctOfBook", snapshot.PositionPctOfBook); err != nil {
		return err
	}
	if err := common.ValidatePercentage("positionPctOfTotalPortfolio", snapshot.PositionPctOfTotalPortfolio); err != nil {
		return err
	}
	if err := common.ValidatePercentage("targetPositionPct", snapshot.TargetPositionPct); err != nil {
		return err
	}
	if err := common.ValidatePercentage("maxPositionPct", snapshot.MaxPositionPct); err != nil {
		return err
	}
	if err := common.ValidatePercentage("underweightVsTargetPct", snapshot.UnderweightVsTargetPct); err != nil {
		return err
	}
	if err := common.ValidatePercentage("overweightVsTargetPct", snapshot.OverweightVsTargetPct); err != nil {
		return err
	}
	if snapshot.MaxPositionPct > 0 && snapshot.TargetPositionPct > snapshot.MaxPositionPct {
		return fmt.Errorf("targetPositionPct cannot exceed maxPositionPct")
	}
	if snapshot.IsOwned && snapshot.Quantity <= 0 {
		return fmt.Errorf("quantity must be greater than zero when the position is owned")
	}
	if !snapshot.IsOwned && snapshot.Quantity > 0 {
		return fmt.Errorf("quantity must be zero when the position is not owned")
	}
	if snapshot.OwnedSinceDate != nil && snapshot.OwnedSinceDate.IsZero() {
		return fmt.Errorf("ownedSinceDate cannot be zero")
	}
	return nil
}

type ReviewChangeLog struct {
	PreviousReviewID         primitive.ObjectID `bson:"previousReviewId,omitempty" json:"previousReviewId,omitempty"`
	WeightedTotalScoreChange float64            `bson:"weightedTotalScoreChange,omitempty" json:"weightedTotalScoreChange,omitempty"`
	BucketChange             string             `bson:"bucketChange,omitempty" json:"bucketChange,omitempty"`
	ActionChange             string             `bson:"actionChange,omitempty" json:"actionChange,omitempty"`
	ThesisStatusChange       string             `bson:"thesisStatusChange,omitempty" json:"thesisStatusChange,omitempty"`
	MajorPositiveChanges     []string           `bson:"majorPositiveChanges,omitempty" json:"majorPositiveChanges,omitempty"`
	MajorNegativeChanges     []string           `bson:"majorNegativeChanges,omitempty" json:"majorNegativeChanges,omitempty"`
	SectionScoreChanges      map[string]float64 `bson:"sectionScoreChanges,omitempty" json:"sectionScoreChanges,omitempty"`
	SubScoreChanges          map[string]float64 `bson:"subScoreChanges,omitempty" json:"subScoreChanges,omitempty"`
	ValuationStateChange     string             `bson:"valuationStateChange,omitempty" json:"valuationStateChange,omitempty"`
	OwnershipRelevanceChange string             `bson:"ownershipRelevanceChange,omitempty" json:"ownershipRelevanceChange,omitempty"`
	RequiresExitReview       bool               `bson:"requiresExitReview" json:"requiresExitReview"`
	ChangeSummary            string             `bson:"changeSummary,omitempty" json:"changeSummary,omitempty"`
}

func (log ReviewChangeLog) Validate() error {
	if err := common.ValidateStringSlice("majorPositiveChanges", log.MajorPositiveChanges); err != nil {
		return err
	}
	if err := common.ValidateStringSlice("majorNegativeChanges", log.MajorNegativeChanges); err != nil {
		return err
	}
	for sectionName := range log.SectionScoreChanges {
		if !common.SectionName(strings.TrimSpace(sectionName)).IsValid() {
			return fmt.Errorf("invalid sectionScoreChanges key %q", sectionName)
		}
	}
	for subScoreName := range log.SubScoreChanges {
		if !common.SubScoreName(strings.TrimSpace(subScoreName)).IsValid() {
			return fmt.Errorf("invalid subScoreChanges key %q", subScoreName)
		}
	}
	return nil
}
