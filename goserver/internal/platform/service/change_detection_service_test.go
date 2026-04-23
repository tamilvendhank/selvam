package service

import (
	"context"
	"testing"

	platformconfig "goserver/internal/platform/config"
	"goserver/internal/platform/domain"
	"goserver/internal/platform/testutil"
)

func TestChangeDetectionRequiresExitReviewOnLargeDrop(t *testing.T) {
	service := NewChangeDetectionService(platformconfig.Default())
	previous := testutil.SampleInvestingReview(8.3, true)
	current := testutil.SampleInvestingReview(6.9, true)
	current.Sections[6].SectionScoreRaw = 6.8

	changeLog, err := service.CompareReviews(context.Background(), current, previous, testutil.SampleThesis())
	if err != nil {
		t.Fatalf("CompareReviews returned error: %v", err)
	}
	if !changeLog.RequiresExitReview {
		t.Fatalf("expected exit review to be required")
	}
}

func TestChangeDetectionCapturesSectionDiffs(t *testing.T) {
	service := NewChangeDetectionService(platformconfig.Default())
	previous := testutil.SampleInvestingReview(8.0, true)
	current := testutil.SampleInvestingReview(8.0, true)
	current.Sections[0].SectionScoreRaw = 7.5

	changeLog, err := service.CompareReviews(context.Background(), current, previous, nil)
	if err != nil {
		t.Fatalf("CompareReviews returned error: %v", err)
	}
	if changeLog.SectionScoreChanges[string(domain.SectionInvestability)] != -0.5 {
		t.Fatalf("expected investability diff -0.5, got %v", changeLog.SectionScoreChanges[string(domain.SectionInvestability)])
	}
}
