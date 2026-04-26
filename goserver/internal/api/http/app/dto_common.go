package app

import (
	"time"

	domaincommon "goserver/internal/domain/common"
)

type PageDTO struct {
	Limit   int  `json:"limit"`
	Offset  int  `json:"offset"`
	HasMore bool `json:"hasMore"`
}

type PagedResponseDTO[T any] struct {
	Items []T     `json:"items"`
	Page  PageDTO `json:"page"`
}

type ErrorResponseDTO struct {
	Error string `json:"error"`
	Code  string `json:"code,omitempty"`
}

type ReviewSummaryDTO struct {
	ReviewID               string                            `json:"reviewId"`
	CompanyID              string                            `json:"companyId,omitempty"`
	Symbol                 string                            `json:"symbol,omitempty"`
	BookType               domaincommon.BookType             `json:"bookType,omitempty"`
	WorkflowRunID          string                            `json:"workflowRunId,omitempty"`
	ReviewDate             time.Time                         `json:"reviewDate"`
	ReviewStatus           domaincommon.ReviewStatus         `json:"reviewStatus,omitempty"`
	ReviewLifecycleState   domaincommon.ReviewLifecycleState `json:"reviewLifecycleState,omitempty"`
	WeightedTotalScore     float64                           `json:"weightedTotalScore,omitempty"`
	ConfidenceScore        float64                           `json:"confidenceScore,omitempty"`
	HardGateFailed         bool                              `json:"hardGateFailed,omitempty"`
	FinalActionAfterReview domaincommon.InvestingActionType  `json:"finalActionAfterReview,omitempty"`
	FinalBucketAfterReview domaincommon.WatchlistBucket      `json:"finalBucketAfterReview,omitempty"`
	ActionRationaleSummary string                            `json:"actionRationaleSummary,omitempty"`
	WhatChangedSummary     string                            `json:"whatChangedSummary,omitempty"`
	ConfigSnapshotID       string                            `json:"configSnapshotId,omitempty"`
	CreatedAt              time.Time                         `json:"createdAt,omitempty"`
	UpdatedAt              time.Time                         `json:"updatedAt,omitempty"`
	FinalizedAt            *time.Time                        `json:"finalizedAt,omitempty"`
}

type CurrentPositionDTO struct {
	PositionID                    string                `json:"positionId"`
	CompanyID                     string                `json:"companyId,omitempty"`
	Symbol                        string                `json:"symbol,omitempty"`
	BookType                      domaincommon.BookType `json:"bookType,omitempty"`
	IsOpen                        bool                  `json:"isOpen"`
	Quantity                      float64               `json:"quantity"`
	AverageCost                   float64               `json:"averageCost"`
	CurrentMarketValue            float64               `json:"currentMarketValue"`
	CurrentPositionPctOfBook      float64               `json:"currentPositionPctOfBook"`
	CurrentPositionPctOfPortfolio float64               `json:"currentPositionPctOfPortfolio"`
	LastUpdatedAt                 time.Time             `json:"lastUpdatedAt"`
}
