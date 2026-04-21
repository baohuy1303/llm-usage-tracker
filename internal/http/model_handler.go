package http

import (
	"encoding/json"
	"net/http"
	"strconv"

	"llm-usage-tracker/internal/service"
)

type ModelHandler struct {
	service *service.ModelService
}

func NewModelHandler(service *service.ModelService) *ModelHandler {
	return &ModelHandler{service: service}
}

func (h *ModelHandler) CreateModel(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Name                  string `json:"name"`
		InputPerMillionCents  int64  `json:"input_per_million_cents"`
		OutputPerMillionCents int64  `json:"output_per_million_cents"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, r, http.StatusBadRequest, "Invalid request body", err)
		return
	}

	model, err := h.service.CreateModel(r.Context(), req.Name, req.InputPerMillionCents, req.OutputPerMillionCents)
	if err != nil {
		writeError(w, r, err)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(model)
}

func (h *ModelHandler) ListModels(w http.ResponseWriter, r *http.Request) {
	models, err := h.service.ListModels(r.Context())
	if err != nil {
		respondError(w, r, http.StatusInternalServerError, "Failed to list models", err)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(models)
}

func (h *ModelHandler) GetModelByID(w http.ResponseWriter, r *http.Request) {
	idStr := r.PathValue("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		respondError(w, r, http.StatusBadRequest, "invalid id", err)
		return
	}

	model, err := h.service.GetModelByID(r.Context(), id)
	if err != nil {
		writeError(w, r, err)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(model)
}

func (h *ModelHandler) UpdateModel(w http.ResponseWriter, r *http.Request) {
	idStr := r.PathValue("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		respondError(w, r, http.StatusBadRequest, "invalid id", err)
		return
	}

	var req struct {
		Name                  *string `json:"name"`
		InputPerMillionCents  *int64  `json:"input_per_million_cents"`
		OutputPerMillionCents *int64  `json:"output_per_million_cents"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, r, http.StatusBadRequest, "Invalid request body", err)
		return
	}

	model, err := h.service.UpdateModel(r.Context(), id, req.Name, req.InputPerMillionCents, req.OutputPerMillionCents)
	if err != nil {
		writeError(w, r, err)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(model)
}

func (h *ModelHandler) DeleteModel(w http.ResponseWriter, r *http.Request) {
	idStr := r.PathValue("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		respondError(w, r, http.StatusBadRequest, "invalid id", err)
		return
	}

	err = h.service.DeleteModel(r.Context(), id)
	if err != nil {
		writeError(w, r, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
