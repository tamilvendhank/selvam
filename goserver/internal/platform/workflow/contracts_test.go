package workflow

import "testing"

func TestInvestingStepNamesOrder(t *testing.T) {
	got := InvestingStepNames()
	want := []string{
		"ScanUniverse",
		"ApplyHardFilters",
		"BuildReviewInputs",
		"CreatePendingReviewRecords",
		"CreateBatchJob",
		"SubmitBatchJob",
		"WaitForAsyncResults",
		"PollAndReconcileBatchResults",
		"ValidateAIOutputs",
		"MaterializeFinalReviews",
		"EvaluateThesisAndChange",
		"MapActions",
		"AssignBuckets",
		"BuildCapitalCandidates",
		"AllocateCapital",
		"PersistOutputs",
		"PublishRunSummary",
	}

	if len(got) != len(want) {
		t.Fatalf("expected %d investing steps, got %d", len(want), len(got))
	}
	for index := range want {
		if got[index] != want[index] {
			t.Fatalf("expected step %d to be %q, got %q", index, want[index], got[index])
		}
	}
}
