package company

import (
	"fmt"
	"strings"
	"time"

	"goserver/internal/domain/common"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

// Company is the canonical company document shared across investing and trading books.
type Company struct {
	ID                    primitive.ObjectID `bson:"_id,omitempty" json:"id,omitempty"`
	Symbol                string             `bson:"symbol" json:"symbol"`
	Exchange              string             `bson:"exchange" json:"exchange"`
	CompanyName           string             `bson:"companyName" json:"companyName"`
	Sector                string             `bson:"sector" json:"sector"`
	Industry              string             `bson:"industry" json:"industry"`
	SubIndustry           string             `bson:"subIndustry,omitempty" json:"subIndustry,omitempty"`
	BusinessSummary       string             `bson:"businessSummary,omitempty" json:"businessSummary,omitempty"`
	ListingDate           time.Time          `bson:"listingDate" json:"listingDate"`
	MarketCapBucket       string             `bson:"marketCapBucket" json:"marketCapBucket"`
	IsInInvestingUniverse bool               `bson:"isInInvestingUniverse" json:"isInInvestingUniverse"`
	IsInTradingUniverse   bool               `bson:"isInTradingUniverse" json:"isInTradingUniverse"`
	StatusActive          bool               `bson:"statusActive" json:"statusActive"`
	CreatedAt             time.Time          `bson:"createdAt" json:"createdAt"`
	UpdatedAt             time.Time          `bson:"updatedAt" json:"updatedAt"`
	SchemaVersion         int                `bson:"schemaVersion" json:"schemaVersion"`
}

func (company Company) Validate() error {
	if err := common.RequireString("symbol", company.Symbol); err != nil {
		return err
	}
	if err := common.RequireString("exchange", company.Exchange); err != nil {
		return err
	}
	if err := common.RequireString("companyName", company.CompanyName); err != nil {
		return err
	}
	if err := common.RequireString("sector", company.Sector); err != nil {
		return err
	}
	if err := common.RequireString("industry", company.Industry); err != nil {
		return err
	}
	if err := common.RequireString("marketCapBucket", company.MarketCapBucket); err != nil {
		return err
	}
	if err := common.RequireTime("listingDate", company.ListingDate); err != nil {
		return err
	}
	if err := common.RequireTime("createdAt", company.CreatedAt); err != nil {
		return err
	}
	if err := common.RequireTime("updatedAt", company.UpdatedAt); err != nil {
		return err
	}
	if err := common.ValidateTimestampOrder("createdAt", company.CreatedAt, "updatedAt", company.UpdatedAt); err != nil {
		return err
	}
	if err := common.ValidateSchemaVersion("schemaVersion", company.SchemaVersion); err != nil {
		return err
	}
	if !company.IsInInvestingUniverse && !company.IsInTradingUniverse {
		return fmt.Errorf("company must belong to at least one book universe")
	}
	if strings.TrimSpace(company.Symbol) != strings.ToUpper(strings.TrimSpace(company.Symbol)) {
		return fmt.Errorf("symbol must be normalized to uppercase")
	}
	return nil
}

func (company Company) InUniverse(bookType common.BookType) bool {
	switch bookType {
	case common.BookTypeInvesting:
		return company.IsInInvestingUniverse
	case common.BookTypeTrading:
		return company.IsInTradingUniverse
	default:
		return false
	}
}
