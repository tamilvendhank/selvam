package continuation

import (
	domaincommon "goserver/internal/domain/common"
	domainworkflow "goserver/internal/domain/workflow"
)

func investingStepSequence() []domaincommon.WorkflowStepName {
	return []domaincommon.WorkflowStepName{
		domaincommon.WorkflowStepNameScanUniverse,
		domaincommon.WorkflowStepNameApplyHardFilters,
		domaincommon.WorkflowStepNameBuildReviewInputs,
		domaincommon.WorkflowStepNameCreatePendingReviewRecords,
		domaincommon.WorkflowStepNameCreateBatchJob,
		domaincommon.WorkflowStepNameSubmitBatchJob,
		domaincommon.WorkflowStepNameWaitForAsyncResults,
		domaincommon.WorkflowStepNamePollAndReconcileBatchResults,
		domaincommon.WorkflowStepNameValidateAIOutputs,
		domaincommon.WorkflowStepNameMaterializeFinalReviews,
		domaincommon.WorkflowStepNameEvaluateThesisAndChange,
		domaincommon.WorkflowStepNameMapActions,
		domaincommon.WorkflowStepNameAssignBuckets,
		domaincommon.WorkflowStepNameBuildCapitalCandidates,
		domaincommon.WorkflowStepNameAllocateCapital,
		domaincommon.WorkflowStepNamePersistOutputs,
		domaincommon.WorkflowStepNamePublishRunSummary,
	}
}

func tradingStepSequence() []domaincommon.WorkflowStepName {
	return []domaincommon.WorkflowStepName{
		domaincommon.WorkflowStepNameRefreshUniverse,
		domaincommon.WorkflowStepNameEvaluateRegime,
		domaincommon.WorkflowStepNameBuildTradingReviewInputs,
		domaincommon.WorkflowStepNameCreateTradingBatchJob,
		domaincommon.WorkflowStepNameSubmitTradingBatchJob,
		domaincommon.WorkflowStepNameWaitForTradingAsyncResults,
		domaincommon.WorkflowStepNamePollAndReconcileTradingResults,
		domaincommon.WorkflowStepNameValidateTradingAIOutputs,
		domaincommon.WorkflowStepNameApproveTradeCandidates,
		domaincommon.WorkflowStepNamePersistTradingReview,
		domaincommon.WorkflowStepNamePublishTradingRunSummary,
	}
}

func stepSequenceForBook(bookType domaincommon.BookType) []domaincommon.WorkflowStepName {
	switch bookType {
	case domaincommon.BookTypeTrading:
		return tradingStepSequence()
	case domaincommon.BookTypeInvesting:
		return investingStepSequence()
	default:
		return nil
	}
}

func stepStatusByName(steps []*domainworkflow.WorkflowStepRun) map[domaincommon.WorkflowStepName]*domainworkflow.WorkflowStepRun {
	byName := make(map[domaincommon.WorkflowStepName]*domainworkflow.WorkflowStepRun, len(steps))
	for _, step := range steps {
		if step == nil {
			continue
		}
		existing := byName[step.StepName]
		if existing == nil || step.UpdatedAt.After(existing.UpdatedAt) {
			byName[step.StepName] = step
		}
	}
	return byName
}

func latestCompletedStep(
	sequence []domaincommon.WorkflowStepName,
	steps map[domaincommon.WorkflowStepName]*domainworkflow.WorkflowStepRun,
) domaincommon.WorkflowStepName {
	var latest domaincommon.WorkflowStepName
	for _, stepName := range sequence {
		if isStepComplete(steps[stepName]) {
			latest = stepName
		}
	}
	return latest
}

func firstIncompleteStep(
	sequence []domaincommon.WorkflowStepName,
	steps map[domaincommon.WorkflowStepName]*domainworkflow.WorkflowStepRun,
) domaincommon.WorkflowStepName {
	for _, stepName := range sequence {
		if !isStepComplete(steps[stepName]) {
			return stepName
		}
	}
	return ""
}

func nextStepAfter(
	sequence []domaincommon.WorkflowStepName,
	stepName domaincommon.WorkflowStepName,
) domaincommon.WorkflowStepName {
	for index, candidate := range sequence {
		if candidate == stepName && index+1 < len(sequence) {
			return sequence[index+1]
		}
	}
	return ""
}

