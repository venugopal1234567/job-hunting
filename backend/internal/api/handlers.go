package api

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"remotehunter/internal/ai"
	"remotehunter/internal/models"
	"remotehunter/internal/resume"
	"remotehunter/internal/scraper"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// Handler holds dependencies for all route handlers
type Handler struct {
	db        *sql.DB
	aiClient  *ai.Client
	scheduler *scraper.Scheduler
}

// NewHandler creates a new handler with injected dependencies
func NewHandler(db *sql.DB, aiClient *ai.Client, scheduler *scraper.Scheduler) *Handler {
	return &Handler{db: db, aiClient: aiClient, scheduler: scheduler}
}

// getActiveModel reads the current AI model from app_settings, falling back to the client default
func (h *Handler) getActiveModel() string {
	var value string
	err := h.db.QueryRow(`SELECT value FROM app_settings WHERE key = 'active_model'`).Scan(&value)
	if err != nil || value == "" {
		return h.aiClient.DefaultModel()
	}
	return value
}

// ─────────────────────────────────────────────────────
// GET /jobs
// ─────────────────────────────────────────────────────

func (h *Handler) GetJobs(c *gin.Context) {
	// Parse query params (default to 30 days window)
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
	if days < 1 {
		days = 30
	}

	// If no skills param provided, fetch active resume skills for candidate relevance
	requestedSkills := []string{}
	if skillsParam != "" {
		for _, s := range strings.Split(skillsParam, ",") {
			s = strings.TrimSpace(s)
			if s != "" {
				requestedSkills = append(requestedSkills, s)
			}
		}
	} else {
		var skillsJSON []byte
		_ = h.db.QueryRow(`SELECT extracted_skills FROM resumes WHERE is_active = TRUE ORDER BY uploaded_at DESC LIMIT 1`).Scan(&skillsJSON)
		if len(skillsJSON) > 0 {
			_ = json.Unmarshal(skillsJSON, &requestedSkills)
		}
	}

	// Build dynamic SQL query
	args := []interface{}{}
	conditions := []string{
		"j.is_active = TRUE",
		`LOWER(j.source_board) IN (
			SELECT CASE 
				WHEN LOWER(board_name) LIKE '%builtin%' THEN 'builtin'
				ELSE LOWER(REGEXP_REPLACE(board_name, '\s+', '', 'g'))
			END
			FROM scraper_configs
			WHERE enabled = TRUE
		)`,
	}
	argIdx := 1

	// Filter by date cutoff (strict for short windows like 24h/3d/7d)
	cutoff := time.Now().AddDate(0, 0, -days)
	if days <= 7 {
		conditions = append(conditions, fmt.Sprintf("j.posted_at >= $%d", argIdx))
		args = append(args, cutoff)
		argIdx++
	} else {
		conditions = append(conditions, fmt.Sprintf("(j.posted_at >= $%d OR (j.posted_at IS NULL AND j.scraped_at >= $%d))", argIdx, argIdx+1))
		args = append(args, cutoff, cutoff)
		argIdx += 2
	}

	// Filter by multiple locations
	if country != "" {
		locParts := strings.Split(country, ",")
		var locConds []string
		for _, part := range locParts {
			part = strings.TrimSpace(part)
			if part != "" {
				locConds = append(locConds, fmt.Sprintf("(j.country ILIKE $%d OR j.location ILIKE $%d)", argIdx, argIdx+1))
				args = append(args, "%"+part+"%", "%"+part+"%")
				argIdx += 2
			}
		}
		if len(locConds) > 0 {
			conditions = append(conditions, "("+strings.Join(locConds, " OR ")+")")
		}
	}

	// Filter by target skills (if requested or inferred from active resume)
	if len(requestedSkills) > 0 {
		skillConds := []string{}
		for _, sk := range requestedSkills {
			skLower := strings.ToLower(sk)
			if skLower == "go" || skLower == "golang" {
				skillConds = append(skillConds,
					fmt.Sprintf("(j.title ILIKE $%d OR j.description ILIKE $%d OR j.title ILIKE $%d OR j.description ILIKE $%d)",
						argIdx, argIdx+1, argIdx+2, argIdx+3),
				)
				args = append(args, "%Go%", "%Go%", "%Golang%", "%Golang%")
				argIdx += 4
			} else {
				skillConds = append(skillConds, fmt.Sprintf("(j.title ILIKE $%d OR j.description ILIKE $%d)", argIdx, argIdx+1))
				args = append(args, "%"+sk+"%", "%"+sk+"%")
				argIdx += 2
			}
		}
		conditions = append(conditions, "("+strings.Join(skillConds, " OR ")+")")
	}

	where := "WHERE " + strings.Join(conditions, " AND ")

	// Count total
	var total int
	countQuery := "SELECT COUNT(*) FROM jobs j " + where
	if err := h.db.QueryRow(countQuery, args...).Scan(&total); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to count jobs"})
		return
	}

	// Fetch paginated results
	offset := (page - 1) * limit
	jobArgs := append(args, limit, offset)
	jobQuery := fmt.Sprintf(`
		SELECT j.id, j.title, j.company, j.location, j.country, j.source_url,
		       j.source_board, j.description, j.salary_range, j.posted_at, j.scraped_at
		FROM jobs j
		%s
		ORDER BY j.scraped_at DESC
		LIMIT $%d OFFSET $%d`, where, argIdx, argIdx+1)

	rows, err := h.db.Query(jobQuery, jobArgs...)
	if err != nil {
		log.Printf("[Handler] GetJobs query error: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to query jobs"})
		return
	}
	defer rows.Close()

	jobs := []models.Job{}
	for rows.Next() {
		var j models.Job
		var postedAt sql.NullTime
		if err := rows.Scan(
			&j.ID, &j.Title, &j.Company, &j.Location, &j.Country,
			&j.SourceURL, &j.SourceBoard, &j.Description, &j.SalaryRange,
			&postedAt, &j.ScrapedAt,
		); err != nil {
			continue
		}
		if postedAt.Valid {
			j.PostedAt = &postedAt.Time
		}

		// Enrich with ATS data if active resume exists
		enrichJobWithATS(h.db, &j, requestedSkills)
		jobs = append(jobs, j)
	}

	c.JSON(http.StatusOK, models.JobsResponse{
		Total: total,
		Page:  page,
		Limit: limit,
		Jobs:  jobs,
	})
}

