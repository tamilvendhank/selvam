package domain

import (
	"fmt"
	"strings"
	"time"
)

type InvestmentThesis struct {
	ID                         string       `json:"id" bson:"-"`
	CompanyID                  string       `json:"companyId" bson:"companyId"`
	ThesisStatus               ThesisStatus `json:"thesisStatus" bson:"thesisStatus"`
	ThesisVersion              int          `json:"thesisVersion" bson:"thesisVersion"`
	CreatedFromReviewID        string       `json:"createdFromReviewId" bson:"createdFromReviewId"`
	LastUpdatedFromReviewID    string       `json:"lastUpdatedFromReviewId" bson:"lastUpdatedFromReviewId"`
	ThesisSummary              string       `json:"thesisSummary" bson:"thesisSummary"`
	WhyThisBusinessCanCompound string       `json:"whyThisBusinessCanCompound" bson:"whyThisBusinessCanCompound"`
	KeyGrowthDrivers           []string     `json:"keyGrowthDrivers,omitempty" bson:"keyGrowthDrivers,omitempty"`
	KeyMoatOrAdvantageFactors  []string     `json:"keyMoatOrAdvantageFactors,omitempty" bson:"keyMoatOrAdvantageFactors,omitempty"`
	WhyNow                     string       `json:"whyNow,omitempty" bson:"whyNow,omitempty"`
	KeyRisks                   []string     `json:"keyRisks,omitempty" bson:"keyRisks,omitempty"`
	DisconfirmingSignals       []string     `json:"disconfirmingSignals,omitempty" bson:"disconfirmingSignals,omitempty"`
	WhatWouldBreakTheThesis    []string     `json:"whatWouldBreakTheThesis,omitempty" bson:"whatWouldBreakTheThesis,omitempty"`
	DesiredHoldingPeriod       string       `json:"desiredHoldingPeriod,omitempty" bson:"desiredHoldingPeriod,omitempty"`
	ConfidenceLevel            float64      `json:"confidenceLevel" bson:"confidenceLevel"`
	ThesisHealthScore          float64      `json:"thesisHealthScore" bson:"thesisHealthScore"`
	ThesisChangeSummary        string       `json:"thesisChangeSummary,omitempty" bson:"thesisChangeSummary,omitempty"`
	NewSupportingEvidence      []string     `json:"newSupportingEvidence,omitempty" bson:"newSupportingEvidence,omitempty"`
	NewContradictingEvidence   []string     `json:"newContradictingEvidence,omitempty" bson:"newContradictingEvidence,omitempty"`
	CurrentPositionRole        PositionRole `json:"currentPositionRole,omitempty" bson:"currentPositionRole,omitempty"`
	SchemaVersion              string       `json:"schemaVersion" bson:"schemaVersion"`
	CreatedAt                  time.Time    `json:"createdAt" bson:"createdAt"`
	UpdatedAt                  time.Time    `json:"updatedAt" bson:"updatedAt"`
}

func (thesis *InvestmentThesis) Validate() error {
	if thesis == nil {
		return fmt.Errorf("investment thesis is required")
	}
	if strings.TrimSpace(thesis.CompanyID) == "" {
		return fmt.Errorf("thesis companyId is required")
	}
	if !IsValidThesisStatus(thesis.ThesisStatus) {
		return fmt.Errorf("invalid thesis status %q", thesis.ThesisStatus)
	}
	if thesis.ThesisVersion <= 0 {
		return fmt.Errorf("thesis version must be greater than zero")
	}
	if strings.TrimSpace(thesis.ThesisSummary) == "" {
		return fmt.Errorf("thesis summary is required")
	}
	if strings.TrimSpace(thesis.WhyThisBusinessCanCompound) == "" {
		return fmt.Errorf("whyThisBusinessCanCompound is required")
	}
	if err := ValidateUnitRange("thesis confidence level", thesis.ConfidenceLevel); err != nil {
		return err
	}
	if thesis.ThesisHealthScore < 0 || thesis.ThesisHealthScore > 10 {
		return fmt.Errorf("thesis health score must be between 0 and 10")
	}
	if thesis.CurrentPositionRole != "" && !IsValidPositionRole(thesis.CurrentPositionRole) {
		return fmt.Errorf("invalid current position role %q", thesis.CurrentPositionRole)
	}
	if strings.TrimSpace(thesis.SchemaVersion) == "" {
		return fmt.Errorf("thesis schema version is required")
	}
	if err := ValidateNonZeroTime("thesis createdAt", thesis.CreatedAt); err != nil {
		return err
	}
	if err := ValidateNonZeroTime("thesis updatedAt", thesis.UpdatedAt); err != nil {
		return err
	}

	return nil
}
