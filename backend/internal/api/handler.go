package api

import (
	"remotehunter/internal/ai"
	"remotehunter/internal/repository"
	"remotehunter/internal/scraper"
)

// Handler holds injected repositories & dependencies for HTTP endpoints
type Handler struct {
	jobRepo      repository.JobRepository
	resumeRepo   repository.ResumeRepository
	settingsRepo repository.SettingsRepository
	aiClient     *ai.Client
	scheduler    *scraper.Scheduler
}

// NewHandler creates a new API Handler instance with repository injection
func NewHandler(
	jobRepo repository.JobRepository,
	resumeRepo repository.ResumeRepository,
	settingsRepo repository.SettingsRepository,
	aiClient *ai.Client,
	scheduler *scraper.Scheduler,
) *Handler {
	return &Handler{
		jobRepo:      jobRepo,
		resumeRepo:   resumeRepo,
		settingsRepo: settingsRepo,
		aiClient:     aiClient,
		scheduler:    scheduler,
	}
}
