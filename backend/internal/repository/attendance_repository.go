package repository

import (
	"context"
	"time"

	"syslog/internal/domain"
)

type AttendanceRepository interface {
	FindByEmployeeAndDate(ctx context.Context, employeeID uint64, attendanceDate time.Time) (*domain.AttendanceRecord, error)
	Save(ctx context.Context, record *domain.AttendanceRecord) error
}
