package http

import (
	"log/slog"
	"net/http"
	"strconv"
	"time"

	"llm-usage-tracker/internal/metrics"
)

// responseWriterRecorder is a custom ResponseWriter that captures the status code.
type responseWriterRecorder struct {
	http.ResponseWriter
	status int
}

func (w *responseWriterRecorder) WriteHeader(status int) {
	w.status = status
	w.ResponseWriter.WriteHeader(status)
}

// MetricsMiddleware records Prometheus HTTP metrics: requests_total,
// request_duration_seconds (histogram), and requests_in_flight (gauge).
//
// The mux is required so we can extract the registered route template
// (e.g. "/projects/{id}/usage") instead of the raw URL path. Using the
// raw path would explode label cardinality.
func MetricsMiddleware(mux *http.ServeMux, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		metrics.HTTPRequestsInFlight.Inc()
		defer metrics.HTTPRequestsInFlight.Dec()

		start := time.Now()
		rec := &responseWriterRecorder{
			ResponseWriter: w,
			status:         http.StatusOK,
		}

		next.ServeHTTP(rec, r)

		// Resolve the route template. Empty pattern means no match (404) —
		// label as "unmatched" to keep cardinality bounded.
		_, pattern := mux.Handler(r)
		if pattern == "" {
			pattern = "unmatched"
		}

		metrics.HTTPRequestsTotal.
			WithLabelValues(r.Method, pattern, strconv.Itoa(rec.status)).
			Inc()
		metrics.HTTPRequestDurationSeconds.
			WithLabelValues(r.Method, pattern).
			Observe(time.Since(start).Seconds())
	})
}

// LoggingMiddleware wraps an http.Handler and logs the request method, path, status, and duration.
func LoggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		
		// Initialize the recorder with 200 OK as default status code
		rec := &responseWriterRecorder{
			ResponseWriter: w,
			status:         http.StatusOK, 
		}

		next.ServeHTTP(rec, r)

		duration := time.Since(start).Milliseconds()

		slog.Info("request",
			"method", r.Method,
			"path", r.URL.Path,
			"status", rec.status,
			"duration_ms", duration,
		)
	})
}
