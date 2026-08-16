package api

import (
	"context"
	"net/http"
	"remotehunter/internal/models"
	"remotehunter/internal/repository"
	"strconv"

	"github.com/gin-gonic/gin"
)

// GET /jobs
func (h *Handler) GetJobs(c *gin.Context) {
	skillsParam := c.DefaultQuery("skills", "")
	daysStr := c.DefaultQuery("days", "30")
	days, _ := strconv.Atoi(daysStr)
	country := c.DefaultQuery("country", "")
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))

	if page < 1 {
		page = 1
	}
	if limit < 1 || limit > 100 {
		limit = 20
	}

	filter := repository.JobFilter{
		Skills:  skillsParam,
		Days:    days,
		Country: country,
		Page:    page,
		Limit:   limit,
	}

	jobs, total, err := h.jobRepo.GetJobs(c.Request.Context(), filter)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"jobs":  jobs,
		"total": total,
		"page":  page,
		"limit": limit,
	})
}

// GET /jobs/:id
func (h *Handler) GetJobByID(c *gin.Context) {
	id := c.Param("id")
	job, err := h.jobRepo.GetJobByID(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if job == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Job not found"})
		return
	}
	c.JSON(http.StatusOK, job)
}

// POST /jobs/:id/analyze or POST /jobs/analyze
func (h *Handler) AnalyzeJob(c *gin.Context) {
	var req struct {
		JobID    string `json:"job_id"`
		ResumeID string `json:"resume_id"`
	}
	_ = c.ShouldBindJSON(&req)

	jobID := c.Param("id")
	if jobID == "" {
		jobID = req.JobID
	}

	if jobID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Job ID is required"})
		return
	}

	ctx := c.Request.Context()

	// Get active resume if not specified
	var resume *models.Resume
	var err error
	if req.ResumeID == "" {
		resume, err = h.resumeRepo.GetActiveResume(ctx)
		if err != nil || resume == nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "No active resume found. Please upload a resume first."})
			return
		}
	} else {
		resume, err = h.resumeRepo.GetActiveResume(ctx)
		if err != nil || resume == nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "Resume not found"})
			return
		}
	}

	job, err := h.jobRepo.GetJobByID(ctx, jobID)
	if err != nil || job == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Job not found"})
		return
	}

	activeModel := h.getActiveModelWithContext(ctx)
	analysis, err := h.aiClient.AnalyzeATSMatch(job, resume, activeModel)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "AI analysis failed: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, analysis)
}

func (h *Handler) getActiveModelWithContext(ctx context.Context) string {
	model, err := h.settingsRepo.GetActiveModel(ctx, h.aiClient.DefaultModel())
	if err != nil || model == "" {
		return h.aiClient.DefaultModel()
	}
	return model
}
