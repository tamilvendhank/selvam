package app

import (
	"time"

	domaincommon "goserver/internal/domain/common"
)

type ReviewListItemDTO struct {
	ReviewSummaryDTO
	Mode              domaincommon.InvestingMode    `json:"mode,omitempty"`
	OwnedBeforeReview bool                          `json:"ownedBeforeReview,omitempty"`
	ReviewPeriodType  domaincommon.ReviewPeriodType `json:"reviewPeriodType,omitempty"`
}

type ReviewDetailDTO struct {
	ReviewListItemDTO
	CurrentBucketBeforeReview domaincommon.WatchlistBucket     `json:"currentBucketBeforeReview,omitempty"`
	CurrentActionBeforeReview domaincommon.InvestingActionType `json:"currentActionBeforeReview,omitempty"`
	HardGateFailureReasons    []string                         `json:"hardGateFailureReasons,omitempty"`
	ReviewerType              domaincommon.ReviewerType        `json:"reviewerType,omitempty"`
	AIModelName               string                           `json:"aiModelName,omitempty"`
	AIPromptVersion           string                           `json:"aiPromptVersion,omitempty"`
	RawAIResultRef            *domaincommon.PayloadReference   `json:"rawAiResultRef,omitempty"`
	Scorecard                 ScorecardDTO                     `json:"scorecard"`
	DecisionAction            *DecisionActionDTO               `json:"decisionAction,omitempty"`
	PositionSnapshot          *PositionSnapshotDTO             `json:"positionSnapshot,omitempty"`
	ChangeLog                 *ReviewDiffDTO                   `json:"changeLog,omitempty"`
	ReviewMetadata            map[string]any                   `json:"reviewMetadata,omitempty"`
}

type ScorecardDTO struct {
	ReviewID               string            `json:"reviewId,omitempty"`
	WeightedTotalScore     float64           `json:"weightedTotalScore"`
	ConfidenceScore        float64           `json:"confidenceScore"`
	HardGateFailed         bool              `json:"hardGateFailed"`
	HardGateFailureReasons []string          `json:"hardGateFailureReasons,omitempty"`
	Sections               []SectionScoreDTO `json:"sections,omitempty"`
}

type SectionScoreDTO struct {
	SectionName               domaincommon.SectionName      `json:"sectionName"`
	SectionWeight             float64                       `json:"sectionWeight"`
	SectionScoreRaw           float64                       `json:"sectionScoreRaw"`
	SectionScoreWeighted      float64                       `json:"sectionScoreWeighted"`
	SectionPassedMinimumCheck bool                          `json:"sectionPassedMinimumCheck"`
	SectionActionCap          domaincommon.SectionActionCap `json:"sectionActionCap,omitempty"`
	Summary                   string                        `json:"summary,omitempty"`
	Strengths                 []string                      `json:"strengths,omitempty"`
	Weaknesses                []string                      `json:"weaknesses,omitempty"`
	Risks                     []string                      `json:"risks,omitempty"`
	ConfidenceScore           float64                       `json:"confidenceScore,omitempty"`
	SubScores                 []SubScoreDTO                 `json:"subScores,omitempty"`
	EvidenceRefs              []EvidenceReferenceDTO        `json:"evidenceRefs,omitempty"`
}

type SubScoreDTO struct {
	Name             domaincommon.SubScoreName     `json:"name"`
	Weight           float64                       `json:"weight"`
	Value            float64                       `json:"value"`
	Summary          string                        `json:"summary,omitempty"`
	TrendDirection   domaincommon.TrendDirection   `json:"trendDirection,omitempty"`
	EvidenceStrength domaincommon.EvidenceStrength `json:"evidenceStrength,omitempty"`
	MetricBasis      domaincommon.MetricBasis      `json:"metricBasis,omitempty"`
	Notes            string                        `json:"notes,omitempty"`
	EvidenceRefIDs   []string                      `json:"evidenceRefIds,omitempty"`
}

