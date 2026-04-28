// Package metrics defines and registers all Prometheus metrics for the app.
// Importing this package triggers init-time registration via promauto, so
// no explicit setup is needed at the call sites or in main.go.
package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

const namespace = "llmtracker"

// HTTP layer (RED method).
var (
	HTTPRequestsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: namespace,
			Subsystem: "http",
			Name:      "requests_total",
			Help:      "Total HTTP requests processed, labeled by method, route template, and status code.",
		},
		[]string{"method", "route", "status"},
	)

	HTTPRequestDurationSeconds = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Namespace: namespace,
			Subsystem: "http",
			Name:      "request_duration_seconds",
			Help:      "HTTP request latency by method and route template.",
			Buckets:   prometheus.DefBuckets, // 5ms..10s
		},
		[]string{"method", "route"},
	)

	HTTPRequestsInFlight = promauto.NewGauge(
		prometheus.GaugeOpts{
			Namespace: namespace,
			Subsystem: "http",
			Name:      "requests_in_flight",
			Help:      "Current number of HTTP requests being served.",
		},
	)
)

// Cache layer.
var (
	CacheHitsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: namespace,
			Subsystem: "cache",
			Name:      "hits_total",
			Help:      "Redis cache hits by operation.",
		},
		[]string{"op"},
	)

	CacheMissesTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: namespace,
			Subsystem: "cache",
			Name:      "misses_total",
			Help:      "Redis cache misses by operation.",
		},
		[]string{"op"},
	)

	RedisErrorsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: namespace,
			Subsystem: "redis",
			Name:      "errors_total",
			Help:      "Redis operation errors by operation name.",
		},
		[]string{"op"},
	)
)

// Business layer.
var (
	UsageEventsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: namespace,
			Subsystem: "usage",
			Name:      "events_total",
			Help:      "Total usage events recorded, labeled by project and model.",
		},
		[]string{"project_id", "model"},
	)

	UsageCostCentsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: namespace,
			Subsystem: "usage",
			Name:      "cost_cents_total",
			Help:      "Cumulative cost in cents, labeled by project and model.",
		},
		[]string{"project_id", "model"},
	)

	UsageTokensTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: namespace,
			Subsystem: "usage",
			Name:      "tokens_total",
			Help:      "Cumulative tokens, labeled by project, model, and direction (in|out).",
		},
		[]string{"project_id", "model", "direction"},
	)

	LLMCallDurationSeconds = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Namespace: namespace,
			Subsystem: "llm",
			Name:      "call_duration_seconds",
			Help:      "Client-reported LLM call latency, labeled by model.",
			// 100ms .. ~100s — wider than HTTP defaults because LLM calls are slower.
			Buckets: prometheus.ExponentialBuckets(0.1, 2, 10),
		},
		[]string{"model"},
	)

	BudgetExceededTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: namespace,
			Subsystem: "budget",
			Name:      "exceeded_total",
			Help:      "Times a usage event pushed a project over its budget, by window (daily|monthly).",
		},
		[]string{"project_id", "window"},
	)
)

// Project metadata gauges. Drive the dashboard project dropdown and budget
// threshold panels. Set on project create/update, deleted on project delete,
// re-hydrated from SQL on app startup.
var (
	ProjectInfo = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace: namespace,
			Subsystem: "project",
			Name:      "info",
			Help:      "1 for each known (non-deleted) project; absent otherwise.",
		},
		[]string{"project_id"},
	)

	ProjectBudgetCents = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace: namespace,
			Subsystem: "project",
			Name:      "budget_cents",
			Help:      "Current budget in cents per window (daily|monthly|total). Absent when not set.",
		},
		[]string{"project_id", "window"},
	)
)
