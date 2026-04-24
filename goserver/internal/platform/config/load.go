package config

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

func Load(path string) (*AppConfig, error) {
	contents, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	config, err := loadBytesWithMeta(contents, filepath.Ext(path))
	if err != nil {
		return nil, err
	}
	return config, nil
}

func LoadFromFile(path string) (AppConfig, error) {
	config, err := Load(path)
	if err != nil {
		return AppConfig{}, err
	}
	return *config, nil
}

func LoadReader(reader io.Reader, format string) (*AppConfig, error) {
	contents, err := io.ReadAll(reader)
	if err != nil {
		return nil, err
	}
	return loadBytesWithMeta(contents, format)
}

func LoadBytes(contents []byte, extension string) (AppConfig, error) {
	config, err := loadBytesWithMeta(contents, extension)
	if err != nil {
		return AppConfig{}, err
	}
	return *config, nil
}

func loadBytesWithMeta(contents []byte, extension string) (*AppConfig, error) {
	config := Default()

	meta, err := detectTopLevelKeys(contents, extension)
	if err != nil {
		return nil, err
	}

	switch strings.ToLower(strings.TrimSpace(extension)) {
	case ".yaml", ".yml":
		decoder := yaml.NewDecoder(bytes.NewReader(contents))
		decoder.KnownFields(true)
		if err := decoder.Decode(&config); err != nil {
			return nil, fmt.Errorf("decode yaml config: %w", err)
		}
	case ".json", "":
		decoder := json.NewDecoder(bytes.NewReader(contents))
		decoder.DisallowUnknownFields()
		if err := decoder.Decode(&config); err != nil {
			return nil, fmt.Errorf("decode json config: %w", err)
		}
	default:
		return nil, fmt.Errorf("%w: %q", ErrUnsupportedConfigFormat, extension)
	}

	if meta.HasAI && !meta.HasAsyncAI {
		config.AsyncAI = config.AI
	}
	config.applyEnvOverrides()
	config.normalizeDerived()
	if err := config.Validate(); err != nil {
		return nil, err
	}

	return &config, nil
}

func (config AppConfig) MarshalSanitizedJSON() (map[string]any, error) {
	document := config.toSanitizedDocument()

	payload, err := json.Marshal(document)
	if err != nil {
		return nil, err
	}

	var out map[string]any
	if err := json.Unmarshal(payload, &out); err != nil {
		return nil, err
	}
	return out, nil
}

type loadMeta struct {
	HasAI      bool
	HasAsyncAI bool
}

func detectTopLevelKeys(contents []byte, extension string) (loadMeta, error) {
	switch strings.ToLower(strings.TrimSpace(extension)) {
	case ".yaml", ".yml":
		var node yaml.Node
		if err := yaml.Unmarshal(contents, &node); err != nil {
			return loadMeta{}, fmt.Errorf("inspect yaml config keys: %w", err)
		}
		if len(node.Content) == 0 {
			return loadMeta{}, nil
		}
		root := node.Content[0]
		meta := loadMeta{}
		for index := 0; index+1 < len(root.Content); index += 2 {
			key := strings.TrimSpace(root.Content[index].Value)
			switch key {
			case "ai":
				meta.HasAI = true
			case "asyncAi":
				meta.HasAsyncAI = true
			}
		}
		return meta, nil
	case ".json", "":
		var root map[string]json.RawMessage
		if err := json.Unmarshal(contents, &root); err != nil {
			return loadMeta{}, fmt.Errorf("inspect json config keys: %w", err)
		}
		_, hasAI := root["ai"]
		_, hasAsyncAI := root["asyncAi"]
		return loadMeta{HasAI: hasAI, HasAsyncAI: hasAsyncAI}, nil
	default:
		return loadMeta{}, fmt.Errorf("%w: %q", ErrUnsupportedConfigFormat, extension)
	}
}

