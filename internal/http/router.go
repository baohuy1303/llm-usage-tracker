package http

import (
	"encoding/json"
	"net/http"
)

// No custom struct needed because we are not holding any state
// We are just passing the handler to the router

func healthz(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}

func NewRouter(ph *ProjectHandler, mh *ModelHandler, uh *UsageHandler) http.Handler {
	mux := http.NewServeMux()

	mux.HandleFunc("GET /healthz", healthz)

	mux.HandleFunc("GET /projects", ph.ListProjects)
	mux.HandleFunc("POST /projects/create", ph.CreateProject)
	mux.HandleFunc("GET /projects/{id}", ph.GetProjectByID)
	mux.HandleFunc("PATCH /projects/{id}", ph.UpdateProject)
	mux.HandleFunc("DELETE /projects/{id}", ph.DeleteProject)

	mux.HandleFunc("POST /models", mh.CreateModel)
	mux.HandleFunc("GET /models", mh.ListModels)
	mux.HandleFunc("GET /models/{id}", mh.GetModelByID)
	mux.HandleFunc("PATCH /models/{id}", mh.UpdateModel)
	mux.HandleFunc("DELETE /models/{id}", mh.DeleteModel)

	mux.HandleFunc("POST /projects/{id}/usage", uh.AddUsage)
	mux.HandleFunc("GET /projects/{id}/usage/daily", uh.GetDailyStats)
	mux.HandleFunc("GET /projects/{id}/usage/monthly", uh.GetMonthlyStats)
	mux.HandleFunc("GET /projects/{id}/usage/range", uh.GetProjectRangeStats)
	mux.HandleFunc("GET /projects/{id}/usage/events", uh.ListProjectEvents)
	mux.HandleFunc("GET /usage/summary", uh.GetUsageSummary)
	mux.HandleFunc("GET /usage/events", uh.ListAllEvents)

	return LoggingMiddleware(mux)
}