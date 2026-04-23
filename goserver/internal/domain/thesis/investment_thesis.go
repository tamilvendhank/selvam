package thesis

import (
	"fmt"
	"time"

	"goserver/internal/domain/common"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

// InvestmentThesis is an investing-book-only document that evolves over time.
type InvestmentThesis struct {
	ID                         primitive.ObjectID  `bson:"_id,omitempty" json:"id,omitempty"`
	CompanyID                  primitive.ObjectID  `bson:"companyId" json:"companyId"`
	ThesisStatus               common.ThesisStatus `bson:"thesisStatus" json:"thesisStatus"`
	ThesisVersion              int                 `bson:"thesisVersion" json:"thesisVersion"`
	CreatedFromReviewID        primitive.ObjectID  `bson:"createdFromReviewId" json:"createdFromReviewId"`
	LastUpdatedFromReviewID    primitive.ObjectID  `bson:"lastUpdatedFromReviewId" json:"lastUpdatedFromReviewId"`
	ThesisSummary              string              `bson:"thesisSummary" json:"thesisSummary"`
	WhyThisBusinessCanCompound string              `bson:"whyThisBusinessCanCompound" json:"whyThisBusinessCanCompound"`
	KeyGrowthDrivers           []string            `bson:"keyGrowthDrivers,omitempty" json:"keyGrowthDrivers,omitempty"`
	KeyMoatOrAdvantageFactors  []string            `bson:"keyMoatOrAdvantageFactors,omitempty" json:"keyMoatOrAdvantageFactors,omitempty"`
	WhyNow                     string              `bson:"whyNow,omitempty" json:"whyNow,omitempty"`
	KeyRisks                   []string            `bson:"keyRisks,omitempty" json:"keyRisks,omitempty"`
	DisconfirmingSignals       []string            `bson:"disconfirmingSignals,omitempty" json:"disconfirmingSignals,omitempty"`
	WhatWouldBreakTheThesis    []string            `bson:"whatWouldBreakTheThesis,omitempty" json:"whatWouldBreakTheThesis,omitempty"`
	DesiredHoldingPeriod       string              `bson:"desiredHoldingPeriod,omitempty" json:"desiredHoldingPeriod,omitempty"`
	ConfidenceLevel            float64             `bson:"confidenceLevel" json:"confidenceLevel"`
	ThesisHealthScore          float64             `bson:"thesisHealthScore" json:"thesisHealthScore"`
	ThesisChangeSummary        string              `bson:"thesisChangeSummary,omitempty" json:"thesisChangeSummary,omitempty"`
	NewSupportingEvidence      []string            `bson:"newSupportingEvidence,omitempty" json:"newSupportingEvidence,omitempty"`
	NewContradictingEvidence   []string            `bson:"newContradictingEvidence,omitempty" json:"newContradictingEvidence,omitempty"`
	CurrentPositionRole        common.PositionRole `bson:"currentPositionRole,omitempty" json:"currentPositionRole,omitempty"`
	CreatedAt                  time.Time           `bson:"createdAt" json:"createdAt"`
	UpdatedAt                  time.Time           `bson:"updatedAt" json:"updatedAt"`
	SchemaVersion              int                 `bson:"schemaVersion" json:"schemaVersion"`
}

var allowedThesisTransitions = map[common.ThesisStatus]map[common.ThesisStatus]struct{}{
	common.ThesisStatusActive: {
		common.ThesisStatusUnderReview: {},
		common.ThesisStatusBroken:      {},
		common.ThesisStatusArchived:    {},
	},
	common.ThesisStatusUnderReview: {
		common.ThesisStatusActive:   {},
		common.ThesisStatusBroken:   {},
		common.ThesisStatusArchived: {},
	},
	common.ThesisStatusBroken: {
		common.ThesisStatusUnderReview: {},
		common.ThesisStatusArchived:    {},
	},
	common.ThesisStatusArchived: {},
}

func (thesis InvestmentThesis) Validate() error {
	if err := common.RequireObjectID("companyId", thesis.CompanyID); err != nil {
		return err
	}
	if !thesis.ThesisStatus.IsValid() {
		return fmt.Errorf("invalid thesisStatus %q", thesis.ThesisStatus)
	}
	if err := common.ValidatePositiveInt("thesisVersion", thesis.ThesisVersion); err != nil {
		return err
	}
	if err := common.RequireObjectID("createdFromReviewId", thesis.CreatedFromReviewID); err != nil {
		return err
	}
	if thesis.ThesisVersion > 1 {
		if err := common.RequireObjectID("lastUpdatedFromReviewId", thesis.LastUpdatedFromReviewID); err != nil {
			return err
		}
	}
	if err := common.RequireString("thesisSummary", thesis.ThesisSummary); err != nil {
		return err
	}
	if err := common.RequireString("whyThisBusinessCanCompound", thesis.WhyThisBusinessCanCompound); err != nil {
		return err
	}
	if err := common.ValidateStringSlice("keyGrowthDrivers", thesis.KeyGrowthDrivers); err != nil {
		return err
	}
	if err := common.ValidateStringSlice("keyMoatOrAdvantageFactors", thesis.KeyMoatOrAdvantageFactors); err != nil {
		return err
	}
	if err := common.ValidateStringSlice("keyRisks", thesis.KeyRisks); err != nil {
		return err
	}
	if err := common.ValidateStringSlice("disconfirmingSignals", thesis.DisconfirmingSignals); err != nil {
		return err
	}
	if err := common.ValidateStringSlice("whatWouldBreakTheThesis", thesis.WhatWouldBreakTheThesis); err != nil {
		return err
	}
	if err := common.ValidateStringSlice("newSupportingEvidence", thesis.NewSupportingEvidence); err != nil {
		return err
	}
	if err := common.ValidateStringSlice("newContradictingEvidence", thesis.NewContradictingEvidence); err != nil {
		return err
	}
	if err := common.ValidateUnitInterval("confidenceLevel", thesis.ConfidenceLevel); err != nil {
		return err
	}
	if err := common.ValidateComputedScore("thesisHealthScore", thesis.ThesisHealthScore); err != nil {
		return err
	}
	if !thesis.CurrentPositionRole.IsValid() {
		return fmt.Errorf("invalid currentPositionRole %q", thesis.CurrentPositionRole)
	}
	if err := common.RequireTime("createdAt", thesis.CreatedAt); err != nil {
		return err
	}
	if err := common.RequireTime("updatedAt", thesis.UpdatedAt); err != nil {
		return err
	}
	if err := common.ValidateTimestampOrder("createdAt", thesis.CreatedAt, "updatedAt", thesis.UpdatedAt); err != nil {
		return err
	}
	if err := common.ValidateSchemaVersion("schemaVersion", thesis.SchemaVersion); err != nil {
		return err
	}
	return nil
}

func (thesis InvestmentThesis) CanTransitionTo(next common.ThesisStatus) bool {
	if thesis.ThesisStatus == next {
		return true
	}
	nextStates, ok := allowedThesisTransitions[thesis.ThesisStatus]
	if !ok {
		return false
	}
	_, ok = nextStates[next]
	return ok
}

func (thesis *InvestmentThesis) TransitionTo(next common.ThesisStatus, at time.Time) error {
	if thesis == nil {
		return fmt.Errorf("thesis is required")
	}
	if !next.IsValid() {
		return fmt.Errorf("invalid next thesis status %q", next)
	}
	if !thesis.CanTransitionTo(next) {
		return fmt.Errorf("invalid thesis transition from %q to %q", thesis.ThesisStatus, next)
	}
	if err := common.RequireTime("transitionAt", at); err != nil {
		return err
	}
	thesis.ThesisStatus = next
	thesis.UpdatedAt = at.UTC()
	return nil
}