// enrichJobWithATS adds matched/missing skills and cached ATS score to a job
func enrichJobWithATS(db *sql.DB, job *models.Job, requestedSkills []string) {
	// Check cached ATS analysis
	var analysisID string
	var score int
	var matchBreakdownJSON []byte
	err := db.QueryRow(`
		SELECT a.id, a.ats_score, a.match_breakdown
		FROM ats_analyses a
		JOIN resumes r ON r.id = a.resume_id
		WHERE a.job_id = $1 AND r.is_active = TRUE
		ORDER BY a.analyzed_at DESC LIMIT 1`, job.ID).
		Scan(&analysisID, &score, &matchBreakdownJSON)

	if err == nil && len(matchBreakdownJSON) > 0 {
		var mb models.MatchBreakdown
		if json.Unmarshal(matchBreakdownJSON, &mb) == nil {
			job.MatchedSkills = mb.MatchedSkills
			job.MissingSkills = mb.MissingSkills
			job.ATSScore = &score
		}
	}
}

// ─────────────────────────────────────────────────────
// GET /jobs/:id
// ─────────────────────────────────────────────────────

func (h *Handler) GetJobByID(c *gin.Context) {
	id := c.Param("id")

	var job models.Job
	var postedAt sql.NullTime
	err := h.db.QueryRow(`
		SELECT id, title, company, location, country, source_url, source_board,
		       description, salary_range, job_type, posted_at, scraped_at, is_active
		FROM jobs WHERE id = $1`, id).
		Scan(&job.ID, &job.Title, &job.Company, &job.Location, &job.Country,
			&job.SourceURL, &job.SourceBoard, &job.Description, &job.SalaryRange,
			&job.JobType, &postedAt, &job.ScrapedAt, &job.IsActive)

	if err == sql.ErrNoRows {
		c.JSON(http.StatusNotFound, gin.H{"error": "job not found"})
		return
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "db error"})
		return
	}
	if postedAt.Valid {
		job.PostedAt = &postedAt.Time
	}

	enrichJobWithATS(h.db, &job, nil)
	c.JSON(http.StatusOK, job)
}

