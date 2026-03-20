package repository

import (
	"context"
	"database/sql"

	"syslog/internal/domain"
)

type SystemSettingRepository interface {
	GetByKey(ctx context.Context, key string) (*domain.SystemSetting, error)
	List(ctx context.Context) ([]domain.SystemSetting, error)
}

type MySQLSystemSettingRepository struct {
	db *sql.DB
}

func NewMySQLSystemSettingRepository(db *sql.DB) *MySQLSystemSettingRepository {
	return &MySQLSystemSettingRepository{db: db}
}

func (r *MySQLSystemSettingRepository) GetByKey(ctx context.Context, key string) (*domain.SystemSetting, error) {
	const query = `
SELECT id, setting_key, setting_value, updated_at
FROM system_settings
WHERE setting_key = ?
LIMIT 1`

	row := r.db.QueryRowContext(ctx, trimSQL(query), key)

	var setting domain.SystemSetting
	if err := row.Scan(&setting.ID, &setting.SettingKey, &setting.SettingValue, &setting.UpdatedAt); err != nil {
		return nil, err
	}

	return &setting, nil
}

func (r *MySQLSystemSettingRepository) List(ctx context.Context) ([]domain.SystemSetting, error) {
	const query = `
SELECT id, setting_key, setting_value, updated_at
FROM system_settings
ORDER BY setting_key ASC`

	rows, err := r.db.QueryContext(ctx, trimSQL(query))
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	settings := make([]domain.SystemSetting, 0)
	for rows.Next() {
		var setting domain.SystemSetting
		if err := rows.Scan(&setting.ID, &setting.SettingKey, &setting.SettingValue, &setting.UpdatedAt); err != nil {
			return nil, err
		}

		settings = append(settings, setting)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return settings, nil
}
