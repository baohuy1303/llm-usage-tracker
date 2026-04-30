# Pulse API

Usage and budget tracker for LLM API calls. Records token counts, costs, and latencies. Aggregates per project, day, month, or arbitrary range. Enforces soft daily/monthly/total budgets.

This doc is a navigation index. Full request/response examples live in [`api.http`](api.http).

## Quick start

```bash
docker compose up --build
```

| Service | URL |
|---------|-----|
| API | http://localhost:8080 |
| Prometheus | http://localhost:9090 |
| Grafana | http://localhost:3000 (admin/admin) |

To use [`api.http`](api.http):

- **VS Code**: install [REST Client](https://marketplace.visualstudio.com/items?itemName=humao.rest-client), click `Send Request` above any `###` block.
- **JetBrains**: open the file and click the `▶` gutter icon.
- **Thunder Client / Bruno**: import as a collection.

## Top 9 endpoints

These cover ~90% of dev-time usage. They live at the top of [`api.http`](api.http) under `QUICKSTART` and are designed to fire top-to-bottom on a fresh DB.

| # | What | Method | Path | Line |
|---|------|--------|------|------|
| 1 | Create Project | `POST` | `/projects/create` | [43](api.http#L43) |
| 2 | List Projects | `GET` | `/projects` | [56](api.http#L56) |
| 3 | Get Project By ID | `GET` | `/projects/{id}` | [59](api.http#L59) |
| 4 | Add Model | `POST` | `/models` | [62](api.http#L62) |
| 5 | List Models | `GET` | `/models` | [73](api.http#L73) |
| 6 | Add Usage Event | `POST` | `/projects/{id}/usage` | [76](api.http#L76) |
| 7 | List Usage Events | `GET` | `/projects/{id}/usage/events` | [88](api.http#L88) |
| 8 | Update Project Budget | `PATCH` | `/projects/{id}` | [92](api.http#L92) |
| 9 | Get Project + budget status | `GET` | `/projects/{id}` | [100](api.http#L100) |

### Example: create a project, log usage, check budget

```bash
# 1. Create a project
curl -X POST http://localhost:8080/projects/create \
  -H "Content-Type: application/json" \
  -d '{"name":"Demo","daily_budget_dollars":5.00}'
# => {"id":1,"name":"Demo","daily_budget_dollars":5.00, ...}

# 2. Log a usage event
curl -X POST http://localhost:8080/projects/1/usage \
  -H "Content-Type: application/json" \
  -d '{"model":"gpt-4o","tokens_in":1500,"tokens_out":500,"latency_ms":350,"tag":"chat"}'
# => {"id":1,"cost_dollars":0.013, ..., "budget_status":{"daily":{"spent_dollars":0.013, "over_budget":false}}}

# 3. Get budget status
curl http://localhost:8080/projects/1
# => {..., "budget_status":{"daily":{"spent_dollars":0.013,"budget_dollars":5.00,"percent":0.26,"over_budget":false}}}
```

## Detailed reference

Past line 105 in [`api.http`](api.http), endpoints are grouped by domain. Each section has happy-path variations followed by an `### --- error cases ---` divider.

| Domain | Line |
|--------|------|
| Projects (CRUD, errors) | [109](api.http#L109) |
| Models (CRUD, errors) | [188](api.http#L188) |
| Usage write (variations, bulk-over-budget) | [232](api.http#L232) |
| Usage stats (daily, monthly) | [324](api.http#L324) |
| Range and summary (arbitrary windows) | [355](api.http#L355) |
| Usage events (pagination, filtering) | [404](api.http#L404) |

## Conventions

| Field | Format |
|-------|--------|
| Money | Dollars as float. `daily_budget_dollars: 5.00`, `cost_dollars: 0.00350`. Stored internally as millicents (1 cent = 1000 millicents). |
| Tokens | Integers. |
| Dates (query params) | `YYYY-MM-DD` or RFC3339. Date-only `from` expands to start-of-day, `to` expands to end-of-day. |
| Pagination | Cursor-based on `/usage/events`. Pass `next_cursor` from the previous response as `?cursor=`. `has_more: false` means you're at the end. |
| Delete | Soft delete. `DELETE /projects/{id}` sets `deleted_at` but preserves usage events for historical stats. |
| Budget enforcement | Warn-only. `POST /usage` always succeeds. Response carries `budget_status` with `over_budget: true/false` per window so the client can decide whether to throttle. |

## Status codes

| Code | Meaning |
|------|---------|
| `200` | OK (read) |
| `201` | Created (write) |
| `204` | No content (delete) |
| `400` | Bad request (validation, malformed body, bad query param) |
| `404` | Not found (project, model) |
| `409` | Conflict (duplicate name) |
| `500` | Internal error (DB, Redis when not degraded) |

Error response shape:

```json
{ "error": "duplicate name" }
```

## Other endpoints

| Path | Purpose |
|------|---------|
| `/healthz` | Liveness check. Returns `{"status":"ok"}`. |
| `/metrics` | Prometheus scrape. `curl http://localhost:8080/metrics \| grep llmtracker_` to see business metrics. |

## Maintenance

When endpoint behavior changes:

1. Update [`api.http`](api.http) (both QUICKSTART and the detailed section if applicable).
2. Update line numbers in the tables above if entries shifted.

If line numbers are stale, the section titles also work for `Ctrl+F`.
