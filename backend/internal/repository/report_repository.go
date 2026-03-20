package repository

import (
	"context"

	"syslog/internal/domain"
)

type ReportRepository interface {
	FindByIdempotencyKey(ctx context.Context, idempotencyKey string) (*domain.AttendanceReport, error)
	Save(ctx context.Context, report *domain.AttendanceReport) error
}
