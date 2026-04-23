package domain

import (
	"fmt"
	"strings"
	"time"
)

type Company struct {
	ID                    string    `json:"id" bson:"-"`
	Symbol                string    `json:"symbol" bson:"symbol"`
	Exchange              string    `json:"exchange" bson:"exchange"`
	CompanyName           string    `json:"companyName" bson:"companyName"`
	Sector                string    `json:"sector" bson:"sector"`
	Industry              string    `json:"industry" bson:"industry"`
	SubIndustry           string    `json:"subIndustry" bson:"subIndustry"`
	BusinessSummary       string    `json:"businessSummary" bson:"businessSummary"`
	ListingDate           time.Time `json:"listingDate" bson:"listingDate"`
	MarketCapBucket       string    `json:"marketCapBucket" bson:"marketCapBucket"`
	IsInInvestingUniverse bool      `json:"isInInvestingUniverse" bson:"isInInvestingUniverse"`
	IsInTradingUniverse   bool      `json:"isInTradingUniverse" bson:"isInTradingUniverse"`
	StatusActive          bool      `json:"statusActive" bson:"statusActive"`
	SchemaVersion         string    `json:"schemaVersion" bson:"schemaVersion"`
	CreatedAt             time.Time `json:"createdAt" bson:"createdAt"`
	UpdatedAt             time.Time `json:"updatedAt" bson:"updatedAt"`
}

func (company *Company) Validate() error {
	if company == nil {
		return fmt.Errorf("company is required")
	}
	if strings.TrimSpace(company.Symbol) == "" {
		return fmt.Errorf("company symbol is required")
	}
	if strings.TrimSpace(company.Exchange) == "" {
		return fmt.Errorf("company exchange is required")
	}
	if strings.TrimSpace(company.CompanyName) == "" {
		return fmt.Errorf("company name is required")
	}
	if strings.TrimSpace(company.SchemaVersion) == "" {
		return fmt.Errorf("company schema version is required")
	}
	if err := ValidateNonZeroTime("company createdAt", company.CreatedAt); err != nil {
		return err
	}
	if err := ValidateNonZeroTime("company updatedAt", company.UpdatedAt); err != nil {
		return err
	}

	return nil
}
