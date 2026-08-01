package main

import (
	"log"
	"remotehunter/internal/ai"
	"remotehunter/internal/api"
	"remotehunter/internal/config"
	"remotehunter/internal/db"
	"remotehunter/internal/scraper"
)

func main() {
	// Load configuration from environment variables
	cfg := config.Load()

	log.Printf("[Main] Starting RemoteHunter API server on :%s", cfg.Server.Port)
	log.Printf("[Main] Database: %s@%s:%s/%s", cfg.Database.User, cfg.Database.Host, cfg.Database.Port, cfg.Database.Name)
	log.Printf("[Main] Ollama: %s (model: %s)", cfg.Ollama.Host, cfg.Ollama.Model)

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

	// Initialize AI client
	aiClient := ai.NewClient(cfg.Ollama.Host, cfg.Ollama.Model)

	// Initialize and start background scheduler
	scheduler := scraper.NewScheduler(database)
	scheduler.Start()
	defer scheduler.Stop()

	// Setup HTTP router and start server
	router := api.SetupRouter(api.NewHandler(database, aiClient, scheduler))

	log.Printf("[Main] Server listening on 0.0.0.0:%s", cfg.Server.Port)
	if err := router.Run(":" + cfg.Server.Port); err != nil {
		log.Fatalf("[Main] Server failed: %v", err)
	}
}
