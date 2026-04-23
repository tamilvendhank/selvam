package dto

import (
	"time"

	"goserver/internal/platform/domain"
	"goserver/internal/platform/ports"
)

type CompanySummary struct {
	ID                    string `json:"id"`
	Symbol                string `json:"symbol"`
	Exchange              string `json:"exchange"`
	CompanyName           string `json:"companyName"`
	Sector                string `json:"sector"`
	Industry              string `json:"industry"`
	MarketCapBucket       string `json:"marketCapBucket"`
	IsInInvestingUniverse bool   `json:"isInInvestingUniverse"`
	IsInTradingUniverse   bool   `json:"isInTradingUniverse"`
	StatusActive          bool   `json:"statusActive"`
}

type ReviewSummary struct {
	ID                     string                 `json:"id"`
	CompanyID              string                 `json:"companyId"`
	Symbol                 string                 `json:"symbol"`
	BookType               domain.BookType        `json:"bookType"`
	ReviewDate             time.Time              `json:"reviewDate"`
	ReviewStatus           domain.ReviewStatus    `json:"reviewStatus"`
	WeightedTotalScore     float64                `json:"weightedTotalScore"`
	ConfidenceScore        float64                `json:"confidenceScore"`
	FinalBucketAfterReview domain.WatchlistBucket `json:"finalBucketAfterReview"`
	FinalActionAfterReview domain.ActionType      `json:"finalActionAfterReview"`
}

type WorkflowRunSummary struct {
	ID                    string                   `json:"id"`
	BookType              domain.BookType          `json:"bookType"`
	RunType               domain.WorkflowRunType   `json:"runType"`
	Status                domain.WorkflowRunStatus `json:"status"`
	StartedAt             time.Time                `json:"startedAt"`
	CompletedAt           *time.Time               `json:"completedAt,omitempty"`
	CompaniesScannedCount int                      `json:"companiesScannedCount"`
	ReviewsCreatedCount   int                      `json:"reviewsCreatedCount"`
	ErrorsCount           int                      `json:"errorsCount"`
	DryRun                bool                     `json:"dryRun"`
}

type CapitalAllocationSummary struct {
	ID                  string          `json:"id"`
	WorkflowRunID       string          `json:"workflowRunId"`
	AllocationDate      time.Time       `json:"allocationDate"`
	BookType            domain.BookType `json:"bookType"`
	AllocatedCashTotal  float64         `json:"allocatedCashTotal"`
	CashLeftUnallocated float64         `json:"cashLeftUnallocated"`
}

type ManualOverrideSummary struct {
	ID               string            `json:"id"`
	CompanyID        string            `json:"companyId"`
	ReviewID         string            `json:"reviewId"`
	BookType         domain.BookType   `json:"bookType"`
	OriginalAction   domain.ActionType `json:"originalAction"`
	OverriddenAction domain.ActionType `json:"overriddenAction"`
	OverrideDate     time.Time         `json:"overrideDate"`
}

type PositionSummary struct {
	ID                          string          `json:"id"`
	CompanyID                   string          `json:"companyId"`
	Symbol                      string          `json:"symbol"`
	BookType                    domain.BookType `json:"bookType"`
	Quantity                    float64         `json:"quantity"`
	MarketValue                 float64         `json:"marketValue"`
	PositionPctOfBook           float64         `json:"positionPctOfBook"`
	PositionPctOfTotalPortfolio float64         `json:"positionPctOfTotalPortfolio"`
	UpdatedAt                   time.Time       `json:"updatedAt"`
}

type StartInvestingWorkflowRequest struct {
	RunType         domain.WorkflowRunType `json:"runType"`
	Mode            domain.InvestingMode   `json:"mode"`
	CompanyIDs      []string               `json:"companyIds,omitempty"`
	Limit           int                    `json:"limit,omitempty"`
	ReplayFromRunID string                 `json:"replayFromRunId,omitempty"`
	IdempotencyKey  string                 `json:"idempotencyKey,omitempty"`
	Notes           string                 `json:"notes,omitempty"`
	RequestedBy     string                 `json:"requestedBy,omitempty"`
}

func (request StartInvestingWorkflowRequest) ToPort(dryRun bool) ports.StartInvestingWorkflowRequest {
	return ports.StartInvestingWorkflowRequest{
		RunType:         request.RunType,
		Mode:            request.Mode,
		CompanyIDs:      request.CompanyIDs,
		Limit:           request.Limit,
		ReplayFromRunID: request.ReplayFromRunID,
		IdempotencyKey:  request.IdempotencyKey,
		DryRun:          dryRun,
		Notes:           request.Notes,
		RequestedBy:     request.RequestedBy,
	}
}

type CreateManualOverrideRequest struct {
	CompanyID        string            `json:"companyId"`
	ReviewID         string            `json:"reviewId"`
	BookType         domain.BookType   `json:"bookType"`
	OriginalAction   domain.ActionType `json:"originalAction"`
	OverriddenAction domain.ActionType `json:"overriddenAction"`
	OverrideReason   string            `json:"overrideReason"`
	OverrideBy       string            `json:"overrideBy"`
}
