package contracts

import (
	"time"

	"goserver/internal/domain/common"
)

type InvestingReviewOutputPayload struct {
	CompanyID                    string                         `json:"company_id"`
	Symbol                       string                         `json:"symbol"`
	ReviewDate                   time.Time                      `json:"review_date"`
	Mode                         common.InvestingMode           `json:"mode"`
	WeightedTotalScore           float64                        `json:"weighted_total_score"`
	ConfidenceScore              float64                        `json:"confidence_score"`
	HardGateFailed               bool                           `json:"hard_gate_failed"`
	HardGateFailureReasons       []string                       `json:"hard_gate_failure_reasons,omitempty"`
	Sections                     []AISectionScoreOutput         `json:"sections"`
	SuggestedAction              common.InvestingActionType     `json:"suggested_action"`
	SuggestedBucket              common.WatchlistBucket         `json:"suggested_bucket"`
	ActionRationaleSummary       string                         `json:"action_rationale_summary"`
	WhatChangedSummary           string                         `json:"what_changed_summary,omitempty"`
	CapitalEligible              bool                           `json:"capital_eligible"`
	CapitalPriorityScore         *float64                       `json:"capital_priority_score,omitempty"`
	RecommendedPositionTargetPct *float64                       `json:"recommended_position_target_pct,omitempty"`
	RecommendedPositionCapPct    *float64                       `json:"recommended_position_cap_pct,omitempty"`
	RecommendedTrancheStyle      common.RecommendedTrancheStyle `json:"recommended_tranche_style,omitempty"`
	ActionConstraints            []string                       `json:"action_constraints,omitempty"`
	ThesisSummaryCandidate       string                         `json:"thesis_summary_candidate,omitempty"`
	WhyThisBusinessCanCompound   string                         `json:"why_this_business_can_compound,omitempty"`
	KeyGrowthDrivers             []string                       `json:"key_growth_drivers,omitempty"`
	KeyMoatOrAdvantageFactors    []string                       `json:"key_moat_or_advantage_factors,omitempty"`
	WhyNow                       string                         `json:"why_now,omitempty"`
	KeyRisks                     []string                       `json:"key_risks,omitempty"`
	DisconfirmingSignals         []string                       `json:"disconfirming_signals,omitempty"`
	WhatWouldBreakTheThesis      []string                       `json:"what_would_break_the_thesis,omitempty"`
	ChangeLog                    AIReviewChangeLogOutput        `json:"change_log,omitempty"`
	MissingDataPoints            []string                       `json:"missing_data_points,omitempty"`
	LowConfidenceAreas           []string                       `json:"low_confidence_areas,omitempty"`
	AssumptionsMade              []string                       `json:"assumptions_made,omitempty"`
	Warnings                     []string                       `json:"warnings,omitempty"`
}

type AISectionScoreOutput struct {
	SectionName               common.SectionName          `json:"section_name"`
	SectionWeight             float64                     `json:"section_weight"`
	SectionScoreRaw           float64                     `json:"section_score_raw"`
	SectionScoreWeighted      *float64                    `json:"section_score_weighted,omitempty"`
	SectionPassedMinimumCheck bool                        `json:"section_passed_minimum_check"`
	SectionActionCap          common.SectionActionCap     `json:"section_action_cap,omitempty"`
	SectionSummary            string                      `json:"section_summary"`
	SectionStrengths          []string                    `json:"section_strengths"`
	SectionWeaknesses         []string                    `json:"section_weaknesses"`
	SectionRisks              []string                    `json:"section_risks"`
	SectionConfidenceScore    float64                     `json:"section_confidence_score"`
	SubScores                 []AISubScoreOutput          `json:"sub_scores"`
	EvidenceRefs              []AIEvidenceReferenceOutput `json:"evidence_refs"`
}

type AISubScoreOutput struct {
	SubScoreName     common.SubScoreName     `json:"sub_score_name"`
	SubScoreWeight   float64                 `json:"sub_score_weight"`
	SubScoreValue    float64                 `json:"sub_score_value"`
	SubScoreSummary  string                  `json:"sub_score_summary"`
	TrendDirection   common.TrendDirection   `json:"trend_direction"`
	EvidenceStrength common.EvidenceStrength `json:"evidence_strength"`
	MetricBasis      common.MetricBasis      `json:"metric_basis"`
	Notes            string                  `json:"notes,omitempty"`
	EvidenceRefIDs   []string                `json:"evidence_ref_ids,omitempty"`
}

type AIEvidenceReferenceOutput struct {
	EvidenceID           string                    `json:"evidence_id"`
	SourceType           common.EvidenceSourceType `json:"source_type"`
	SourceDate           *time.Time                `json:"source_date,omitempty"`
	SourceTitle          string                    `json:"source_title,omitempty"`
	SourcePeriod         string                    `json:"source_period,omitempty"`
	SourceURLOrPath      string                    `json:"source_url_or_path,omitempty"`
	ExcerptOrMetricName  string                    `json:"excerpt_or_metric_name,omitempty"`
	ExcerptOrMetricValue string                    `json:"excerpt_or_metric_value,omitempty"`
	EvidenceSummary      string                    `json:"evidence_summary"`
	EvidenceDirection    common.EvidenceDirection  `json:"evidence_direction"`
}

type AIReviewChangeLogOutput struct {
	PreviousReviewID         string                          `json:"previous_review_id,omitempty"`
	WeightedTotalScoreChange *float64                        `json:"weighted_total_score_change,omitempty"`
	BucketChange             string                          `json:"bucket_change,omitempty"`
	ActionChange             string                          `json:"action_change,omitempty"`
	ThesisStatusChange       string                          `json:"thesis_status_change,omitempty"`
	MajorPositiveChanges     []string                        `json:"major_positive_changes,omitempty"`
	MajorNegativeChanges     []string                        `json:"major_negative_changes,omitempty"`
	SectionScoreChanges      map[common.SectionName]float64  `json:"section_score_changes,omitempty"`
	SubScoreChanges          map[common.SubScoreName]float64 `json:"sub_score_changes,omitempty"`
	ValuationStateChange     string                          `json:"valuation_state_change,omitempty"`
	OwnershipRelevanceChange string                          `json:"ownership_relevance_change,omitempty"`
	RequiresExitReview       bool                            `json:"requires_exit_review"`
	ChangeSummary            string                          `json:"change_summary,omitempty"`
}
