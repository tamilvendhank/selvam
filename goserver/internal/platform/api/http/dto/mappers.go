package dto

import "goserver/internal/platform/domain"

func MapCompanySummaries(companies []*domain.Company) []CompanySummary {
	result := make([]CompanySummary, 0, len(companies))
	for _, company := range companies {
		if company == nil {
			continue
		}
		result = append(result, CompanySummary{
			ID:                    company.ID,
			Symbol:                company.Symbol,
			Exchange:              company.Exchange,
			CompanyName:           company.CompanyName,
			Sector:                company.Sector,
			Industry:              company.Industry,
			MarketCapBucket:       company.MarketCapBucket,
			IsInInvestingUniverse: company.IsInInvestingUniverse,
			IsInTradingUniverse:   company.IsInTradingUniverse,
			StatusActive:          company.StatusActive,
		})
	}

	return result
}

func MapReviewSummaries(reviews []*domain.CompanyReview) []ReviewSummary {
	result := make([]ReviewSummary, 0, len(reviews))
	for _, review := range reviews {
		if review == nil {
			continue
		}
		result = append(result, ReviewSummary{
			ID:                     review.ID,
			CompanyID:              review.CompanyID,
			Symbol:                 review.Symbol,
			BookType:               review.BookType,
			ReviewDate:             review.ReviewDate,
			ReviewStatus:           review.ReviewStatus,
			WeightedTotalScore:     review.WeightedTotalScore,
			ConfidenceScore:        review.ConfidenceScore,
			FinalBucketAfterReview: review.FinalBucketAfterReview,
			FinalActionAfterReview: review.FinalActionAfterReview,
		})
	}

	return result
}

func MapWorkflowRunSummaries(runs []*domain.WorkflowRun) []WorkflowRunSummary {
	result := make([]WorkflowRunSummary, 0, len(runs))
	for _, run := range runs {
		if run == nil {
			continue
		}
		result = append(result, WorkflowRunSummary{
			ID:                    run.ID,
			BookType:              run.BookType,
			RunType:               run.RunType,
			Status:                run.Status,
			StartedAt:             run.StartedAt,
			CompletedAt:           run.CompletedAt,
			CompaniesScannedCount: run.CompaniesScannedCount,
			ReviewsCreatedCount:   run.ReviewsCreatedCount,
			ErrorsCount:           run.ErrorsCount,
			DryRun:                run.DryRun,
		})
	}

	return result
}

func MapCapitalAllocationSummaries(runs []*domain.CapitalAllocationRun) []CapitalAllocationSummary {
	result := make([]CapitalAllocationSummary, 0, len(runs))
	for _, run := range runs {
		if run == nil {
			continue
		}
		result = append(result, CapitalAllocationSummary{
			ID:                  run.ID,
			WorkflowRunID:       run.WorkflowRunID,
			AllocationDate:      run.AllocationDate,
			BookType:            run.BookType,
			AllocatedCashTotal:  run.AllocatedCashTotal,
			CashLeftUnallocated: run.CashLeftUnallocated,
		})
	}

	return result
}

func MapManualOverrideSummaries(overrides []*domain.ManualOverride) []ManualOverrideSummary {
	result := make([]ManualOverrideSummary, 0, len(overrides))
	for _, override := range overrides {
		if override == nil {
			continue
		}
		result = append(result, ManualOverrideSummary{
			ID:               override.ID,
			CompanyID:        override.CompanyID,
			ReviewID:         override.ReviewID,
			BookType:         override.BookType,
			OriginalAction:   override.OriginalAction,
			OverriddenAction: override.OverriddenAction,
			OverrideDate:     override.OverrideDate,
		})
	}

	return result
}

func MapPositionSummaries(positions []*domain.CurrentPosition) []PositionSummary {
	result := make([]PositionSummary, 0, len(positions))
	for _, position := range positions {
		if position == nil {
			continue
		}
		result = append(result, PositionSummary{
			ID:                          position.ID,
			CompanyID:                   position.CompanyID,
			Symbol:                      position.Symbol,
			BookType:                    position.BookType,
			Quantity:                    position.Quantity,
			MarketValue:                 position.MarketValue,
			PositionPctOfBook:           position.PositionPctOfBook,
			PositionPctOfTotalPortfolio: position.PositionPctOfTotalPortfolio,
			UpdatedAt:                   position.UpdatedAt,
		})
	}

	return result
}
