package api

import (
	"fmt"
	"net/http"
	"remotehunter/internal/models"
	"remotehunter/internal/pdf"

	"github.com/gin-gonic/gin"
)

// POST /chat
func (h *Handler) ChatResume(c *gin.Context) {
	var req models.ChatRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body"})
		return
	}

	ctx := c.Request.Context()
	activeModel := h.getActiveModelWithContext(ctx)

	jobContext := ""
	if req.CustomJD != "" {
		jobContext = fmt.Sprintf("Custom Job Description / Context:\n%s", req.CustomJD)
	} else if req.JobID != "" {
		job, err := h.jobRepo.GetJobByID(ctx, req.JobID)
		if err == nil && job != nil {
			jobContext = fmt.Sprintf("Title: %s\nCompany: %s\nDescription:\n%s", job.Title, job.Company, job.Description)
		}
	}

	chatResp, err := h.aiClient.ChatWithResume(&req, jobContext, activeModel)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "AI chat failed: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, chatResp)
}

// GET /ai/models
func (h *Handler) GetAIModels(c *gin.Context) {
	models, err := h.aiClient.ListModels()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"models": models})
}

// GET /ai/settings
func (h *Handler) GetAISettings(c *gin.Context) {
	ctx := c.Request.Context()
	activeModel := h.getActiveModelWithContext(ctx)
	provider := "nvidia"
	c.JSON(http.StatusOK, gin.H{
		"active_model": activeModel,
		"provider":     provider,
	})
}

// POST /ai/settings
func (h *Handler) UpdateAISettings(c *gin.Context) {
	var body struct {
		ActiveModel string `json:"active_model"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body"})
		return
	}

	if err := h.settingsRepo.SetSetting(c.Request.Context(), "active_model", body.ActiveModel); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update active model: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message":      "AI settings updated successfully",
		"active_model": body.ActiveModel,
	})
}

// POST /resume/convert-template
func (h *Handler) ConvertResumeTemplate(c *gin.Context) {
	var req struct {
		Text          string `json:"text"`
		Model         string `json:"model"`
		Format        string `json:"format"`
		FitSinglePage bool   `json:"fit_single_page"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body"})
		return
	}

	ctx := c.Request.Context()
	rawText := req.Text
	if rawText == "" {
		var err error
		rawText, err = h.resumeRepo.GetResumeFullText(ctx)
		if err != nil || rawText == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "No resume text provided and no active resume found"})
			return
		}
	}

	modelToUse := req.Model
	if modelToUse == "" {
		modelToUse = h.getActiveModelWithContext(ctx)
	}

	structRes, htmlContent, err := h.aiClient.ConvertResumeToTemplate(rawText, modelToUse, req.FitSinglePage)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to convert resume: " + err.Error()})
		return
	}

	// Auditing by Recruiter AI
	var recruiterAudit *models.RecruiterValidationResult
	valRes, auditErr := h.aiClient.ValidateResumeWithRecruiter(rawText, structRes, modelToUse)
	if auditErr == nil {
		recruiterAudit = valRes
	}

	if req.Format == "pdf" {
		pdfService, pdfErr := pdf.New("/tmp")
		if pdfErr != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "PDF service initialization failed: " + pdfErr.Error()})
			return
		}
		pdfBytes, err := pdfService.GenerateFromHTML(ctx, htmlContent)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate PDF: " + err.Error()})
			return
		}
		c.Header("Content-Type", "application/pdf")
		c.Header("Content-Disposition", "attachment; filename=\"Tailored_Resume.pdf\"")
		c.Data(http.StatusOK, "application/pdf", pdfBytes)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"structured_resume": structRes,
		"html":              htmlContent,
		"recruiter_audit":   recruiterAudit,
	})
}
