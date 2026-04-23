package domain_test

import (
	"testing"

	"goserver/internal/platform/testutil"
)

func TestCompanyReviewValidate(t *testing.T) {
	review := testutil.SampleInvestingReview(8.1, true)
	if err := review.Validate(); err != nil {
		t.Fatalf("expected valid review, got error: %v", err)
	}
}

func TestCompanyReviewValidateRejectsInvalidSubScore(t *testing.T) {
	review := testutil.SampleInvestingReview(8.1, true)
	review.Sections[0].SubScores[0].SubScoreValue = 11

	if err := review.Validate(); err == nil {
		t.Fatalf("expected validation error for invalid sub-score range")
	}
}
