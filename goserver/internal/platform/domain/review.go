package domain

import (
	"fmt"
	"strings"
	"time"
)

type EvidenceReference struct {
	ID                   string             `json:"id" bson:"id"`
	SourceType           EvidenceSourceType `json:"sourceType" bson:"sourceType"`
	SourceDate           *time.Time         `json:"sourceDate,omitempty" bson:"sourceDate,omitempty"`
	SourceTitle          string             `json:"sourceTitle,omitempty" bson:"sourceTitle,omitempty"`
	SourcePeriod         string             `json:"sourcePeriod,omitempty" bson:"sourcePeriod,omitempty"`
	SourceURLOrPath      string             `json:"sourceUrlOrPath,omitempty" bson:"sourceUrlOrPath,omitempty"`
	ExcerptOrMetricName  string             `json:"excerptOrMetricName,omitempty" bson:"excerptOrMetricName,omitempty"`
	ExcerptOrMetricValue string             `json:"excerptOrMetricValue,omitempty" bson:"excerptOrMetricValue,omitempty"`
	EvidenceSummary      string             `json:"evidenceSummary,omitempty" bson:"evidenceSummary,omitempty"`
	EvidenceDirection    EvidenceDirection  `json:"evidenceDirection,omitempty" bson:"evidenceDirection,omitempty"`
}

func (reference *EvidenceReference) Validate() error {
	if reference == nil {
		return nil
	}
	if strings.TrimSpace(reference.ID) == "" {
		return fmt.Errorf("evidence reference id is required")
	}
	if !IsValidEvidenceSourceType(reference.SourceType) {
		return fmt.Errorf("invalid evidence source type %q", reference.SourceType)
	}
	if reference.EvidenceDirection != "" && !IsValidEvidenceDirection(reference.EvidenceDirection) {
		return fmt.Errorf("invalid evidence direction %q", reference.EvidenceDirection)
	}

	return nil
}

type SubScore struct {
	SubScoreName     string           `json:"subScoreName" bson:"subScoreName"`
	SubScoreWeight   float64          `json:"subScoreWeight" bson:"subScoreWeight"`
	SubScoreValue    float64          `json:"subScoreValue" bson:"subScoreValue"`
	SubScoreSummary  string           `json:"subScoreSummary,omitempty" bson:"subScoreSummary,omitempty"`
	TrendDirection   TrendDirection   `json:"trendDirection,omitempty" bson:"trendDirection,omitempty"`
	EvidenceStrength EvidenceStrength `json:"evidenceStrength,omitempty" bson:"evidenceStrength,omitempty"`
	MetricBasis      MetricBasis      `json:"metricBasis,omitempty" bson:"metricBasis,omitempty"`
	Notes            string           `json:"notes,omitempty" bson:"notes,omitempty"`
	EvidenceRefIDs   []string         `json:"evidenceRefIds,omitempty" bson:"evidenceRefIds,omitempty"`
}

func (subScore *SubScore) Validate(sectionName string) error {
	if subScore == nil {
		return nil
	}
	if strings.TrimSpace(subScore.SubScoreName) == "" {
		return fmt.Errorf("sub-score name is required")
	}
	if !IsValidInvestingSubScore(sectionName, subScore.SubScoreName) {
		return fmt.Errorf("invalid sub-score %q for section %q", subScore.SubScoreName, sectionName)
	}
	if err := ValidatePercentRange("sub-score weight", subScore.SubScoreWeight); err != nil {
		return err
	}
	if err := ValidateScoreRange("sub-score value", subScore.SubScoreValue); err != nil {
		return err
	}
	if subScore.TrendDirection != "" && !IsValidTrendDirection(subScore.TrendDirection) {
		return fmt.Errorf("invalid sub-score trend direction %q", subScore.TrendDirection)
	}
	if subScore.EvidenceStrength != "" && !IsValidEvidenceStrength(subScore.EvidenceStrength) {
		return fmt.Errorf("invalid evidence strength %q", subScore.EvidenceStrength)
	}
	if subScore.MetricBasis != "" && !IsValidMetricBasis(subScore.MetricBasis) {
		return fmt.Errorf("invalid metric basis %q", subScore.MetricBasis)
	}

	return nil
}

