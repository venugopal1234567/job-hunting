package api

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"path/filepath"
	"remotehunter/internal/models"
	"remotehunter/internal/pdf"
	"remotehunter/internal/resume"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// POST /resume/upload
func (h *Handler) UploadResume(c *gin.Context) {
	fileHeader, err := c.FormFile("file")
	if err != nil {
		fileHeader, err = c.FormFile("resume")
	}
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "No resume file uploaded"})
		return
	}

	ext := filepath.Ext(fileHeader.Filename)
	if ext != ".pdf" && ext != ".docx" && ext != ".txt" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Unsupported file format. Please upload PDF, DOCX, or TXT."})
		return
	}

	file, err := fileHeader.Open()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to open uploaded file"})
		return
	}
	defer file.Close()

	fileBytes, err := io.ReadAll(file)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to read file content"})
		return
	}

	parsedText, err := resume.ParseText(bytes.NewReader(fileBytes))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to extract text from resume: " + err.Error()})
		return
	}

	extractedSkills := resume.ExtractSkills(parsedText)

	resObj := &models.Resume{
		ID:              uuid.New().String(),
		Filename:        fileHeader.Filename,
		RawText:         parsedText,
		ExtractedSkills: extractedSkills,
		RawTextLength:   len(parsedText),
		UploadedAt:      time.Now(),
		IsActive:        true,
	}
	if ext == ".pdf" {
		resObj.PDFBytes = fileBytes
		resObj.HasPDF = true
	}

	ctx := c.Request.Context()
	activeModel := h.getActiveModelWithContext(ctx)

	// Phase 2: Convert to Structured template via AI immediately
	structRes, _, aiErr := h.aiClient.ConvertResumeToTemplate(parsedText, activeModel, false)
	if aiErr == nil && structRes != nil {
		resObj.InitialStructured = structRes
	}

	if err := h.resumeRepo.SaveResume(ctx, resObj); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to save resume: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message":            "Resume uploaded and parsed successfully",
		"id":                 resObj.ID,
		"filename":           resObj.Filename,
		"skills":             resObj.ExtractedSkills,
		"initial_structured": resObj.InitialStructured,
	})
}

// GET /resume/active
func (h *Handler) GetActiveResume(c *gin.Context) {
	resObj, err := h.resumeRepo.GetActiveResume(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if resObj == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "No active resume uploaded"})
		return
	}
	c.JSON(http.StatusOK, resObj)
}

// GET /resume/active/pdf
func (h *Handler) GetActiveResumePDF(c *gin.Context) {
	ctx := c.Request.Context()
	resObj, err := h.resumeRepo.GetActiveResume(ctx)
	if err != nil || resObj == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "No active resume found"})
		return
	}

	if len(resObj.PDFBytes) > 0 {
		c.Header("Content-Type", "application/pdf")
		c.Header("Content-Disposition", fmt.Sprintf("inline; filename=\"%s\"", resObj.Filename))
		c.Data(http.StatusOK, "application/pdf", resObj.PDFBytes)
		return
	}

	// Fallback to dynamic PDF rendering if no raw PDF bytes stored
	activeModel := h.getActiveModelWithContext(ctx)
	_, htmlContent, err := h.aiClient.ConvertResumeToTemplate(resObj.RawText, activeModel, false)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to convert resume to template: " + err.Error()})
		return
	}

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
	c.Header("Content-Disposition", "inline; filename=\"Active_Resume.pdf\"")
	c.Data(http.StatusOK, "application/pdf", pdfBytes)
}

// GET /resume/text
func (h *Handler) GetResumeFullText(c *gin.Context) {
	ctx := c.Request.Context()
	structured, err := h.resumeRepo.GetResumeContent(ctx)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	rawText, _ := h.resumeRepo.GetResumeFullText(ctx)
	activeResume, _ := h.resumeRepo.GetActiveResume(ctx)

	hasEdits := false
	if activeResume != nil && activeResume.EditedStructured != nil {
		hasEdits = true
	}

	c.JSON(http.StatusOK, gin.H{
		"structured":   structured,
		"has_raw_text": rawText != "",
		"has_edits":    hasEdits,
	})
}

// PUT /resume/text
func (h *Handler) UpdateResumeText(c *gin.Context) {
	var body models.UpdateResumeRequest
	if err := c.ShouldBindJSON(&body); err != nil || body.Structured == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body or empty structured resume"})
		return
	}

	if err := h.resumeRepo.UpdateResumeStructured(c.Request.Context(), body.Structured); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Resume updated successfully", "structured": body.Structured})
}

// POST /resume/revert
func (h *Handler) RevertResumeText(c *gin.Context) {
	revertedStruct, err := h.resumeRepo.RevertResume(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message":    "Reverted to previous version",
		"structured": revertedStruct,
	})
}

// POST /resume/active/analyze
func (h *Handler) AnalyzeResume(c *gin.Context) {
	ctx := c.Request.Context()
	rawText, err := h.resumeRepo.GetResumeFullText(ctx)
	if err != nil || rawText == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "No active resume raw text found to analyze"})
		return
	}

	activeModel := h.getActiveModelWithContext(ctx)
	structRes, _, err := h.aiClient.ConvertResumeToTemplate(rawText, activeModel, false)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "AI structuring failed: " + err.Error()})
		return
	}

	if err := h.resumeRepo.SetInitialStructured(ctx, structRes); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to save initial structured: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"structured": structRes})
}

// GET /resume/versions
func (h *Handler) GetResumeVersions(c *gin.Context) {
	ctx := c.Request.Context()
	resumeID, err := h.resumeRepo.GetActiveResumeID(ctx)
	if err != nil || resumeID == "" {
		c.JSON(http.StatusNotFound, gin.H{"error": "No active resume found"})
		return
	}

	versions, err := h.resumeRepo.GetVersions(ctx, resumeID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"versions": versions})
}

// POST /resume/versions
func (h *Handler) SaveResumeVersion(c *gin.Context) {
	ctx := c.Request.Context()
	resumeID, err := h.resumeRepo.GetActiveResumeID(ctx)
	if err != nil || resumeID == "" {
		c.JSON(http.StatusNotFound, gin.H{"error": "No active resume found"})
		return
	}

	var req struct {
		Label      string                  `json:"label"`
		Structured *models.StructuredResume `json:"snapshot_structured"`
	}
	if err := c.ShouldBindJSON(&req); err != nil || req.Structured == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Snapshot structured resume is required"})
		return
	}
	if req.Label == "" {
		req.Label = "Manual Version " + time.Now().Format("Jan 02 15:04")
	}

	version := &models.ResumeVersion{
		ID:                 uuid.New().String(),
		ResumeID:           resumeID,
		Label:              req.Label,
		SnapshotText:       "", // keep text empty since snapshot_structured is populated
		SnapshotStructured: req.Structured,
		AppliedAt:          time.Now(),
	}

	if err := h.resumeRepo.SaveVersion(ctx, version); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Version saved", "version": version})
}

// GET /resume/versions/:id/text
func (h *Handler) GetVersionText(c *gin.Context) {
	versionID := c.Param("id")
	version, err := h.resumeRepo.GetVersionByID(c.Request.Context(), versionID)
	if err != nil || version == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Version not found"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"structured": version.SnapshotStructured})
}
