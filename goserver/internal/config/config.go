package config

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"time"

	platformconfig "goserver/internal/platform/config"
)

type Config struct {
	Port     int
	MongoDB  MongoConfig
	OpenAI   OpenAIConfig
	Frontend FrontendConfig
	Worker   WorkerConfig
	Logging  LoggingConfig
	Platform platformconfig.AppConfig
}

type MongoConfig struct {
	URI                            string
	DBName                         string
	JobsCollectionName             string
	SubmissionIterationsCollection string
	ProceduresCollectionName       string
	ProcedureExecutionsCollection  string
}

type OpenAIConfig struct {
	APIKey               string
	Model                string
	ResponseInstructions string
	BatchEndpoint        string
	CompletionWindow     string
	BaseURL              string
}

type FrontendConfig struct {
	RootDir string
}

type WorkerConfig struct {
	Enabled                   bool
	SubmissionRefreshInterval time.Duration
	MinBatchRefreshAge        time.Duration
	FollowUpClaimTimeout      time.Duration
	MaxBatchesPerPass         int
}

type LoggingConfig struct {
	Level       string
	Encoding    string
	Development bool
}

func Load() (Config, error) {
	return loadConfig(true)
}

func LoadOffline() (Config, error) {
	return loadConfig(false)
}

func loadConfig(requireAPIKey bool) (Config, error) {
	if err := loadDefaultEnvFiles(); err != nil {
		return Config{}, err
	}

	apiKey := strings.TrimSpace(os.Getenv("OPENAI_API_KEY"))
	if requireAPIKey {
		var err error
		apiKey, err = requireEnv("OPENAI_API_KEY")
		if err != nil {
			return Config{}, err
		}
	}

	port, err := parsePort(os.Getenv("PORT"))
	if err != nil {
		return Config{}, err
	}

	refreshInterval, err := parseDuration(
		envOrDefault("SUBMISSION_REFRESH_INTERVAL", "15s"),
		"SUBMISSION_REFRESH_INTERVAL",
	)
	if err != nil {
		return Config{}, err
	}

	followUpClaimTimeout, err := parseDuration(
		envOrDefault("SUBMISSION_FOLLOW_UP_CLAIM_TIMEOUT", "2m"),
		"SUBMISSION_FOLLOW_UP_CLAIM_TIMEOUT",
	)
	if err != nil {
		return Config{}, err
	}

	minBatchRefreshAge, err := parseDuration(
		envOrDefault("SUBMISSION_MIN_BATCH_REFRESH_AGE", "30s"),
		"SUBMISSION_MIN_BATCH_REFRESH_AGE",
	)
	if err != nil {
		return Config{}, err
	}

	maxBatchesPerPass, err := parseNonNegativeInt(
		envOrDefault("SUBMISSION_MAX_BATCHES_PER_PASS", "20"),
		"SUBMISSION_MAX_BATCHES_PER_PASS",
	)
	if err != nil {
		return Config{}, err
	}

	config := Config{
		Port: port,
		MongoDB: MongoConfig{
			URI:                            envOrDefault("MONGODB_URI", "mongodb://127.0.0.1:27017"),
			DBName:                         envOrDefault("MONGODB_DB_NAME", "openai_batch_webapp"),
			JobsCollectionName:             "query_jobs",
			SubmissionIterationsCollection: "submissions_iterations",
			ProceduresCollectionName:       "procedures",
			ProcedureExecutionsCollection:  "procedure_executions",
		},
		OpenAI: OpenAIConfig{
			APIKey:               apiKey,
			Model:                envOrDefault("OPENAI_MODEL", "gpt-4.1-mini"),
			ResponseInstructions: envOrDefault("OPENAI_RESPONSE_INSTRUCTIONS", "Answer the user's query clearly and concisely."),
			BatchEndpoint:        "/v1/responses",
			CompletionWindow:     "24h",
			BaseURL:              envOrDefault("OPENAI_BASE_URL", "https://api.openai.com"),
		},
		Frontend: FrontendConfig{
			RootDir: envOrDefault("FRONTEND_ROOT", defaultFrontendRoot()),
		},
		Worker: WorkerConfig{
			Enabled:                   parseBool(envOrDefault("SUBMISSION_REFRESH_WORKER_ENABLED", "true")),
			SubmissionRefreshInterval: refreshInterval,
			MinBatchRefreshAge:        minBatchRefreshAge,
			FollowUpClaimTimeout:      followUpClaimTimeout,
			MaxBatchesPerPass:         maxBatchesPerPass,
		},
		Logging: LoggingConfig{
			Level:       envOrDefault("LOG_LEVEL", "info"),
			Encoding:    envOrDefault("LOG_ENCODING", "console"),
			Development: parseBool(envOrDefault("LOG_DEVELOPMENT", "false")),
		},
	}

	platform, err := buildPlatformConfig(config)
	if err != nil {
		return Config{}, err
	}
	config.Platform = platform

	return config, nil
}

