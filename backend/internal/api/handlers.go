package api

import (
	"database/sql"
	"encoding/json"
	"fmt"
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

	// Filter by country
	if country != "" {
		conditions = append(conditions, fmt.Sprintf("(j.country ILIKE $%d OR j.location ILIKE $%d)", argIdx, argIdx+1))
		args = append(args, "%"+country+"%", "%"+country+"%")
		argIdx += 2
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
			return
		}
	}

	// Fallback: compute simple skill match from requested skills
	if len(requestedSkills) > 0 {
		matched := []string{}
		missing := []string{}
		descLower := strings.ToLower(job.Description + " " + job.Title)
		for _, sk := range requestedSkills {
			if strings.Contains(descLower, strings.ToLower(sk)) {
				matched = append(matched, sk)
			} else {
				missing = append(missing, sk)
			}
		}
		job.MatchedSkills = matched
		job.MissingSkills = missing
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
		INSERT INTO resumes (filename, raw_text, extracted_skills, is_active)
		VALUES ($1, $2, $3, TRUE)
		RETURNING id, uploaded_at`,
		filename, rawText, string(skillsJSON),
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

	err := h.db.QueryRow(`
		SELECT id, filename, raw_text, extracted_skills, uploaded_at
		FROM resumes WHERE is_active = TRUE ORDER BY uploaded_at DESC LIMIT 1`).
		Scan(&r.ID, &r.Filename, &r.RawText, &skillsJSON, &r.UploadedAt)

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
	r.RawText = "" // Don't return full text in list response
	c.JSON(http.StatusOK, r)
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

	// Check for cached analysis
	var cached models.ATSAnalysis
	var mbJSON, suggestJSON, gapJSON []byte
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

	// Run AI analysis
	analysis, err := h.aiClient.AnalyzeATSMatch(&job, &res)
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
