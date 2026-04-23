package mongo

import (
	"testing"
	"time"

	"goserver/internal/domain/common"
	companypkg "goserver/internal/domain/company"
	reviewpkg "goserver/internal/domain/review"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

func TestCompanyDocumentRoundTrip(t *testing.T) {
	original := sampleCompany()
	document := toCompanyDocument(original)

	encoded, err := bson.Marshal(document)
	if err != nil {
		t.Fatalf("Marshal returned error: %v", err)
	}

	var decoded companyDocument
	if err := bson.Unmarshal(encoded, &decoded); err != nil {
		t.Fatalf("Unmarshal returned error: %v", err)
	}

	roundTrip := fromCompanyDocument(&decoded)
	if roundTrip.Symbol != original.Symbol {
		t.Fatalf("expected symbol %s, got %s", original.Symbol, roundTrip.Symbol)
	}
	if roundTrip.ID != original.ID {
		t.Fatalf("expected id %s, got %s", original.ID.Hex(), roundTrip.ID.Hex())
	}
}

func TestCompanyReviewDocumentRoundTrip(t *testing.T) {
	original := sampleReview()
	document := toCompanyReviewDocument(original)

	encoded, err := bson.Marshal(document)
	if err != nil {
		t.Fatalf("Marshal returned error: %v", err)
	}

	var decoded companyReviewDocument
	if err := bson.Unmarshal(encoded, &decoded); err != nil {
		t.Fatalf("Unmarshal returned error: %v", err)
	}

	roundTrip := fromCompanyReviewDocument(&decoded)
	if roundTrip.Symbol != original.Symbol {
		t.Fatalf("expected symbol %s, got %s", original.Symbol, roundTrip.Symbol)
	}
	if len(roundTrip.Sections) != len(original.Sections) {
		t.Fatalf("expected %d sections, got %d", len(original.Sections), len(roundTrip.Sections))
	}
}

func sampleCompany() *companypkg.Company {
	now := time.Date(2026, 4, 23, 0, 0, 0, 0, time.UTC)
	return &companypkg.Company{
		ID:                    primitive.NewObjectID(),
		Symbol:                "INFY",
		Exchange:              "NSE",
		CompanyName:           "Infosys Limited",
		Sector:                "Information Technology",
		Industry:              "IT Services",
		SubIndustry:           "Digital Services",
		BusinessSummary:       "Large-cap India-listed technology services exporter.",
		ListingDate:           time.Date(1993, 6, 14, 0, 0, 0, 0, time.UTC),
		MarketCapBucket:       "large_cap",
		IsInInvestingUniverse: true,
		IsInTradingUniverse:   true,
		StatusActive:          true,
		CreatedAt:             now,
		UpdatedAt:             now,
		SchemaVersion:         common.SchemaVersion1,
	}
}

func sampleReview() *reviewpkg.CompanyReview {
	now := time.Date(2026, 4, 23, 0, 0, 0, 0, time.UTC)
	evidenceID := primitive.NewObjectID()
	companyID := primitive.NewObjectID()
	configSnapshotID := primitive.NewObjectID()

	return &reviewpkg.CompanyReview{
		ID:                   primitive.NewObjectID(),
		CompanyID:            companyID,
		Symbol:               "INFY",
		BookType:             common.BookTypeInvesting,
		ReviewDate:           now,
		ReviewPeriodType:     common.ReviewPeriodTypeMonthly,
		ConfigSnapshotID:     configSnapshotID,
		ReviewStatus:         common.ReviewStatusDraft,
		ReviewLifecycleState: common.ReviewLifecycleStateAIValidated,
		Mode:                 common.InvestingModeBalanced,
		ReviewerType:         common.ReviewerTypeHybrid,
		WeightedTotalScore:   8,
		ConfidenceScore:      0.8,
		DecisionAction: &reviewpkg.DecisionAction{
			ActionType:          common.InvestingActionTypeBuy,
			BucketAfterAction:   common.WatchlistBucketBuyReady,
			ActionReasonPrimary: "Strong business quality.",
		},
		FinalBucketAfterReview: common.WatchlistBucketBuyReady,
		FinalActionAfterReview: common.InvestingActionTypeBuy,
		Sections: []reviewpkg.SectionScore{
			{
				SectionName:               common.SectionNameInvestability,
				SectionWeight:             100,
				SectionScoreRaw:           8,
				SectionScoreWeighted:      8,
				SectionPassedMinimumCheck: true,
				SectionActionCap:          common.SectionActionCapNone,
				SectionConfidenceScore:    0.8,
				SubScores: []reviewpkg.SubScore{
					{
						SubScoreName:     common.SubScoreNameLiquidity,
						SubScoreWeight:   100,
						SubScoreValue:    8,
						TrendDirection:   common.TrendDirectionStable,
						EvidenceStrength: common.EvidenceStrengthMedium,
						MetricBasis:      common.MetricBasisHybrid,
						EvidenceRefIDs:   []primitive.ObjectID{evidenceID},
					},
				},
				EvidenceRefs: []reviewpkg.EvidenceReference{
					{
						ID:                evidenceID,
						SourceType:        common.EvidenceSourceTypeAnnualReport,
						SourceDate:        &now,
						SourceTitle:       "Annual Report FY26",
						EvidenceSummary:   "Sample evidence.",
						EvidenceDirection: common.EvidenceDirectionPositive,
					},
				},
			},
		},
		CreatedAt:     now,
		UpdatedAt:     now,
		SchemaVersion: common.SchemaVersion1,
	}
}
