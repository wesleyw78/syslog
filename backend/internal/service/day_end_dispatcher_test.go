package service

import (
	"context"
	"database/sql"
	"errors"
	"testing"
	"time"

	"syslog/internal/domain"
	"syslog/internal/repository"
)

type fakeDayEndRunRepo struct {
	completedDates map[string]domain.DayEndRun
	saved          []domain.DayEndRun
}

func (r *fakeDayEndRunRepo) FindByDate(_ context.Context, date time.Time) (*domain.DayEndRun, error) {
	if r.completedDates == nil {
		return nil, sql.ErrNoRows
	}
	item, ok := r.completedDates[date.Format("2006-01-02")]
	if !ok {
		return nil, sql.ErrNoRows
	}
	copied := item
	return &copied, nil
}

func (r *fakeDayEndRunRepo) Save(_ context.Context, run *domain.DayEndRun) error {
	copied := *run
	r.saved = append(r.saved, copied)
	if r.completedDates == nil {
		r.completedDates = map[string]domain.DayEndRun{}
	}
	r.completedDates[run.BusinessDate.Format("2006-01-02")] = copied
	return nil
}

func (r *fakeDayEndRunRepo) WithTx(*sql.Tx) repository.DayEndRunRepository { return r }

type dayEndAttendanceRepo struct {
	records []domain.AttendanceRecord
	saved   []domain.AttendanceRecord
}

func (r *dayEndAttendanceRepo) FindByID(context.Context, uint64) (*domain.AttendanceRecord, error) {
	return nil, sql.ErrNoRows
}

func (r *dayEndAttendanceRepo) FindByEmployeeAndDate(context.Context, uint64, time.Time) (*domain.AttendanceRecord, error) {
	return nil, sql.ErrNoRows
}

func (r *dayEndAttendanceRepo) Save(_ context.Context, record *domain.AttendanceRecord) error {
	copied := *record
	r.saved = append(r.saved, copied)
	return nil
}

func (r *dayEndAttendanceRepo) ListByDateRange(_ context.Context, from, to time.Time) ([]domain.AttendanceRecord, error) {
	if from.Format("2006-01-02") != to.Format("2006-01-02") {
		return nil, errors.New("expected same-day range")
	}
	items := make([]domain.AttendanceRecord, len(r.records))
	copy(items, r.records)
	return items, nil
}

func (r *dayEndAttendanceRepo) WithTx(*sql.Tx) repository.AttendanceRepository { return r }

type dayEndReportRepo struct {
	existing map[string]domain.AttendanceReport
	saved    []domain.AttendanceReport
}

func (r *dayEndReportRepo) FindByIdempotencyKey(_ context.Context, key string) (*domain.AttendanceReport, error) {
	if r.existing == nil {
		return nil, sql.ErrNoRows
	}
	item, ok := r.existing[key]
	if !ok {
		return nil, sql.ErrNoRows
	}
	copied := item
	return &copied, nil
}

func (r *dayEndReportRepo) FindLatestSuccessfulByAttendanceRecordAndType(context.Context, uint64, string) (*domain.AttendanceReport, error) {
	return nil, sql.ErrNoRows
}

func (r *dayEndReportRepo) Save(_ context.Context, report *domain.AttendanceReport) error {
	copied := *report
	r.saved = append(r.saved, copied)
	if r.existing == nil {
		r.existing = map[string]domain.AttendanceReport{}
	}
	r.existing[report.IdempotencyKey] = copied
	return nil
}

func (r *dayEndReportRepo) ListDispatchable(context.Context, int, uint32) ([]domain.AttendanceReport, error) {
	return nil, nil
}

func (r *dayEndReportRepo) ListNotificationDispatchable(context.Context, int, uint32) ([]domain.AttendanceReport, error) {
	return nil, nil
}

func (r *dayEndReportRepo) ListByAttendanceRecordID(context.Context, uint64) ([]domain.AttendanceReport, error) {
	return nil, nil
}

func (r *dayEndReportRepo) WithTx(*sql.Tx) repository.ReportRepository { return r }

