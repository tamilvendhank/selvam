package domain

import (
	"fmt"
	"math"
	"strings"
	"time"
)

const (
	SchemaVersionV1Alpha1 = "v1alpha1"
)

type BookType string

const (
	BookTypeInvesting BookType = "investing"
	BookTypeTrading   BookType = "trading"
)

type ReviewPeriodType string

const (
	ReviewPeriodMonthly     ReviewPeriodType = "monthly"
	ReviewPeriodQuarterly   ReviewPeriodType = "quarterly"
	ReviewPeriodEventDriven ReviewPeriodType = "event_driven"
	ReviewPeriodManual      ReviewPeriodType = "manual"
)

type ReviewStatus string

const (
	ReviewStatusDraft      ReviewStatus = "draft"
	ReviewStatusFinal      ReviewStatus = "final"
	ReviewStatusSuperseded ReviewStatus = "superseded"
)

type InvestingMode string

const (
	InvestingModeEarlyHunter         InvestingMode = "early_hunter"
	InvestingModeBalanced            InvestingMode = "balanced"
	InvestingModeConfirmedCompounder InvestingMode = "confirmed_compounder"
)

type ReviewerType string

const (
	ReviewerTypeAI     ReviewerType = "ai"
	ReviewerTypeHuman  ReviewerType = "human"
	ReviewerTypeHybrid ReviewerType = "hybrid"
)

type TrendDirection string

const (
	TrendDirectionImproving TrendDirection = "improving"
	TrendDirectionStable    TrendDirection = "stable"
	TrendDirectionWeakening TrendDirection = "weakening"
	TrendDirectionMixed     TrendDirection = "mixed"
)

type MetricBasis string

const (
	MetricBasisQuant  MetricBasis = "quant"
	MetricBasisText   MetricBasis = "text"
	MetricBasisHybrid MetricBasis = "hybrid"
)

type EvidenceStrength string

const (
	EvidenceStrengthLow    EvidenceStrength = "low"
	EvidenceStrengthMedium EvidenceStrength = "medium"
	EvidenceStrengthHigh   EvidenceStrength = "high"
)

type EvidenceSourceType string

const (
	EvidenceSourceAnnualReport         EvidenceSourceType = "annual_report"
	EvidenceSourceConcall              EvidenceSourceType = "concall"
	EvidenceSourceInvestorPresentation EvidenceSourceType = "investor_presentation"
	EvidenceSourceExchangeFiling       EvidenceSourceType = "exchange_filing"
	EvidenceSourceFinancialData        EvidenceSourceType = "financial_data"
	EvidenceSourcePriceData            EvidenceSourceType = "price_data"
	EvidenceSourceManualNote           EvidenceSourceType = "manual_note"
)

type EvidenceDirection string

const (
	EvidenceDirectionPositive EvidenceDirection = "positive"
	EvidenceDirectionNegative EvidenceDirection = "negative"
	EvidenceDirectionNeutral  EvidenceDirection = "neutral"
)

type WatchlistBucket string

const (
	WatchlistBucketResearch   WatchlistBucket = "research"
	WatchlistBucketWatch      WatchlistBucket = "watch"
	WatchlistBucketBuyReady   WatchlistBucket = "buy_ready"
	WatchlistBucketHold       WatchlistBucket = "hold"
	WatchlistBucketExitReview WatchlistBucket = "exit_review"
)

type ActionType string

const (
	ActionBuy    ActionType = "buy"
	ActionWatch  ActionType = "watch"
	ActionHold   ActionType = "hold"
	ActionTrim   ActionType = "trim"
	ActionSell   ActionType = "sell"
	ActionReject ActionType = "reject"
)

type SectionActionCap string

const (
	SectionActionCapCannotBuy      SectionActionCap = "cannot_buy"
	SectionActionCapWatchOnly      SectionActionCap = "watch_only"
	SectionActionCapExitReviewOnly SectionActionCap = "exit_review_only"
)

type ThesisStatus string

const (
	ThesisStatusActive      ThesisStatus = "active"
	ThesisStatusUnderReview ThesisStatus = "under_review"
	ThesisStatusBroken      ThesisStatus = "broken"
	ThesisStatusArchived    ThesisStatus = "archived"
)

