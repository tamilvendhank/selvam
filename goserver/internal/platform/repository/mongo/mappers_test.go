package mongo

import (
	"testing"

	"goserver/internal/platform/testutil"

	"go.mongodb.org/mongo-driver/bson"
)

func TestCompanyDocumentRoundTrip(t *testing.T) {
	original := testutil.SampleCompany()
	document := toCompanyDocument(original)
	document.ObjectID = newDocumentID()

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
}

func TestCompanyReviewDocumentRoundTrip(t *testing.T) {
	original := testutil.SampleInvestingReview(8.2, true)
	document := toCompanyReviewDocument(original)
	document.ObjectID = newDocumentID()

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
