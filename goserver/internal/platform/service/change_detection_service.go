package service

import (
	"context"
	"fmt"

	platformconfig "goserver/internal/platform/config"
	"goserver/internal/platform/domain"
)

type DefaultChangeDetectionService struct {
	config platformconfig.AppConfig
}

func NewChangeDetectionService(config platformconfig.AppConfig) *DefaultChangeDetectionService {
	return &DefaultChangeDetectionService{config: config}
}

func (service *DefaultChangeDetectionService) CompareReviews(_ context.Context, current, previous *domain.CompanyReview, thesis *domain.InvestmentThesis) (*domain.ReviewChangeLog, error) {
	if current == nil {
		return nil, fmt.Errorf("current review is required")
	}
	if previous == nil {
		return &domain.ReviewChangeLog{
			ChangeSummary: "Initial review snapshot.",
		}, nil
	}

	changeLog := &domain.ReviewChangeLog{
		PreviousReviewID:         previous.ID,
		WeightedTotalScoreChange: domain.NormalizeScore(current.WeightedTotalScore - previous.WeightedTotalScore),
		SectionScoreChanges:      map[string]float64{},
		SubScoreChanges:          map[string]float64{},
	}

	if previous.FinalBucketAfterReview != current.FinalBucketAfterReview {
		changeLog.BucketChange = fmt.Sprintf("%s -> %s", previous.FinalBucketAfterReview, current.FinalBucketAfterReview)
	}
	if previous.FinalActionAfterReview != current.FinalActionAfterReview {
		changeLog.ActionChange = fmt.Sprintf("%s -> %s", previous.FinalActionAfterReview, current.FinalActionAfterReview)
	}
	if thesis != nil {
		changeLog.ThesisStatusChange = string(thesis.ThesisStatus)
	}

	requiresExitReview := false
	for _, section := range current.Sections {
		previousSection := domain.FindSection(previous, domain.InvestingSectionName(section.SectionName))
		if previousSection == nil {
			continue
		}

		diff := domain.NormalizeScore(section.SectionScoreRaw - previousSection.SectionScoreRaw)
		if diff != 0 {
			changeLog.SectionScoreChanges[section.SectionName] = diff
		}

		if section.SectionName == string(domain.SectionManagementGovernance) && diff <= -service.config.Investing.ActionThresholds.ExitReviewManagementDrop {
			requiresExitReview = true
		}
		if isConfiguredCoreSection(service.config, section.SectionName) && diff <= -service.config.Investing.ActionThresholds.ExitReviewCoreDrop {
			requiresExitReview = true
		}

		for _, subScore := range section.SubScores {
			for _, previousSubScore := range previousSection.SubScores {
				if previousSubScore.SubScoreName != subScore.SubScoreName {
					continue
				}
				subDiff := domain.NormalizeScore(subScore.SubScoreValue - previousSubScore.SubScoreValue)
				if subDiff != 0 {
					key := fmt.Sprintf("%s::%s", section.SectionName, subScore.SubScoreName)
					changeLog.SubScoreChanges[key] = subDiff
				}
			}
		}
	}

	if changeLog.WeightedTotalScoreChange <= -service.config.Investing.ActionThresholds.ExitReviewTotalDrop {
		requiresExitReview = true
	}
	changeLog.RequiresExitReview = requiresExitReview
	if requiresExitReview {
		changeLog.ChangeSummary = "Meaningful deterioration detected. Exit review recommended."
	} else {
		changeLog.ChangeSummary = "Change set captured without automatic exit review escalation."
	}

	return changeLog, nil
}

func isConfiguredCoreSection(config platformconfig.AppConfig, sectionName string) bool {
	for _, section := range config.Investing.CoreSections {
		if section == sectionName {
			return true
		}
	}

	return false
}