// ─────────────────────────────────────────────────────
// POST /jobs/trigger-scrape
// ─────────────────────────────────────────────────────

func (h *Handler) TriggerScrape(c *gin.Context) {
	go h.scheduler.TriggerAll()
	c.JSON(http.StatusOK, gin.H{"message": "scrape triggered for all enabled sources"})
}

// ─────────────────────────────────────────────────────
// POST /resume/upload
// ─────────────────────────────────────────────────────

func (h *Handler) UploadResume(c *gin.Context) {
	file, header, err := c.Request.FormFile("file")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "file field required"})
		return
	}
	defer file.Close()

	// Check extension
	filename := header.Filename
	if !strings.HasSuffix(strings.ToLower(filename), ".pdf") &&
		!strings.HasSuffix(strings.ToLower(filename), ".txt") {
		c.JSON(http.StatusBadRequest, gin.H{"error": "only .pdf or .txt files accepted"})
		return
	}

	// Read raw bytes of the file for database storage
	var pdfData []byte
	if strings.HasSuffix(strings.ToLower(filename), ".pdf") {
		var buf bytes.Buffer
		if _, err := io.Copy(&buf, file); err == nil {
			pdfData = buf.Bytes()
		}
		// Reset file seek pointer so parsing can read it from the beginning
		if seeker, ok := file.(io.ReadSeeker); ok {
			_, _ = seeker.Seek(0, io.SeekStart)
		}
	}

	// Parse text content
	rawText, err := resume.ParseText(file)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to parse resume"})
		return
	}

	// Extract skills
	skills := resume.ExtractSkills(rawText)
	skillsJSON, _ := json.Marshal(skills)

	// Deactivate previous resumes
	h.db.Exec(`UPDATE resumes SET is_active = FALSE`)

	// Insert new resume
	var id string
	var uploadedAt time.Time
	err = h.db.QueryRow(`
		INSERT INTO resumes (filename, raw_text, extracted_skills, is_active, pdf_data)
		VALUES ($1, $2, $3, TRUE, $4)
		RETURNING id, uploaded_at`,
		filename, rawText, string(skillsJSON), pdfData,
	).Scan(&id, &uploadedAt)

	if err != nil {
		log.Printf("[Handler] UploadResume insert error: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to save resume"})
		return
	}

	c.JSON(http.StatusOK, models.Resume{
		ID:              id,
		Filename:        filename,
		ExtractedSkills: skills,
		RawTextLength:   len(rawText),
		UploadedAt:      uploadedAt,
		IsActive:        true,
	})
}

// ─────────────────────────────────────────────────────
// GET /resume/active
// ─────────────────────────────────────────────────────

func (h *Handler) GetActiveResume(c *gin.Context) {
	var r models.Resume
	var skillsJSON []byte
	var pdfData []byte

	err := h.db.QueryRow(`
		SELECT id, filename, raw_text, extracted_skills, uploaded_at, pdf_data
		FROM resumes WHERE is_active = TRUE ORDER BY uploaded_at DESC LIMIT 1`).
		Scan(&r.ID, &r.Filename, &r.RawText, &skillsJSON, &r.UploadedAt, &pdfData)

	if err == sql.ErrNoRows {
		c.JSON(http.StatusNotFound, gin.H{"error": "no active resume"})
		return
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "db error"})
		return
	}

	json.Unmarshal(skillsJSON, &r.ExtractedSkills)
	r.RawTextLength = len(r.RawText)
	r.IsActive = true
	r.HasPDF = len(pdfData) > 0
	r.RawText = "" // Don't return full text in list response
	c.JSON(http.StatusOK, r)
}

