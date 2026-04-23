package position

import (
	"fmt"
	"time"

	"goserver/internal/domain/common"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

type CurrentPosition struct {
	ID                            primitive.ObjectID `bson:"_id,omitempty" json:"id,omitempty"`
	CompanyID                     primitive.ObjectID `bson:"companyId" json:"companyId"`
	BookType                      common.BookType    `bson:"bookType" json:"bookType"`
	IsOpen                        bool               `bson:"isOpen" json:"isOpen"`
	Quantity                      float64            `bson:"quantity" json:"quantity"`
	AverageCost                   float64            `bson:"averageCost" json:"averageCost"`
	CurrentMarketValue            float64            `bson:"currentMarketValue,omitempty" json:"currentMarketValue,omitempty"`
	CurrentPositionPctOfBook      float64            `bson:"currentPositionPctOfBook,omitempty" json:"currentPositionPctOfBook,omitempty"`
	CurrentPositionPctOfPortfolio float64            `bson:"currentPositionPctOfPortfolio,omitempty" json:"currentPositionPctOfPortfolio,omitempty"`
	LastUpdatedAt                 time.Time          `bson:"lastUpdatedAt" json:"lastUpdatedAt"`
	SchemaVersion                 int                `bson:"schemaVersion" json:"schemaVersion"`
}

func (position CurrentPosition) Validate() error {
	if err := common.RequireObjectID("companyId", position.CompanyID); err != nil {
		return err
	}
	if !position.BookType.IsValid() {
		return fmt.Errorf("invalid bookType %q", position.BookType)
	}
	if err := common.ValidateNonNegativeFloat("quantity", position.Quantity); err != nil {
		return err
	}
	if err := common.ValidateNonNegativeFloat("averageCost", position.AverageCost); err != nil {
		return err
	}
	if err := common.ValidateNonNegativeFloat("currentMarketValue", position.CurrentMarketValue); err != nil {
		return err
	}
	if err := common.ValidatePercentage("currentPositionPctOfBook", position.CurrentPositionPctOfBook); err != nil {
		return err
	}
	if err := common.ValidatePercentage("currentPositionPctOfPortfolio", position.CurrentPositionPctOfPortfolio); err != nil {
		return err
	}
	if position.IsOpen && position.Quantity <= 0 {
		return fmt.Errorf("open positions must have quantity greater than zero")
	}
	if !position.IsOpen && position.Quantity > 0 {
		return fmt.Errorf("closed positions must have zero quantity")
	}
	if err := common.RequireTime("lastUpdatedAt", position.LastUpdatedAt); err != nil {
		return err
	}
	if err := common.ValidateSchemaVersion("schemaVersion", position.SchemaVersion); err != nil {
		return err
	}
	return nil
}
