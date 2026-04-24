package mongo

import (
	"strings"

	platformconfig "goserver/internal/platform/config"
)

const (
	CompaniesCollectionName             = "companies"
	CompanyReviewsCollectionName        = "company_reviews"
	InvestmentThesesCollectionName      = "investment_theses"
	WorkflowRunsCollectionName          = "workflow_runs"
	WorkflowStepRunsCollectionName      = "workflow_step_runs"
	ConfigSnapshotsCollectionName       = "config_snapshots"
	CapitalAllocationRunsCollectionName = "capital_allocation_runs"
	ManualOverridesCollectionName       = "manual_overrides"
	CurrentPositionsCollectionName      = "current_positions"
	AIBatchJobsCollectionName           = "ai_batch_jobs"
	AIBatchItemsCollectionName          = "ai_batch_items"
	JobReconciliationLogsCollectionName = "job_reconciliation_logs"
)

// DefaultIndexCollections returns the collection names used when the index bootstrap
// layer is called without an explicit collection-name override.
func DefaultIndexCollections() platformconfig.CollectionConfig {
	return platformconfig.CollectionConfig{
		Companies:             CompaniesCollectionName,
		CompanyReviews:        CompanyReviewsCollectionName,
		InvestmentTheses:      InvestmentThesesCollectionName,
		WorkflowRuns:          WorkflowRunsCollectionName,
		WorkflowStepRuns:      WorkflowStepRunsCollectionName,
		ConfigSnapshots:       ConfigSnapshotsCollectionName,
		CapitalAllocationRuns: CapitalAllocationRunsCollectionName,
		ManualOverrides:       ManualOverridesCollectionName,
		CurrentPositions:      CurrentPositionsCollectionName,
		AIBatchJobs:           AIBatchJobsCollectionName,
		AIBatchItems:          AIBatchItemsCollectionName,
		JobReconciliationLogs: JobReconciliationLogsCollectionName,
	}
}

func resolveIndexCollections(overrides ...platformconfig.CollectionConfig) platformconfig.CollectionConfig {
	resolved := DefaultIndexCollections()
	if len(overrides) == 0 {
		return resolved
	}

	override := overrides[0]
	if name := strings.TrimSpace(override.Companies); name != "" {
		resolved.Companies = name
	}
	if name := strings.TrimSpace(override.CompanyReviews); name != "" {
		resolved.CompanyReviews = name
	}
	if name := strings.TrimSpace(override.InvestmentTheses); name != "" {
		resolved.InvestmentTheses = name
	}
	if name := strings.TrimSpace(override.WorkflowRuns); name != "" {
		resolved.WorkflowRuns = name
	}
	if name := strings.TrimSpace(override.WorkflowStepRuns); name != "" {
		resolved.WorkflowStepRuns = name
	}
	if name := strings.TrimSpace(override.ConfigSnapshots); name != "" {
		resolved.ConfigSnapshots = name
	}
	if name := strings.TrimSpace(override.CapitalAllocationRuns); name != "" {
		resolved.CapitalAllocationRuns = name
	}
	if name := strings.TrimSpace(override.ManualOverrides); name != "" {
		resolved.ManualOverrides = name
	}
	if name := strings.TrimSpace(override.CurrentPositions); name != "" {
		resolved.CurrentPositions = name
	}
	if name := strings.TrimSpace(override.AIBatchJobs); name != "" {
		resolved.AIBatchJobs = name
	}
	if name := strings.TrimSpace(override.AIBatchItems); name != "" {
		resolved.AIBatchItems = name
	}
	if name := strings.TrimSpace(override.JobReconciliationLogs); name != "" {
		resolved.JobReconciliationLogs = name
	}

	return resolved
}