type SectionScore struct {
	SectionName               string              `json:"sectionName" bson:"sectionName"`
	SectionWeight             float64             `json:"sectionWeight" bson:"sectionWeight"`
	SectionScoreRaw           float64             `json:"sectionScoreRaw" bson:"sectionScoreRaw"`
	SectionScoreWeighted      float64             `json:"sectionScoreWeighted" bson:"sectionScoreWeighted"`
	SectionPassedMinimumCheck bool                `json:"sectionPassedMinimumCheck" bson:"sectionPassedMinimumCheck"`
	SectionActionCap          SectionActionCap    `json:"sectionActionCap,omitempty" bson:"sectionActionCap,omitempty"`
	SectionSummary            string              `json:"sectionSummary,omitempty" bson:"sectionSummary,omitempty"`
	SectionStrengths          []string            `json:"sectionStrengths,omitempty" bson:"sectionStrengths,omitempty"`
	SectionWeaknesses         []string            `json:"sectionWeaknesses,omitempty" bson:"sectionWeaknesses,omitempty"`
	SectionRisks              []string            `json:"sectionRisks,omitempty" bson:"sectionRisks,omitempty"`
	SectionConfidenceScore    float64             `json:"sectionConfidenceScore" bson:"sectionConfidenceScore"`
	SubScores                 []SubScore          `json:"subScores" bson:"subScores"`
	EvidenceRefs              []EvidenceReference `json:"evidenceRefs,omitempty" bson:"evidenceRefs,omitempty"`
}

func (section *SectionScore) Validate() error {
	if section == nil {
		return nil
	}
	if !IsValidInvestingSectionName(section.SectionName) {
		return fmt.Errorf("invalid section name %q", section.SectionName)
	}
	if err := ValidatePercentRange("section weight", section.SectionWeight); err != nil {
		return err
	}
	if err := ValidateScoreRange("section raw score", section.SectionScoreRaw); err != nil {
		return err
	}
	if section.SectionScoreWeighted < 0 || section.SectionScoreWeighted > 10 {
		return fmt.Errorf("section weighted score must be between 0 and 10")
	}
	if err := ValidateUnitRange("section confidence score", section.SectionConfidenceScore); err != nil {
		return err
	}
	if !IsValidSectionActionCap(section.SectionActionCap) {
		return fmt.Errorf("invalid section action cap %q", section.SectionActionCap)
	}

	subScoreNames := make(map[string]struct{}, len(section.SubScores))
	var subScoreWeightTotal float64
	for index := range section.SubScores {
		if err := section.SubScores[index].Validate(section.SectionName); err != nil {
			return err
		}
		if _, exists := subScoreNames[section.SubScores[index].SubScoreName]; exists {
			return fmt.Errorf("duplicate sub-score %q", section.SubScores[index].SubScoreName)
		}
		subScoreNames[section.SubScores[index].SubScoreName] = struct{}{}
		subScoreWeightTotal += section.SubScores[index].SubScoreWeight
	}
	if len(section.SubScores) == 0 {
		return fmt.Errorf("section %q must include sub-scores", section.SectionName)
	}
	if NormalizeScore(subScoreWeightTotal) != 100 {
		return fmt.Errorf("section %q sub-score weights must total 100", section.SectionName)
	}

	for index := range section.EvidenceRefs {
		if err := section.EvidenceRefs[index].Validate(); err != nil {
			return err
		}
	}

	return nil
}

type DecisionAction struct {
	ActionType                   ActionType      `json:"actionType" bson:"actionType"`
	BucketAfterAction            WatchlistBucket `json:"bucketAfterAction,omitempty" bson:"bucketAfterAction,omitempty"`
	ActionPriorityRank           int             `json:"actionPriorityRank,omitempty" bson:"actionPriorityRank,omitempty"`
	ActionReasonPrimary          string          `json:"actionReasonPrimary,omitempty" bson:"actionReasonPrimary,omitempty"`
	ActionReasonSecondary        string          `json:"actionReasonSecondary,omitempty" bson:"actionReasonSecondary,omitempty"`
	ActionConstraints            []string        `json:"actionConstraints,omitempty" bson:"actionConstraints,omitempty"`
	CapitalEligible              bool            `json:"capitalEligible" bson:"capitalEligible"`
	CapitalPriorityScore         float64         `json:"capitalPriorityScore,omitempty" bson:"capitalPriorityScore,omitempty"`
	RecommendedPositionTargetPct float64         `json:"recommendedPositionTargetPct,omitempty" bson:"recommendedPositionTargetPct,omitempty"`
	RecommendedPositionCapPct    float64         `json:"recommendedPositionCapPct,omitempty" bson:"recommendedPositionCapPct,omitempty"`
	RecommendedTrancheStyle      string          `json:"recommendedTrancheStyle,omitempty" bson:"recommendedTrancheStyle,omitempty"`
	Notes                        string          `json:"notes,omitempty" bson:"notes,omitempty"`
}

