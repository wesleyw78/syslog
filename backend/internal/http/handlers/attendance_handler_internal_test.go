package handlers

import (
	"context"
	"database/sql"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"syslog/internal/domain"
	"syslog/internal/repository"
)

type captureAttendanceWindowRepo struct {
	from    time.Time
	to      time.Time
	calls   int
	records []domain.AttendanceRecord
}

func (r *captureAttendanceWindowRepo) FindByEmployeeAndDate(context.Context, uint64, time.Time) (*domain.AttendanceRecord, error) {
	return nil, nil
}

func (r *captureAttendanceWindowRepo) FindByID(context.Context, uint64) (*domain.AttendanceRecord, error) {
	return nil, nil
}

func (r *captureAttendanceWindowRepo) Save(context.Context, *domain.AttendanceRecord) error {
	return nil
}

func (r *captureAttendanceWindowRepo) ListByDateRange(_ context.Context, from, to time.Time) ([]domain.AttendanceRecord, error) {
	r.calls++
	r.from = from
	r.to = to
	return append([]domain.AttendanceRecord(nil), r.records...), nil
}

func (r *captureAttendanceWindowRepo) WithTx(*sql.Tx) repository.AttendanceRepository {
	return r
}

func TestNewAttendanceHandlerUsesDayBoundaries(t *testing.T) {
	originalNow := attendanceNow
	defer func() { attendanceNow = originalNow }()

	fixedNow := time.Date(2026, 3, 21, 15, 4, 0, 0, asiaShanghai)
	attendanceNow = func() time.Time { return fixedNow }

	repo := &captureAttendanceWindowRepo{
		records: []domain.AttendanceRecord{{ID: 1, EmployeeID: 42}},
	}

	handler := NewAttendanceHandler(repo)
	req := httptest.NewRequest(http.MethodGet, "/api/attendance", nil)
	resp := httptest.NewRecorder()

	handler.ServeHTTP(resp, req)

	if resp.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, resp.Code)
	}
	wantFrom := time.Date(2026, 2, 20, 0, 0, 0, 0, asiaShanghai)
	wantTo := time.Date(2026, 3, 21, 23, 59, 59, 999999999, asiaShanghai)

	if repo.calls != 1 {
		t.Fatalf("expected one list call, got %d", repo.calls)
	}
	if !repo.from.Equal(wantFrom) {
		t.Fatalf("expected from %s, got %s", wantFrom, repo.from)
	}
	if !repo.to.Equal(wantTo) {
		t.Fatalf("expected to %s, got %s", wantTo, repo.to)
	}
}
