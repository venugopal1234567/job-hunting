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
	var pdfBytes []byte
	var skillsJSON []byte
	var uploadedAt time.Time
	var initStructuredJSON, editStructuredJSON []byte

	err := r.db.QueryRowContext(ctx, `
		SELECT id, filename, raw_text, edited_text, pdf_data, extracted_skills, uploaded_at, is_active, initial_structured, edited_structured
		FROM resumes WHERE is_active = true ORDER BY uploaded_at DESC LIMIT 1
	`).Scan(
		&res.ID, &res.Filename, &rawText, &editedText, &pdfBytes,
		&skillsJSON, &uploadedAt, &res.IsActive, &initStructuredJSON, &editStructuredJSON,
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

	if len(pdfBytes) > 0 {
		res.HasPDF = true
		res.PDFBytes = pdfBytes
	}

	if len(skillsJSON) > 0 {
		var skills []string
		_ = json.Unmarshal(skillsJSON, &skills)
		res.ExtractedSkills = skills
	}

	if len(initStructuredJSON) > 0 {
		var initStruct models.StructuredResume
		if err := json.Unmarshal(initStructuredJSON, &initStruct); err == nil {
			res.InitialStructured = &initStruct
		}
	}

	if len(editStructuredJSON) > 0 {
		var editStruct models.StructuredResume
		if err := json.Unmarshal(editStructuredJSON, &editStruct); err == nil {
			res.EditedStructured = &editStruct
		}
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
	var initStructuredJSON []byte
	if resume.InitialStructured != nil {
		initStructuredJSON, _ = json.Marshal(resume.InitialStructured)
	}
	now := time.Now()
	err = tx.QueryRowContext(ctx, `
		INSERT INTO resumes (id, filename, raw_text, pdf_data, extracted_skills, is_active, uploaded_at, initial_structured)
		VALUES ($1, $2, $3, $4, $5, true, $6, $7)
		RETURNING id
	`, resume.ID, resume.Filename, resume.RawText, resume.PDFBytes, skillsJSON, now, initStructuredJSON).Scan(&resume.ID)
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

	// Reset edited_text to NULL so it reverts completely to raw_text of the original uploaded PDF
	_, err = r.db.ExecContext(ctx, `UPDATE resumes SET edited_text = NULL, edited_structured = NULL WHERE id = $1`, resumeID)
	if err != nil {
		return "", err
	}

	var rawText string
	err = r.db.QueryRowContext(ctx, `SELECT raw_text FROM resumes WHERE id = $1`, resumeID).Scan(&rawText)
	if err != nil {
		return "", err
	}

	return rawText, nil
}

func (r *ResumeRepo) GetVersions(ctx context.Context, resumeID string) ([]models.ResumeVersion, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT id, resume_id, label, snapshot_text, applied_at, snapshot_structured
		FROM resume_versions WHERE resume_id = $1 ORDER BY applied_at DESC
	`, resumeID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var versions []models.ResumeVersion
	for rows.Next() {
		var v models.ResumeVersion
		var structJSON []byte
		if err := rows.Scan(&v.ID, &v.ResumeID, &v.Label, &v.SnapshotText, &v.AppliedAt, &structJSON); err != nil {
			return nil, err
		}
		if len(structJSON) > 0 {
			var sr models.StructuredResume
			if err := json.Unmarshal(structJSON, &sr); err == nil {
				v.SnapshotStructured = &sr
			}
		}
		versions = append(versions, v)
	}
	if versions == nil {
		versions = []models.ResumeVersion{}
	}
	return versions, nil
}

func (r *ResumeRepo) SaveVersion(ctx context.Context, version *models.ResumeVersion) error {
	var structJSON []byte
	if version.SnapshotStructured != nil {
		structJSON, _ = json.Marshal(version.SnapshotStructured)
	}
	_, err := r.db.ExecContext(ctx, `
		INSERT INTO resume_versions (id, resume_id, label, snapshot_text, applied_at, snapshot_structured)
		VALUES ($1, $2, $3, $4, $5, $6)
	`, version.ID, version.ResumeID, version.Label, version.SnapshotText, version.AppliedAt, structJSON)
	return err
}

func (r *ResumeRepo) GetVersionByID(ctx context.Context, versionID string) (*models.ResumeVersion, error) {
	var v models.ResumeVersion
	var structJSON []byte
	err := r.db.QueryRowContext(ctx, `
		SELECT id, resume_id, label, snapshot_text, applied_at, snapshot_structured
		FROM resume_versions WHERE id = $1
	`, versionID).Scan(&v.ID, &v.ResumeID, &v.Label, &v.SnapshotText, &v.AppliedAt, &structJSON)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	if len(structJSON) > 0 {
		var sr models.StructuredResume
		if err := json.Unmarshal(structJSON, &sr); err == nil {
			v.SnapshotStructured = &sr
		}
	}
	return &v, nil
}

func (r *ResumeRepo) GetResumeContent(ctx context.Context) (*models.StructuredResume, error) {
	var editJSON, initJSON []byte
	err := r.db.QueryRowContext(ctx, `
		SELECT edited_structured, initial_structured
		FROM resumes WHERE is_active = true ORDER BY uploaded_at DESC LIMIT 1
	`).Scan(&editJSON, &initJSON)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	var targetJSON []byte
	if len(editJSON) > 0 {
		targetJSON = editJSON
	} else if len(initJSON) > 0 {
		targetJSON = initJSON
	} else {
		return nil, nil
	}

	var sr models.StructuredResume
	if err := json.Unmarshal(targetJSON, &sr); err != nil {
		return nil, err
	}
	return &sr, nil
}

func (r *ResumeRepo) UpdateResumeStructured(ctx context.Context, sr *models.StructuredResume) error {
	resumeID, err := r.GetActiveResumeID(ctx)
	if err != nil || resumeID == "" {
		return fmt.Errorf("no active resume found: %w", err)
	}

	// Backup current version
	var currentEditJSON []byte
	_ = r.db.QueryRowContext(ctx, `SELECT edited_structured FROM resumes WHERE id = $1`, resumeID).Scan(&currentEditJSON)
	if len(currentEditJSON) > 0 {
		var currentStruct models.StructuredResume
		if err := json.Unmarshal(currentEditJSON, &currentStruct); err == nil {
			vID := uuid.New().String()
			vLabel := fmt.Sprintf("Backup %s", time.Now().Format("Jan 02 15:04"))
			_, _ = r.db.ExecContext(ctx, `
				INSERT INTO resume_versions (id, resume_id, label, snapshot_text, applied_at, snapshot_structured)
				VALUES ($1, $2, $3, '', NOW(), $4)
			`, vID, resumeID, vLabel, currentEditJSON)
		}
	}

	newJSON, err := json.Marshal(sr)
	if err != nil {
		return err
	}

	_, err = r.db.ExecContext(ctx, `UPDATE resumes SET edited_structured = $1 WHERE id = $2`, newJSON, resumeID)
	return err
}

func (r *ResumeRepo) SetInitialStructured(ctx context.Context, sr *models.StructuredResume) error {
	resumeID, err := r.GetActiveResumeID(ctx)
	if err != nil || resumeID == "" {
		return fmt.Errorf("no active resume found: %w", err)
	}

	newJSON, err := json.Marshal(sr)
	if err != nil {
		return err
	}

	_, err = r.db.ExecContext(ctx, `UPDATE resumes SET initial_structured = $1 WHERE id = $2`, newJSON, resumeID)
	return err
}

func (r *ResumeRepo) RevertResume(ctx context.Context) (*models.StructuredResume, error) {
	resumeID, err := r.GetActiveResumeID(ctx)
	if err != nil || resumeID == "" {
		return nil, fmt.Errorf("no active resume found")
	}

	_, err = r.db.ExecContext(ctx, `UPDATE resumes SET edited_structured = NULL WHERE id = $1`, resumeID)
	if err != nil {
		return nil, err
	}

	var initJSON []byte
	err = r.db.QueryRowContext(ctx, `SELECT initial_structured FROM resumes WHERE id = $1`, resumeID).Scan(&initJSON)
	if err != nil {
		return nil, err
	}

	if len(initJSON) == 0 {
		return nil, fmt.Errorf("no initial structured resume to revert to")
	}

	var sr models.StructuredResume
	if err := json.Unmarshal(initJSON, &sr); err != nil {
		return nil, err
	}

	return &sr, nil
}
