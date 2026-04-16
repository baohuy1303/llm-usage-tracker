package http

import(
	"encoding/json"
	"net/http"
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
	err = h.service.CreateProject(req.Name, req.Budget)
	if err != nil {
		http.Error(w, "Failed to create project", http.StatusInternalServerError)
		return
	}

	// Explicitly set the status code to 201 Created
	// If we don't set like this, it will default to 200 OK
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)

	// Encode the response body, and send it to the client (appear in response when we do the request)
	json.NewEncoder(w).Encode(req)
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