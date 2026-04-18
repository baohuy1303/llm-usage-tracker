package main

import(
	"context"
	"log/slog"
	"net/http"
	"os"
	"llm-usage-tracker/internal/store"
	"llm-usage-tracker/internal/service"
	appredis "llm-usage-tracker/internal/redis"
	apphttp "llm-usage-tracker/internal/http" // Renamed to avoid conflict with stdlib http
)

func main(){
	// Get the database path from the environment variable or use the default
	dbPath := os.Getenv("DATABASE_URL")
	if dbPath == "" {
		// Create the data directory if it doesn't exist
		os.MkdirAll("./data", 0755)
		dbPath = "./data/app.db"
	}

	// Initialize the database
	db, err := store.NewSQLite(dbPath)
	if err != nil {
		slog.Error("Failed to initialize database", "err", err)
		os.Exit(1)
	}
	defer db.Close()

	// Initialize the schema
	err = store.InitSchema(db)
	if err != nil {
		slog.Error("Failed to initialize schema", "err", err)
		os.Exit(1)
	}

	// Initialize Redis
	ctx := context.Background()
	rdb, err := appredis.NewClient(ctx, "localhost:6379")
	if err != nil {
		slog.Error("Failed to initialize Redis", "err", err)
		os.Exit(1)
	}
	defer rdb.Close()
	slog.Info("Redis connected")

	// Create the repository and service
	projectRepo := store.NewProjectRepo(db)
	projectService := service.NewProjectService(projectRepo)

	// Create the handler
	projectHandler := apphttp.NewProjectHandler(projectService)

	// Create the router
	router := apphttp.NewRouter(projectHandler)

	// Start the server
	slog.Info("Server started on :8080")
	if err := http.ListenAndServe(":8080", router); err != nil {
		slog.Error("Failed to start server", "err", err)
		os.Exit(1)
	}
}