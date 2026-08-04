package api

import (
	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
)

// SetupRouter configures all routes and middleware
func SetupRouter(h *Handler) *gin.Engine {
	r := gin.Default()

	// CORS - allow frontend origin
	r.Use(cors.New(cors.Config{
		AllowOrigins:     []string{"http://localhost:3000", "http://localhost:5173"},
		AllowMethods:     []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Authorization"},
		ExposeHeaders:    []string{"Content-Length"},
		AllowCredentials: true,
	}))

	v1 := r.Group("/api/v1")
	{
		// Health check
		v1.GET("/health", h.Health)

		// Job endpoints
		jobs := v1.Group("/jobs")
		{
			jobs.GET("", h.GetJobs)
			jobs.GET("/:id", h.GetJobByID)
			jobs.POST("/trigger-scrape", h.TriggerScrape)
			jobs.POST("/:id/analyze", h.AnalyzeJob)
		}

		// Resume endpoints
		res := v1.Group("/resume")
		{
			res.POST("/upload", h.UploadResume)
			res.GET("/active", h.GetActiveResume)
			res.GET("/active/text", h.GetResumeFullText)
			res.GET("/active/pdf", h.GetActiveResumePDF)
			res.PUT("/active", h.UpdateResumeText)
			res.POST("/revert", h.RevertResumeText)
			res.POST("/chat", h.ChatResume)
			res.GET("/versions", h.GetResumeVersions)
			res.POST("/versions", h.SaveResumeVersion)
			res.GET("/versions/:id/text", h.GetVersionText)
		}

		// Settings endpoints
		settings := v1.Group("/settings")
		{
			settings.GET("", h.GetSettings)
			settings.PUT("/sources", h.UpdateSources)
			settings.GET("/ai", h.GetAISettings)
			settings.PUT("/ai", h.UpdateAISettings)
		}

		// AI utility endpoints
		v1.GET("/ai/models", h.GetAIModels)
	}

	return r
}