type EvidenceReferenceDTO struct {
	EvidenceID           string                          `json:"evidenceId,omitempty"`
	SectionName          domaincommon.SectionName        `json:"sectionName,omitempty"`
	SubScoreName         domaincommon.SubScoreName       `json:"subScoreName,omitempty"`
	SourceType           domaincommon.EvidenceSourceType `json:"sourceType,omitempty"`
	SourceDate           *time.Time                      `json:"sourceDate,omitempty"`
	SourceTitle          string                          `json:"sourceTitle,omitempty"`
	SourcePeriod         string                          `json:"sourcePeriod,omitempty"`
	SourceURLOrPath      string                          `json:"sourceUrlOrPath,omitempty"`
	ExcerptOrMetricName  string                          `json:"excerptOrMetricName,omitempty"`
	ExcerptOrMetricValue string                          `json:"excerptOrMetricValue,omitempty"`
	EvidenceSummary      string                          `json:"evidenceSummary,omitempty"`
	EvidenceDirection    domaincommon.EvidenceDirection  `json:"evidenceDirection,omitempty"`
}

type DecisionActionDTO struct {
	ActionType                   domaincommon.InvestingActionType     `json:"actionType,omitempty"`
	BucketAfterAction            domaincommon.WatchlistBucket         `json:"bucketAfterAction,omitempty"`
	ActionPriorityRank           int                                  `json:"actionPriorityRank,omitempty"`
	ActionReasonPrimary          string                               `json:"actionReasonPrimary,omitempty"`
	ActionReasonSecondary        string                               `json:"actionReasonSecondary,omitempty"`
	ActionConstraints            []string                             `json:"actionConstraints,omitempty"`
	CapitalEligible              bool                                 `json:"capitalEligible"`
	CapitalPriorityScore         float64                              `json:"capitalPriorityScore,omitempty"`
	RecommendedPositionTargetPct float64                              `json:"recommendedPositionTargetPct,omitempty"`
	RecommendedPositionCapPct    float64                              `json:"recommendedPositionCapPct,omitempty"`
	RecommendedTrancheStyle      domaincommon.RecommendedTrancheStyle `json:"recommendedTrancheStyle,omitempty"`
	Notes                        string                               `json:"notes,omitempty"`
}

type PositionSnapshotDTO struct {
	IsOwned                     bool       `json:"isOwned"`
	Quantity                    float64    `json:"quantity"`
	AverageCost                 float64    `json:"averageCost"`
	MarketPriceAtReview         float64    `json:"marketPriceAtReview"`
	MarketValue                 float64    `json:"marketValue"`
	PositionPctOfBook           float64    `json:"positionPctOfBook"`
	PositionPctOfTotalPortfolio float64    `json:"positionPctOfTotalPortfolio"`
	UnrealizedPnLAbs            float64    `json:"unrealizedPnlAbs"`
	UnrealizedPnLPct            float64    `json:"unrealizedPnlPct"`
	TargetPositionPct           float64    `json:"targetPositionPct"`
	MaxPositionPct              float64    `json:"maxPositionPct"`
	UnderweightVsTargetPct      float64    `json:"underweightVsTargetPct,omitempty"`
	OverweightVsTargetPct       float64    `json:"overweightVsTargetPct,omitempty"`
	OwnedSinceDate              *time.Time `json:"ownedSinceDate,omitempty"`
}

type ReviewDiffDTO struct {
	PreviousReviewID         string             `json:"previousReviewId,omitempty"`
	WeightedTotalScoreChange float64            `json:"weightedTotalScoreChange,omitempty"`
	SectionScoreChanges      map[string]float64 `json:"sectionScoreChanges,omitempty"`
	SubScoreChanges          map[string]float64 `json:"subScoreChanges,omitempty"`
	BucketChange             string             `json:"bucketChange,omitempty"`
	ActionChange             string             `json:"actionChange,omitempty"`
	ThesisStatusChange       string             `json:"thesisStatusChange,omitempty"`
	MajorPositiveChanges     []string           `json:"majorPositiveChanges,omitempty"`
	MajorNegativeChanges     []string           `json:"majorNegativeChanges,omitempty"`
	ValuationStateChange     string             `json:"valuationStateChange,omitempty"`
	OwnershipRelevanceChange string             `json:"ownershipRelevanceChange,omitempty"`
	RequiresExitReview       bool               `json:"requiresExitReview"`
	ChangeSummary            string             `json:"changeSummary,omitempty"`
}
