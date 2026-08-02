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
	ID              string    `json:"id"`
	Filename        string    `json:"filename"`
	RawText         string    `json:"raw_text,omitempty"`
	ExtractedSkills []string  `json:"extracted_skills"`
	RawTextLength   int       `json:"raw_text_length"`
	UploadedAt      time.Time `json:"uploaded_at"`
	IsActive        bool      `json:"is_active"`
	HasPDF          bool      `json:"has_pdf"`
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
	Message    string `json:"message"`
	ResumeText string `json:"resume_text"`
	JobID      string `json:"job_id,omitempty"`
}

// ChatResponse is the AI assistant's response
type ChatResponse struct {
	Message       string              `json:"message"`
	ProposedEdits []ProposedEdit      `json:"proposed_edits,omitempty"`
	GapPrompts    []GapQuestionPrompt `json:"gap_prompts,omitempty"`
}

// ResumeVersion is a snapshot of a resume at the time of applying
type ResumeVersion struct {
	ID           string     `json:"id"`
	ResumeID     string     `json:"resume_id"`
	JobID        string     `json:"job_id,omitempty"`
	JobTitle     string     `json:"job_title,omitempty"`
	JobCompany   string     `json:"job_company,omitempty"`
	SnapshotText string     `json:"snapshot_text"`
	Label        string     `json:"label"`
	AppliedAt    time.Time  `json:"applied_at"`
	Source       string     `json:"source"`
}
