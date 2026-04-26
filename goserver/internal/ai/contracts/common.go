package contracts

import (
	"time"

	"goserver/internal/domain/common"
)

type ReviewType string

const (
	ReviewTypeInvestingCompanyReview ReviewType = "investing_company_review"
	ReviewTypeThesisUpdate           ReviewType = "thesis_update"
	ReviewTypeChangeSummary          ReviewType = "change_summary"
	ReviewTypeEvidenceSummary        ReviewType = "evidence_summary"
	ReviewTypeTradingCandidateReview ReviewType = "trading_candidate_review"
)

func (reviewType ReviewType) IsValid() bool {
	switch reviewType {
	case ReviewTypeInvestingCompanyReview,
		ReviewTypeThesisUpdate,
		ReviewTypeChangeSummary,
		ReviewTypeEvidenceSummary,
		ReviewTypeTradingCandidateReview:
		return true
	default:
		return false
	}
}

type CompanyIdentityInput struct {
	CompanyID       string `json:"company_id"`
	Symbol          string `json:"symbol"`
	Exchange        string `json:"exchange,omitempty"`
	CompanyName     string `json:"company_name"`
	Sector          string `json:"sector,omitempty"`
	Industry        string `json:"industry,omitempty"`
	SubIndustry     string `json:"sub_industry,omitempty"`
	BusinessSummary string `json:"business_summary,omitempty"`
}

type ReviewContextInput struct {
	ReviewDate                time.Time                  `json:"review_date"`
	Mode                      common.InvestingMode       `json:"mode"`
	YearsLookback             int                        `json:"years_lookback"`
	RecentQuarterLookback     int                        `json:"recent_quarter_lookback"`
	CompareAgainst            string                     `json:"compare_against"`
	BookType                  common.BookType            `json:"book_type"`
	OwnedBeforeReview         bool                       `json:"owned_before_review"`
	CurrentBucketBeforeReview common.WatchlistBucket     `json:"current_bucket_before_review,omitempty"`
	CurrentActionBeforeReview common.InvestingActionType `json:"current_action_before_review,omitempty"`
}

type CurrentPositionInput struct {
	IsOwned                     bool       `json:"is_owned"`
	PositionPctOfBook           float64    `json:"position_pct_of_book,omitempty"`
	PositionPctOfTotalPortfolio float64    `json:"position_pct_of_total_portfolio,omitempty"`
	TargetPositionPct           float64    `json:"target_position_pct,omitempty"`
	MaxPositionPct              float64    `json:"max_position_pct,omitempty"`
	UnrealizedPnLPct            float64    `json:"unrealized_pnl_pct,omitempty"`
	OwnedSinceDate              *time.Time `json:"owned_since_date,omitempty"`
}

type AnnualFinancialMetricsInput struct {
	FiscalYear         string   `json:"fiscal_year"`
	Revenue            *float64 `json:"revenue,omitempty"`
	RevenueGrowthPct   *float64 `json:"revenue_growth_pct,omitempty"`
	OperatingProfit    *float64 `json:"operating_profit,omitempty"`
	OperatingMarginPct *float64 `json:"operating_margin_pct,omitempty"`
	PAT                *float64 `json:"pat,omitempty"`
	PATGrowthPct       *float64 `json:"pat_growth_pct,omitempty"`
	OperatingCashFlow  *float64 `json:"operating_cash_flow,omitempty"`
	FreeCashFlow       *float64 `json:"free_cash_flow,omitempty"`
	ROCEPct            *float64 `json:"roce_pct,omitempty"`
	ROICPct            *float64 `json:"roic_pct,omitempty"`
	DebtToEquity       *float64 `json:"debt_to_equity,omitempty"`
	InterestCoverage   *float64 `json:"interest_coverage,omitempty"`
	WorkingCapitalDays *float64 `json:"working_capital_days,omitempty"`
	ReceivableDays     *float64 `json:"receivable_days,omitempty"`
	InventoryDays      *float64 `json:"inventory_days,omitempty"`
	PayableDays        *float64 `json:"payable_days,omitempty"`
	SharesOutstanding  *float64 `json:"shares_outstanding,omitempty"`
	DilutionPct        *float64 `json:"dilution_pct,omitempty"`
	Capex              *float64 `json:"capex,omitempty"`
	FCFYieldPct        *float64 `json:"fcf_yield_pct,omitempty"`
}

