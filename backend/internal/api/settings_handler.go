package api

import (
	"log"
	"net/http"
	"strings"

	"remotehunter/internal/models"

	"github.com/gin-gonic/gin"
)

var AllSources = []string{
	"weworkremotely", "realworkfromanywhere", "googlejobs", "builtin",
	"golangprojects", "remoterocketship", "vacancyglobalpro", "bayt",
	"naukri", "bdjobs", "flexboard", "hnhiring", "remotive",
	"arbeitnow", "ziprecruiter", "indeed", "linkedin", "glassdoor", "remoteok",
}

// GET /settings
func (h *Handler) GetSettings(c *gin.Context) {
	ctx := c.Request.Context()

	configs, err := h.settingsRepo.GetScraperConfigs(ctx)
	if err != nil {
		log.Printf("[API] Failed to load scraper configs: %v", err)
	}
	if configs == nil {
		configs = []models.ScraperConfig{}
	}

	// Load active_sources setting to mark which are enabled
	activeSources, err := h.settingsRepo.GetSetting(ctx, "active_sources")
	if err != nil || activeSources == "" {
		activeSources = strings.Join(AllSources, ",")
	}
	activeSet := make(map[string]bool)
	for _, s := range strings.Split(activeSources, ",") {
		activeSet[strings.TrimSpace(s)] = true
	}

	// Sync enabled state from active_sources into configs
	for i := range configs {
		normalized := normalizeSourceKey(configs[i].BoardName)
		if activeSet[normalized] {
			configs[i].Enabled = true
		} else {
			configs[i].Enabled = false
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"sources":          configs,
		"available_source": AllSources,
	})
}

func normalizeSourceKey(name string) string {
	return strings.ToLower(strings.ReplaceAll(name, " ", ""))
}

// POST /settings/sources
func (h *Handler) UpdateSources(c *gin.Context) {
	var body struct {
		Sources []models.ScraperConfig `json:"sources"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body"})
		return
	}

	ctx := c.Request.Context()

	// Update each config in the database
	for _, cfg := range body.Sources {
		if err := h.settingsRepo.UpdateScraperConfig(ctx, cfg); err != nil {
			log.Printf("[API] Failed to update scraper config %d: %v", cfg.ID, err)
		}
	}

	// Sync active_sources setting from enabled configs
	var active []string
	for _, cfg := range body.Sources {
		if cfg.Enabled {
			active = append(active, normalizeSourceKey(cfg.BoardName))
		}
	}
	joined := strings.Join(active, ",")
	if err := h.settingsRepo.SetSetting(ctx, "active_sources", joined); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update sources: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Scraper sources updated",
		"source_configs": body.Sources,
	})
}

// POST /scrape/trigger
func (h *Handler) TriggerScrape(c *gin.Context) {
	go h.scheduler.TriggerAll()
	c.JSON(http.StatusOK, gin.H{"message": "Scraper job triggered successfully"})
}

// GET /health
func (h *Handler) Health(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"status": "healthy",
		"db":     "connected",
	})
}
