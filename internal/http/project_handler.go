package http

import (
	"encoding/json"
	"net/http"
	"strconv"

	"llm-usage-tracker/internal/service"
)

type ProjectHandler struct {
	service *service.ProjectService
}

func NewProjectHandler(service *service.ProjectService) *ProjectHandler {
	return &ProjectHandler{service: service}
}

func (h *ProjectHandler) CreateProject(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Name               string `json:"name"`
		DailyBudgetCents   *int64 `json:"daily_budget_cents"`
		MonthlyBudgetCents *int64 `json:"monthly_budget_cents"`
		TotalBudgetCents   *int64 `json:"total_budget_cents"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, r, http.StatusBadRequest, "Invalid request body", err)
		return
	}

	project, err := h.service.CreateProject(r.Context(), req.Name, req.DailyBudgetCents, req.MonthlyBudgetCents, req.TotalBudgetCents)
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
		Name               *string `json:"name"`
		DailyBudgetCents   *int64  `json:"daily_budget_cents"`
		MonthlyBudgetCents *int64  `json:"monthly_budget_cents"`
		TotalBudgetCents   *int64  `json:"total_budget_cents"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, r, http.StatusBadRequest, "Invalid request body", err)
		return
	}

	project, err := h.service.UpdateProject(r.Context(), id, req.Name, req.DailyBudgetCents, req.MonthlyBudgetCents, req.TotalBudgetCents)
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