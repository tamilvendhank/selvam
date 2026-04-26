package app

import (
	"time"

	domaincommon "goserver/internal/domain/common"
)

type CompanyListItemDTO struct {
	CompanyID             string                           `json:"companyId"`
	Symbol                string                           `json:"symbol"`
	Exchange              string                           `json:"exchange,omitempty"`
	CompanyName           string                           `json:"companyName,omitempty"`
	Sector                string                           `json:"sector,omitempty"`
	Industry              string                           `json:"industry,omitempty"`
	SubIndustry           string                           `json:"subIndustry,omitempty"`
	MarketCapBucket       string                           `json:"marketCapBucket,omitempty"`
	IsInInvestingUniverse bool                             `json:"isInInvestingUniverse"`
	IsInTradingUniverse   bool                             `json:"isInTradingUniverse"`
	StatusActive          bool                             `json:"statusActive"`
	LatestReviewSummary   *ReviewSummaryDTO                `json:"latestReviewSummary,omitempty"`
	LatestAction          domaincommon.InvestingActionType `json:"latestAction,omitempty"`
	LatestBucket          domaincommon.WatchlistBucket     `json:"latestBucket,omitempty"`
	CreatedAt             time.Time                        `json:"createdAt,omitempty"`
	UpdatedAt             time.Time                        `json:"updatedAt,omitempty"`
}

type CompanyDetailDTO struct {
	CompanyID                 string               `json:"companyId"`
	Symbol                    string               `json:"symbol"`
	Exchange                  string               `json:"exchange,omitempty"`
	CompanyName               string               `json:"companyName,omitempty"`
	Sector                    string               `json:"sector,omitempty"`
	Industry                  string               `json:"industry,omitempty"`
	SubIndustry               string               `json:"subIndustry,omitempty"`
	BusinessSummary           string               `json:"businessSummary,omitempty"`
	ListingDate               time.Time            `json:"listingDate,omitempty"`
	MarketCapBucket           string               `json:"marketCapBucket,omitempty"`
	IsInInvestingUniverse     bool                 `json:"isInInvestingUniverse"`
	IsInTradingUniverse       bool                 `json:"isInTradingUniverse"`
	StatusActive              bool                 `json:"statusActive"`
	LatestInvestingReview     *ReviewSummaryDTO    `json:"latestInvestingReview,omitempty"`
	LatestTradingReview       *ReviewSummaryDTO    `json:"latestTradingReview,omitempty"`
	ActiveThesis              *ThesisListItemDTO   `json:"activeThesis,omitempty"`
	CurrentPositions          []CurrentPositionDTO `json:"currentPositions,omitempty"`
	LatestAllocationRelevance []AllocationItemDTO  `json:"latestAllocationRelevance,omitempty"`
	CreatedAt                 time.Time            `json:"createdAt,omitempty"`
	UpdatedAt                 time.Time            `json:"updatedAt,omitempty"`
}

type CompanyHistorySummaryDTO struct {
	CompanyID         string                     `json:"companyId"`
	ReviewCount       int                        `json:"reviewCount"`
	LatestReviewDate  *time.Time                 `json:"latestReviewDate,omitempty"`
	ScoreTrend        []ScorePointDTO            `json:"scoreTrend,omitempty"`
	ActionHistory     []ActionHistoryItemDTO     `json:"actionHistory,omitempty"`
	ThesisHistory     []ThesisHistoryItemDTO     `json:"thesisStatusHistory,omitempty"`
	AllocationHistory []AllocationHistoryItemDTO `json:"allocationHistory,omitempty"`
}

type ScorePointDTO struct {
	ReviewID           string                `json:"reviewId"`
	BookType           domaincommon.BookType `json:"bookType,omitempty"`
	ReviewDate         time.Time             `json:"reviewDate"`
	WeightedTotalScore float64               `json:"weightedTotalScore"`
}

type ActionHistoryItemDTO struct {
	ReviewID string                           `json:"reviewId"`
	Date     time.Time                        `json:"date"`
	Action   domaincommon.InvestingActionType `json:"action,omitempty"`
	Bucket   domaincommon.WatchlistBucket     `json:"bucket,omitempty"`
	Summary  string                           `json:"summary,omitempty"`
}

type AllocationHistoryItemDTO struct {
	AllocationRunID string    `json:"allocationRunId"`
	WorkflowRunID   string    `json:"workflowRunId,omitempty"`
	AllocationDate  time.Time `json:"allocationDate"`
	Amount          float64   `json:"amount"`
	Blocked         bool      `json:"blocked,omitempty"`
	Reason          string    `json:"reason,omitempty"`
}