type PositionRole string

const (
	PositionRoleStarter       PositionRole = "starter"
	PositionRoleBuilding      PositionRole = "building"
	PositionRoleCore          PositionRole = "core"
	PositionRoleTrimCandidate PositionRole = "trim_candidate"
	PositionRoleExitCandidate PositionRole = "exit_candidate"
)

type WorkflowRunType string

const (
	WorkflowRunTypeMonthlyScan      WorkflowRunType = "monthly_scan"
	WorkflowRunTypeQuarterlyRefresh WorkflowRunType = "quarterly_refresh"
	WorkflowRunTypeEventRefresh     WorkflowRunType = "event_refresh"
	WorkflowRunTypeManual           WorkflowRunType = "manual"
)

type WorkflowRunStatus string

const (
	WorkflowRunStatusDraft        WorkflowRunStatus = "draft"
	WorkflowRunStatusRunning      WorkflowRunStatus = "running"
	WorkflowRunStatusWaitingAsync WorkflowRunStatus = "waiting_async"
	WorkflowRunStatusCompleted    WorkflowRunStatus = "completed"
	WorkflowRunStatusFailed       WorkflowRunStatus = "failed"
)

type WorkflowStepStatusType string

const (
	WorkflowStepStatusPending      WorkflowStepStatusType = "pending"
	WorkflowStepStatusRunning      WorkflowStepStatusType = "running"
	WorkflowStepStatusWaitingAsync WorkflowStepStatusType = "waiting_async"
	WorkflowStepStatusCompleted    WorkflowStepStatusType = "completed"
	WorkflowStepStatusFailed       WorkflowStepStatusType = "failed"
	WorkflowStepStatusSkipped      WorkflowStepStatusType = "skipped"
)

type AsyncTaskStatus string

const (
	AsyncTaskStatusPending     AsyncTaskStatus = "pending"
	AsyncTaskStatusQueued      AsyncTaskStatus = "queued"
	AsyncTaskStatusInProgress  AsyncTaskStatus = "in_progress"
	AsyncTaskStatusCompleted   AsyncTaskStatus = "completed"
	AsyncTaskStatusFailed      AsyncTaskStatus = "failed"
	AsyncTaskStatusUnavailable AsyncTaskStatus = "unavailable"
)

type InvestingSectionName string

const (
	SectionInvestability                      InvestingSectionName = "Investability"
	SectionBusinessTraction                   InvestingSectionName = "Business Traction"
	SectionProfitConversion                   InvestingSectionName = "Profit Conversion"
	SectionCapitalEfficiencyFinancialStrength InvestingSectionName = "Capital Efficiency / Financial Strength"
	SectionStructuralSectorAttractiveness     InvestingSectionName = "Structural Sector Attractiveness"
	SectionRunwayIndustryPositioning          InvestingSectionName = "Runway / Industry Positioning"
	SectionManagementGovernance               InvestingSectionName = "Management / Governance"
	SectionMarketConfirmation                 InvestingSectionName = "Market Confirmation"
	SectionValuationEntryAttractiveness       InvestingSectionName = "Valuation / Entry Attractiveness"
)

var InvestingSectionsInOrder = []InvestingSectionName{
	SectionInvestability,
	SectionBusinessTraction,
	SectionProfitConversion,
	SectionCapitalEfficiencyFinancialStrength,
	SectionStructuralSectorAttractiveness,
	SectionRunwayIndustryPositioning,
	SectionManagementGovernance,
	SectionMarketConfirmation,
	SectionValuationEntryAttractiveness,
}

