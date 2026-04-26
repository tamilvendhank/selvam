package app

import (
	"time"

	domaincommon "goserver/internal/domain/common"
)

type ThesisListItemDTO struct {
	ThesisID                string                    `json:"thesisId"`
	CompanyID               string                    `json:"companyId,omitempty"`
	ThesisStatus            domaincommon.ThesisStatus `json:"thesisStatus,omitempty"`
	ThesisVersion           int                       `json:"thesisVersion,omitempty"`
	ThesisSummary           string                    `json:"thesisSummary,omitempty"`
	ConfidenceLevel         float64                   `json:"confidenceLevel,omitempty"`
	ThesisHealthScore       float64                   `json:"thesisHealthScore,omitempty"`
	CurrentPositionRole     domaincommon.PositionRole `json:"currentPositionRole,omitempty"`
	CreatedFromReviewID     string                    `json:"createdFromReviewId,omitempty"`
	LastUpdatedFromReviewID string                    `json:"lastUpdatedFromReviewId,omitempty"`
	CreatedAt               time.Time                 `json:"createdAt,omitempty"`
	UpdatedAt               time.Time                 `json:"updatedAt,omitempty"`
}

type ThesisDetailDTO struct {
	ThesisListItemDTO
	WhyThisBusinessCanCompound string   `json:"whyThisBusinessCanCompound,omitempty"`
	KeyGrowthDrivers           []string `json:"keyGrowthDrivers,omitempty"`
	KeyMoatOrAdvantageFactors  []string `json:"keyMoatOrAdvantageFactors,omitempty"`
	WhyNow                     string   `json:"whyNow,omitempty"`
	KeyRisks                   []string `json:"keyRisks,omitempty"`
	DisconfirmingSignals       []string `json:"disconfirmingSignals,omitempty"`
	WhatWouldBreakTheThesis    []string `json:"whatWouldBreakTheThesis,omitempty"`
	DesiredHoldingPeriod       string   `json:"desiredHoldingPeriod,omitempty"`
	ThesisChangeSummary        string   `json:"thesisChangeSummary,omitempty"`
	NewSupportingEvidence      []string `json:"supportingEvidenceSummaries,omitempty"`
	NewContradictingEvidence   []string `json:"contradictingEvidenceSummaries,omitempty"`
	LinkedReviewIDs            []string `json:"linkedReviewIds,omitempty"`
}

type ThesisHistoryItemDTO struct {
	ThesisID                string                    `json:"thesisId"`
	ThesisStatus            domaincommon.ThesisStatus `json:"thesisStatus,omitempty"`
	ThesisVersion           int                       `json:"thesisVersion,omitempty"`
	ThesisSummary           string                    `json:"thesisSummary,omitempty"`
	ThesisHealthScore       float64                   `json:"thesisHealthScore,omitempty"`
	ThesisChangeSummary     string                    `json:"thesisChangeSummary,omitempty"`
	LastUpdatedFromReviewID string                    `json:"lastUpdatedFromReviewId,omitempty"`
	UpdatedAt               time.Time                 `json:"updatedAt,omitempty"`
}