func (action *DecisionAction) Validate() error {
	if action == nil {
		return nil
	}
	if !IsValidActionType(action.ActionType) {
		return fmt.Errorf("invalid decision action type %q", action.ActionType)
	}
	if action.BucketAfterAction != "" && !IsValidBucket(action.BucketAfterAction) {
		return fmt.Errorf("invalid bucket after action %q", action.BucketAfterAction)
	}
	if action.CapitalPriorityScore < 0 || action.CapitalPriorityScore > 10 {
		return fmt.Errorf("capital priority score must be between 0 and 10")
	}
	if err := ValidatePercentRange("recommended position target pct", action.RecommendedPositionTargetPct); err != nil {
		return err
	}
	if err := ValidatePercentRange("recommended position cap pct", action.RecommendedPositionCapPct); err != nil {
		return err
	}

	return nil
}

type PositionSnapshot struct {
	IsOwned                     bool       `json:"isOwned" bson:"isOwned"`
	Quantity                    float64    `json:"quantity,omitempty" bson:"quantity,omitempty"`
	AverageCost                 float64    `json:"averageCost,omitempty" bson:"averageCost,omitempty"`
	MarketPriceAtReview         float64    `json:"marketPriceAtReview,omitempty" bson:"marketPriceAtReview,omitempty"`
	MarketValue                 float64    `json:"marketValue,omitempty" bson:"marketValue,omitempty"`
	PositionPctOfBook           float64    `json:"positionPctOfBook,omitempty" bson:"positionPctOfBook,omitempty"`
	PositionPctOfTotalPortfolio float64    `json:"positionPctOfTotalPortfolio,omitempty" bson:"positionPctOfTotalPortfolio,omitempty"`
	UnrealizedPnLAbs            float64    `json:"unrealizedPnlAbs,omitempty" bson:"unrealizedPnlAbs,omitempty"`
	UnrealizedPnLPct            float64    `json:"unrealizedPnlPct,omitempty" bson:"unrealizedPnlPct,omitempty"`
	TargetPositionPct           float64    `json:"targetPositionPct,omitempty" bson:"targetPositionPct,omitempty"`
	MaxPositionPct              float64    `json:"maxPositionPct,omitempty" bson:"maxPositionPct,omitempty"`
	UnderweightVsTargetPct      float64    `json:"underweightVsTargetPct,omitempty" bson:"underweightVsTargetPct,omitempty"`
	OverweightVsTargetPct       float64    `json:"overweightVsTargetPct,omitempty" bson:"overweightVsTargetPct,omitempty"`
	OwnedSinceDate              *time.Time `json:"ownedSinceDate,omitempty" bson:"ownedSinceDate,omitempty"`
}

func (snapshot *PositionSnapshot) Validate() error {
	if snapshot == nil {
		return nil
	}
	if err := ValidatePercentRange("position pct of book", snapshot.PositionPctOfBook); err != nil {
		return err
	}
	if err := ValidatePercentRange("position pct of total portfolio", snapshot.PositionPctOfTotalPortfolio); err != nil {
		return err
	}
	if err := ValidatePercentRange("target position pct", snapshot.TargetPositionPct); err != nil {
		return err
	}
	if err := ValidatePercentRange("max position pct", snapshot.MaxPositionPct); err != nil {
		return err
	}

	return nil
}

