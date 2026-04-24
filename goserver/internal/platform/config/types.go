package config

import "time"

const (
	defaultEnvironment = "development"
	defaultTimezone    = "Asia/Kolkata"
	defaultCurrency    = "INR"
)

// AppConfig is the strongly typed runtime configuration consumed by the platform.
// It keeps the existing runtime knobs for server and mongo setup while focusing
// on the business-facing investing, trading, AI, and UI config surface.
type AppConfig struct {
	SchemaVersion string `json:"schemaVersion" yaml:"schemaVersion"`
	Environment   string `json:"environment" yaml:"environment"`
	Timezone      string `json:"timezone,omitempty" yaml:"timezone,omitempty"`

	Server ServerConfig `json:"server,omitempty" yaml:"server,omitempty"`
	Mongo  MongoConfig  `json:"mongo,omitempty" yaml:"mongo,omitempty"`

	Global    GlobalConfig    `json:"global" yaml:"global"`
	Investing InvestingConfig `json:"investing" yaml:"investing"`
	Trading   TradingConfig   `json:"trading" yaml:"trading"`
	UI        UIConfig        `json:"ui" yaml:"ui"`

	// AI is the preferred external serialization key.
	AI AIConfig `json:"ai,omitempty" yaml:"ai,omitempty"`
	// AsyncAI remains available for compatibility with existing runtime code and
	// legacy config files.
	AsyncAI AIConfig `json:"asyncAi,omitempty" yaml:"asyncAi,omitempty"`
}

type ServerConfig struct {
	Port              int           `json:"port" yaml:"port"`
	FrontendRootDir   string        `json:"frontendRootDir,omitempty" yaml:"frontendRootDir,omitempty"`
	ReadHeaderTimeout time.Duration `json:"readHeaderTimeout" yaml:"readHeaderTimeout"`
}

type MongoConfig struct {
	URI         string           `json:"uri" yaml:"uri"`
	Database    string           `json:"database" yaml:"database"`
	Collections CollectionConfig `json:"collections" yaml:"collections"`
}

type CollectionConfig struct {
	Companies               string `json:"companies" yaml:"companies"`
	CompanyReviews          string `json:"companyReviews" yaml:"companyReviews"`
	InvestmentTheses        string `json:"investmentTheses" yaml:"investmentTheses"`
	WorkflowRuns            string `json:"workflowRuns" yaml:"workflowRuns"`
	WorkflowStepRuns        string `json:"workflowStepRuns" yaml:"workflowStepRuns"`
	ConfigSnapshots         string `json:"configSnapshots" yaml:"configSnapshots"`
	CapitalAllocationRuns   string `json:"capitalAllocationRuns" yaml:"capitalAllocationRuns"`
	ManualOverrides         string `json:"manualOverrides" yaml:"manualOverrides"`
	CurrentPositions        string `json:"currentPositions" yaml:"currentPositions"`
	AIBatchJobs             string `json:"aiBatchJobs" yaml:"aiBatchJobs"`
	AIBatchItems            string `json:"aiBatchItems" yaml:"aiBatchItems"`
	JobReconciliationLogs   string `json:"jobReconciliationLogs" yaml:"jobReconciliationLogs"`
	ProviderBatchJobs       string `json:"providerBatchJobs" yaml:"providerBatchJobs"`
	ProviderBatchIterations string `json:"providerBatchIterations" yaml:"providerBatchIterations"`
}