func envOrDefault(name, fallback string) string {
	if value := os.Getenv(name); value != "" {
		return value
	}

	return fallback
}

func requireEnv(name string) (string, error) {
	value := os.Getenv(name)
	if value == "" {
		return "", fmt.Errorf("missing required environment variable: %s", name)
	}

	return value, nil
}

func parsePort(raw string) (int, error) {
	if raw == "" {
		return 3000, nil
	}

	value, err := strconv.Atoi(raw)
	if err != nil {
		return 0, fmt.Errorf("invalid PORT value: %w", err)
	}

	return value, nil
}

func parseDuration(raw, name string) (time.Duration, error) {
	value, err := time.ParseDuration(strings.TrimSpace(raw))
	if err != nil {
		return 0, fmt.Errorf("invalid %s value: %w", name, err)
	}

	if value <= 0 {
		return 0, fmt.Errorf("%s must be greater than zero", name)
	}

	return value, nil
}

func parseNonNegativeInt(raw, name string) (int, error) {
	value, err := strconv.Atoi(strings.TrimSpace(raw))
	if err != nil {
		return 0, fmt.Errorf("invalid %s value: %w", name, err)
	}

	if value < 0 {
		return 0, fmt.Errorf("%s must be zero or greater", name)
	}

	return value, nil
}

func parseBool(raw string) bool {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "0", "false", "no", "off":
		return false
	default:
		return true
	}
}

func loadDefaultEnvFiles() error {
	for _, path := range []string{
		defaultEnvFilePath(),
		defaultEnvLocalFilePath(),
	} {
		if err := loadEnvFile(path); err != nil {
			return err
		}
	}

	return nil
}

func loadEnvFile(path string) error {
	file, err := os.Open(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}

		return err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		key, value, ok := strings.Cut(line, "=")
		if !ok {
			continue
		}

		key = strings.TrimSpace(key)
		if key == "" {
			continue
		}

		if _, exists := os.LookupEnv(key); exists {
			continue
		}

		os.Setenv(key, normalizeEnvValue(value))
	}

	return scanner.Err()
}

func normalizeEnvValue(value string) string {
	trimmed := strings.TrimSpace(value)
	if len(trimmed) >= 2 {
		if (trimmed[0] == '"' && trimmed[len(trimmed)-1] == '"') || (trimmed[0] == '\'' && trimmed[len(trimmed)-1] == '\'') {
			return trimmed[1 : len(trimmed)-1]
		}
	}

	return trimmed
}

func defaultFrontendRoot() string {
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		return "../webapp"
	}

	configDir := filepath.Dir(file)
	serverRoot := filepath.Clean(filepath.Join(configDir, "..", ".."))
	return filepath.Clean(filepath.Join(serverRoot, "..", "webapp"))
}

func defaultEnvFilePath() string {
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		return ".env"
	}

	configDir := filepath.Dir(file)
	serverRoot := filepath.Clean(filepath.Join(configDir, "..", ".."))
	return filepath.Join(serverRoot, ".env")
}

func defaultEnvLocalFilePath() string {
	return defaultEnvFilePath() + ".local"
}

