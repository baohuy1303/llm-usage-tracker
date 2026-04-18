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

func NewRouter(ph *ProjectHandler) http.Handler {
	mux := http.NewServeMux()
	
	mux.HandleFunc("GET /healthz", healthz)

	mux.HandleFunc("GET /projects", ph.ListProjects)
	mux.HandleFunc("POST /projects/create", ph.CreateProject)
	mux.HandleFunc("GET /projects/{id}", ph.GetProjectByID)
	mux.HandleFunc("PATCH /projects/{id}", ph.UpdateProject)
	mux.HandleFunc("DELETE /projects/{id}", ph.DeleteProject)
	
	return LoggingMiddleware(mux)
}