// GetActiveResumePDF returns the original uploaded PDF file bytes
func (h *Handler) GetActiveResumePDF(c *gin.Context) {
	var pdfData []byte
	err := h.db.QueryRow(`
		SELECT pdf_data FROM resumes 
		WHERE is_active = TRUE AND pdf_data IS NOT NULL 
		ORDER BY uploaded_at DESC LIMIT 1`).Scan(&pdfData)

	if err != nil {
		if err == sql.ErrNoRows {
			c.JSON(http.StatusNotFound, gin.H{"error": "no active PDF file found"})
		} else {
			log.Printf("[Handler] GetActiveResumePDF error: %v", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch PDF"})
		}
		return
	}

	c.Header("Content-Type", "application/pdf")
	c.Header("Content-Disposition", "inline; filename=resume.pdf")
	c.Header("Content-Length", fmt.Sprintf("%d", len(pdfData)))
	c.Writer.Write(pdfData)
}

// ─────────────────────────────────────────────────────
// POST /jobs/:id/analyze
// ─────────────────────────────────────────────────────

func (h *Handler) AnalyzeJob(c *gin.Context) {
	jobID := c.Param("id")

	var reqBody struct {
		ResumeID string `json:"resume_id"`
	}
	c.ShouldBindJSON(&reqBody)

	// Fetch job
	var job models.Job
	var postedAt sql.NullTime
	err := h.db.QueryRow(`
		SELECT id, title, company, description, source_board, source_url, location, country, salary_range
		FROM jobs WHERE id = $1`, jobID).
		Scan(&job.ID, &job.Title, &job.Company, &job.Description,
			&job.SourceBoard, &job.SourceURL, &job.Location, &job.Country, &job.SalaryRange)
	if err == sql.ErrNoRows {
		c.JSON(http.StatusNotFound, gin.H{"error": "job not found"})
		return
	}
	_ = postedAt

	// Fetch resume
	resumeQuery := `SELECT id, filename, raw_text, extracted_skills, uploaded_at FROM resumes WHERE is_active = TRUE ORDER BY uploaded_at DESC LIMIT 1`
	if reqBody.ResumeID != "" {
		resumeQuery = `SELECT id, filename, raw_text, extracted_skills, uploaded_at FROM resumes WHERE id = '` + reqBody.ResumeID + `'`
	}

	var res models.Resume
	var skillsJSON []byte
	err = h.db.QueryRow(resumeQuery).Scan(&res.ID, &res.Filename, &res.RawText, &skillsJSON, &res.UploadedAt)
	if err == sql.ErrNoRows {
		c.JSON(http.StatusBadRequest, gin.H{"error": "no active resume. Please upload a resume first."})
		return
	}
	json.Unmarshal(skillsJSON, &res.ExtractedSkills)

	var mbJSON, suggestJSON, gapJSON []byte

	// Check for cached analysis unless forced
	if c.Query("force") != "true" {
		var cached models.ATSAnalysis
		cacheErr := h.db.QueryRow(`
			SELECT id, ats_score, match_breakdown, actionable_suggestions, gap_questions, analyzed_at
			FROM ats_analyses WHERE job_id = $1 AND resume_id = $2`,
			jobID, res.ID).
			Scan(&cached.ID, &cached.ATSScore, &mbJSON, &suggestJSON, &gapJSON, &cached.AnalyzedAt)

		if cacheErr == nil {
			json.Unmarshal(mbJSON, &cached.MatchBreakdown)
			json.Unmarshal(suggestJSON, &cached.ActionableSuggestions)
			json.Unmarshal(gapJSON, &cached.GapQuestions)
			cached.JobID = jobID
			cached.ResumeID = res.ID
			c.JSON(http.StatusOK, cached)
			return
		}
	}

	// Run AI analysis using the persisted active model
	activeModel := h.getActiveModel()
	analysis, err := h.aiClient.AnalyzeATSMatch(&job, &res, activeModel)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "analysis failed"})
		return
	}
	analysis.JobID = jobID
	analysis.ResumeID = res.ID

	// Persist analysis
	mbJSON, _ = json.Marshal(analysis.MatchBreakdown)
	suggestJSON, _ = json.Marshal(analysis.ActionableSuggestions)
	gapJSON, _ = json.Marshal(analysis.GapQuestions)

	var analysisID string
	err = h.db.QueryRow(`
		INSERT INTO ats_analyses (job_id, resume_id, ats_score, match_breakdown, actionable_suggestions, gap_questions)
		VALUES ($1, $2, $3, $4, $5, $6)
		ON CONFLICT (job_id, resume_id) DO UPDATE
		SET ats_score = EXCLUDED.ats_score,
		    match_breakdown = EXCLUDED.match_breakdown,
		    actionable_suggestions = EXCLUDED.actionable_suggestions,
		    gap_questions = EXCLUDED.gap_questions,
		    analyzed_at = CURRENT_TIMESTAMP
		RETURNING id`,
		jobID, res.ID, analysis.ATSScore, string(mbJSON), string(suggestJSON), string(gapJSON),
	).Scan(&analysisID)

	if err != nil {
		log.Printf("[Handler] AnalyzeJob insert error: %v", err)
	} else {
		analysis.ID = analysisID
	}

	c.JSON(http.StatusOK, analysis)
}