func (config *AppConfig) applyEnvOverrides() {
	if config == nil {
		return
	}

	config.SchemaVersion = envOrDefault("PLATFORM_CONFIG_SCHEMA_VERSION", config.SchemaVersion)
	config.Environment = envOrDefault("PLATFORM_ENVIRONMENT", config.Environment)
	config.Timezone = envOrDefault("PLATFORM_TIMEZONE", config.Timezone)
	config.Global.DefaultTimezone = envOrDefault("PLATFORM_DEFAULT_TIMEZONE", config.Global.DefaultTimezone)
	config.Global.DefaultCurrency = envOrDefault("PLATFORM_DEFAULT_CURRENCY", config.Global.DefaultCurrency)
	config.Global.ReviewSchemaVersion = envOrDefault("PLATFORM_REVIEW_SCHEMA_VERSION", config.Global.ReviewSchemaVersion)
	config.Global.PromptSchemaVersion = envOrDefault("PLATFORM_PROMPT_SCHEMA_VERSION", config.Global.PromptSchemaVersion)
	config.Global.AIProviders.DefaultProvider = envOrDefault("PLATFORM_AI_PROVIDER", config.Global.AIProviders.DefaultProvider)
	config.Global.AIProviders.DefaultModel = envOrDefault("PLATFORM_OPENAI_MODEL", config.Global.AIProviders.DefaultModel)
	config.Global.AIProviders.ReviewPromptVersion = envOrDefault("PLATFORM_REVIEW_PROMPT_VERSION", config.Global.AIProviders.ReviewPromptVersion)
	config.Global.FeatureFlags.EnableAsyncAIReview = envBoolOrDefault("PLATFORM_ENABLE_ASYNC_AI_REVIEW", config.Global.FeatureFlags.EnableAsyncAIReview)
	config.Global.FeatureFlags.EnableCurrentPositionProjection = envBoolOrDefault("PLATFORM_ENABLE_CURRENT_POSITION_PROJECTION", config.Global.FeatureFlags.EnableCurrentPositionProjection)
	config.Global.FeatureFlags.EnableTradingWorkflow = envBoolOrDefault("PLATFORM_ENABLE_TRADING_WORKFLOW", config.Global.FeatureFlags.EnableTradingWorkflow)

	config.Server.Port = envIntOrDefault("PLATFORM_PORT", config.Server.Port)
	config.Server.FrontendRootDir = envOrDefault("PLATFORM_FRONTEND_ROOT", config.Server.FrontendRootDir)
	config.Server.ReadHeaderTimeout = envDurationOrDefault("PLATFORM_READ_HEADER_TIMEOUT", config.Server.ReadHeaderTimeout)

	config.Mongo.URI = envOrDefault("PLATFORM_MONGODB_URI", config.Mongo.URI)
	config.Mongo.Database = envOrDefault("PLATFORM_MONGODB_DB_NAME", config.Mongo.Database)
	config.Mongo.Collections.Companies = envOrDefault("PLATFORM_COMPANIES_COLLECTION", config.Mongo.Collections.Companies)
	config.Mongo.Collections.CompanyReviews = envOrDefault("PLATFORM_COMPANY_REVIEWS_COLLECTION", config.Mongo.Collections.CompanyReviews)
	config.Mongo.Collections.InvestmentTheses = envOrDefault("PLATFORM_INVESTMENT_THESES_COLLECTION", config.Mongo.Collections.InvestmentTheses)
	config.Mongo.Collections.WorkflowRuns = envOrDefault("PLATFORM_WORKFLOW_RUNS_COLLECTION", config.Mongo.Collections.WorkflowRuns)
	config.Mongo.Collections.WorkflowStepRuns = envOrDefault("PLATFORM_WORKFLOW_STEP_RUNS_COLLECTION", config.Mongo.Collections.WorkflowStepRuns)
	config.Mongo.Collections.ConfigSnapshots = envOrDefault("PLATFORM_CONFIG_SNAPSHOTS_COLLECTION", config.Mongo.Collections.ConfigSnapshots)
	config.Mongo.Collections.CapitalAllocationRuns = envOrDefault("PLATFORM_CAPITAL_ALLOCATIONS_COLLECTION", config.Mongo.Collections.CapitalAllocationRuns)
	config.Mongo.Collections.ManualOverrides = envOrDefault("PLATFORM_MANUAL_OVERRIDES_COLLECTION", config.Mongo.Collections.ManualOverrides)
	config.Mongo.Collections.CurrentPositions = envOrDefault("PLATFORM_CURRENT_POSITIONS_COLLECTION", config.Mongo.Collections.CurrentPositions)
	config.Mongo.Collections.AIBatchJobs = envOrDefault("PLATFORM_AI_BATCH_JOBS_COLLECTION", config.Mongo.Collections.AIBatchJobs)
	config.Mongo.Collections.AIBatchItems = envOrDefault("PLATFORM_AI_BATCH_ITEMS_COLLECTION", config.Mongo.Collections.AIBatchItems)
	config.Mongo.Collections.JobReconciliationLogs = envOrDefault("PLATFORM_JOB_RECONCILIATION_LOGS_COLLECTION", config.Mongo.Collections.JobReconciliationLogs)
	config.Mongo.Collections.ProviderBatchJobs = envOrDefault("PLATFORM_PROVIDER_BATCH_JOBS_COLLECTION", config.Mongo.Collections.ProviderBatchJobs)
	config.Mongo.Collections.ProviderBatchIterations = envOrDefault("PLATFORM_PROVIDER_BATCH_ITERATIONS_COLLECTION", config.Mongo.Collections.ProviderBatchIterations)

	config.AsyncAI.Enabled = envBoolOrDefault("PLATFORM_ASYNC_AI_ENABLED", config.AsyncAI.Enabled)
	config.AsyncAI.ProviderName = envOrDefault("PLATFORM_AI_PROVIDER", firstNonEmptyString(config.AsyncAI.ProviderName, config.AsyncAI.Provider))
	config.AsyncAI.DefaultModelName = envOrDefault("PLATFORM_OPENAI_MODEL", firstNonEmptyString(config.AsyncAI.DefaultModelName, config.AsyncAI.Model))
	config.AsyncAI.PromptVersion = envOrDefault("PLATFORM_REVIEW_PROMPT_VERSION", config.AsyncAI.PromptVersion)
	config.AsyncAI.SchemaVersion = envOrDefault("PLATFORM_AI_SCHEMA_VERSION", config.AsyncAI.SchemaVersion)
	config.AsyncAI.Snapshot.ReviewSchemaVersion = envOrDefault("PLATFORM_REVIEW_SCHEMA_VERSION", config.AsyncAI.Snapshot.ReviewSchemaVersion)
	config.AsyncAI.Snapshot.OutputSchemaVersion = envOrDefault("PLATFORM_OUTPUT_SCHEMA_VERSION", config.AsyncAI.Snapshot.OutputSchemaVersion)
	config.AsyncAI.BaseURL = envOrDefault("PLATFORM_OPENAI_BASE_URL", config.AsyncAI.BaseURL)
	config.AsyncAI.APIKey = envOrDefault("PLATFORM_AI_API_KEY", config.AsyncAI.APIKey)
	config.AsyncAI.ResponseInstructions = envOrDefault("PLATFORM_OPENAI_RESPONSE_INSTRUCTIONS", config.AsyncAI.ResponseInstructions)
	config.AsyncAI.BatchEndpoint = envOrDefault("PLATFORM_BATCH_ENDPOINT", config.AsyncAI.BatchEndpoint)
	config.AsyncAI.CompletionWindow = envOrDefault("PLATFORM_BATCH_COMPLETION_WINDOW", config.AsyncAI.CompletionWindow)
	config.AsyncAI.Worker.Enabled = envBoolOrDefault("PLATFORM_ASYNC_AI_WORKERS_ENABLED", config.AsyncAI.Worker.Enabled)
	config.AsyncAI.Worker.RefreshInterval = envDurationOrDefault("PLATFORM_ASYNC_AI_REFRESH_INTERVAL", config.AsyncAI.Worker.RefreshInterval)
	config.AsyncAI.Worker.MinBatchRefreshAge = envDurationOrDefault("PLATFORM_ASYNC_AI_MIN_BATCH_REFRESH_AGE", config.AsyncAI.Worker.MinBatchRefreshAge)
	config.AsyncAI.Worker.FollowUpClaimTimeout = envDurationOrDefault("PLATFORM_ASYNC_AI_FOLLOW_UP_CLAIM_TIMEOUT", config.AsyncAI.Worker.FollowUpClaimTimeout)
	config.AsyncAI.Worker.MaxBatchesPerPass = envIntOrDefault("PLATFORM_ASYNC_AI_MAX_BATCHES_PER_PASS", config.AsyncAI.Worker.MaxBatchesPerPass)
	config.AsyncAI.Worker.EnableBatchSubmissionWorker = envBoolOrDefault("PLATFORM_ENABLE_BATCH_SUBMISSION_WORKER", config.AsyncAI.Worker.EnableBatchSubmissionWorker)
	config.AsyncAI.Worker.EnablePollingWorker = envBoolOrDefault("PLATFORM_ENABLE_POLLING_WORKER", config.AsyncAI.Worker.EnablePollingWorker)
	config.AsyncAI.Worker.EnableReconciliationWorker = envBoolOrDefault("PLATFORM_ENABLE_RECONCILIATION_WORKER", config.AsyncAI.Worker.EnableReconciliationWorker)
	config.AsyncAI.Worker.EnableWorkflowContinuationWorker = envBoolOrDefault("PLATFORM_ENABLE_WORKFLOW_CONTINUATION_WORKER", config.AsyncAI.Worker.EnableWorkflowContinuationWorker)
	config.AsyncAI.syncCompatibilityFields()
	config.AI = config.AsyncAI
}

