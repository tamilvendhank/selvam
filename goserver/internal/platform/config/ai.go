package config

import "time"

type AIConfig struct {
	Enabled          bool               `json:"enabled" yaml:"enabled"`
	ProviderName     string             `json:"providerName,omitempty" yaml:"providerName,omitempty"`
	DefaultModelName string             `json:"defaultModelName,omitempty" yaml:"defaultModelName,omitempty"`
	PromptVersion    string             `json:"promptVersion,omitempty" yaml:"promptVersion,omitempty"`
	SchemaVersion    string             `json:"schemaVersion,omitempty" yaml:"schemaVersion,omitempty"`
	EnabledBooks     AllowedBooksConfig `json:"enabledBooks,omitempty" yaml:"enabledBooks,omitempty"`
	Batch            AIBatchConfig      `json:"batch" yaml:"batch"`
	Worker           AIWorkerConfig     `json:"worker" yaml:"worker"`
	Snapshot         AISnapshotConfig   `json:"snapshot" yaml:"snapshot"`

	ResponseInstructions string `json:"responseInstructions,omitempty" yaml:"responseInstructions,omitempty"`
	BatchEndpoint        string `json:"batchEndpoint,omitempty" yaml:"batchEndpoint,omitempty"`
	CompletionWindow     string `json:"completionWindow,omitempty" yaml:"completionWindow,omitempty"`
	BaseURL              string `json:"baseUrl,omitempty" yaml:"baseUrl,omitempty"`
	APIKey               string `json:"apiKey,omitempty" yaml:"apiKey,omitempty"`

	// Compatibility aliases consumed by existing runtime code.
	Provider string `json:"-" yaml:"-"`
	Model    string `json:"-" yaml:"-"`
}

type AIBatchConfig struct {
	MaxBatchSize               int           `json:"maxBatchSize" yaml:"maxBatchSize"`
	MaxItemsPerBatch           int           `json:"maxItemsPerBatch" yaml:"maxItemsPerBatch"`
	SubmissionRetryLimit       int           `json:"submissionRetryLimit" yaml:"submissionRetryLimit"`
	PollRetryLimit             int           `json:"pollRetryLimit" yaml:"pollRetryLimit"`
	ItemRetryLimit             int           `json:"itemRetryLimit" yaml:"itemRetryLimit"`
	PollInterval               time.Duration `json:"pollInterval" yaml:"pollInterval"`
	ReconciliationInterval     time.Duration `json:"reconciliationInterval" yaml:"reconciliationInterval"`
	ValidationFailureRetryable bool          `json:"validationFailureRetryable" yaml:"validationFailureRetryable"`
	ResultFetchTimeout         time.Duration `json:"resultFetchTimeout" yaml:"resultFetchTimeout"`
	BatchTimeout               time.Duration `json:"batchTimeout" yaml:"batchTimeout"`
}

type AIWorkerConfig struct {
	Enabled                          bool          `json:"enabled" yaml:"enabled"`
	RefreshInterval                  time.Duration `json:"refreshInterval" yaml:"refreshInterval"`
	MinBatchRefreshAge               time.Duration `json:"minBatchRefreshAge" yaml:"minBatchRefreshAge"`
	FollowUpClaimTimeout             time.Duration `json:"followUpClaimTimeout" yaml:"followUpClaimTimeout"`
	MaxBatchesPerPass                int           `json:"maxBatchesPerPass" yaml:"maxBatchesPerPass"`
	EnableBatchSubmissionWorker      bool          `json:"enableBatchSubmissionWorker" yaml:"enableBatchSubmissionWorker"`
	EnablePollingWorker              bool          `json:"enablePollingWorker" yaml:"enablePollingWorker"`
	EnableReconciliationWorker       bool          `json:"enableReconciliationWorker" yaml:"enableReconciliationWorker"`
	EnableWorkflowContinuationWorker bool          `json:"enableWorkflowContinuationWorker" yaml:"enableWorkflowContinuationWorker"`
}

type AISnapshotConfig struct {
	PromptVersion       string `json:"promptVersion,omitempty" yaml:"promptVersion,omitempty"`
	ReviewSchemaVersion string `json:"reviewSchemaVersion,omitempty" yaml:"reviewSchemaVersion,omitempty"`
	OutputSchemaVersion string `json:"outputSchemaVersion,omitempty" yaml:"outputSchemaVersion,omitempty"`
}