// ─────────────────────────────────────────────────────
// GET /settings
// ─────────────────────────────────────────────────────

func (h *Handler) GetSettings(c *gin.Context) {
	rows, err := h.db.Query(`SELECT id, board_name, target_url, enabled, cron_schedule, last_run_at FROM scraper_configs ORDER BY id`)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "db error"})
		return
	}
	defer rows.Close()

	var sources []models.ScraperConfig
	for rows.Next() {
		var sc models.ScraperConfig
		var lastRunAt sql.NullTime
		rows.Scan(&sc.ID, &sc.BoardName, &sc.TargetURL, &sc.Enabled, &sc.CronSchedule, &lastRunAt)
		if lastRunAt.Valid {
			sc.LastRunAt = &lastRunAt.Time
		}
		sources = append(sources, sc)
	}

	c.JSON(http.StatusOK, models.Settings{
		Sources:        sources,
		DefaultSkills:  []string{"Go", "PostgreSQL", "Docker", "Kubernetes", "AWS"},
		ScrapeInterval: "@every 1h",
	})
}

// ─────────────────────────────────────────────────────
// PUT /settings/sources
// ─────────────────────────────────────────────────────

func (h *Handler) UpdateSources(c *gin.Context) {
	var sources []models.ScraperConfig
	if err := c.ShouldBindJSON(&sources); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
		return
	}

	for _, src := range sources {
		_, err := h.db.Exec(`
			INSERT INTO scraper_configs (board_name, target_url, enabled, cron_schedule)
			VALUES ($1, $2, $3, $4)
			ON CONFLICT (board_name) DO UPDATE
			SET target_url = EXCLUDED.target_url,
			    enabled = EXCLUDED.enabled,
			    cron_schedule = EXCLUDED.cron_schedule`,
			src.BoardName, src.TargetURL, src.Enabled, src.CronSchedule,
		)
		if err != nil {
			log.Printf("[Handler] UpdateSources error for '%s': %v", src.BoardName, err)
		}
	}

	c.JSON(http.StatusOK, gin.H{"message": fmt.Sprintf("updated %d sources", len(sources))})
}

// ─────────────────────────────────────────────────────
// GET /health
// ─────────────────────────────────────────────────────

func (h *Handler) Health(c *gin.Context) {
	if err := h.db.Ping(); err != nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"status": "unhealthy", "db": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"status":    "healthy",
		"timestamp": time.Now().UTC(),
		"version":   "1.0.0",
	})
}

// generateUUID is a simple UUID generator helper
func generateUUID() string {
	return uuid.New().String()
}

// ─────────────────────────────────────────────────────
// GET /resume/active/text  — full raw text for editor
// ─────────────────────────────────────────────────────

func (h *Handler) GetResumeFullText(c *gin.Context) {
	var id, rawText string
	var editedText sql.NullString

	err := h.db.QueryRow(`
		SELECT id, raw_text, edited_text
		FROM resumes WHERE is_active = TRUE ORDER BY uploaded_at DESC LIMIT 1`).
		Scan(&id, &rawText, &editedText)

	if err == sql.ErrNoRows {
		c.JSON(http.StatusNotFound, gin.H{"error": "no active resume"})
		return
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "db error"})
		return
	}

	// Return edited_text if present, else raw_text
	text := rawText
	if editedText.Valid && editedText.String != "" {
		text = editedText.String
	}

	c.JSON(http.StatusOK, gin.H{"id": id, "text": text, "has_edits": editedText.Valid && editedText.String != ""})
}

// ─────────────────────────────────────────────────────
// PUT /resume/active  — save edited text + re-extract skills
// ─────────────────────────────────────────────────────

