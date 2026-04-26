package contracts

type InvestingReviewInputPayload struct {
	Company               CompanyIdentityInput             `json:"company"`
	ReviewContext         ReviewContextInput               `json:"review_context"`
	CurrentPosition       *CurrentPositionInput            `json:"current_position,omitempty"`
	AnnualMetrics         []AnnualFinancialMetricsInput    `json:"annual_metrics"`
	QuarterlyMetrics      []QuarterlyFinancialMetricsInput `json:"quarterly_metrics,omitempty"`
	Valuation             ValuationMetricsInput            `json:"valuation,omitempty"`
	MarketConfirmation    MarketConfirmationMetricsInput   `json:"market_confirmation,omitempty"`
	TextEvidenceSummaries []TextEvidenceSummaryInput       `json:"text_evidence_summaries,omitempty"`
	PreviousReview        *PreviousReviewContextInput      `json:"previous_review,omitempty"`
	Weights               ScorecardWeightConfigInput       `json:"weights"`
}