func (config AppConfig) toSanitizedDocument() map[string]any {
	ai := config.AsyncAI
	ai.APIKey = ""
	ai.syncCompatibilityFields()

	return map[string]any{
		"schemaVersion": config.SchemaVersion,
		"environment":   config.Environment,
		"timezone":      config.EffectiveTimezone(),
		"server":        config.Server,
		"mongo":         config.Mongo,
		"global":        config.Global,
		"investing":     config.Investing,
		"trading":       config.Trading,
		"ui":            config.UI,
		"ai": map[string]any{
			"enabled":              ai.Enabled,
			"providerName":         ai.ProviderName,
			"defaultModelName":     ai.DefaultModelName,
			"promptVersion":        ai.PromptVersion,
			"schemaVersion":        ai.SchemaVersion,
			"enabledBooks":         ai.EnabledBooks,
			"batch":                ai.Batch,
			"worker":               ai.Worker,
			"snapshot":             ai.Snapshot,
			"responseInstructions": ai.ResponseInstructions,
			"batchEndpoint":        ai.BatchEndpoint,
			"completionWindow":     ai.CompletionWindow,
			"baseUrl":              ai.BaseURL,
		},
	}
}

func envOrDefault(name, fallback string) string {
	if value, exists := os.LookupEnv(name); exists && strings.TrimSpace(value) != "" {
		return strings.TrimSpace(value)
	}
	return fallback
}

func envBoolOrDefault(name string, fallback bool) bool {
	value, exists := os.LookupEnv(name)
	if !exists || strings.TrimSpace(value) == "" {
		return fallback
	}
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "1", "true", "yes", "on":
		return true
	case "0", "false", "no", "off":
		return false
	default:
		return fallback
	}
}

func envIntOrDefault(name string, fallback int) int {
	value, exists := os.LookupEnv(name)
	if !exists || strings.TrimSpace(value) == "" {
		return fallback
	}
	parsed, err := strconv.Atoi(strings.TrimSpace(value))
	if err != nil {
		return fallback
	}
	return parsed
}

func envDurationOrDefault(name string, fallback time.Duration) time.Duration {
	value, exists := os.LookupEnv(name)
	if !exists || strings.TrimSpace(value) == "" {
		return fallback
	}
	parsed, err := time.ParseDuration(strings.TrimSpace(value))
	if err != nil {
		return fallback
	}
	return parsed
}
