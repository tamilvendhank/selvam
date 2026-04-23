package domain

import (
	"fmt"
	"strings"
	"time"
)

type ManualOverride struct {
	ID               string     `json:"id" bson:"-"`
	CompanyID        string     `json:"companyId" bson:"companyId"`
	ReviewID         string     `json:"reviewId" bson:"reviewId"`
	BookType         BookType   `json:"bookType" bson:"bookType"`
	OriginalAction   ActionType `json:"originalAction" bson:"originalAction"`
	OverriddenAction ActionType `json:"overriddenAction" bson:"overriddenAction"`
	OverrideReason   string     `json:"overrideReason" bson:"overrideReason"`
	OverrideBy       string     `json:"overrideBy" bson:"overrideBy"`
	OverrideDate     time.Time  `json:"overrideDate" bson:"overrideDate"`
	SchemaVersion    string     `json:"schemaVersion" bson:"schemaVersion"`
	CreatedAt        time.Time  `json:"createdAt" bson:"createdAt"`
}

func (override *ManualOverride) Validate() error {
	if override == nil {
		return fmt.Errorf("manual override is required")
	}
	if strings.TrimSpace(override.CompanyID) == "" {
		return fmt.Errorf("manual override companyId is required")
	}
	if strings.TrimSpace(override.ReviewID) == "" {
		return fmt.Errorf("manual override reviewId is required")
	}
	if !IsValidBookType(override.BookType) {
		return fmt.Errorf("invalid manual override book type %q", override.BookType)
	}
	if !IsValidActionType(override.OriginalAction) {
		return fmt.Errorf("invalid original action %q", override.OriginalAction)
	}
	if !IsValidActionType(override.OverriddenAction) {
		return fmt.Errorf("invalid overridden action %q", override.OverriddenAction)
	}
	if strings.TrimSpace(override.OverrideReason) == "" {
		return fmt.Errorf("manual override reason is required")
	}
	if strings.TrimSpace(override.OverrideBy) == "" {
		return fmt.Errorf("manual override overrideBy is required")
	}
	if strings.TrimSpace(override.SchemaVersion) == "" {
		return fmt.Errorf("manual override schema version is required")
	}
	if err := ValidateNonZeroTime("manual override overrideDate", override.OverrideDate); err != nil {
		return err
	}
	if err := ValidateNonZeroTime("manual override createdAt", override.CreatedAt); err != nil {
		return err
	}

	return nil
}

type CurrentPosition struct {
	ID                          string     `json:"id" bson:"-"`
	CompanyID                   string     `json:"companyId" bson:"companyId"`
	Symbol                      string     `json:"symbol" bson:"symbol"`
	BookType                    BookType   `json:"bookType" bson:"bookType"`
	Quantity                    float64    `json:"quantity" bson:"quantity"`
	AverageCost                 float64    `json:"averageCost" bson:"averageCost"`
	MarketPrice                 float64    `json:"marketPrice" bson:"marketPrice"`
	MarketValue                 float64    `json:"marketValue" bson:"marketValue"`
	PositionPctOfBook           float64    `json:"positionPctOfBook" bson:"positionPctOfBook"`
	PositionPctOfTotalPortfolio float64    `json:"positionPctOfTotalPortfolio" bson:"positionPctOfTotalPortfolio"`
	TargetPositionPct           float64    `json:"targetPositionPct" bson:"targetPositionPct"`
	MaxPositionPct              float64    `json:"maxPositionPct" bson:"maxPositionPct"`
	LastReviewID                string     `json:"lastReviewId,omitempty" bson:"lastReviewId,omitempty"`
	OwnedSinceDate              *time.Time `json:"ownedSinceDate,omitempty" bson:"ownedSinceDate,omitempty"`
	SchemaVersion               string     `json:"schemaVersion" bson:"schemaVersion"`
	AsOf                        time.Time  `json:"asOf" bson:"asOf"`
	CreatedAt                   time.Time  `json:"createdAt" bson:"createdAt"`
	UpdatedAt                   time.Time  `json:"updatedAt" bson:"updatedAt"`
}

func (position *CurrentPosition) Validate() error {
	if position == nil {
		return fmt.Errorf("current position is required")
	}
	if strings.TrimSpace(position.CompanyID) == "" {
		return fmt.Errorf("current position companyId is required")
	}
	if strings.TrimSpace(position.Symbol) == "" {
		return fmt.Errorf("current position symbol is required")
	}
	if !IsValidBookType(position.BookType) {
		return fmt.Errorf("invalid current position book type %q", position.BookType)
	}
	if err := ValidatePercentRange("position pct of book", position.PositionPctOfBook); err != nil {
		return err
	}
	if err := ValidatePercentRange("position pct of total portfolio", position.PositionPctOfTotalPortfolio); err != nil {
		return err
	}
	if err := ValidatePercentRange("target position pct", position.TargetPositionPct); err != nil {
		return err
	}
	if err := ValidatePercentRange("max position pct", position.MaxPositionPct); err != nil {
		return err
	}
	if strings.TrimSpace(position.SchemaVersion) == "" {
		return fmt.Errorf("current position schema version is required")
	}
	if err := ValidateNonZeroTime("current position asOf", position.AsOf); err != nil {
		return err
	}
	if err := ValidateNonZeroTime("current position createdAt", position.CreatedAt); err != nil {
		return err
	}
	if err := ValidateNonZeroTime("current position updatedAt", position.UpdatedAt); err != nil {
		return err
	}

	return nil
}