func (h *Handler) UpdateResumeText(c *gin.Context) {
	var body struct {
		Text string `json:"text"`
	}
	if err := c.ShouldBindJSON(&body); err != nil || body.Text == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "text field required"})
		return
	}

	// Re-extract skills from new text
	skills := resume.ExtractSkills(body.Text)
	skillsJSON, _ := json.Marshal(skills)

	var id string
	err := h.db.QueryRow(`
		UPDATE resumes SET edited_text = $1, extracted_skills = $2
		WHERE is_active = TRUE
		RETURNING id`, body.Text, string(skillsJSON)).
		Scan(&id)

	if err != nil {
		log.Printf("[Handler] UpdateResumeText error: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to save"})
		return
	}

	// Invalidate cached ATS analyses for this resume so they re-run
	h.db.Exec(`DELETE FROM ats_analyses WHERE resume_id = $1`, id)

	c.JSON(http.StatusOK, gin.H{"id": id, "skills": skills, "message": "saved"})
}

// ─────────────────────────────────────────────────────
// POST /resume/revert  — revert edited text to original
// ─────────────────────────────────────────────────────

func (h *Handler) RevertResumeText(c *gin.Context) {
	var id, rawText string
	err := h.db.QueryRow(`
		SELECT id, raw_text FROM resumes WHERE is_active = TRUE ORDER BY uploaded_at DESC LIMIT 1
	`).Scan(&id, &rawText)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "no active resume"})
		return
	}

	skills := resume.ExtractSkills(rawText)
	skillsJSON, _ := json.Marshal(skills)

	_, err = h.db.Exec(`
		UPDATE resumes SET edited_text = NULL, extracted_skills = $1
		WHERE id = $2`, string(skillsJSON), id)
	if err != nil {
		log.Printf("[Handler] RevertResumeText error: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to revert"})
		return
	}

	// Invalidate cached ATS analyses for this resume so they re-run
	h.db.Exec(`DELETE FROM ats_analyses WHERE resume_id = $1`, id)

	c.JSON(http.StatusOK, gin.H{"id": id, "text": rawText, "skills": skills, "message": "reverted"})
}


// ─────────────────────────────────────────────────────
// POST /resume/chat  — AI resume editor chat
// ─────────────────────────────────────────────────────

func (h *Handler) ChatResume(c *gin.Context) {
	var req models.ChatRequest
	if err := c.ShouldBindJSON(&req); err != nil || req.Message == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "message field required"})
		return
	}

	// Fetch job context if job_id provided
	jobContext := ""
	if req.JobID != "" {
		var title, company, description string
		err := h.db.QueryRow(`SELECT title, company, description FROM jobs WHERE id = $1`, req.JobID).
			Scan(&title, &company, &description)
		if err == nil {
			jobContext = fmt.Sprintf("Job Title: %s\nCompany: %s\n\n%s", title, company, description)
		}
	}

	// Resolve model: per-request override > DB active model > client default
	modelOverride := req.Model
	if modelOverride == "" {
		modelOverride = h.getActiveModel()
	}
	response, err := h.aiClient.ChatWithResume(&req, jobContext, modelOverride)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "chat failed"})
		return
	}

	c.JSON(http.StatusOK, response)
}

// ─────────────────────────────────────────────────────
// GET /resume/versions  — list all applied snapshots
// ─────────────────────────────────────────────────────

func (h *Handler) GetResumeVersions(c *gin.Context) {
	rows, err := h.db.Query(`
		SELECT rv.id, rv.resume_id, rv.job_id, j.title, j.company,
		       rv.label, rv.applied_at, rv.source
		FROM resume_versions rv
		LEFT JOIN jobs j ON j.id = rv.job_id
		ORDER BY rv.applied_at DESC
		LIMIT 50`)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "db error"})
		return
	}
	defer rows.Close()

	versions := []models.ResumeVersion{}
	for rows.Next() {
		var v models.ResumeVersion
		var jobID, jobTitle, jobCompany sql.NullString
		if err := rows.Scan(&v.ID, &v.ResumeID, &jobID, &jobTitle, &jobCompany, &v.Label, &v.AppliedAt, &v.Source); err != nil {
			continue
		}
		if jobID.Valid {
			v.JobID = jobID.String
		}
		if jobTitle.Valid {
			v.JobTitle = jobTitle.String
		}
		if jobCompany.Valid {
			v.JobCompany = jobCompany.String
		}
		versions = append(versions, v)
	}

	c.JSON(http.StatusOK, gin.H{"versions": versions})
}

