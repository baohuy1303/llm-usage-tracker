package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"

	"github.com/joho/godotenv"
	apphttp "llm-usage-tracker/internal/http"
	"llm-usage-tracker/internal/cache"
	appredis "llm-usage-tracker/internal/redis"
	"llm-usage-tracker/internal/service"
	"llm-usage-tracker/internal/store"
)

func main() {
	// .env is optional — in production, variables come from the environment directly.
	godotenv.Load()

	logLevel := slog.LevelInfo
	if os.Getenv("LOG_LEVEL") == "debug" {
		logLevel = slog.LevelDebug
	}
	slog.SetDefault(slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: logLevel})))
	dbPath := os.Getenv("DATABASE_URL")
	if dbPath == "" {
		os.MkdirAll("./data", 0755)
		dbPath = "./data/app.db"
	}

	db, err := store.NewSQLite(dbPath)
	if err != nil {
		slog.Error("Failed to initialize database", "err", err)
		os.Exit(1)
	}
	defer db.Close()

	if err := store.InitSchema(db); err != nil {
		slog.Error("Failed to initialize schema", "err", err)
		os.Exit(1)
	}

	ctx := context.Background()

	redisAddr := os.Getenv("REDIS_ADDR")
	if redisAddr == "" {
		redisAddr = "localhost:6379"
	}

	var usageCache *cache.UsageCache
	rdb, err := appredis.NewClient(ctx, redisAddr)
	if err != nil {
		slog.Warn("Redis unavailable, running without cache", "err", err)
	} else {
		defer rdb.Close()
		slog.Info("Redis connected")
		usageCache = cache.NewUsageCache(rdb)
	}

	projectRepo := store.NewProjectRepo(db)
	projectService := service.NewProjectService(projectRepo)
	projectHandler := apphttp.NewProjectHandler(projectService)

	modelRepo := store.NewModelRepo(db)
	modelService := service.NewModelService(modelRepo)
	modelHandler := apphttp.NewModelHandler(modelService)

	usageRepo := store.NewUsageRepo(db)
	usageService := service.NewUsageService(usageRepo, projectRepo, modelRepo, usageCache)
	usageHandler := apphttp.NewUsageHandler(usageService)

	router := apphttp.NewRouter(projectHandler, modelHandler, usageHandler)

	slog.Info("Server started on :8080")
	if err := http.ListenAndServe(":8080", router); err != nil {
		slog.Error("Failed to start server", "err", err)
		os.Exit(1)
	}
}
