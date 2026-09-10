package postgres

import (
	"context"
	"database/sql"
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
	if err == sql.ErrNoRows {
		return "", nil
	}
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
	// Accept any non-empty model name
	if val == "" {
		return defaultModel, nil
	}
	return val, nil
}
