package main

import (
	"log"
	"net/http"
	"time"

	"remotehunter/internal/ai"
	"remotehunter/internal/api"
	"remotehunter/internal/config"
	"remotehunter/internal/db"
	"remotehunter/internal/repository/postgres"
	"remotehunter/internal/scraper"
)

func main() {
	// Load configuration from environment variables
	cfg := config.Load()

	log.Printf("[Main] Starting RemoteHunter API server on :%s", cfg.Server.Port)
	log.Printf("[Main] Database: %s@%s:%s/%s", cfg.Database.User, cfg.Database.Host, cfg.Database.Port, cfg.Database.Name)
	log.Printf("[Main] AI LLM: %s (model: %s)", cfg.AI.BaseURL, cfg.AI.Model)

	if cfg.AI.APIKey == "" || cfg.AI.Model == "" {
		log.Fatalf("[Main] Missing required AI config: set AI_API_KEY and AI_MODEL in .env")
	}

	// Connect to PostgreSQL
	database, err := db.Connect(cfg.Database.DSN())
	if err != nil {
		log.Fatalf("[Main] Database connection failed: %v", err)
	}
	defer database.Close()

	// Run migrations
	if err := db.Migrate(database); err != nil {
		log.Fatalf("[Main] Migration failed: %v", err)
	}

	// Initialize Repositories
	jobRepo := postgres.NewJobRepo(database)
	resumeRepo := postgres.NewResumeRepo(database)
	settingsRepo := postgres.NewSettingsRepo(database)

	// Initialize AI client
	aiClient := ai.NewClient(cfg.AI.APIKey, cfg.AI.BaseURL, cfg.AI.Model)

	// Initialize and start background scheduler
	scheduler := scraper.NewScheduler(database)
	scheduler.Start()
	defer scheduler.Stop()

	// Initialize API Handler with repository injection
	handler := api.NewHandler(jobRepo, resumeRepo, settingsRepo, aiClient, scheduler)

	// Setup HTTP router and start server with extended timeouts (10 mins for AI requests)
	router := api.SetupRouter(handler)

	server := &http.Server{
		Addr:         ":" + cfg.Server.Port,
		Handler:      router,
		ReadTimeout:  10 * time.Minute,
		WriteTimeout: 10 * time.Minute,
		IdleTimeout:  10 * time.Minute,
	}

	log.Printf("[Main] Server listening on 0.0.0.0:%s", cfg.Server.Port)
	if err := server.ListenAndServe(); err != nil {
		log.Fatalf("[Main] Server failed: %v", err)
	}
}
