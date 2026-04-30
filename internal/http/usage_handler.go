package http

import (
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"llm-usage-tracker/internal/service"
)

type UsageHandler struct {
	service *service.UsageService
}

func NewUsageHandler(service *service.UsageService) *UsageHandler {
	return &UsageHandler{service: service}
}

func (h *UsageHandler) AddUsage(w http.ResponseWriter, r *http.Request) {
	idStr := r.PathValue("id")
	projectID, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		respondError(w, r, http.StatusBadRequest, "invalid id", err)
		return
	}

	var req struct {
		Model     string `json:"model"`
		TokensIn  int64  `json:"tokens_in"`
		TokensOut int64  `json:"tokens_out"`
		LatencyMs *int64 `json:"latency_ms"`
		Tag       string `json:"tag"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, r, http.StatusBadRequest, "Invalid request body", err)
		return
	}

	usage, err := h.service.AddUsage(r.Context(), projectID, req.Model, req.TokensIn, req.TokensOut, req.LatencyMs, req.Tag)
	if err != nil {
		writeError(w, r, err)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(usage)
}

// GET /projects/{id}/usage/daily?date=2026-04-21
func (h *UsageHandler) GetDailyStats(w http.ResponseWriter, r *http.Request) {
	projectID, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		respondError(w, r, http.StatusBadRequest, "invalid id", err)
		return
	}

	dateStr := r.URL.Query().Get("date")
	if dateStr == "" {
		dateStr = time.Now().UTC().Format("2006-01-02")
	}

	date, err := time.Parse("2006-01-02", dateStr)
	if err != nil {
		respondError(w, r, http.StatusBadRequest, "invalid date, expected YYYY-MM-DD", err)
		return
	}

	stats, err := h.service.GetDailyStats(r.Context(), projectID, date)
	if err != nil {
		writeError(w, r, err)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(stats)
}

// GET /projects/{id}/usage/monthly?month=2026-04
func (h *UsageHandler) GetMonthlyStats(w http.ResponseWriter, r *http.Request) {
	projectID, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		respondError(w, r, http.StatusBadRequest, "invalid id", err)
		return
	}

	monthStr := r.URL.Query().Get("month")
	if monthStr == "" {
		monthStr = time.Now().UTC().Format("2006-01")
	}

	month, err := time.Parse("2006-01", monthStr)
	if err != nil {
		respondError(w, r, http.StatusBadRequest, "invalid month, expected YYYY-MM", err)
		return
	}

	stats, err := h.service.GetMonthlyStats(r.Context(), projectID, month)
	if err != nil {
		writeError(w, r, err)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(stats)
}

// GET /projects/{id}/usage/recent-latency?limit=5
// Returns avg latency over the latest N events, ignoring null latency values.
func (h *UsageHandler) GetRecentLatencyStats(w http.ResponseWriter, r *http.Request) {
	projectID, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		respondError(w, r, http.StatusBadRequest, "invalid id", err)
		return
	}

	limit := 0
	if s := r.URL.Query().Get("limit"); s != "" {
		n, err := strconv.Atoi(s)
		if err != nil || n <= 0 {
			respondError(w, r, http.StatusBadRequest, "invalid limit, expected positive integer", err)
			return
		}
		limit = n
	}

	stats, err := h.service.GetRecentLatencyStats(r.Context(), projectID, limit)
	if err != nil {
		writeError(w, r, err)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(stats)
}

// parseTimeRange pulls required ?from=&to= params supporting either YYYY-MM-DD
// (day-level, expanded to start/end of day UTC) or RFC3339 (exact timestamp).
// On any error it writes a 400 to the response and returns ok=false.
func parseTimeRange(w http.ResponseWriter, r *http.Request) (from, to time.Time, ok bool) {
	fromStr := r.URL.Query().Get("from")
	toStr := r.URL.Query().Get("to")
	if fromStr == "" || toStr == "" {
		respondError(w, r, http.StatusBadRequest, "from and to are required (YYYY-MM-DD or RFC3339)", nil)
		return
	}

	var err error
	from, err = parseFlexTime(fromStr, false)
	if err != nil {
		respondError(w, r, http.StatusBadRequest, "invalid from, expected YYYY-MM-DD or RFC3339", err)
		return
	}
	to, err = parseFlexTime(toStr, true)
	if err != nil {
		respondError(w, r, http.StatusBadRequest, "invalid to, expected YYYY-MM-DD or RFC3339", err)
		return
	}
	if from.After(to) {
		respondError(w, r, http.StatusBadRequest, "from must be on or before to", nil)
		return
	}
	ok = true
	return
}

// parseFlexTime accepts RFC3339 (e.g. "2026-04-21T10:00:00Z") or YYYY-MM-DD.
// For date-only input, endOfDay=true expands to 23:59:59 UTC; otherwise 00:00:00 UTC.
func parseFlexTime(s string, endOfDay bool) (time.Time, error) {
	if t, err := time.Parse(time.RFC3339, s); err == nil {
		return t.UTC(), nil
	}
	t, err := time.Parse("2006-01-02", s)
	if err != nil {
		return time.Time{}, err
	}
	if endOfDay {
		t = t.Add(24*time.Hour - time.Second)
	}
	return t.UTC(), nil
}

// GET /projects/{id}/usage/range?from=2026-04-01&to=2026-04-21
func (h *UsageHandler) GetProjectRangeStats(w http.ResponseWriter, r *http.Request) {
	projectID, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		respondError(w, r, http.StatusBadRequest, "invalid id", err)
		return
	}

	from, to, ok := parseTimeRange(w, r)
	if !ok {
		return
	}

	stats, err := h.service.GetProjectRangeStats(r.Context(), projectID, from, to)
	if err != nil {
		writeError(w, r, err)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(stats)
}

// parseOptionalTimeRange reads ?from= and ?to= if present. Unlike parseTimeRange,
// missing params are legal — the corresponding pointer is returned as nil.
// On malformed input it writes a 400 and returns ok=false.
func parseOptionalTimeRange(w http.ResponseWriter, r *http.Request) (from, to *time.Time, ok bool) {
	if s := r.URL.Query().Get("from"); s != "" {
		t, err := parseFlexTime(s, false)
		if err != nil {
			respondError(w, r, http.StatusBadRequest, "invalid from, expected YYYY-MM-DD or RFC3339", err)
			return
		}
		from = &t
	}
	if s := r.URL.Query().Get("to"); s != "" {
		t, err := parseFlexTime(s, true)
		if err != nil {
			respondError(w, r, http.StatusBadRequest, "invalid to, expected YYYY-MM-DD or RFC3339", err)
			return
		}
		to = &t
	}
	if from != nil && to != nil && from.After(*to) {
		respondError(w, r, http.StatusBadRequest, "from must be on or before to", nil)
		return
	}
	ok = true
	return
}

// listEvents handles both /projects/{id}/usage/events (scoped) and /usage/events (all-projects).
// projectID is nil when called from the all-projects route.
func (h *UsageHandler) listEvents(w http.ResponseWriter, r *http.Request, projectID *int64) {
	from, to, ok := parseOptionalTimeRange(w, r)
	if !ok {
		return
	}

	limit := 0
	if s := r.URL.Query().Get("limit"); s != "" {
		n, err := strconv.Atoi(s)
		if err != nil || n <= 0 {
			respondError(w, r, http.StatusBadRequest, "invalid limit, expected positive integer", err)
			return
		}
		limit = n
	}

	cursor := r.URL.Query().Get("cursor")

	page, err := h.service.ListEvents(r.Context(), projectID, from, to, cursor, limit)
	if err != nil {
		writeError(w, r, err)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(page)
}

// GET /projects/{id}/usage/events?from=&to=&limit=&cursor=
func (h *UsageHandler) ListProjectEvents(w http.ResponseWriter, r *http.Request) {
	projectID, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		respondError(w, r, http.StatusBadRequest, "invalid id", err)
		return
	}
	h.listEvents(w, r, &projectID)
}

// GET /usage/events?from=&to=&limit=&cursor=
func (h *UsageHandler) ListAllEvents(w http.ResponseWriter, r *http.Request) {
	h.listEvents(w, r, nil)
}

// GET /usage/summary?from=2026-04-01&to=2026-04-21
func (h *UsageHandler) GetUsageSummary(w http.ResponseWriter, r *http.Request) {
	from, to, ok := parseTimeRange(w, r)
	if !ok {
		return
	}

	stats, err := h.service.GetAllProjectsSummary(r.Context(), from, to)
	if err != nil {
		writeError(w, r, err)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(stats)
}
