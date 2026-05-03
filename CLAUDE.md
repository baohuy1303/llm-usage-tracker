# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Common commands

```bash
# Run the full stack (API + Redis + Prometheus + Grafana)
docker compose up --build

# Run just the API locally (Redis + Prometheus + Grafana still need to be up via docker if you want their features)
go run ./cmd/api

# Build everything
go build ./...

# Tests (none exist yet; the user is aware)
go test ./...

# Wipe everything and start fresh
docker compose down -v
redis-cli FLUSHDB        # if you kept Redis but wiped SQL

# Regenerate after a schema change in migrations/001_init.sql
rm -rf ./data/app.db
docker compose up --build
```

`docker compose up` without `--build` reuses the cached image and ignores Go code changes. Always pass `--build` after editing source.

## Architecture

Strict 4-layer separation, top to bottom: `cmd/api/main.go` wires everything; `internal/http/` handles HTTP; `internal/service/` is business logic; `internal/store/` and `internal/cache/` are persistence. The dev follows this rigidly per `learn.md`. A handler never touches a repo; a repo never returns business types.

```
HTTP layer (internal/http)         routes, request/response, middleware, /metrics, /swagger
       ↓
Service layer (internal/service)   validation, orchestration, output DTOs
       ↓                ↓
Store (internal/store)   Cache (internal/cache)
   SQLite                  Redis (atomic Lua)
```

### Two parallel persistence stores

- **SQLite is source of truth.** Every write goes here first. Aggregations for reporting (`GET /usage/range`, `/usage/summary`, `/usage/events`) read from SQL only.
- **Redis is a counter cache.** A Lua script (`internal/cache/scripts/incr_usage.lua`, `//go:embed`'d) atomically `INCRBY`s daily/monthly cost and `HINCRBY`s the daily tokens hash, then checks daily and monthly spend against the project's budgets. One round-trip, race-free.
- **Cache failure does not fail the request.** If Redis errors during `IncrUsageWithBudget`, we log and call `DeleteUsageKeys` to invalidate the keys so subsequent reads fall back to SQL. SQL has already committed the row.

### Two parallel data sources for the dashboard

The Grafana dashboard at `grafana/dashboards/pulse-overview.json` reads from both:

- **Prometheus** for cumulative/aggregate metrics (cost, events, tokens, budget %, exceedances).
- **Infinity plugin** for live event-level data (the Latest 5 Calls table, the Avg Latency stat). These hit the API directly, bypassing Prometheus.

Prometheus is pre-aggregated so it can't list individual events. The API can list events but isn't built for fast time-series math. Each endpoint is used for what it's good at.

### Money is stored as millicents

`1 cent = 1000 millicents`, `$1 = 100,000 millicents`. Stored as `int64`. Reason: integer cents truncated cheap LLM calls (e.g. 100 tokens at $0.15/M) to 0. The API exposes dollars via custom `MarshalJSON` on `store.Usage` and `store.Project` (see `internal/store/models.go`). Helpers `MillicentsToDollars` and `DollarsToMillicents` live there too. Internal types use the `Millicents` suffix; JSON field names use `_dollars`.

### Cost is computed server-side

Clients post `tokens_in` and `tokens_out` only. Server multiplies by the model's `input_per_million_cents` / `output_per_million_cents` and divides by 1000 to get millicents. See `UsageService.AddUsage`. Clients can never lie about cost.

### Budget is warn-only enforcement

`POST /usage` always succeeds (the LLM call already happened, blocking is pointless). The response carries a `budget_status` block with `over_budget: true|false` per window. Per-project, budgets are nullable per window (daily/monthly/total). Nil means "no enforcement for that window" and is encoded in the Lua script as `-1`.

### Soft delete on projects

`DELETE /projects/{id}` sets `deleted_at`. Usage events keep their FK intact so historical aggregates still work. `getProject` (internal helper, used by `Update`/`Delete` for existence checks) filters `deleted_at IS NULL`. `GetProjectByID` (the public endpoint method) wraps this and attaches `BudgetStatus`.

## Service wiring order matters

In `cmd/api/main.go`, `UsageService` must be constructed before `ProjectService` because `ProjectService` depends on `UsageService` for budget status computation. Reverse the order and the build fails.

## Metric rehydration on startup

After SQL init in `cmd/api/main.go`:
- `projectService.RehydrateMetrics(ctx)` re-publishes `ProjectInfo` and `ProjectBudgetMillicents` for every non-deleted project so the dashboard dropdown isn't empty after restart.
- `usageService.RehydrateCounterMetrics(ctx)` replays cumulative `UsageEventsTotal`, `UsageCostMillicentsTotal`, `UsageTokensTotal{in,out}` from SQL via `AggregateForMetrics`.

`BudgetExceededTotal` and `LLMCallDurationSeconds` are not rehydrated. Documented in `RehydrateCounterMetrics` doc comment.

## Specific gotchas

- **Redis `WRONGTYPE` errors** after Lua key shape changes (e.g., when the daily tokens key changed from string to hash). Run `redis-cli FLUSHDB` or wait for the 48h TTL to clear old keys.
- **Schema changes** require deleting `./data/app.db` since there's no migration tracking. The single migration file is `internal/store/migrations/001_init.sql` and is `//go:embed`'d into the binary.
- **Prometheus query writing**: never use `increase(metric[Xh])` on this codebase's counters. Sparse-data extrapolation produces bouncing values. Use `sum(metric)` for all-time and `sum(metric) - (sum(metric offset Xh) or vector(0))` for windowed deltas. The dashboard already follows this; preserve it on edits.
- **Grafana display units**: `unit: "short"` and `unit: "ms"` auto-scale (1500 becomes "1.5K" or "1.5 s"). For exact display, use `unit: "none"` (raw with locale separators) or `unit: "locale"` (always with thousand separators) and put the unit name in the panel/column title.
- **`/metrics` lives on the outer mux** (router.go), not the inner one wrapped by `MetricsMiddleware`. Otherwise every Prometheus scrape inflates `http_requests_total`. Same pattern for `/swagger` if/when added.

## Key files for getting oriented

- `cmd/api/main.go` for service wiring and startup order
- `internal/http/router.go` for the route map
- `internal/service/usage_service.go` for the AddUsage flow (SQL → metrics → Lua → budget status response)
- `internal/cache/scripts/incr_usage.lua` for the atomic budget check
- `internal/store/models.go` for the millicents-to-dollars MarshalJSON pattern
- `grafana/dashboards/pulse-overview.json` for the dashboard JSON; bump `version` when editing so Grafana picks up changes cleanly

## API surface

Top-9 endpoints documented at the top of `api.http` under a QUICKSTART section. Full navigation in `api.md`. Detailed CRUD plus error cases live below line 105 of `api.http`. When adding a new endpoint, add to `router.go` and add an example to `api.http`. If it's a top-9 candidate, add to the QUICKSTART section and update the table in `api.md`.
