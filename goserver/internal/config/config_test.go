package config

import (
	"testing"
	"time"
)

func TestBuildPlatformConfigUsesPrimaryServerSettings(t *testing.T) {
	base := Config{
		Port: 3000,
		MongoDB: MongoConfig{
			URI:                            "mongodb://127.0.0.1:27017",
			DBName:                         "openai_batch_webapp",
			JobsCollectionName:             "query_jobs",
			SubmissionIterationsCollection: "submissions_iterations",
		},
		OpenAI: OpenAIConfig{
			APIKey:               "secret",
			Model:                "gpt-5.4-mini",
			ResponseInstructions: "Return structured JSON only.",
			BatchEndpoint:        "/v1/responses",
			CompletionWindow:     "24h",
			BaseURL:              "https://api.openai.com",
		},
		Frontend: FrontendConfig{
			RootDir: "../webapp",
		},
		Worker: WorkerConfig{
			Enabled:                   true,
			SubmissionRefreshInterval: 15 * time.Second,
			MinBatchRefreshAge:        30 * time.Second,
			FollowUpClaimTimeout:      2 * time.Minute,
			MaxBatchesPerPass:         20,
		},
	}

	platform, err := buildPlatformConfig(base)
	if err != nil {
		t.Fatalf("buildPlatformConfig returned error: %v", err)
	}
	if platform.Mongo.URI != base.MongoDB.URI {
		t.Fatalf("expected platform mongo URI %q, got %q", base.MongoDB.URI, platform.Mongo.URI)
	}
	if platform.Mongo.Database != base.MongoDB.DBName {
		t.Fatalf("expected platform mongo database %q, got %q", base.MongoDB.DBName, platform.Mongo.Database)
	}
	if platform.Mongo.Collections.ProviderBatchJobs != base.MongoDB.JobsCollectionName {
		t.Fatalf("expected provider batch jobs collection %q, got %q", base.MongoDB.JobsCollectionName, platform.Mongo.Collections.ProviderBatchJobs)
	}
	if platform.Mongo.Collections.AIBatchJobs != "ai_batch_jobs" {
		t.Fatalf("expected platform async batch jobs collection %q, got %q", "ai_batch_jobs", platform.Mongo.Collections.AIBatchJobs)
	}
	if platform.AsyncAI.APIKey != base.OpenAI.APIKey {
		t.Fatalf("expected platform async ai api key to be carried over")
	}
	if platform.AsyncAI.Worker.RefreshInterval != base.Worker.SubmissionRefreshInterval {
		t.Fatalf("expected refresh interval %s, got %s", base.Worker.SubmissionRefreshInterval, platform.AsyncAI.Worker.RefreshInterval)
	}
	if platform.Server.FrontendRootDir != base.Frontend.RootDir {
		t.Fatalf("expected frontend root %q, got %q", base.Frontend.RootDir, platform.Server.FrontendRootDir)
	}
}

func TestBuildPlatformConfigHonorsPlatformOverrides(t *testing.T) {
	t.Setenv("PLATFORM_MONGODB_DB_NAME", "selvam_platform")
	t.Setenv("PLATFORM_COMPANIES_COLLECTION", "listed_companies")

	base := Config{
		Port: 3000,
		MongoDB: MongoConfig{
			URI:                            "mongodb://127.0.0.1:27017",
			DBName:                         "openai_batch_webapp",
			JobsCollectionName:             "query_jobs",
			SubmissionIterationsCollection: "submissions_iterations",
		},
		OpenAI: OpenAIConfig{
			Model:                "gpt-5.4-mini",
			ResponseInstructions: "Return structured JSON only.",
			BatchEndpoint:        "/v1/responses",
			CompletionWindow:     "24h",
			BaseURL:              "https://api.openai.com",
		},
		Frontend: FrontendConfig{
			RootDir: "../webapp",
		},
		Worker: WorkerConfig{
			Enabled:                   true,
			SubmissionRefreshInterval: 15 * time.Second,
			MinBatchRefreshAge:        30 * time.Second,
			FollowUpClaimTimeout:      2 * time.Minute,
			MaxBatchesPerPass:         20,
		},
	}

	platform, err := buildPlatformConfig(base)
	if err != nil {
		t.Fatalf("buildPlatformConfig returned error: %v", err)
	}
	if platform.Mongo.Database != "selvam_platform" {
		t.Fatalf("expected overridden platform database, got %q", platform.Mongo.Database)
	}
	if platform.Mongo.Collections.Companies != "listed_companies" {
		t.Fatalf("expected overridden companies collection, got %q", platform.Mongo.Collections.Companies)
	}
}
