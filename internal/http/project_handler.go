package http

import(
	"encoding/json"
	"net/http"
	"llm-usage-tracker/internal/service"
	"strconv"
	"database/sql"
	"errors"
)

type ProjectHandler struct {
	service *service.ProjectService
}

func NewProjectHandler(service *service.ProjectService) *ProjectHandler {
	return &ProjectHandler{service: service}
}

func (h *ProjectHandler) CreateProject(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Name string `json:"name"`
		Budget int `json:"budget"`
	}

	// Decode the request body into the req struct
	err := json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Call the service
	project, err := h.service.CreateProject(r.Context(), req.Name, req.Budget)
	if err != nil {
		if errors.Is(err, service.ErrDuplicateName) {
			http.Error(w, "Duplicate name", http.StatusConflict)
			return
		}
		http.Error(w, "Failed to create project", http.StatusInternalServerError)
		return
	}

	// Explicitly set the status code to 201 Created
	// If we don't set like this, it will default to 200 OK
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)

	// Encode the response body, and send it to the client (appear in response when we do the request)
	json.NewEncoder(w).Encode(project)
}

func (h *ProjectHandler) ListProjects(w http.ResponseWriter, r *http.Request) {
	projects, err := h.service.ListProjects()
	if err != nil {
		http.Error(w, "Failed to list projects", http.StatusInternalServerError)
		return
	}

	//No need to set status code to 200 OK because it is the default
	//Same as before
	json.NewEncoder(w).Encode(projects)
}

func (h *ProjectHandler) GetProjectByID(w http.ResponseWriter, r *http.Request) {
	// query := r.URL.Query()
	idStr := r.PathValue("id")

    // idStr := query.Get("id")

	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		http.Error(w, "invalid id", http.StatusBadRequest)
		return
	}

	project, err := h.service.GetProjectByID(r.Context(), id)
	if err != nil {
		if(errors.Is(err, sql.ErrNoRows)) {
			http.Error(w, "Project not found", http.StatusNotFound)
			return
		}
		http.Error(w, "Failed to get project", http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(project)
}