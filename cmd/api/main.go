package main

import(
	"log"
	"net/http"
	"os"
	"llm-usage-tracker/internal/store"
	"llm-usage-tracker/internal/service"
	apphttp "llm-usage-tracker/internal/http" // Renamed to avoid conflict with stdlib http
)

func main(){
	// Get the database path from the environment variable or use the default
	dbPath := os.Getenv("DATABASE_URL")
	if dbPath == "" {
		dbPath = "./data/app.db"
	}

	// Initialize the database
	db, err := store.NewSQLite(dbPath)
	if err != nil {
		log.Fatalf("Failed to initialize database: %v", err)
	}
	defer db.Close()

	// Initialize the schema
	err = store.InitSchema(db)
	if err != nil {
		log.Fatalf("Failed to initialize schema: %v", err)
	}

	// Create the repository and service
	projectRepo := store.NewProjectRepo(db)
	projectService := service.NewProjectService(projectRepo)

	// Create the handler
	projectHandler := apphttp.NewProjectHandler(projectService)

	// Create the router
	router := apphttp.NewRouter(projectHandler)

	// Start the server
	log.Println("Server started on :8080")
	if err := http.ListenAndServe(":8080", router); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}