package api

import (
	"net/http"
	"strings"

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
	activeSources, err := h.settingsRepo.GetSetting(ctx, "active_sources")
	if err != nil || activeSources == "" {
		activeSources = "weworkremotely,realworkfromanywhere,googlejobs,builtin,golangprojects,remoterocketship,vacancyglobalpro,bayt,naukri,bdjobs,flexboard,hnhiring,remotive,arbeitnow,ziprecruiter,indeed,linkedin,glassdoor,remoteok"
	}

	sourcesList := strings.Split(activeSources, ",")
	for i, s := range sourcesList {
		sourcesList[i] = strings.TrimSpace(s)
	}

	c.JSON(http.StatusOK, gin.H{
		"sources":          sourcesList,
		"available_source": AllSources,
	})
}

// POST /settings/sources
func (h *Handler) UpdateSources(c *gin.Context) {
	var body struct {
		Sources []string `json:"sources"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body"})
		return
	}

	joined := strings.Join(body.Sources, ",")
	if err := h.settingsRepo.SetSetting(c.Request.Context(), "active_sources", joined); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update sources: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Scraper sources updated",
		"sources": body.Sources,
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
