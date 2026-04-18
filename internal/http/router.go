package http

import (
	"net/http"
)

// No custom struct needed because we are not holding any state
// We are just passing the handler to the router

func NewRouter(ph *ProjectHandler) *http.ServeMux {
	mux := http.NewServeMux()
	mux.HandleFunc("/projects", ph.ListProjects)
	mux.HandleFunc("/projects/create", ph.CreateProject)
	mux.HandleFunc("/projects/{id}", ph.GetProjectByID)
	return mux
}