package postgres

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"remotehunter/internal/models"
	"remotehunter/internal/repository"
	"time"

	"github.com/google/uuid"
)

type ResumeRepo struct {
	db *sql.DB
}

func NewResumeRepo(db *sql.DB) repository.ResumeRepository {
	return &ResumeRepo{db: db}
}

func (r *ResumeRepo) GetActiveResume(ctx context.Context) (*models.Resume, error) {
	var res models.Resume
	var rawText, editedText sql.NullString
	var skillsJSON []byte
	var uploadedAt time.Time
	err := r.db.QueryRowContext(ctx, `
		SELECT id, filename, raw_text, edited_text, extracted_skills, uploaded_at, is_active
		FROM resumes WHERE is_active = true ORDER BY uploaded_at DESC LIMIT 1
	`).Scan(
		&res.ID, &res.Filename, &rawText, &editedText,
		&skillsJSON, &uploadedAt, &res.IsActive,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("get active resume error: %w", err)
	}

	if editedText.Valid && editedText.String != "" {
		res.RawText = editedText.String
	} else if rawText.Valid {
		res.RawText = rawText.String
	}

	if len(skillsJSON) > 0 {
		var skills []string
		_ = json.Unmarshal(skillsJSON, &skills)
		res.ExtractedSkills = skills
	}

	res.RawTextLength = len(res.RawText)
	res.UploadedAt = uploadedAt
	return &res, nil
}

func (r *ResumeRepo) GetActiveResumeID(ctx context.Context) (string, error) {
	var resumeID string
	err := r.db.QueryRowContext(ctx, `SELECT id FROM resumes WHERE is_active = true ORDER BY uploaded_at DESC LIMIT 1`).Scan(&resumeID)
	if err == sql.ErrNoRows {
		return "", nil
	}
	return resumeID, err
}

func (r *ResumeRepo) SaveResume(ctx context.Context, resume *models.Resume) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	// Deactivate all existing resumes
	_, err = tx.ExecContext(ctx, `UPDATE resumes SET is_active = false`)
	if err != nil {
		return err
	}

	skillsJSON, _ := json.Marshal(resume.ExtractedSkills)
	now := time.Now()
	err = tx.QueryRowContext(ctx, `
		INSERT INTO resumes (id, filename, raw_text, extracted_skills, is_active, uploaded_at)
		VALUES ($1, $2, $3, $4, true, $5)
		RETURNING id
	`, resume.ID, resume.Filename, resume.RawText, skillsJSON, now).Scan(&resume.ID)
	if err != nil {
		return err
	}

	return tx.Commit()
}

func (r *ResumeRepo) GetResumeFullText(ctx context.Context) (string, error) {
	var rawText, editedText sql.NullString
	err := r.db.QueryRowContext(ctx, `SELECT raw_text, edited_text FROM resumes WHERE is_active = true ORDER BY uploaded_at DESC LIMIT 1`).Scan(&rawText, &editedText)
	if err == sql.ErrNoRows {
		return "", nil
	}
	if err != nil {
		return "", err
	}
	if editedText.Valid && editedText.String != "" {
		return editedText.String, nil
	}
	return rawText.String, nil
}

func (r *ResumeRepo) UpdateResumeText(ctx context.Context, text string) error {
	resumeID, err := r.GetActiveResumeID(ctx)
	if err != nil || resumeID == "" {
		return fmt.Errorf("no active resume found: %w", err)
	}

	// Backup current version before updating
	var currentRawText, currentEditedText sql.NullString
	err = r.db.QueryRowContext(ctx, `SELECT raw_text, edited_text FROM resumes WHERE id = $1`, resumeID).Scan(&currentRawText, &currentEditedText)
	curText := currentRawText.String
	if currentEditedText.Valid && currentEditedText.String != "" {
		curText = currentEditedText.String
	}
	if err == nil && curText != "" {
		vID := uuid.New().String()
		vLabel := fmt.Sprintf("Backup %s", time.Now().Format("Jan 02 15:04"))
		_, _ = r.db.ExecContext(ctx, `
			INSERT INTO resume_versions (id, resume_id, label, snapshot_text, applied_at)
			VALUES ($1, $2, $3, $4, NOW())
		`, vID, resumeID, vLabel, curText)
	}

	_, err = r.db.ExecContext(ctx, `UPDATE resumes SET edited_text = $1 WHERE id = $2`, text, resumeID)
	return err
}

func (r *ResumeRepo) RevertResumeText(ctx context.Context) (string, error) {
	resumeID, err := r.GetActiveResumeID(ctx)
	if err != nil || resumeID == "" {
		return "", fmt.Errorf("no active resume found")
	}

	var latestVersionText string
	err = r.db.QueryRowContext(ctx, `
		SELECT snapshot_text FROM resume_versions 
		WHERE resume_id = $1 ORDER BY applied_at DESC LIMIT 1
	`, resumeID).Scan(&latestVersionText)
	if err == sql.ErrNoRows {
		return "", fmt.Errorf("no versions available to revert to")
	}
	if err != nil {
		return "", err
	}

	_, err = r.db.ExecContext(ctx, `UPDATE resumes SET edited_text = $1 WHERE id = $2`, latestVersionText, resumeID)
	if err != nil {
		return "", err
	}

	return latestVersionText, nil
}

func (r *ResumeRepo) GetVersions(ctx context.Context, resumeID string) ([]models.ResumeVersion, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT id, resume_id, label, snapshot_text, applied_at
		FROM resume_versions WHERE resume_id = $1 ORDER BY applied_at DESC
	`, resumeID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var versions []models.ResumeVersion
	for rows.Next() {
		var v models.ResumeVersion
		if err := rows.Scan(&v.ID, &v.ResumeID, &v.Label, &v.SnapshotText, &v.AppliedAt); err != nil {
			return nil, err
		}
		versions = append(versions, v)
	}
	if versions == nil {
		versions = []models.ResumeVersion{}
	}
	return versions, nil
}

func (r *ResumeRepo) SaveVersion(ctx context.Context, version *models.ResumeVersion) error {
	_, err := r.db.ExecContext(ctx, `
		INSERT INTO resume_versions (id, resume_id, label, snapshot_text, applied_at)
		VALUES ($1, $2, $3, $4, $5)
	`, version.ID, version.ResumeID, version.Label, version.SnapshotText, version.AppliedAt)
	return err
}

func (r *ResumeRepo) GetVersionByID(ctx context.Context, versionID string) (*models.ResumeVersion, error) {
	var v models.ResumeVersion
	err := r.db.QueryRowContext(ctx, `
		SELECT id, resume_id, label, snapshot_text, applied_at
		FROM resume_versions WHERE id = $1
	`, versionID).Scan(&v.ID, &v.ResumeID, &v.Label, &v.SnapshotText, &v.AppliedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &v, nil
}
