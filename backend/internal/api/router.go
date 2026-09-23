package api

import (
	"crypto/subtle"
	"log"
	"net/http"
	"strings"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
)

// authMiddleware requires a bearer token when authToken is non-empty. An empty
// token disables the check so local development keeps working; that case is
// logged once at startup by SetupRouter.
func authMiddleware(authToken string) gin.HandlerFunc {
	return func(c *gin.Context) {
		if authToken == "" {
			c.Next()
			return
		}

		// Require the literal "Bearer " scheme prefix (RFC 6750) — a bare token
		// with no scheme must not authenticate.
		const bearerPrefix = "Bearer "
		header := c.GetHeader("Authorization")
		if !strings.HasPrefix(header, bearerPrefix) {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
			return
		}
		provided := strings.TrimPrefix(header, bearerPrefix)

		// Constant-time compare so the token cannot be guessed byte by byte.
		if subtle.ConstantTimeCompare([]byte(provided), []byte(authToken)) != 1 {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
			return
		}
		c.Next()
	}
}

// SetupRouter configures all routes and middleware
func SetupRouter(h *Handler, authToken string) *gin.Engine {
	r := gin.Default()

	if authToken == "" {
		log.Println("[API] WARNING: API_AUTH_TOKEN is not set — API endpoints are unauthenticated. Set it before exposing this server beyond localhost.")
	}

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
		// Health check stays unauthenticated so orchestrators/load balancers can probe it
		v1.GET("/health", h.Health)

		// Everything else requires the bearer token when one is configured
		authed := v1.Group("", authMiddleware(authToken))

		// Job endpoints
		jobs := authed.Group("/jobs")
		{
			jobs.GET("", h.GetJobs)
			jobs.GET("/:id", h.GetJobByID)
			jobs.POST("/trigger-scrape", h.TriggerScrape)
			jobs.POST("/:id/analyze", h.AnalyzeJob)
		}

		// Resume endpoints
		res := authed.Group("/resume")
		{
			res.POST("/upload", h.UploadResume)
			res.GET("/active", h.GetActiveResume)
			res.GET("/active/text", h.GetResumeFullText)
			res.GET("/active/pdf", h.GetActiveResumePDF)
			res.PUT("/active", h.UpdateResumeText)
			res.POST("/revert", h.RevertResumeText)
			res.POST("/active/analyze", h.AnalyzeResume)
			res.POST("/chat", h.ChatResume)
			res.GET("/versions", h.GetResumeVersions)
			res.POST("/versions", h.SaveResumeVersion)
			res.GET("/versions/:id/text", h.GetVersionText)
			res.POST("/convert-template", h.ConvertResumeTemplate)
		}

		// Settings endpoints
		settings := authed.Group("/settings")
		{
			settings.GET("", h.GetSettings)
			settings.PUT("/sources", h.UpdateSources)
			settings.GET("/ai", h.GetAISettings)
			settings.PUT("/ai", h.UpdateAISettings)
		}

		// AI utility endpoints
		authed.GET("/ai/models", h.GetAIModels)
	}

	return r
}