type ReviewChangeLog struct {
	PreviousReviewID         string             `json:"previousReviewId,omitempty" bson:"previousReviewId,omitempty"`
	WeightedTotalScoreChange float64            `json:"weightedTotalScoreChange,omitempty" bson:"weightedTotalScoreChange,omitempty"`
	BucketChange             string             `json:"bucketChange,omitempty" bson:"bucketChange,omitempty"`
	ActionChange             string             `json:"actionChange,omitempty" bson:"actionChange,omitempty"`
	ThesisStatusChange       string             `json:"thesisStatusChange,omitempty" bson:"thesisStatusChange,omitempty"`
	MajorPositiveChanges     []string           `json:"majorPositiveChanges,omitempty" bson:"majorPositiveChanges,omitempty"`
	MajorNegativeChanges     []string           `json:"majorNegativeChanges,omitempty" bson:"majorNegativeChanges,omitempty"`
	SectionScoreChanges      map[string]float64 `json:"sectionScoreChanges,omitempty" bson:"sectionScoreChanges,omitempty"`
	SubScoreChanges          map[string]float64 `json:"subScoreChanges,omitempty" bson:"subScoreChanges,omitempty"`
	ValuationStateChange     string             `json:"valuationStateChange,omitempty" bson:"valuationStateChange,omitempty"`
	OwnershipRelevanceChange string             `json:"ownershipRelevanceChange,omitempty" bson:"ownershipRelevanceChange,omitempty"`
	RequiresExitReview       bool               `json:"requiresExitReview" bson:"requiresExitReview"`
	ChangeSummary            string             `json:"changeSummary,omitempty" bson:"changeSummary,omitempty"`
}

type CompanyReview struct {
	ID                        string            `json:"id" bson:"-"`
	CompanyID                 string            `json:"companyId" bson:"companyId"`
	Symbol                    string            `json:"symbol" bson:"symbol"`
	BookType                  BookType          `json:"bookType" bson:"bookType"`
	ReviewDate                time.Time         `json:"reviewDate" bson:"reviewDate"`
	ReviewPeriodType          ReviewPeriodType  `json:"reviewPeriodType" bson:"reviewPeriodType"`
	WorkflowRunID             string            `json:"workflowRunId,omitempty" bson:"workflowRunId,omitempty"`
	ConfigSnapshotID          string            `json:"configSnapshotId" bson:"configSnapshotId"`
	ReviewStatus              ReviewStatus      `json:"reviewStatus" bson:"reviewStatus"`
	Mode                      InvestingMode     `json:"mode,omitempty" bson:"mode,omitempty"`
	OwnedBeforeReview         bool              `json:"ownedBeforeReview" bson:"ownedBeforeReview"`
	CurrentBucketBeforeReview WatchlistBucket   `json:"currentBucketBeforeReview,omitempty" bson:"currentBucketBeforeReview,omitempty"`
	CurrentActionBeforeReview ActionType        `json:"currentActionBeforeReview,omitempty" bson:"currentActionBeforeReview,omitempty"`
	WeightedTotalScore        float64           `json:"weightedTotalScore" bson:"weightedTotalScore"`
	HardGateFailed            bool              `json:"hardGateFailed" bson:"hardGateFailed"`
	HardGateFailureReasons    []string          `json:"hardGateFailureReasons,omitempty" bson:"hardGateFailureReasons,omitempty"`
	ConfidenceScore           float64           `json:"confidenceScore" bson:"confidenceScore"`
	FinalBucketAfterReview    WatchlistBucket   `json:"finalBucketAfterReview,omitempty" bson:"finalBucketAfterReview,omitempty"`
	FinalActionAfterReview    ActionType        `json:"finalActionAfterReview,omitempty" bson:"finalActionAfterReview,omitempty"`
	ActionRationaleSummary    string            `json:"actionRationaleSummary,omitempty" bson:"actionRationaleSummary,omitempty"`
	WhatChangedSummary        string            `json:"whatChangedSummary,omitempty" bson:"whatChangedSummary,omitempty"`
	ReviewerType              ReviewerType      `json:"reviewerType" bson:"reviewerType"`
	AIModelName               string            `json:"aiModelName,omitempty" bson:"aiModelName,omitempty"`
	AIPromptVersion           string            `json:"aiPromptVersion,omitempty" bson:"aiPromptVersion,omitempty"`
	SchemaVersion             string            `json:"schemaVersion" bson:"schemaVersion"`
	ReviewMetadata            map[string]any    `json:"reviewMetadata,omitempty" bson:"reviewMetadata,omitempty"`
	Sections                  []SectionScore    `json:"sections" bson:"sections"`
	DecisionAction            *DecisionAction   `json:"decisionAction,omitempty" bson:"decisionAction,omitempty"`
	PositionSnapshot          *PositionSnapshot `json:"positionSnapshot,omitempty" bson:"positionSnapshot,omitempty"`
	ChangeLog                 *ReviewChangeLog  `json:"changeLog,omitempty" bson:"changeLog,omitempty"`
	CreatedAt                 time.Time         `json:"createdAt" bson:"createdAt"`
	UpdatedAt                 time.Time         `json:"updatedAt" bson:"updatedAt"`
}

