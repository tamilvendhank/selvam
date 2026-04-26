package app

import (
	"goserver/internal/domain/allocation"
	domaincommon "goserver/internal/domain/common"
	"goserver/internal/domain/company"
	"goserver/internal/domain/position"
	"goserver/internal/domain/review"
	"goserver/internal/domain/thesis"
	platformrepo "goserver/internal/platform/repository"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

func pageDTO(page platformrepo.PageInfo) PageDTO {
	return PageDTO{Limit: page.PageSize, Offset: page.Offset, HasMore: page.HasMore}
}

func objectIDString(id primitive.ObjectID) string {
	if id.IsZero() {
		return ""
	}
	return id.Hex()
}

func objectIDStrings(ids []primitive.ObjectID) []string {
	if len(ids) == 0 {
		return nil
	}
	values := make([]string, 0, len(ids))
	for _, id := range ids {
		if !id.IsZero() {
			values = append(values, id.Hex())
		}
	}
	return values
}

func mapCompanyListItem(company *company.Company) CompanyListItemDTO {
	if company == nil {
		return CompanyListItemDTO{}
	}
	return CompanyListItemDTO{
		CompanyID:             objectIDString(company.ID),
		Symbol:                company.Symbol,
		Exchange:              company.Exchange,
		CompanyName:           company.CompanyName,
		Sector:                company.Sector,
		Industry:              company.Industry,
		SubIndustry:           company.SubIndustry,
		MarketCapBucket:       company.MarketCapBucket,
		IsInInvestingUniverse: company.IsInInvestingUniverse,
		IsInTradingUniverse:   company.IsInTradingUniverse,
		StatusActive:          company.StatusActive,
		CreatedAt:             company.CreatedAt,
		UpdatedAt:             company.UpdatedAt,
	}
}

func mapCompanyListItems(companies []*company.Company) []CompanyListItemDTO {
	items := make([]CompanyListItemDTO, 0, len(companies))
	for _, item := range companies {
		if item != nil {
			items = append(items, mapCompanyListItem(item))
		}
	}
	return items
}

func mapReviewSummary(review *review.CompanyReview) ReviewSummaryDTO {
	if review == nil {
		return ReviewSummaryDTO{}
	}
	return ReviewSummaryDTO{
		ReviewID:               objectIDString(review.ID),
		CompanyID:              objectIDString(review.CompanyID),
		Symbol:                 review.Symbol,
		BookType:               review.BookType,
		WorkflowRunID:          objectIDString(review.WorkflowRunID),
		ReviewDate:             review.ReviewDate,
		ReviewStatus:           review.ReviewStatus,
		ReviewLifecycleState:   review.ReviewLifecycleState,
		WeightedTotalScore:     review.WeightedTotalScore,
		ConfidenceScore:        review.ConfidenceScore,
		HardGateFailed:         review.HardGateFailed,
		FinalActionAfterReview: review.FinalActionAfterReview,
		FinalBucketAfterReview: review.FinalBucketAfterReview,
		ActionRationaleSummary: review.ActionRationaleSummary,
		WhatChangedSummary:     review.WhatChangedSummary,
		ConfigSnapshotID:       objectIDString(review.ConfigSnapshotID),
		CreatedAt:              review.CreatedAt,
		UpdatedAt:              review.UpdatedAt,
		FinalizedAt:            review.FinalizedAt,
	}
}

func mapReviewSummaryFromRepository(summary *platformrepo.CompanyReviewSummary) ReviewSummaryDTO {
	if summary == nil {
		return ReviewSummaryDTO{}
	}
	return ReviewSummaryDTO{
		ReviewID:               objectIDString(summary.ID),
		CompanyID:              objectIDString(summary.CompanyID),
		Symbol:                 summary.Symbol,
		BookType:               summary.BookType,
		WorkflowRunID:          objectIDString(summary.WorkflowRunID),
		ReviewDate:             summary.ReviewDate,
		ReviewStatus:           summary.ReviewStatus,
		ReviewLifecycleState:   summary.ReviewLifecycleState,
		WeightedTotalScore:     summary.WeightedTotalScore,
		FinalActionAfterReview: summary.FinalActionAfterReview,
		FinalBucketAfterReview: summary.FinalBucketAfterReview,
		UpdatedAt:              summary.UpdatedAt,
		FinalizedAt:            summary.FinalizedAt,
	}
}

func mapReviewListItem(review *review.CompanyReview) ReviewListItemDTO {
	if review == nil {
		return ReviewListItemDTO{}
	}
	return ReviewListItemDTO{
		ReviewSummaryDTO:  mapReviewSummary(review),
		Mode:              review.Mode,
		OwnedBeforeReview: review.OwnedBeforeReview,
		ReviewPeriodType:  review.ReviewPeriodType,
	}
}

func mapReviewListItems(reviews []*review.CompanyReview) []ReviewListItemDTO {
	items := make([]ReviewListItemDTO, 0, len(reviews))
	for _, review := range reviews {
		if review != nil {
			items = append(items, mapReviewListItem(review))
		}
	}
	return items
}

func mapReviewListItemsFromSummaries(summaries []*platformrepo.CompanyReviewSummary) []ReviewListItemDTO {
	items := make([]ReviewListItemDTO, 0, len(summaries))
	for _, summary := range summaries {
		if summary != nil {
			items = append(items, ReviewListItemDTO{ReviewSummaryDTO: mapReviewSummaryFromRepository(summary)})
		}
	}
	return items
}

func mapReviewDetail(review *review.CompanyReview) ReviewDetailDTO {
	if review == nil {
		return ReviewDetailDTO{}
	}
	return ReviewDetailDTO{
		ReviewListItemDTO:         mapReviewListItem(review),
		CurrentBucketBeforeReview: review.CurrentBucketBeforeReview,
		CurrentActionBeforeReview: review.CurrentActionBeforeReview,
		HardGateFailureReasons:    append([]string(nil), review.HardGateFailureReasons...),
		ReviewerType:              review.ReviewerType,
		AIModelName:               review.AIModelName,
		AIPromptVersion:           review.AIPromptVersion,
		RawAIResultRef:            review.RawAIResultRef,
		Scorecard:                 mapScorecard(review),
		DecisionAction:            mapDecisionAction(review.DecisionAction),
		PositionSnapshot:          mapPositionSnapshot(review.PositionSnapshot),
		ChangeLog:                 mapReviewChangeLog(review.ChangeLog),
		ReviewMetadata:            review.ReviewMetadata,
	}
}

func mapScorecard(review *review.CompanyReview) ScorecardDTO {
	if review == nil {
		return ScorecardDTO{}
	}
	return ScorecardDTO{
		ReviewID:               objectIDString(review.ID),
		WeightedTotalScore:     review.WeightedTotalScore,
		ConfidenceScore:        review.ConfidenceScore,
		HardGateFailed:         review.HardGateFailed,
		HardGateFailureReasons: append([]string(nil), review.HardGateFailureReasons...),
		Sections:               mapSections(review.Sections),
	}
}

func mapSections(sections []review.SectionScore) []SectionScoreDTO {
	result := make([]SectionScoreDTO, 0, len(sections))
	for _, section := range sections {
		result = append(result, SectionScoreDTO{
			SectionName:               section.SectionName,
			SectionWeight:             section.SectionWeight,
			SectionScoreRaw:           section.SectionScoreRaw,
			SectionScoreWeighted:      section.SectionScoreWeighted,
			SectionPassedMinimumCheck: section.SectionPassedMinimumCheck,
			SectionActionCap:          section.SectionActionCap,
			Summary:                   section.SectionSummary,
			Strengths:                 append([]string(nil), section.SectionStrengths...),
			Weaknesses:                append([]string(nil), section.SectionWeaknesses...),
			Risks:                     append([]string(nil), section.SectionRisks...),
			ConfidenceScore:           section.SectionConfidenceScore,
			SubScores:                 mapSubScores(section.SubScores),
			EvidenceRefs:              mapEvidenceRefs(section.SectionName, "", section.EvidenceRefs),
		})
	}
	return result
}

func mapSubScores(subScores []review.SubScore) []SubScoreDTO {
	result := make([]SubScoreDTO, 0, len(subScores))
	for _, subScore := range subScores {
		result = append(result, SubScoreDTO{
			Name:             subScore.SubScoreName,
			Weight:           subScore.SubScoreWeight,
			Value:            subScore.SubScoreValue,
			Summary:          subScore.SubScoreSummary,
			TrendDirection:   subScore.TrendDirection,
			EvidenceStrength: subScore.EvidenceStrength,
			MetricBasis:      subScore.MetricBasis,
			Notes:            subScore.Notes,
			EvidenceRefIDs:   objectIDStrings(subScore.EvidenceRefIDs),
		})
	}
	return result
}

func mapEvidenceRefs(sectionName domaincommon.SectionName, subScoreName domaincommon.SubScoreName, refs []review.EvidenceReference) []EvidenceReferenceDTO {
	result := make([]EvidenceReferenceDTO, 0, len(refs))
	for _, ref := range refs {
		result = append(result, EvidenceReferenceDTO{
			EvidenceID:           objectIDString(ref.ID),
			SectionName:          sectionName,
			SubScoreName:         subScoreName,
			SourceType:           ref.SourceType,
			SourceDate:           ref.SourceDate,
			SourceTitle:          ref.SourceTitle,
			SourcePeriod:         ref.SourcePeriod,
			SourceURLOrPath:      ref.SourceURLOrPath,
			ExcerptOrMetricName:  ref.ExcerptOrMetricName,
			ExcerptOrMetricValue: ref.ExcerptOrMetricValue,
			EvidenceSummary:      ref.EvidenceSummary,
			EvidenceDirection:    ref.EvidenceDirection,
		})
	}
	return result
}

func mapDecisionAction(decision *review.DecisionAction) *DecisionActionDTO {
	if decision == nil {
		return nil
	}
	return &DecisionActionDTO{
		ActionType:                   decision.ActionType,
		BucketAfterAction:            decision.BucketAfterAction,
		ActionPriorityRank:           decision.ActionPriorityRank,
		ActionReasonPrimary:          decision.ActionReasonPrimary,
		ActionReasonSecondary:        decision.ActionReasonSecondary,
		ActionConstraints:            append([]string(nil), decision.ActionConstraints...),
		CapitalEligible:              decision.CapitalEligible,
		CapitalPriorityScore:         decision.CapitalPriorityScore,
		RecommendedPositionTargetPct: decision.RecommendedPositionTargetPct,
		RecommendedPositionCapPct:    decision.RecommendedPositionCapPct,
		RecommendedTrancheStyle:      decision.RecommendedTrancheStyle,
		Notes:                        decision.Notes,
	}
}

func mapPositionSnapshot(snapshot *review.PositionSnapshot) *PositionSnapshotDTO {
	if snapshot == nil {
		return nil
	}
	return &PositionSnapshotDTO{
		IsOwned:                     snapshot.IsOwned,
		Quantity:                    snapshot.Quantity,
		AverageCost:                 snapshot.AverageCost,
		MarketPriceAtReview:         snapshot.MarketPriceAtReview,
		MarketValue:                 snapshot.MarketValue,
		PositionPctOfBook:           snapshot.PositionPctOfBook,
		PositionPctOfTotalPortfolio: snapshot.PositionPctOfTotalPortfolio,
		UnrealizedPnLAbs:            snapshot.UnrealizedPnLAbs,
		UnrealizedPnLPct:            snapshot.UnrealizedPnLPct,
		TargetPositionPct:           snapshot.TargetPositionPct,
		MaxPositionPct:              snapshot.MaxPositionPct,
		UnderweightVsTargetPct:      snapshot.UnderweightVsTargetPct,
		OverweightVsTargetPct:       snapshot.OverweightVsTargetPct,
		OwnedSinceDate:              snapshot.OwnedSinceDate,
	}
}

func mapReviewChangeLog(log *review.ReviewChangeLog) *ReviewDiffDTO {
	if log == nil {
		return nil
	}
	return &ReviewDiffDTO{
		PreviousReviewID:         objectIDString(log.PreviousReviewID),
		WeightedTotalScoreChange: log.WeightedTotalScoreChange,
		SectionScoreChanges:      log.SectionScoreChanges,
		SubScoreChanges:          log.SubScoreChanges,
		BucketChange:             log.BucketChange,
		ActionChange:             log.ActionChange,
		ThesisStatusChange:       log.ThesisStatusChange,
		MajorPositiveChanges:     append([]string(nil), log.MajorPositiveChanges...),
		MajorNegativeChanges:     append([]string(nil), log.MajorNegativeChanges...),
		ValuationStateChange:     log.ValuationStateChange,
		OwnershipRelevanceChange: log.OwnershipRelevanceChange,
		RequiresExitReview:       log.RequiresExitReview,
		ChangeSummary:            log.ChangeSummary,
	}
}

func mapThesisListItem(thesis *thesis.InvestmentThesis) ThesisListItemDTO {
	if thesis == nil {
		return ThesisListItemDTO{}
	}
	return ThesisListItemDTO{
		ThesisID:                objectIDString(thesis.ID),
		CompanyID:               objectIDString(thesis.CompanyID),
		ThesisStatus:            thesis.ThesisStatus,
		ThesisVersion:           thesis.ThesisVersion,
		ThesisSummary:           thesis.ThesisSummary,
		ConfidenceLevel:         thesis.ConfidenceLevel,
		ThesisHealthScore:       thesis.ThesisHealthScore,
		CurrentPositionRole:     thesis.CurrentPositionRole,
		CreatedFromReviewID:     objectIDString(thesis.CreatedFromReviewID),
		LastUpdatedFromReviewID: objectIDString(thesis.LastUpdatedFromReviewID),
		CreatedAt:               thesis.CreatedAt,
		UpdatedAt:               thesis.UpdatedAt,
	}
}

func mapThesisListItems(theses []*thesis.InvestmentThesis) []ThesisListItemDTO {
	items := make([]ThesisListItemDTO, 0, len(theses))
	for _, thesis := range theses {
		if thesis != nil {
			items = append(items, mapThesisListItem(thesis))
		}
	}
	return items
}

func mapThesisDetail(thesis *thesis.InvestmentThesis) ThesisDetailDTO {
	if thesis == nil {
		return ThesisDetailDTO{}
	}
	linked := []primitive.ObjectID{thesis.CreatedFromReviewID}
	if !thesis.LastUpdatedFromReviewID.IsZero() && thesis.LastUpdatedFromReviewID != thesis.CreatedFromReviewID {
		linked = append(linked, thesis.LastUpdatedFromReviewID)
	}
	return ThesisDetailDTO{
		ThesisListItemDTO:          mapThesisListItem(thesis),
		WhyThisBusinessCanCompound: thesis.WhyThisBusinessCanCompound,
		KeyGrowthDrivers:           append([]string(nil), thesis.KeyGrowthDrivers...),
		KeyMoatOrAdvantageFactors:  append([]string(nil), thesis.KeyMoatOrAdvantageFactors...),
		WhyNow:                     thesis.WhyNow,
		KeyRisks:                   append([]string(nil), thesis.KeyRisks...),
		DisconfirmingSignals:       append([]string(nil), thesis.DisconfirmingSignals...),
		WhatWouldBreakTheThesis:    append([]string(nil), thesis.WhatWouldBreakTheThesis...),
		DesiredHoldingPeriod:       thesis.DesiredHoldingPeriod,
		ThesisChangeSummary:        thesis.ThesisChangeSummary,
		NewSupportingEvidence:      append([]string(nil), thesis.NewSupportingEvidence...),
		NewContradictingEvidence:   append([]string(nil), thesis.NewContradictingEvidence...),
		LinkedReviewIDs:            objectIDStrings(linked),
	}
}

func mapThesisHistoryItem(thesis *thesis.InvestmentThesis) ThesisHistoryItemDTO {
	if thesis == nil {
		return ThesisHistoryItemDTO{}
	}
	return ThesisHistoryItemDTO{
		ThesisID:                objectIDString(thesis.ID),
		ThesisStatus:            thesis.ThesisStatus,
		ThesisVersion:           thesis.ThesisVersion,
		ThesisSummary:           thesis.ThesisSummary,
		ThesisHealthScore:       thesis.ThesisHealthScore,
		ThesisChangeSummary:     thesis.ThesisChangeSummary,
		LastUpdatedFromReviewID: objectIDString(thesis.LastUpdatedFromReviewID),
		UpdatedAt:               thesis.UpdatedAt,
	}
}

func mapAllocationRunListItem(run *allocation.CapitalAllocationRun) AllocationRunListItemDTO {
	if run == nil {
		return AllocationRunListItemDTO{}
	}
	return AllocationRunListItemDTO{
		AllocationRunID:       objectIDString(run.ID),
		WorkflowRunID:         objectIDString(run.WorkflowRunID),
		AllocationDate:        run.AllocationDate,
		BookType:              run.BookType,
		AvailableCashStart:    run.AvailableCashStart,
		FreshMonthlyCash:      run.FreshMonthlyCash,
		SellProceedsAvailable: run.SellProceedsAvailable,
		CarryForwardCash:      run.CarryForwardCash,
		TargetDeployableCash:  run.TargetDeployableCash,
		AllocatedCashTotal:    run.AllocatedCashTotal,
		CashLeftUnallocated:   run.CashLeftUnallocated,
		ItemCount:             len(run.Items),
		CreatedAt:             run.CreatedAt,
	}
}

func mapAllocationRunListItems(runs []*allocation.CapitalAllocationRun) []AllocationRunListItemDTO {
	items := make([]AllocationRunListItemDTO, 0, len(runs))
	for _, run := range runs {
		if run != nil {
			items = append(items, mapAllocationRunListItem(run))
		}
	}
	return items
}

func mapAllocationRunDetail(run *allocation.CapitalAllocationRun) AllocationRunDetailDTO {
	if run == nil {
		return AllocationRunDetailDTO{}
	}
	items := mapAllocationItems(run.Items)
	blocked := make([]AllocationItemDTO, 0)
	for _, item := range items {
		if item.BlockedByConstraint {
			blocked = append(blocked, item)
		}
	}
	return AllocationRunDetailDTO{
		AllocationRunListItemDTO: mapAllocationRunListItem(run),
		AllocationNotes:          run.AllocationNotes,
		Items:                    items,
		BlockedCandidates:        blocked,
	}
}

func mapAllocationItems(items []allocation.CapitalAllocationItem) []AllocationItemDTO {
	result := make([]AllocationItemDTO, 0, len(items))
	for _, item := range items {
		result = append(result, AllocationItemDTO{
			CompanyID:                     objectIDString(item.CompanyID),
			DecisionReviewID:              objectIDString(item.DecisionReviewID),
			ActionType:                    item.ActionType,
			BuyPriorityRank:               item.BuyPriorityRank,
			CapitalPriorityScore:          item.CapitalPriorityScore,
			RecommendedAllocationAmount:   item.RecommendedAllocationAmount,
			RecommendedAllocationPctOfRun: item.RecommendedAllocationPctOfRun,
			RecommendedTrancheNumber:      item.RecommendedTrancheNumber,
			AllocationReason:              item.AllocationReason,
			BlockedByConstraint:           item.BlockedByConstraint,
			ConstraintReason:              item.ConstraintReason,
		})
	}
	return result
}

func mapCurrentPosition(position *position.CurrentPosition, symbol string) CurrentPositionDTO {
	if position == nil {
		return CurrentPositionDTO{}
	}
	return CurrentPositionDTO{
		PositionID:                    objectIDString(position.ID),
		CompanyID:                     objectIDString(position.CompanyID),
		Symbol:                        symbol,
		BookType:                      position.BookType,
		IsOpen:                        position.IsOpen,
		Quantity:                      position.Quantity,
		AverageCost:                   position.AverageCost,
		CurrentMarketValue:            position.CurrentMarketValue,
		CurrentPositionPctOfBook:      position.CurrentPositionPctOfBook,
		CurrentPositionPctOfPortfolio: position.CurrentPositionPctOfPortfolio,
		LastUpdatedAt:                 position.LastUpdatedAt,
	}
}

func mapCurrentPositions(positions []*position.CurrentPosition) []CurrentPositionDTO {
	items := make([]CurrentPositionDTO, 0, len(positions))
	for _, position := range positions {
		if position != nil {
			items = append(items, mapCurrentPosition(position, ""))
		}
	}
	return items
}