var InvestingSectionSubScores = map[InvestingSectionName][]string{
	SectionInvestability: {
		"Liquidity",
		"Data quality / completeness",
		"Basic investability suitability",
		"Listing / operating history sufficiency",
	},
	SectionBusinessTraction: {
		"Revenue growth strength",
		"Revenue growth consistency",
		"Recent 12-quarter acceleration / deterioration",
		"Evidence of expanding demand",
	},
	SectionProfitConversion: {
		"Operating margin quality / trend",
		"Profit growth strength",
		"Cash conversion quality",
		"Recent operating leverage / margin direction",
	},
	SectionCapitalEfficiencyFinancialStrength: {
		"ROCE / ROIC quality",
		"Balance-sheet strength",
		"Working-capital efficiency",
		"Dilution / capital-allocation discipline",
	},
	SectionStructuralSectorAttractiveness: {
		"Demand tailwind strength",
		"Industry economics quality",
		"Policy / formalization support",
		"Cyclicality risk",
	},
	SectionRunwayIndustryPositioning: {
		"Market opportunity size",
		"Share-gain potential",
		"Expansion optionality",
		"Competitive positioning strength",
	},
	SectionManagementGovernance: {
		"Capital allocation quality",
		"Execution consistency",
		"Shareholder alignment / trustworthiness",
		"Disclosure quality",
	},
	SectionMarketConfirmation: {
		"Relative strength",
		"Trend quality",
		"Drawdown / resilience behavior",
		"Reaction to results / news",
	},
	SectionValuationEntryAttractiveness: {
		"Historical valuation attractiveness",
		"Valuation support vs current quality",
		"Overvaluation risk",
		"Entry timing suitability",
	},
}

var CoreInvestingSections = []InvestingSectionName{
	SectionBusinessTraction,
	SectionProfitConversion,
	SectionCapitalEfficiencyFinancialStrength,
	SectionRunwayIndustryPositioning,
	SectionManagementGovernance,
}

func IsValidBookType(value BookType) bool {
	return value == BookTypeInvesting || value == BookTypeTrading
}

func IsValidActionType(value ActionType) bool {
	switch value {
	case ActionBuy, ActionWatch, ActionHold, ActionTrim, ActionSell, ActionReject:
		return true
	default:
		return false
	}
}

func IsValidBucket(value WatchlistBucket) bool {
	switch value {
	case WatchlistBucketResearch, WatchlistBucketWatch, WatchlistBucketBuyReady, WatchlistBucketHold, WatchlistBucketExitReview:
		return true
	default:
		return false
	}
}

func IsValidSectionActionCap(value SectionActionCap) bool {
	switch value {
	case "", SectionActionCapCannotBuy, SectionActionCapWatchOnly, SectionActionCapExitReviewOnly:
		return true
	default:
		return false
	}
}

func IsValidReviewStatus(value ReviewStatus) bool {
	switch value {
	case ReviewStatusDraft, ReviewStatusFinal, ReviewStatusSuperseded:
		return true
	default:
		return false
	}
}

func IsValidReviewPeriod(value ReviewPeriodType) bool {
	switch value {
	case ReviewPeriodMonthly, ReviewPeriodQuarterly, ReviewPeriodEventDriven, ReviewPeriodManual:
		return true
	default:
		return false
	}
}

func IsValidInvestingMode(value InvestingMode) bool {
	switch value {
	case InvestingModeEarlyHunter, InvestingModeBalanced, InvestingModeConfirmedCompounder:
		return true
	default:
		return false
	}
}

func IsValidReviewerType(value ReviewerType) bool {
	switch value {
	case ReviewerTypeAI, ReviewerTypeHuman, ReviewerTypeHybrid:
		return true
	default:
		return false
	}
}

func IsValidTrendDirection(value TrendDirection) bool {
	switch value {
	case TrendDirectionImproving, TrendDirectionStable, TrendDirectionWeakening, TrendDirectionMixed:
		return true
	default:
		return false
	}
}

func IsValidMetricBasis(value MetricBasis) bool {
	switch value {
	case MetricBasisQuant, MetricBasisText, MetricBasisHybrid:
		return true
	default:
		return false
	}
}

func IsValidEvidenceStrength(value EvidenceStrength) bool {
	switch value {
	case EvidenceStrengthLow, EvidenceStrengthMedium, EvidenceStrengthHigh:
		return true
	default:
		return false
	}
}

func IsValidEvidenceSourceType(value EvidenceSourceType) bool {
	switch value {
	case EvidenceSourceAnnualReport,
		EvidenceSourceConcall,
		EvidenceSourceInvestorPresentation,
		EvidenceSourceExchangeFiling,
		EvidenceSourceFinancialData,
		EvidenceSourcePriceData,
		EvidenceSourceManualNote:
		return true
	default:
		return false
	}
}

