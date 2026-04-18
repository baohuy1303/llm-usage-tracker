package http

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"strconv"
	"errors"
	"llm-usage-tracker/internal/service"
)

type ProjectHandler struct {
	service *service.ProjectService
}

func NewProjectHandler(service *service.ProjectService) *ProjectHandler {
	return &ProjectHandler{service: service}
}

func respondError(w http.ResponseWriter, r *http.Request, status int, message string, err error) {
	if err != nil {
		slog.Error("request error", "err", err, "path", r.URL.Path, "method", r.Method)
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(map[string]string{"error": message})
}

func writeError(w http.ResponseWriter, r *http.Request, err error) {
    switch {
    case errors.Is(err, service.ErrDuplicateName):
        respondError(w, r, http.StatusConflict, "Duplicate name", err)

    case errors.Is(err, service.ErrInvalidName):
        respondError(w, r, http.StatusBadRequest, "Name cannot be empty", err)

    case errors.Is(err, service.ErrInvalidBudget):
        respondError(w, r, http.StatusBadRequest, "Budget must be positive", err)

	case errors.Is(err, service.ErrNotFound):
		respondError(w, r, http.StatusNotFound, "Project not found", err)

    default:
        respondError(w, r, http.StatusInternalServerError, "Internal server error", err)
    }
}

func (h *ProjectHandler) CreateProject(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Name string `json:"name"`
		Budget int `json:"budget"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, r, http.StatusBadRequest, "Invalid request body", err)
		return
	}

	project, err := h.service.CreateProject(r.Context(), req.Name, req.Budget)
	if err != nil {
		writeError(w, r, err)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(project)
}

func (h *ProjectHandler) ListProjects(w http.ResponseWriter, r *http.Request) {
	projects, err := h.service.ListProjects()
	if err != nil {
		respondError(w, r, http.StatusInternalServerError, "Failed to list projects", err)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(projects)
}

func (h *ProjectHandler) GetProjectByID(w http.ResponseWriter, r *http.Request) {
	idStr := r.PathValue("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		respondError(w, r, http.StatusBadRequest, "invalid id", err)
		return
	}

	project, err := h.service.GetProjectByID(r.Context(), id)
	if err != nil {
		writeError(w, r, err)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(project)
}

func (h *ProjectHandler) UpdateProject(w http.ResponseWriter, r *http.Request) {
	idStr := r.PathValue("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		respondError(w, r, http.StatusBadRequest, "invalid id", err)
		return
	}

	var req struct {
		Name *string `json:"name"`
		Budget *int `json:"budget"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, r, http.StatusBadRequest, "Invalid request body", err)
		return
	}

	project, err := h.service.UpdateProject(r.Context(), id, req.Name, req.Budget)
	if err != nil {
		writeError(w, r, err)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(project)
}

func (h *ProjectHandler) DeleteProject(w http.ResponseWriter, r *http.Request) {
	idStr := r.PathValue("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		respondError(w, r, http.StatusBadRequest, "invalid id", err)
		return
	}

	err = h.service.DeleteProject(r.Context(), id)
	if err != nil {
		writeError(w, r, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}