func (review *CompanyReview) Validate() error {
	if review == nil {
		return fmt.Errorf("review is required")
	}
	if strings.TrimSpace(review.CompanyID) == "" {
		return fmt.Errorf("review companyId is required")
	}
	if strings.TrimSpace(review.Symbol) == "" {
		return fmt.Errorf("review symbol is required")
	}
	if !IsValidBookType(review.BookType) {
		return fmt.Errorf("invalid review book type %q", review.BookType)
	}
	if err := ValidateNonZeroTime("review date", review.ReviewDate); err != nil {
		return err
	}
	if !IsValidReviewPeriod(review.ReviewPeriodType) {
		return fmt.Errorf("invalid review period type %q", review.ReviewPeriodType)
	}
	if strings.TrimSpace(review.ConfigSnapshotID) == "" {
		return fmt.Errorf("config snapshot id is required")
	}
	if !IsValidReviewStatus(review.ReviewStatus) {
		return fmt.Errorf("invalid review status %q", review.ReviewStatus)
	}
	if review.BookType == BookTypeInvesting && !IsValidInvestingMode(review.Mode) {
		return fmt.Errorf("invalid investing mode %q", review.Mode)
	}
	if review.CurrentBucketBeforeReview != "" && !IsValidBucket(review.CurrentBucketBeforeReview) {
		return fmt.Errorf("invalid current bucket before review %q", review.CurrentBucketBeforeReview)
	}
	if review.CurrentActionBeforeReview != "" && !IsValidActionType(review.CurrentActionBeforeReview) {
		return fmt.Errorf("invalid current action before review %q", review.CurrentActionBeforeReview)
	}
	if review.WeightedTotalScore < 0 || review.WeightedTotalScore > 10 {
		return fmt.Errorf("weighted total score must be between 0 and 10")
	}
	if err := ValidateUnitRange("review confidence score", review.ConfidenceScore); err != nil {
		return err
	}
	if review.FinalBucketAfterReview != "" && !IsValidBucket(review.FinalBucketAfterReview) {
		return fmt.Errorf("invalid final bucket %q", review.FinalBucketAfterReview)
	}
	if review.FinalActionAfterReview != "" && !IsValidActionType(review.FinalActionAfterReview) {
		return fmt.Errorf("invalid final action %q", review.FinalActionAfterReview)
	}
	if !IsValidReviewerType(review.ReviewerType) {
		return fmt.Errorf("invalid reviewer type %q", review.ReviewerType)
	}
	if strings.TrimSpace(review.SchemaVersion) == "" {
		return fmt.Errorf("review schema version is required")
	}

	sectionNames := make(map[string]struct{}, len(review.Sections))
	var sectionWeightTotal float64
	for index := range review.Sections {
		if err := review.Sections[index].Validate(); err != nil {
			return err
		}
		if _, exists := sectionNames[review.Sections[index].SectionName]; exists {
			return fmt.Errorf("duplicate section %q", review.Sections[index].SectionName)
		}
		sectionNames[review.Sections[index].SectionName] = struct{}{}
		sectionWeightTotal += review.Sections[index].SectionWeight
	}
	if review.BookType == BookTypeInvesting && len(review.Sections) != len(InvestingSectionsInOrder) {
		return fmt.Errorf("investing reviews must include %d sections", len(InvestingSectionsInOrder))
	}
	if len(review.Sections) > 0 && NormalizeScore(sectionWeightTotal) != 100 {
		return fmt.Errorf("section weights must total 100")
	}

	if err := review.DecisionAction.Validate(); err != nil {
		return err
	}
	if err := review.PositionSnapshot.Validate(); err != nil {
		return err
	}
	if err := ValidateNonZeroTime("review createdAt", review.CreatedAt); err != nil {
		return err
	}
	if err := ValidateNonZeroTime("review updatedAt", review.UpdatedAt); err != nil {
		return err
	}

	return nil
}

func (review *CompanyReview) IsMutable() bool {
	return review != nil && review.ReviewStatus == ReviewStatusDraft
}

func (review *CompanyReview) FlattenEvidence() []EvidenceReference {
	if review == nil {
		return nil
	}

	flattened := make([]EvidenceReference, 0)
	for _, section := range review.Sections {
		flattened = append(flattened, section.EvidenceRefs...)
	}

	return flattened
}