func IsValidEvidenceDirection(value EvidenceDirection) bool {
	switch value {
	case EvidenceDirectionPositive, EvidenceDirectionNegative, EvidenceDirectionNeutral:
		return true
	default:
		return false
	}
}

func IsValidThesisStatus(value ThesisStatus) bool {
	switch value {
	case ThesisStatusActive, ThesisStatusUnderReview, ThesisStatusBroken, ThesisStatusArchived:
		return true
	default:
		return false
	}
}

func IsValidPositionRole(value PositionRole) bool {
	switch value {
	case PositionRoleStarter, PositionRoleBuilding, PositionRoleCore, PositionRoleTrimCandidate, PositionRoleExitCandidate:
		return true
	default:
		return false
	}
}

func IsValidWorkflowRunType(value WorkflowRunType) bool {
	switch value {
	case WorkflowRunTypeMonthlyScan, WorkflowRunTypeQuarterlyRefresh, WorkflowRunTypeEventRefresh, WorkflowRunTypeManual:
		return true
	default:
		return false
	}
}

func IsValidWorkflowRunStatus(value WorkflowRunStatus) bool {
	switch value {
	case WorkflowRunStatusDraft, WorkflowRunStatusRunning, WorkflowRunStatusWaitingAsync, WorkflowRunStatusCompleted, WorkflowRunStatusFailed:
		return true
	default:
		return false
	}
}

func IsValidWorkflowStepStatus(value WorkflowStepStatusType) bool {
	switch value {
	case WorkflowStepStatusPending,
		WorkflowStepStatusRunning,
		WorkflowStepStatusWaitingAsync,
		WorkflowStepStatusCompleted,
		WorkflowStepStatusFailed,
		WorkflowStepStatusSkipped:
		return true
	default:
		return false
	}
}

func IsValidAsyncTaskStatus(value AsyncTaskStatus) bool {
	switch value {
	case AsyncTaskStatusPending, AsyncTaskStatusQueued, AsyncTaskStatusInProgress, AsyncTaskStatusCompleted, AsyncTaskStatusFailed, AsyncTaskStatusUnavailable:
		return true
	default:
		return false
	}
}

func IsValidInvestingSectionName(value string) bool {
	for _, section := range InvestingSectionsInOrder {
		if string(section) == value {
			return true
		}
	}

	return false
}

func IsValidInvestingSubScore(sectionName, subScoreName string) bool {
	section := InvestingSectionName(sectionName)
	subScores, ok := InvestingSectionSubScores[section]
	if !ok {
		return false
	}

	for _, candidate := range subScores {
		if candidate == subScoreName {
			return true
		}
	}

	return false
}

func NormalizeScore(value float64) float64 {
	return math.Round(value*100) / 100
}

func ValidateScoreRange(name string, value float64) error {
	if value < 1 || value > 10 {
		return fmt.Errorf("%s must be between 1 and 10", name)
	}

	return nil
}

func ValidatePercentRange(name string, value float64) error {
	if value < 0 || value > 100 {
		return fmt.Errorf("%s must be between 0 and 100", name)
	}

	return nil
}

func ValidateUnitRange(name string, value float64) error {
	if value < 0 || value > 1 {
		return fmt.Errorf("%s must be between 0 and 1", name)
	}

	return nil
}

func ValidateNonZeroTime(name string, value time.Time) error {
	if value.IsZero() {
		return fmt.Errorf("%s is required", name)
	}

	return nil
}

func FindSection(review *CompanyReview, sectionName InvestingSectionName) *SectionScore {
	if review == nil {
		return nil
	}

	for index := range review.Sections {
		if strings.EqualFold(review.Sections[index].SectionName, string(sectionName)) {
			return &review.Sections[index]
		}
	}

	return nil
}

func CoreSectionScores(review *CompanyReview) map[InvestingSectionName]float64 {
	result := make(map[InvestingSectionName]float64, len(CoreInvestingSections))
	if review == nil {
		return result
	}

	for _, sectionName := range CoreInvestingSections {
		if section := FindSection(review, sectionName); section != nil {
			result[sectionName] = section.SectionScoreRaw
		}
	}

	return result
}
