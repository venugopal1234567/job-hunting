package postgres

import (
	"context"
	"database/sql"
	"fmt"
	"remotehunter/internal/models"
	"remotehunter/internal/repository"
	"strings"
)

type JobRepo struct {
	db *sql.DB
}

func NewJobRepo(db *sql.DB) repository.JobRepository {
	return &JobRepo{db: db}
}

func (r *JobRepo) GetJobs(ctx context.Context, filter repository.JobFilter) ([]models.Job, int, error) {
	where := []string{"1=1"}
	args := []interface{}{}
	argIdx := 1

	if filter.Days > 0 {
		where = append(where, fmt.Sprintf("(posted_at >= NOW() - INTERVAL '%d days' OR (posted_at IS NULL AND scraped_at >= NOW() - INTERVAL '%d days'))", filter.Days, filter.Days))
	}

	// Match either column: boards like builtin put the work mode ("Remote",
	// "In-Office or Remote") in location and the country only in country, so
	// filtering on location alone hid every one of their postings.
	if filter.Country != "" {
		countryList := strings.Split(filter.Country, ",")
		var countryConditions []string
		for _, c := range countryList {
			c = strings.TrimSpace(c)
			if c != "" {
				countryConditions = append(countryConditions, fmt.Sprintf("(LOWER(country) LIKE $%d OR LOWER(location) LIKE $%d)", argIdx, argIdx))
				args = append(args, "%"+strings.ToLower(c)+"%")
				argIdx++
			}
		}
		if len(countryConditions) > 0 {
			where = append(where, "("+strings.Join(countryConditions, " OR ")+")")
		}
	}

	if filter.Skills != "" {
		skillList := strings.Split(filter.Skills, ",")
		var skillConditions []string
		for _, s := range skillList {
			s = strings.TrimSpace(s)
			if s != "" {
				skillConditions = append(skillConditions, fmt.Sprintf("(LOWER(title) LIKE $%d OR LOWER(description) LIKE $%d)", argIdx, argIdx))
				args = append(args, "%"+strings.ToLower(s)+"%")
				argIdx++
			}
		}
		if len(skillConditions) > 0 {
			where = append(where, "("+strings.Join(skillConditions, " OR ")+")")
		}
	}

	if filter.OnlyEnabled {
		// Filter to only boards the user has enabled in Settings. The
		// setting is stored as a comma-delimited list in app_settings.
		rows, err := r.db.QueryContext(ctx, `SELECT value FROM app_settings WHERE key = 'active_sources'`)
		if err == nil {
			var settingVal string
			if rows.Next() {
				_ = rows.Scan(&settingVal)
			}
			rows.Close()
			if settingVal != "" {
				srcs := strings.Split(settingVal, ",")
				for _, src := range srcs {
					src = strings.TrimSpace(src)
					if src == "" {
						continue
					}
					where = append(where, fmt.Sprintf("source_board = $%d", argIdx))
					args = append(args, src)
					argIdx++
				}
			}
		}
	}

	if len(filter.Sources) > 0 {
		var srcConds []string
		for _, s := range filter.Sources {
			s = strings.TrimSpace(s)
			if s == "" {
				continue
			}
			srcConds = append(srcConds, fmt.Sprintf("source_board = $%d", argIdx))
			args = append(args, s)
			argIdx++
		}
		if len(srcConds) > 0 {
			where = append(where, "("+strings.Join(srcConds, " OR ")+")")
		}
	}

	whereClause := strings.Join(where, " AND ")

	var total int
	countQuery := fmt.Sprintf("SELECT COUNT(*) FROM jobs WHERE %s", whereClause)
	err := r.db.QueryRowContext(ctx, countQuery, args...).Scan(&total)
	if err != nil {
		return nil, 0, fmt.Errorf("count jobs error: %w", err)
	}

	offset := (filter.Page - 1) * filter.Limit
	query := fmt.Sprintf(`
		SELECT id, title, company, location, country, description, source_url, source_board, posted_at, salary_range, scraped_at
		FROM jobs
		WHERE %s
		ORDER BY scraped_at DESC NULLS LAST, posted_at DESC NULLS LAST
		LIMIT $%d OFFSET $%d
	`, whereClause, argIdx, argIdx+1)
	args = append(args, filter.Limit, offset)

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("query jobs error: %w", err)
	}
	defer rows.Close()

	var jobs []models.Job
	for rows.Next() {
		var j models.Job
		err := rows.Scan(
			&j.ID, &j.Title, &j.Company, &j.Location, &j.Country, &j.Description,
			&j.SourceURL, &j.SourceBoard, &j.PostedAt, &j.SalaryRange, &j.ScrapedAt,
		)
		if err != nil {
			return nil, 0, fmt.Errorf("scan job error: %w", err)
		}
		jobs = append(jobs, j)
	}
	if jobs == nil {
		jobs = []models.Job{}
	}

	return jobs, total, nil
}

func (r *JobRepo) GetJobByID(ctx context.Context, id string) (*models.Job, error) {
	var j models.Job
	err := r.db.QueryRowContext(ctx, `
		SELECT id, title, company, location, description, source_url, source_board, posted_at, salary_range, scraped_at
		FROM jobs WHERE id = $1
	`, id).Scan(
		&j.ID, &j.Title, &j.Company, &j.Location, &j.Description,
		&j.SourceURL, &j.SourceBoard, &j.PostedAt, &j.SalaryRange, &j.ScrapedAt,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("get job by id error: %w", err)
	}
	return &j, nil
}

func (r *JobRepo) SaveJobs(ctx context.Context, jobs []models.Job) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	stmt, err := tx.PrepareContext(ctx, `
		INSERT INTO jobs (id, job_hash, title, company, location, country, source_url, source_board, description, salary_range, job_type, posted_at, is_active)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13)
		ON CONFLICT (job_hash) DO UPDATE SET
			title = EXCLUDED.title,
			company = EXCLUDED.company,
			location = EXCLUDED.location,
			country = EXCLUDED.country,
			source_url = EXCLUDED.source_url,
			source_board = EXCLUDED.source_board,
			description = EXCLUDED.description,
			salary_range = EXCLUDED.salary_range,
			job_type = EXCLUDED.job_type,
			posted_at = EXCLUDED.posted_at,
			is_active = EXCLUDED.is_active
	`)
	if err != nil {
		return err
	}
	defer stmt.Close()

	for _, j := range jobs {
		_, err := stmt.ExecContext(ctx,
			j.ID, j.JobHash, j.Title, j.Company, j.Location, j.Country,
			j.SourceURL, j.SourceBoard, j.Description, j.SalaryRange, j.JobType,
			j.PostedAt, j.IsActive,
		)
		if err != nil {
			return err
		}
	}

	return tx.Commit()
}
