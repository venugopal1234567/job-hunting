package models

import "time"

// Job represents a remote job posting
type Job struct {
	ID          string     `json:"id"`
	JobHash     string     `json:"-"`
	Title       string     `json:"title"`
	Company     string     `json:"company"`
	Location    string     `json:"location"`
	Country     string     `json:"country"`
	SourceURL   string     `json:"source_url"`
	SourceBoard string     `json:"source_board"`
	Description string     `json:"description"`
	SalaryRange string     `json:"salary_range"`
	JobType     string     `json:"job_type"`
	PostedAt    *time.Time `json:"posted_at"`
	ScrapedAt   time.Time  `json:"scraped_at"`
	IsActive    bool       `json:"is_active"`

	// Computed from ATS analysis
	MatchedSkills []string `json:"matched_skills,omitempty"`
	MissingSkills []string `json:"missing_skills,omitempty"`
	ATSScore      *int     `json:"ats_score,omitempty"`
}

// JobsResponse is the paginated response for GET /jobs
type JobsResponse struct {
	Total int   `json:"total"`
	Page  int   `json:"page"`
	Limit int   `json:"limit"`
	Jobs  []Job `json:"jobs"`
}

// Resume represents an uploaded candidate resume
type Resume struct {
	ID                string            `json:"id"`
	Filename          string            `json:"filename"`
	RawText           string            `json:"raw_text,omitempty"`
	ExtractedSkills   []string          `json:"extracted_skills"`
	RawTextLength     int               `json:"raw_text_length"`
	UploadedAt        time.Time         `json:"uploaded_at"`
	IsActive          bool              `json:"is_active"`
	HasPDF            bool              `json:"has_pdf"`
	PDFBytes          []byte            `json:"-"`
	InitialStructured *StructuredResume `json:"initial_structured,omitempty"`
	EditedStructured  *StructuredResume `json:"edited_structured,omitempty"`
}

// GapQuestion represents an interview preparation question for a missing skill
type GapQuestion struct {
	Skill    string `json:"skill"`
	Question string `json:"question"`
}

// MatchBreakdown contains skill matching details
type MatchBreakdown struct {
	MatchedSkills []string `json:"matched_skills"`
	MissingSkills []string `json:"missing_skills"`
}

// ATSAnalysis is the result of an AI-powered ATS report
type ATSAnalysis struct {
	ID                   string         `json:"id"`
	JobID                string         `json:"job_id"`
	ResumeID             string         `json:"resume_id,omitempty"`
	ATSScore             int            `json:"ats_score"`
	MatchBreakdown       MatchBreakdown `json:"match_breakdown"`
	ActionableSuggestions []string      `json:"actionable_suggestions"`
	GapQuestions         []GapQuestion  `json:"gap_questions"`
	AnalyzedAt           time.Time      `json:"analyzed_at"`
}

// ScraperConfig represents a configured job board scraper target
type ScraperConfig struct {
	ID           int        `json:"id"`
	BoardName    string     `json:"board_name"`
	TargetURL    string     `json:"target_url"`
	Enabled      bool       `json:"enabled"`
	CronSchedule string     `json:"cron_schedule"`
	LastRunAt    *time.Time `json:"last_run_at"`
}

// Settings holds current application settings
type Settings struct {
	Sources        []ScraperConfig `json:"sources"`
	DefaultSkills  []string        `json:"default_skills"`
	ScrapeInterval string          `json:"scrape_interval"`
}

// JobFilterParams defines query parameters for GET /jobs
type JobFilterParams struct {
	Skills  []string
	Days    int
	Country string
	Page    int
	Limit   int
}

// ProposedEdit is an AI-suggested change to a specific section of the resume
type ProposedEdit struct {
	ID          string `json:"id"`
	Original    string `json:"original"`
	Replacement string `json:"replacement"`
	Reason      string `json:"reason"`
}

// GapQuestionPrompt is an interactive AI question about a missing skill
type GapQuestionPrompt struct {
	Skill    string `json:"skill"`
	Question string `json:"question"`
}

// ChatRequest is the request payload for POST /resume/chat
type ChatRequest struct {
	Message          string            `json:"message"`
	ResumeText       string            `json:"resume_text,omitempty"`
	ResumeStructured *StructuredResume `json:"resume_structured,omitempty"`
	JobID            string            `json:"job_id,omitempty"`
	CustomJD         string            `json:"custom_jd,omitempty"`
	Model            string            `json:"model,omitempty"` // optional per-request model override
}

// StructuredResume represents complete AI generated HTML structured components
type StructuredResume struct {
	Name              string          `json:"name"`
	Title             string          `json:"title"`
	ContactItems      []string        `json:"contact_items"`
	Summary           string          `json:"summary"`
	Skills            []SkillCategory `json:"skills"`
	WorkExperience    []JobExperience `json:"work_experience"`
	Education         []EducationItem `json:"education"`
	HighlightKeywords []string        `json:"highlight_keywords"`
}

type SkillCategory struct {
	Category string `json:"category"`
	Items    string `json:"items"`
}

type JobExperience struct {
	Title     string   `json:"title"`
	Date      string   `json:"date"`
	Company   string   `json:"company"`
	Location  string   `json:"location"`
	Bullets   []string `json:"bullets"`
	TechStack string   `json:"tech_stack"`
}

type EducationItem struct {
	Institution string `json:"institution"`
	Date        string `json:"date"`
	Degree      string `json:"degree"`
}

// ChatResponse is the AI assistant's response
type ChatResponse struct {
	Message          string              `json:"message"`
	ProposedEdits    []ProposedEdit      `json:"proposed_edits,omitempty"`
	GapPrompts       []GapQuestionPrompt `json:"gap_prompts,omitempty"`
	StructuredResume *StructuredResume   `json:"structured_resume,omitempty"`
}

// ResumeVersion is a snapshot of a resume at the time of applying
type ResumeVersion struct {
	ID                 string            `json:"id"`
	ResumeID           string            `json:"resume_id"`
	JobID              string            `json:"job_id,omitempty"`
	JobTitle           string            `json:"job_title,omitempty"`
	JobCompany         string            `json:"job_company,omitempty"`
	SnapshotText       string            `json:"snapshot_text"`
	Label              string            `json:"label"`
	AppliedAt          time.Time         `json:"applied_at"`
	Source             string            `json:"source"`
	SnapshotStructured *StructuredResume `json:"snapshot_structured,omitempty"`
}

type UpdateResumeRequest struct {
	Structured *StructuredResume `json:"structured"`
}

// NvidiaModel represents an available NVIDIA AI model
type NvidiaModel struct {
	Name       string    `json:"name"`
	Size       int64     `json:"size"`
	ModifiedAt time.Time `json:"modified_at"`
	Family     string    `json:"family,omitempty"`
}

// AISettings holds the current AI model configuration
type AISettings struct {
	ActiveModel     string        `json:"active_model"`
	DefaultModel    string        `json:"default_model"`
	AvailableModels []NvidiaModel `json:"available_models"`
}

// RecruiterValidationResult represents an independent recruiter AI audit of a generated resume
type RecruiterValidationResult struct {
	IsValid         bool     `json:"is_valid"`
	Hallucinations  []string `json:"hallucinations"`  // Any fake companies/roles/degrees added
	Omissions       []string `json:"omissions"`       // Any real companies/roles omitted
	DummyData       []string `json:"dummy_data"`       // Any placeholder or dummy strings found
	QualityFeedback string   `json:"quality_feedback"` // Senior recruiter detailed assessment
	RecruiterScore  int      `json:"recruiter_score"`  // 0-100 rating from recruiter standpoint
}