func TestDayEndDispatcherRunOnceCreatesClockOutReportsFromLastDisconnect(t *testing.T) {
	location, err := time.LoadLocation("Asia/Shanghai")
	if err != nil {
		t.Fatalf("load location: %v", err)
	}
	lastDisconnect := time.Date(2026, 3, 23, 18, 5, 0, 0, location)
	attendanceRepo := &dayEndAttendanceRepo{
		records: []domain.AttendanceRecord{
			{
				ID:               41,
				EmployeeID:       7,
				AttendanceDate:   time.Date(2026, 3, 23, 0, 0, 0, 0, location),
				FirstConnectAt:   timePointer(time.Date(2026, 3, 23, 8, 30, 0, 0, location)),
				LastDisconnectAt: &lastDisconnect,
				ClockInStatus:    "done",
				ClockOutStatus:   "pending",
				ExceptionStatus:  "missing_disconnect",
				SourceMode:       "syslog",
				Version:          1,
			},
		},
	}
	reportRepo := &dayEndReportRepo{}
	settingsRepo := &dispatcherSettingsRepo{
		values: map[string]string{
			"day_end_time": "18:30",
		},
	}
	runRepo := &fakeDayEndRunRepo{}

	dispatcher := NewDayEndDispatcher(DayEndDispatcherDeps{
		Attendance: attendanceRepo,
		Reports:    reportRepo,
		Settings:   settingsRepo,
		Runs:       runRepo,
		Location:   location,
		ReportSvc:  NewReportService(),
		Now: func() time.Time {
			return time.Date(2026, 3, 23, 18, 31, 0, 0, location)
		},
	})

	if err := dispatcher.RunOnce(context.Background()); err != nil {
		t.Fatalf("expected dispatcher to succeed, got %v", err)
	}
	if len(attendanceRepo.saved) != 1 {
		t.Fatalf("expected one attendance save, got %d", len(attendanceRepo.saved))
	}
	if attendanceRepo.saved[0].ClockOutStatus != "done" {
		t.Fatalf("expected clock out status done, got %q", attendanceRepo.saved[0].ClockOutStatus)
	}
	if attendanceRepo.saved[0].ExceptionStatus != "none" {
		t.Fatalf("expected exception status none, got %q", attendanceRepo.saved[0].ExceptionStatus)
	}
	if len(reportRepo.saved) != 1 {
		t.Fatalf("expected one clock_out report, got %d", len(reportRepo.saved))
	}
	if reportRepo.saved[0].ReportType != "clock_out" {
		t.Fatalf("expected clock_out report, got %q", reportRepo.saved[0].ReportType)
	}
	if len(runRepo.saved) != 1 {
		t.Fatalf("expected one day end run record, got %d", len(runRepo.saved))
	}
}

func TestDayEndDispatcherRunOnceSkipsBeforeConfiguredTime(t *testing.T) {
	location, err := time.LoadLocation("Asia/Shanghai")
	if err != nil {
		t.Fatalf("load location: %v", err)
	}
	attendanceRepo := &dayEndAttendanceRepo{
		records: []domain.AttendanceRecord{
			{ID: 1, EmployeeID: 7, AttendanceDate: time.Date(2026, 3, 23, 0, 0, 0, 0, location)},
		},
	}
	reportRepo := &dayEndReportRepo{}
	settingsRepo := &dispatcherSettingsRepo{values: map[string]string{"day_end_time": "23:59"}}
	runRepo := &fakeDayEndRunRepo{}
	dispatcher := NewDayEndDispatcher(DayEndDispatcherDeps{
		Attendance: attendanceRepo,
		Reports:    reportRepo,
		Settings:   settingsRepo,
		Runs:       runRepo,
		Location:   location,
		ReportSvc:  NewReportService(),
		Now: func() time.Time {
			return time.Date(2026, 3, 23, 18, 30, 0, 0, location)
		},
	})

	if err := dispatcher.RunOnce(context.Background()); err != nil {
		t.Fatalf("expected dispatcher to skip cleanly, got %v", err)
	}
	if len(attendanceRepo.saved) != 0 || len(reportRepo.saved) != 0 || len(runRepo.saved) != 0 {
		t.Fatalf("expected no writes before cutoff, got attendance=%d reports=%d runs=%d", len(attendanceRepo.saved), len(reportRepo.saved), len(runRepo.saved))
	}
}

