package mongo

import (
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
)

func companyIndexModels() []mongo.IndexModel {
	return []mongo.IndexModel{
		// The repository contract resolves companies by symbol globally, so the storage layer
		// enforces global symbol uniqueness instead of exchange+symbol uniqueness.
		newUniqueIndex(
			"ux_companies_symbol",
			bson.D{{Key: "symbol", Value: 1}},
		),
		newIndex(
			"ix_companies_investing_universe_status_updated_at_desc",
			bson.D{
				{Key: "isInInvestingUniverse", Value: 1},
				{Key: "statusActive", Value: 1},
				{Key: "updatedAt", Value: -1},
			},
		),
		newIndex(
			"ix_companies_trading_universe_status_updated_at_desc",
			bson.D{
				{Key: "isInTradingUniverse", Value: 1},
				{Key: "statusActive", Value: 1},
				{Key: "updatedAt", Value: -1},
			},
		),
		newIndex(
			"ix_companies_sector_industry_sub_industry_updated_at_desc",
			bson.D{
				{Key: "sector", Value: 1},
				{Key: "industry", Value: 1},
				{Key: "subIndustry", Value: 1},
				{Key: "updatedAt", Value: -1},
			},
		),
		newIndex(
			"ix_companies_market_cap_status_updated_at_desc",
			bson.D{
				{Key: "marketCapBucket", Value: 1},
				{Key: "statusActive", Value: 1},
				{Key: "updatedAt", Value: -1},
			},
		),
		newIndex(
			"ix_companies_updated_at_desc",
			bson.D{{Key: "updatedAt", Value: -1}},
		),
	}
}
