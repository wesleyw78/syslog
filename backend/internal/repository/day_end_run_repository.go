package repository

import (
	"context"
	"database/sql"
	"time"

	"syslog/internal/domain"
)

type DayEndRunRepository interface {
	FindByDate(ctx context.Context, businessDate time.Time) (*domain.DayEndRun, error)
	Save(ctx context.Context, run *domain.DayEndRun) error
	WithTx(tx *sql.Tx) DayEndRunRepository
}

type MySQLDayEndRunRepository struct {
	db sqlExecutor
}

func NewMySQLDayEndRunRepository(db *sql.DB) *MySQLDayEndRunRepository {
	return &MySQLDayEndRunRepository{db: db}
}

func (r *MySQLDayEndRunRepository) WithTx(tx *sql.Tx) DayEndRunRepository {
	return &MySQLDayEndRunRepository{db: tx}
}

func (r *MySQLDayEndRunRepository) FindByDate(ctx context.Context, businessDate time.Time) (*domain.DayEndRun, error) {
	const query = `
SELECT id, business_date, cutoff_time, executed_at
FROM day_end_runs
WHERE business_date = ?
LIMIT 1`

	row := r.db.QueryRowContext(ctx, trimSQL(query), businessDate)

	var run domain.DayEndRun
	if err := row.Scan(&run.ID, &run.BusinessDate, &run.CutoffTime, &run.ExecutedAt); err != nil {
		return nil, err
	}

	return &run, nil
}

func (r *MySQLDayEndRunRepository) Save(ctx context.Context, run *domain.DayEndRun) error {
	const query = `
INSERT INTO day_end_runs (
	business_date,
	cutoff_time,
	executed_at
) VALUES (?, ?, ?)
ON DUPLICATE KEY UPDATE
	id = LAST_INSERT_ID(id),
	cutoff_time = VALUES(cutoff_time),
	executed_at = VALUES(executed_at)`

	result, err := r.db.ExecContext(
		ctx,
		trimSQL(query),
		run.BusinessDate,
		run.CutoffTime,
		run.ExecutedAt,
	)
	if err != nil {
		return err
	}

	id, err := parseInsertedID(result)
	if err != nil {
		return err
	}

	run.ID = id
	return nil
}
