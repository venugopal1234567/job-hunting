package postgres

import (
	"context"
	"database/sql"
	"log"
	"remotehunter/internal/models"
	"remotehunter/internal/repository"
)

type SettingsRepo struct {
	db *sql.DB
}

func NewSettingsRepo(db *sql.DB) repository.SettingsRepository {
	return &SettingsRepo{db: db}
}

func (r *SettingsRepo) GetSetting(ctx context.Context, key string) (string, error) {
	var val string
	err := r.db.QueryRowContext(ctx, `SELECT value FROM app_settings WHERE key = $1`, key).Scan(&val)
	return val, err
}

func (r *SettingsRepo) SetSetting(ctx context.Context, key, value string) error {
	_, err := r.db.ExecContext(ctx, `
		INSERT INTO app_settings (key, value, updated_at)
		VALUES ($1, $2, NOW())
		ON CONFLICT (key) DO UPDATE SET value = EXCLUDED.value, updated_at = NOW()
	`, key, value)
	return err
}

func (r *SettingsRepo) GetAllSettings(ctx context.Context) (map[string]string, error) {
	rows, err := r.db.QueryContext(ctx, `SELECT key, value FROM app_settings`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	settings := make(map[string]string)
	for rows.Next() {
		var k, v string
		if err := rows.Scan(&k, &v); err != nil {
			return nil, err
		}
		settings[k] = v
	}
	return settings, nil
}

func (r *SettingsRepo) GetActiveModel(ctx context.Context, defaultModel string) (string, error) {
	val, err := r.GetSetting(ctx, "active_model")
	if err != nil || val == "" {
		return defaultModel, nil
	}
	return val, nil
}

func (r *SettingsRepo) GetScraperConfigs(ctx context.Context) ([]models.ScraperConfig, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT id, board_name, target_url, enabled, cron_schedule, last_run_at
		FROM scraper_configs
		ORDER BY id
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var configs []models.ScraperConfig
	for rows.Next() {
		var c models.ScraperConfig
		var lastRunAt sql.NullTime
		if err := rows.Scan(&c.ID, &c.BoardName, &c.TargetURL, &c.Enabled, &c.CronSchedule, &lastRunAt); err != nil {
			log.Printf("[SettingsRepo] Error scanning scraper config: %v", err)
			continue
		}
		if lastRunAt.Valid {
			c.LastRunAt = &lastRunAt.Time
		}
		configs = append(configs, c)
	}

	if configs == nil {
		configs = []models.ScraperConfig{}
	}
	return configs, nil
}

func (r *SettingsRepo) UpdateScraperConfig(ctx context.Context, cfg models.ScraperConfig) error {
	_, err := r.db.ExecContext(ctx, `
		UPDATE scraper_configs
		SET target_url = $1, enabled = $2, cron_schedule = $3
		WHERE id = $4
	`, cfg.TargetURL, cfg.Enabled, cfg.CronSchedule, cfg.ID)
	return err
}
