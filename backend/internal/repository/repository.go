package repository

import (
	"context"
	"remotehunter/internal/models"
)

// JobFilter defines options for querying jobs
type JobFilter struct {
	Skills  string
	Days    int
	Country string
	Page    int
	Limit   int
}

// JobRepository handles data operations for jobs
type JobRepository interface {
	GetJobs(ctx context.Context, filter JobFilter) ([]models.Job, int, error)
	GetJobByID(ctx context.Context, id string) (*models.Job, error)
	SaveJobs(ctx context.Context, jobs []models.Job) error
}

// ResumeRepository handles data operations for resumes and resume versions
type ResumeRepository interface {
	GetActiveResume(ctx context.Context) (*models.Resume, error)
	GetActiveResumeID(ctx context.Context) (string, error)
	SaveResume(ctx context.Context, resume *models.Resume) error
	GetResumeFullText(ctx context.Context) (string, error)
	UpdateResumeText(ctx context.Context, text string) error
	RevertResumeText(ctx context.Context) (string, error)
	GetVersions(ctx context.Context, resumeID string) ([]models.ResumeVersion, error)
	SaveVersion(ctx context.Context, version *models.ResumeVersion) error
	GetVersionByID(ctx context.Context, versionID string) (*models.ResumeVersion, error)
}

// SettingsRepository handles data operations for app settings and AI configs
type SettingsRepository interface {
	GetSetting(ctx context.Context, key string) (string, error)
	SetSetting(ctx context.Context, key, value string) error
	GetAllSettings(ctx context.Context) (map[string]string, error)
	GetActiveModel(ctx context.Context, defaultModel string) (string, error)
}