func buildPlatformConfig(base Config) (platformconfig.AppConfig, error) {
	platform := platformconfig.Default()

	platform.Environment = envOrDefault("PLATFORM_ENVIRONMENT", platform.Environment)
	platform.Server.Port = base.Port
	platform.Server.FrontendRootDir = base.Frontend.RootDir

	platform.Mongo.URI = envOrDefault("PLATFORM_MONGODB_URI", base.MongoDB.URI)
	platform.Mongo.Database = envOrDefault("PLATFORM_MONGODB_DB_NAME", base.MongoDB.DBName)
	platform.Mongo.Collections.Companies = envOrDefault("PLATFORM_COMPANIES_COLLECTION", platform.Mongo.Collections.Companies)
	platform.Mongo.Collections.CompanyReviews = envOrDefault("PLATFORM_COMPANY_REVIEWS_COLLECTION", platform.Mongo.Collections.CompanyReviews)
	platform.Mongo.Collections.InvestmentTheses = envOrDefault("PLATFORM_INVESTMENT_THESES_COLLECTION", platform.Mongo.Collections.InvestmentTheses)
	platform.Mongo.Collections.WorkflowRuns = envOrDefault("PLATFORM_WORKFLOW_RUNS_COLLECTION", platform.Mongo.Collections.WorkflowRuns)
	platform.Mongo.Collections.ConfigSnapshots = envOrDefault("PLATFORM_CONFIG_SNAPSHOTS_COLLECTION", platform.Mongo.Collections.ConfigSnapshots)
	platform.Mongo.Collections.CapitalAllocationRuns = envOrDefault("PLATFORM_CAPITAL_ALLOCATIONS_COLLECTION", platform.Mongo.Collections.CapitalAllocationRuns)
	platform.Mongo.Collections.ManualOverrides = envOrDefault("PLATFORM_MANUAL_OVERRIDES_COLLECTION", platform.Mongo.Collections.ManualOverrides)
	platform.Mongo.Collections.CurrentPositions = envOrDefault("PLATFORM_CURRENT_POSITIONS_COLLECTION", platform.Mongo.Collections.CurrentPositions)
	platform.Mongo.Collections.AIBatchJobs = base.MongoDB.JobsCollectionName
	platform.Mongo.Collections.AIBatchIterations = base.MongoDB.SubmissionIterationsCollection

	platform.Global.DefaultTimezone = envOrDefault("PLATFORM_DEFAULT_TIMEZONE", platform.Global.DefaultTimezone)
	platform.Global.AIProviders.DefaultProvider = envOrDefault("PLATFORM_AI_PROVIDER", platform.Global.AIProviders.DefaultProvider)
	platform.Global.AIProviders.DefaultModel = envOrDefault("PLATFORM_OPENAI_MODEL", firstNonEmptyString(base.OpenAI.Model, platform.Global.AIProviders.DefaultModel))
	platform.Global.AIProviders.ReviewPromptVersion = envOrDefault("PLATFORM_REVIEW_PROMPT_VERSION", platform.Global.AIProviders.ReviewPromptVersion)
	platform.Global.FeatureFlags.EnableAsyncAIReview = parseBool(envOrDefault(
		"PLATFORM_ENABLE_ASYNC_AI_REVIEW",
		strconv.FormatBool(platform.Global.FeatureFlags.EnableAsyncAIReview),
	))
	platform.Global.FeatureFlags.EnableCurrentPositionProjection = parseBool(envOrDefault(
		"PLATFORM_ENABLE_CURRENT_POSITION_PROJECTION",
		strconv.FormatBool(platform.Global.FeatureFlags.EnableCurrentPositionProjection),
	))
	platform.Global.FeatureFlags.EnableTradingWorkflow = parseBool(envOrDefault(
		"PLATFORM_ENABLE_TRADING_WORKFLOW",
		strconv.FormatBool(platform.Global.FeatureFlags.EnableTradingWorkflow),
	))

	platform.AsyncAI.Enabled = parseBool(envOrDefault("PLATFORM_ASYNC_AI_ENABLED", strconv.FormatBool(platform.AsyncAI.Enabled)))
	platform.AsyncAI.Provider = envOrDefault("PLATFORM_AI_PROVIDER", platform.AsyncAI.Provider)
	platform.AsyncAI.Model = envOrDefault("PLATFORM_OPENAI_MODEL", firstNonEmptyString(base.OpenAI.Model, platform.AsyncAI.Model))
	platform.AsyncAI.PromptVersion = envOrDefault("PLATFORM_REVIEW_PROMPT_VERSION", platform.AsyncAI.PromptVersion)
	platform.AsyncAI.ResponseInstructions = envOrDefault("PLATFORM_OPENAI_RESPONSE_INSTRUCTIONS", platform.AsyncAI.ResponseInstructions)
	platform.AsyncAI.BatchEndpoint = firstNonEmptyString(base.OpenAI.BatchEndpoint, platform.AsyncAI.BatchEndpoint)
	platform.AsyncAI.CompletionWindow = firstNonEmptyString(base.OpenAI.CompletionWindow, platform.AsyncAI.CompletionWindow)
	platform.AsyncAI.BaseURL = firstNonEmptyString(base.OpenAI.BaseURL, platform.AsyncAI.BaseURL)
	platform.AsyncAI.APIKey = base.OpenAI.APIKey
	platform.AsyncAI.Worker.Enabled = base.Worker.Enabled
	platform.AsyncAI.Worker.RefreshInterval = base.Worker.SubmissionRefreshInterval
	platform.AsyncAI.Worker.MinBatchRefreshAge = base.Worker.MinBatchRefreshAge
	platform.AsyncAI.Worker.FollowUpClaimTimeout = base.Worker.FollowUpClaimTimeout
	platform.AsyncAI.Worker.MaxBatchesPerPass = base.Worker.MaxBatchesPerPass

	if err := platform.Validate(); err != nil {
		return platformconfig.AppConfig{}, fmt.Errorf("invalid platform config: %w", err)
	}

	return platform, nil
}

func firstNonEmptyString(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}

	return ""
}
