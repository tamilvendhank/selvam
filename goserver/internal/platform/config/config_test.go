package config

import "testing"

func TestLoadBytesYAML(t *testing.T) {
	input := []byte(`
schemaVersion: v1alpha1
environment: test
timezone: Asia/Kolkata
server:
  port: 9090
  readHeaderTimeout: 5s
mongo:
  uri: mongodb://127.0.0.1:27017
  database: testdb
  collections:
    companies: companies
    companyReviews: company_reviews
    investmentTheses: investment_theses
    workflowRuns: workflow_runs
    workflowStepRuns: workflow_step_runs
    configSnapshots: config_snapshots
    capitalAllocationRuns: capital_allocation_runs
    manualOverrides: manual_overrides
    currentPositions: current_positions
    aiBatchJobs: ai_batch_jobs
    aiBatchItems: ai_batch_items
    jobReconciliationLogs: job_reconciliation_logs
    providerBatchJobs: query_jobs
    providerBatchIterations: submissions_iterations
global:
  defaultTimezone: Asia/Kolkata
  dataSources:
    financialDataProvider: a
    priceDataProvider: b
    textDocumentProvider: c
  aiProviders:
    defaultProvider: openai-batch
    defaultModel: gpt-5.4-mini
    reviewPromptVersion: investing-review-v1
    batchEnabled: true
  featureFlags:
    enableAsyncAiReview: true
    enableCurrentPositionProjection: true
    enableTradingWorkflow: true
`)

	config, err := LoadBytes(input, ".yaml")
	if err != nil {
		t.Fatalf("LoadBytes returned error: %v", err)
	}
	if config.Server.Port != 9090 {
		t.Fatalf("expected port 9090, got %d", config.Server.Port)
	}
	if config.Investing.DefaultMode == "" {
		t.Fatalf("expected defaults to fill investing config")
	}
}

func TestValidateRejectsBrokenSectionWeights(t *testing.T) {
	config := Default()
	config.Investing.SectionWeights.Investability = 99

	if err := config.Validate(); err == nil {
		t.Fatalf("expected validation error for invalid section weights")
	}
}