type QuarterlyFinancialMetricsInput struct {
	Period              string   `json:"period"`
	Revenue             *float64 `json:"revenue,omitempty"`
	RevenueGrowthYoYPct *float64 `json:"revenue_growth_yoy_pct,omitempty"`
	OperatingProfit     *float64 `json:"operating_profit,omitempty"`
	OperatingMarginPct  *float64 `json:"operating_margin_pct,omitempty"`
	PAT                 *float64 `json:"pat,omitempty"`
	PATGrowthYoYPct     *float64 `json:"pat_growth_yoy_pct,omitempty"`
	OperatingCashFlow   *float64 `json:"operating_cash_flow,omitempty"`
	EPS                 *float64 `json:"eps,omitempty"`
}

type ValuationRangeInput struct {
	Min               *float64 `json:"min,omitempty"`
	P25               *float64 `json:"p25,omitempty"`
	Median            *float64 `json:"median,omitempty"`
	P75               *float64 `json:"p75,omitempty"`
	Max               *float64 `json:"max,omitempty"`
	CurrentPercentile *float64 `json:"current_percentile,omitempty"`
}

type ValuationMetricsInput struct {
	CurrentPE                 *float64            `json:"current_pe,omitempty"`
	HistoricalPERange         ValuationRangeInput `json:"historical_pe_range,omitempty"`
	CurrentEVEBITDA           *float64            `json:"current_ev_ebitda,omitempty"`
	HistoricalEVEBITDARange   ValuationRangeInput `json:"historical_ev_ebitda_range,omitempty"`
	CurrentPB                 *float64            `json:"current_pb,omitempty"`
	HistoricalPBRange         ValuationRangeInput `json:"historical_pb_range,omitempty"`
	CurrentPriceSales         *float64            `json:"current_price_sales,omitempty"`
	HistoricalPriceSalesRange ValuationRangeInput `json:"historical_price_sales_range,omitempty"`
	CurrentFCFYield           *float64            `json:"current_fcf_yield,omitempty"`
	HistoricalFCFYieldRange   ValuationRangeInput `json:"historical_fcf_yield_range,omitempty"`
	ExtraMetrics              map[string]*float64 `json:"extra_metrics,omitempty"`
	Notes                     []string            `json:"notes,omitempty"`
}

type MarketConfirmationMetricsInput struct {
	RelativeStrengthScore   *float64 `json:"relative_strength_score,omitempty"`
	TrendQualityScore       *float64 `json:"trend_quality_score,omitempty"`
	DrawdownFromHighPct     *float64 `json:"drawdown_from_high_pct,omitempty"`
	MaxDrawdown1YPct        *float64 `json:"max_drawdown_1y_pct,omitempty"`
	PriceVs200DMAPct        *float64 `json:"price_vs_200dma_pct,omitempty"`
	MarketConfirmationNotes []string `json:"market_confirmation_notes,omitempty"`
}

type TextEvidenceSummaryInput struct {
	SourceID             string                    `json:"source_id"`
	SourceType           common.EvidenceSourceType `json:"source_type"`
	SourceDate           *time.Time                `json:"source_date,omitempty"`
	SourceTitle          string                    `json:"source_title,omitempty"`
	SourcePeriod         string                    `json:"source_period,omitempty"`
	SourceURLOrPath      string                    `json:"source_url_or_path,omitempty"`
	Summary              string                    `json:"summary"`
	KeyPoints            []string                  `json:"key_points,omitempty"`
	RisksMentioned       []string                  `json:"risks_mentioned,omitempty"`
	ManagementClaims     []string                  `json:"management_claims,omitempty"`
	ExtractedCompetitors []string                  `json:"extracted_competitors,omitempty"`
	ConfidenceScore      *float64                  `json:"confidence_score,omitempty"`
}

type PreviousReviewContextInput struct {
	PreviousReviewID           string                         `json:"previous_review_id,omitempty"`
	PreviousWeightedTotalScore *float64                       `json:"previous_weighted_total_score,omitempty"`
	PreviousAction             common.InvestingActionType     `json:"previous_action,omitempty"`
	PreviousBucket             common.WatchlistBucket         `json:"previous_bucket,omitempty"`
	PreviousSectionScores      map[common.SectionName]float64 `json:"previous_section_scores,omitempty"`
	PreviousThesisStatus       common.ThesisStatus            `json:"previous_thesis_status,omitempty"`
	PreviousSummary            string                         `json:"previous_summary,omitempty"`
}

type ScorecardWeightConfigInput struct {
	SectionWeights  map[common.SectionName]float64                         `json:"section_weights"`
	SubScoreWeights map[common.SectionName]map[common.SubScoreName]float64 `json:"sub_score_weights"`
}
