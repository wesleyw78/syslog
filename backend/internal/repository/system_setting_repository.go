package repository

import (
	"context"
	"database/sql"

	"syslog/internal/domain"
)

type SystemSettingRepository interface {
	GetByKey(ctx context.Context, key string) (*domain.SystemSetting, error)
	List(ctx context.Context) ([]domain.SystemSetting, error)
	Save(ctx context.Context, setting *domain.SystemSetting) error
	WithTx(tx *sql.Tx) SystemSettingRepository
}

type MySQLSystemSettingRepository struct {
	db sqlExecutor
}

func NewMySQLSystemSettingRepository(db *sql.DB) *MySQLSystemSettingRepository {
	return &MySQLSystemSettingRepository{db: db}
}

func (r *MySQLSystemSettingRepository) WithTx(tx *sql.Tx) SystemSettingRepository {
	return &MySQLSystemSettingRepository{db: tx}
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

func (r *MySQLSystemSettingRepository) Save(ctx context.Context, setting *domain.SystemSetting) error {
	const query = `
INSERT INTO system_settings (
	setting_key,
	setting_value
) VALUES (?, ?)
ON DUPLICATE KEY UPDATE
	id = LAST_INSERT_ID(id),
	setting_value = VALUES(setting_value)`

	result, err := r.db.ExecContext(ctx, trimSQL(query), setting.SettingKey, setting.SettingValue)
	if err != nil {
		return err
	}

	id, err := parseInsertedID(result)
	if err != nil {
		return err
	}

	setting.ID = id
	return nil
}
