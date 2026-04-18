package http

import (
	"log/slog"
	"net/http"
	"time"
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
