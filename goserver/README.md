# Selvam Platform Foundation

This phase builds the production-grade backend skeleton for an AI-assisted India equities and ETFs investing and trading platform. It is intentionally backend-first and async-first: it persists historical review truth, preserves workflow/config snapshots, and keeps AI orchestration behind versioned workflow and provider contracts. The platform is integrated into the existing Go server instead of running as a separate API process.

## Architecture Summary

- `investing` and `trading` are ring-fenced with separate workflow contracts, configs, state, and persistence references.
- `company_reviews` is the immutable historical aggregate root for review snapshots. Sections, sub-scores, evidence, decision action, position snapshot, and change log are embedded so a finalized review loads in one read.
- `companies`, `investment_theses`, `workflow_runs`, `config_snapshots`, `capital_allocation_runs`, `manual_overrides`, and `current_positions` stay in separate collections because their lifecycle and query patterns differ.
- Async AI flows are orchestration-first and batch-only. The platform creates pending review shells, `ai_batch_jobs`, `ai_batch_items`, and `workflow_step_runs`, then a worker loop submits, polls, reconciles, validates, and materializes results later.
- Async AI uses a provider interface plus a concrete adapter over the existing OpenAI batch job subsystem. HTTP requests only start, inspect, resume, or reconcile workflows; they never wait for final AI completion.
- `/api/v1` exposes stable contracts for future web UI work: companies, reviews, theses, workflow runs, capital allocation, config inspection, overrides, and positions.
- The same server still exposes the legacy batch automation routes, so `/submissions` and `/platform` now share one origin and one configuration tree.

## Project Tree

```text
cmd/
  goserver/
configs/
  platform.example.yaml
internal/
  platform/
    api/http/
      dto/
    app/
    config/
    domain/
    ports/
    provider/
      ai/
    repository/mongo/
    service/
    testutil/
    worker/
    workflow/
      investing/
      trading/
testdata/
  examples/
```

## Key Design Choices

- Historical reviews are append-only in practice. Review shells move through `pending_input -> pending_ai -> ai_completed_unvalidated -> ai_validated -> finalized`, and `finalized` / `superseded` reviews are immutable.
- Hard gates and section action caps can override weighted totals during action mapping.
- Config snapshots are created per workflow run from sanitized config JSON so decisions remain replayable without storing provider secrets.
- Workflow runs persist typed step snapshots, timing, errors, and async task references to support dry runs, idempotency keys, retries, and replay/resume work.
- AI inputs and outputs are persisted at item level so partial completion, validation failures, and manual retry/skip flows are inspectable in the UI.
- AI scoring and data ingestion are placeholders in this phase. The async contract, persistence model, and reconciliation path are real; the intelligence behind them remains intentionally stubbed.

## Collections

- `companies`
- `company_reviews`
- `investment_theses`
- `workflow_runs`
- `workflow_step_runs`
- `config_snapshots`
- `capital_allocation_runs`
- `manual_overrides`
- `current_positions`
- `ai_batch_jobs`
- `ai_batch_items`
- `job_reconciliation_logs`
- Reused legacy provider collections:
  - `query_jobs`
  - `submissions_iterations`

## Async Lifecycle

1. `POST /api/v1/workflow-runs/investing/start` creates a workflow run and returns immediately.
2. The workflow creates pending `company_reviews` shells plus `ai_batch_jobs` and `ai_batch_items`.
3. The in-process worker loop submits pending provider batches using the existing batch job subsystem.
4. Polling and reconciliation update job/item state without blocking requests.
5. AI outputs are parsed, schema-validated, business-validated, and only then transformed into finalized immutable reviews and thesis updates.
6. Invalid outputs remain persisted as failed or invalid items for retry or manual inspection.

## Running

1. Start MongoDB locally.
2. Set the usual main-server environment variables. Platform settings are now derived from the primary config object and can be tuned with `PLATFORM_*` environment overrides when needed.
3. Run:

```bash
go run ./cmd/goserver
```

The server will expose both the existing batch automation routes and the new platform routes.

If `asyncAi.worker.enabled=true`, the existing server process also runs the initial in-process worker supervisor that:
- submits created batch jobs
- polls running jobs
- reconciles results
- continues waiting workflows

## API Surface

- Legacy batch APIs remain available under `/api/...`.
- `GET /api/v1/companies`
- `GET /api/v1/companies/{id}`
- `GET /api/v1/companies/{id}/reviews`
- `GET /api/v1/companies/{id}/thesis`
- `GET /api/v1/companies/{id}/history-summary`
- `GET /api/v1/reviews`
- `GET /api/v1/reviews/{id}`
- `GET /api/v1/reviews/{id}/diff`
- `GET /api/v1/reviews/{id}/evidence`
- `GET /api/v1/workflow-runs`
- `GET /api/v1/workflow-runs/{id}`
- `GET /api/v1/workflow-runs/{id}/steps`
- `GET /api/v1/workflow-runs/{id}/status`
- `POST /api/v1/workflow-runs/investing/start`
- `POST /api/v1/workflow-runs/investing/dry-run`
- `POST /api/v1/workflow-runs/{id}/resume`
- `POST /api/v1/workflow-runs/{id}/reconcile`
- `GET /api/v1/workflow-runs/{id}/summary`
- `GET /api/v1/ai-batch-jobs`
- `GET /api/v1/ai-batch-jobs/{id}`
- `GET /api/v1/ai-batch-jobs/{id}/items`
- `POST /api/v1/ai-batch-jobs/{id}/retry`
- `POST /api/v1/ai-batch-items/{id}/retry`
- `POST /api/v1/ai-batch-items/{id}/skip`
- `GET /api/v1/capital-allocations`
- `GET /api/v1/capital-allocations/{id}`
- `GET /api/v1/config/current`
- `GET /api/v1/config/snapshots`
- `GET /api/v1/config/snapshots/{id}`
- `POST /api/v1/overrides`
- `GET /api/v1/overrides`
- `GET /api/v1/overrides/{id}`
- `GET /api/v1/positions`
- `GET /api/v1/positions/{book_type}`

## Test Coverage

- Config parsing and validation
- Review schema validation
- Action mapping
- Change detection
- Persistence document round-trips
- Workflow contract ordering and async investing orchestration behavior
- HTTP handler basics
- Manual override validation
- Config snapshot sanitization/persistence behavior

## Assumptions

- Percentage fields use whole-percentage units, so `1` means `1%` and `70` means `70%`.
- Config snapshots intentionally exclude secrets such as provider API keys.
- The phase-1 server focuses on stable contracts and persistence, not final scoring formulas, broker execution, or final front-end UX.
- The reference config file is documentation for the platform section shape; the real runtime uses the integrated main server config and same-origin routes.
- `configs/platform.example.yaml` remains as a reference snapshot of platform defaults, but the integrated server now sources platform settings from the primary server config object.