func isStepComplete(step *domainworkflow.WorkflowStepRun) bool {
	if step == nil {
		return false
	}
	return step.Status == domaincommon.WorkflowStepStatusCompleted ||
		step.Status == domaincommon.WorkflowStepStatusSkipped
}

func isStepFailed(step *domainworkflow.WorkflowStepRun) bool {
	return step != nil && step.Status == domaincommon.WorkflowStepStatusFailed
}

func isStepAtOrAfter(
	sequence []domaincommon.WorkflowStepName,
	candidate domaincommon.WorkflowStepName,
	pivot domaincommon.WorkflowStepName,
) bool {
	candidateIndex := stepIndex(sequence, candidate)
	pivotIndex := stepIndex(sequence, pivot)
	return candidateIndex >= 0 && pivotIndex >= 0 && candidateIndex >= pivotIndex
}

func stepIndex(sequence []domaincommon.WorkflowStepName, stepName domaincommon.WorkflowStepName) int {
	for index, candidate := range sequence {
		if candidate == stepName {
			return index
		}
	}
	return -1
}

func submitBatchStep(bookType domaincommon.BookType) domaincommon.WorkflowStepName {
	if bookType == domaincommon.BookTypeTrading {
		return domaincommon.WorkflowStepNameSubmitTradingBatchJob
	}
	return domaincommon.WorkflowStepNameSubmitBatchJob
}

func createBatchStep(bookType domaincommon.BookType) domaincommon.WorkflowStepName {
	if bookType == domaincommon.BookTypeTrading {
		return domaincommon.WorkflowStepNameCreateTradingBatchJob
	}
	return domaincommon.WorkflowStepNameCreateBatchJob
}

func pollAndReconcileStep(bookType domaincommon.BookType) domaincommon.WorkflowStepName {
	if bookType == domaincommon.BookTypeTrading {
		return domaincommon.WorkflowStepNamePollAndReconcileTradingResults
	}
	return domaincommon.WorkflowStepNamePollAndReconcileBatchResults
}

func validateAIOutputsStep(bookType domaincommon.BookType) domaincommon.WorkflowStepName {
	if bookType == domaincommon.BookTypeTrading {
		return domaincommon.WorkflowStepNameValidateTradingAIOutputs
	}
	return domaincommon.WorkflowStepNameValidateAIOutputs
}

func materializeReviewsStep(bookType domaincommon.BookType) domaincommon.WorkflowStepName {
	if bookType == domaincommon.BookTypeTrading {
		return domaincommon.WorkflowStepNamePersistTradingReview
	}
	return domaincommon.WorkflowStepNameMaterializeFinalReviews
}

func firstPostAIStep(bookType domaincommon.BookType) domaincommon.WorkflowStepName {
	if bookType == domaincommon.BookTypeTrading {
		return domaincommon.WorkflowStepNameApproveTradeCandidates
	}
	return domaincommon.WorkflowStepNameEvaluateThesisAndChange
}

func determineNextSuggestedStep(
	bookType domaincommon.BookType,
	sequence []domaincommon.WorkflowStepName,
	steps map[domaincommon.WorkflowStepName]*domainworkflow.WorkflowStepRun,
	counts WorkflowContinuationCounts,
) domaincommon.WorkflowStepName {
	firstIncomplete := firstIncompleteStep(sequence, steps)
	if counts.BatchJobs.Created > 0 {
		return submitBatchStep(bookType)
	}
	if counts.BatchJobs.Submitted+counts.BatchJobs.Running+counts.BatchJobs.PartiallyCompleted > 0 {
		return pollAndReconcileStep(bookType)
	}
	if counts.BatchItems.Unreconciled > 0 && counts.BatchJobs.Completed+counts.BatchJobs.Failed+counts.BatchJobs.TimedOut > 0 {
		return pollAndReconcileStep(bookType)
	}
	if counts.BatchItems.PendingValidation > 0 {
		return validateAIOutputsStep(bookType)
	}
	if counts.BatchItems.Materializable > 0 || counts.Reviews.MaterializationIncomplete > 0 {
		return materializeReviewsStep(bookType)
	}
	if counts.Reviews.FinalizationIncomplete > 0 {
		return materializeReviewsStep(bookType)
	}
	if firstIncomplete == "" {
		return ""
	}
	return firstIncomplete
}
