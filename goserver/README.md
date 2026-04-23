# Selvam Platform Foundation

This phase builds the production-grade backend skeleton for an AI-assisted India equities and ETFs investing and trading platform. It is intentionally backend-first and async-first: it persists historical review truth, preserves workflow/config snapshots, and keeps AI orchestration behind versioned workflow and provider contracts. The platform is integrated into the existing Go server instead of running as a separate API process.

## Architecture Summary

- `investing` and `trading` are ring-fenced with separate workflow contracts, configs, state, and persistence references.
- `company_reviews` is the immutable historical aggregate root for review snapshots. Sections, sub-scores, evidence, decision action, position snapshot, and change log are embedded so a finalized review loads in one read.
- `companies`, `investment_theses`, `workflow_runs`, `config_snapshots`, `capital_allocation_runs`, `manual_overrides`, and `current_positions` stay in separate collections because their lifecycle and query patterns differ.
- Async AI flows use a provider interface and a concrete adapter over the existing OpenAI batch job subsystem. Workflow steps store pollable async task references rather than waiting synchronously.
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
    workflow/
      investing/
      trading/
testdata/
  examples/
```

## Key Design Choices

- Historical reviews are append-only in practice. `UpdateDraft` only works while a review is `draft`; `final` reviews are treated as immutable snapshots.
- Hard gates and section action caps can override weighted totals during action mapping.
- Config snapshots are created per workflow run from sanitized config JSON so decisions remain replayable without storing provider secrets.
- Workflow runs persist typed step snapshots, timing, errors, and async task references to support dry runs, idempotency keys, and future replay/resume work.
- AI scoring and data ingestion are placeholders in this phase. The async contract is real; the intelligence behind it is intentionally stubbed.

## Collections

- `companies`
- `company_reviews`
- `investment_theses`
- `workflow_runs`
- `config_snapshots`
- `capital_allocation_runs`
- `manual_overrides`
- `current_positions`
- Reused legacy async batch collections:
  - `query_jobs`
  - `submissions_iterations`

## Running

1. Start MongoDB locally.
2. Set the usual main-server environment variables. Platform settings are now derived from the primary config object and can be tuned with `PLATFORM_*` environment overrides when needed.
3. Run:

```bash
go run ./cmd/goserver
```

The server will expose both the existing batch automation routes and the new platform routes.

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
- `POST /api/v1/workflow-runs/investing/start`
- `POST /api/v1/workflow-runs/investing/dry-run`
- `GET /api/v1/workflow-runs/{id}/summary`
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
- Workflow contract ordering and investing dry-run behavior
- HTTP handler basics
- Manual override validation
- Config snapshot sanitization/persistence behavior

## Assumptions

- Percentage fields use whole-percentage units, so `1` means `1%` and `70` means `70%`.
- Config snapshots intentionally exclude secrets such as provider API keys.
- The phase-1 server focuses on stable contracts and persistence, not final scoring formulas, broker execution, or final front-end UX.
- `configs/platform.example.yaml` remains as a reference snapshot of platform defaults, but the integrated server now sources platform settings from the primary server config object.