// ─────────────────────────────────────────────────────
// POST /resume/versions  — save an applied snapshot
// ─────────────────────────────────────────────────────

func (h *Handler) SaveResumeVersion(c *gin.Context) {
	var body struct {
		SnapshotText string `json:"snapshot_text"`
		JobID        string `json:"job_id"`
		Label        string `json:"label"`
		Source       string `json:"source"` // "editor" | "upload"
	}
	if err := c.ShouldBindJSON(&body); err != nil || body.SnapshotText == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "snapshot_text required"})
		return
	}

	// Get active resume id
	var resumeID string
	err := h.db.QueryRow(`SELECT id FROM resumes WHERE is_active = TRUE ORDER BY uploaded_at DESC LIMIT 1`).
		Scan(&resumeID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "no active resume"})
		return
	}

	if body.Source == "" {
		body.Source = "editor"
	}

	var versionID string
	var jobIDArg interface{} = nil
	if body.JobID != "" {
		jobIDArg = body.JobID
	}

	err = h.db.QueryRow(`
		INSERT INTO resume_versions (resume_id, job_id, snapshot_text, label, source)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id`,
		resumeID, jobIDArg, body.SnapshotText, body.Label, body.Source).
		Scan(&versionID)

	if err != nil {
		log.Printf("[Handler] SaveResumeVersion error: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to save version"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"id": versionID, "message": "version saved"})
}

// ─────────────────────────────────────────────────────
// GET /resume/versions/:id/text — fetch snapshot text
// ─────────────────────────────────────────────────────

func (h *Handler) GetVersionText(c *gin.Context) {
	versionID := c.Param("id")
	var text string
	err := h.db.QueryRow(`SELECT snapshot_text FROM resume_versions WHERE id = $1`, versionID).Scan(&text)
	if err == sql.ErrNoRows {
		c.JSON(http.StatusNotFound, gin.H{"error": "version not found"})
		return
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "db error"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"text": text})
}

// ─────────────────────────────────────────────────────
// GET /ai/models  — list locally available Ollama models
// ─────────────────────────────────────────────────────

func (h *Handler) GetAIModels(c *gin.Context) {
	availableModels, err := h.aiClient.ListModels()
	if err != nil {
		log.Printf("[Handler] GetAIModels error: %v", err)
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "failed to reach Ollama: " + err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"models": availableModels})
}

// ─────────────────────────────────────────────────────
// GET /settings/ai  — get current AI model configuration
// ─────────────────────────────────────────────────────

func (h *Handler) GetAISettings(c *gin.Context) {
	activeModel := h.getActiveModel()

	// Try to list available models (non-fatal if Ollama is down)
	availableModels, _ := h.aiClient.ListModels()

	c.JSON(http.StatusOK, models.AISettings{
		ActiveModel:     activeModel,
		DefaultModel:    h.aiClient.DefaultModel(),
		AvailableModels: availableModels,
	})
}

// ─────────────────────────────────────────────────────
// PUT /settings/ai  — update active AI model
// ─────────────────────────────────────────────────────

func (h *Handler) UpdateAISettings(c *gin.Context) {
	var body struct {
		ActiveModel string `json:"active_model"`
	}
	if err := c.ShouldBindJSON(&body); err != nil || body.ActiveModel == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "active_model field required"})
		return
	}

	_, err := h.db.Exec(`
		INSERT INTO app_settings (key, value, updated_at)
		VALUES ('active_model', $1, CURRENT_TIMESTAMP)
		ON CONFLICT (key) DO UPDATE
		SET value = EXCLUDED.value,
		    updated_at = CURRENT_TIMESTAMP`,
		body.ActiveModel,
	)
	if err != nil {
		log.Printf("[Handler] UpdateAISettings error: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to save setting"})
		return
	}

	log.Printf("[AI] Active model switched to: %s", body.ActiveModel)
	c.JSON(http.StatusOK, gin.H{"active_model": body.ActiveModel, "message": "model updated"})
}