func TestDayEndDispatcherRunOnceMarksMissingWithoutGeneratingReport(t *testing.T) {
	location, err := time.LoadLocation("Asia/Shanghai")
	if err != nil {
		t.Fatalf("load location: %v", err)
	}
	attendanceRepo := &dayEndAttendanceRepo{
		records: []domain.AttendanceRecord{
			{
				ID:              42,
				EmployeeID:      8,
				AttendanceDate:  time.Date(2026, 3, 23, 0, 0, 0, 0, location),
				ClockInStatus:   "done",
				ClockOutStatus:  "pending",
				ExceptionStatus: "none",
				SourceMode:      "syslog",
				Version:         1,
			},
		},
	}
	reportRepo := &dayEndReportRepo{}
	settingsRepo := &dispatcherSettingsRepo{values: map[string]string{"day_end_time": "18:30"}}
	runRepo := &fakeDayEndRunRepo{}
	dispatcher := NewDayEndDispatcher(DayEndDispatcherDeps{
		Attendance: attendanceRepo,
		Reports:    reportRepo,
		Settings:   settingsRepo,
		Runs:       runRepo,
		Location:   location,
		ReportSvc:  NewReportService(),
		Now: func() time.Time {
			return time.Date(2026, 3, 23, 18, 31, 0, 0, location)
		},
	})

	if err := dispatcher.RunOnce(context.Background()); err != nil {
		t.Fatalf("expected dispatcher to succeed, got %v", err)
	}
	if len(attendanceRepo.saved) != 1 {
		t.Fatalf("expected one attendance save, got %d", len(attendanceRepo.saved))
	}
	if attendanceRepo.saved[0].ClockOutStatus != "missing" {
		t.Fatalf("expected missing clock out status, got %q", attendanceRepo.saved[0].ClockOutStatus)
	}
	if attendanceRepo.saved[0].ExceptionStatus != "missing_disconnect" {
		t.Fatalf("expected missing_disconnect exception, got %q", attendanceRepo.saved[0].ExceptionStatus)
	}
	if len(reportRepo.saved) != 0 {
		t.Fatalf("expected no report for missing disconnect, got %d", len(reportRepo.saved))
	}
}

func TestDayEndDispatcherRunOnceDoesNotRepeatForSameBusinessDate(t *testing.T) {
	location, err := time.LoadLocation("Asia/Shanghai")
	if err != nil {
		t.Fatalf("load location: %v", err)
	}
	attendanceRepo := &dayEndAttendanceRepo{
		records: []domain.AttendanceRecord{
			{ID: 1, EmployeeID: 7, AttendanceDate: time.Date(2026, 3, 23, 0, 0, 0, 0, location)},
		},
	}
	reportRepo := &dayEndReportRepo{}
	settingsRepo := &dispatcherSettingsRepo{values: map[string]string{"day_end_time": "18:30"}}
	runRepo := &fakeDayEndRunRepo{
		completedDates: map[string]domain.DayEndRun{
			"2026-03-23": {
				ID:           1,
				BusinessDate: time.Date(2026, 3, 23, 0, 0, 0, 0, location),
				CutoffTime:   "18:30",
				ExecutedAt:   time.Date(2026, 3, 23, 18, 31, 0, 0, location),
			},
		},
	}
	dispatcher := NewDayEndDispatcher(DayEndDispatcherDeps{
		Attendance: attendanceRepo,
		Reports:    reportRepo,
		Settings:   settingsRepo,
		Runs:       runRepo,
		Location:   location,
		ReportSvc:  NewReportService(),
		Now: func() time.Time {
			return time.Date(2026, 3, 23, 20, 0, 0, 0, location)
		},
	})

	if err := dispatcher.RunOnce(context.Background()); err != nil {
		t.Fatalf("expected dispatcher to skip cleanly, got %v", err)
	}
	if len(attendanceRepo.saved) != 0 || len(reportRepo.saved) != 0 || len(runRepo.saved) != 0 {
		t.Fatalf("expected no repeated writes for same business date")
	}
}
