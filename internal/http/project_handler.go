package http

import (
	"encoding/json"
	"net/http"
	"strconv"

	"llm-usage-tracker/internal/service"
	"llm-usage-tracker/internal/store"
)

type ProjectHandler struct {
	service *service.ProjectService
}

func NewProjectHandler(service *service.ProjectService) *ProjectHandler {
	return &ProjectHandler{service: service}
}

// dollarsToMillicentsPtr converts an optional dollar float to optional millicent
// int for the service layer. Nil in -> nil out (= "no budget set").
func dollarsToMillicentsPtr(d *float64) *int64 {
	if d == nil {
		return nil
	}
	v := store.DollarsToMillicents(*d)
	return &v
}

func (h *ProjectHandler) CreateProject(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Name                 string   `json:"name"`
		DailyBudgetDollars   *float64 `json:"daily_budget_dollars"`
		MonthlyBudgetDollars *float64 `json:"monthly_budget_dollars"`
		TotalBudgetDollars   *float64 `json:"total_budget_dollars"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, r, http.StatusBadRequest, "Invalid request body", err)
		return
	}

	project, err := h.service.CreateProject(r.Context(), req.Name,
		dollarsToMillicentsPtr(req.DailyBudgetDollars),
		dollarsToMillicentsPtr(req.MonthlyBudgetDollars),
		dollarsToMillicentsPtr(req.TotalBudgetDollars),
	)
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
		Name                 *string  `json:"name"`
		DailyBudgetDollars   *float64 `json:"daily_budget_dollars"`
		MonthlyBudgetDollars *float64 `json:"monthly_budget_dollars"`
		TotalBudgetDollars   *float64 `json:"total_budget_dollars"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, r, http.StatusBadRequest, "Invalid request body", err)
		return
	}

	project, err := h.service.UpdateProject(r.Context(), id, req.Name,
		dollarsToMillicentsPtr(req.DailyBudgetDollars),
		dollarsToMillicentsPtr(req.MonthlyBudgetDollars),
		dollarsToMillicentsPtr(req.TotalBudgetDollars),
	)
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
