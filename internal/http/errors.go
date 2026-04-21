package http

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"

	"llm-usage-tracker/internal/service"
)

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
		respondError(w, r, http.StatusConflict, err.Error(), err)
	case errors.Is(err, service.ErrInvalidName):
		respondError(w, r, http.StatusBadRequest, err.Error(), err)
	case errors.Is(err, service.ErrInvalidBudget):
		respondError(w, r, http.StatusBadRequest, err.Error(), err)
	case errors.Is(err, service.ErrInvalidPricing):
		respondError(w, r, http.StatusBadRequest, err.Error(), err)
	case errors.Is(err, service.ErrNotFound):
		respondError(w, r, http.StatusNotFound, err.Error(), err)
	case errors.Is(err, service.ErrModelNotFound):
		respondError(w, r, http.StatusNotFound, err.Error(), err)
	default:
		respondError(w, r, http.StatusInternalServerError, "Internal server error", err)
	}
}
