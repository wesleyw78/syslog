package service

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"syslog/internal/domain"
	"syslog/internal/repository"
)

const (
	dayEndTimeSettingKey      = "day_end_time"
	defaultDayEndTime         = "23:59"
	defaultDayEndPollInterval = time.Minute
)

type DayEndDispatcherDeps struct {
	DB           *sql.DB
	Attendance   repository.AttendanceRepository
	Reports      repository.ReportRepository
	Settings     repository.SystemSettingRepository
	Runs         repository.DayEndRunRepository
	Location     *time.Location
	PollInterval time.Duration
	ReportSvc    *ReportService
	DayEndSvc    *DayEndService
	Now          func() time.Time
}

type DayEndDispatcher struct {
	db           *sql.DB
	attendance   repository.AttendanceRepository
	reports      repository.ReportRepository
	settings     repository.SystemSettingRepository
	runs         repository.DayEndRunRepository
	location     *time.Location
	pollInterval time.Duration
	reportSvc    *ReportService
	dayEndSvc    *DayEndService
	now          func() time.Time
}

type dayEndRunTxRepository interface {
	WithTx(*sql.Tx) repository.DayEndRunRepository
}

func NewDayEndDispatcher(deps DayEndDispatcherDeps) *DayEndDispatcher {
	if deps.Location == nil {
		deps.Location = time.Local
	}
	if deps.PollInterval <= 0 {
		deps.PollInterval = defaultDayEndPollInterval
	}
	if deps.ReportSvc == nil {
		deps.ReportSvc = NewReportService()
	}
	if deps.DayEndSvc == nil {
		deps.DayEndSvc = NewDayEndService()
	}
	if deps.Now == nil {
		deps.Now = time.Now
	}

	return &DayEndDispatcher{
		db:           deps.DB,
		attendance:   deps.Attendance,
		reports:      deps.Reports,
		settings:     deps.Settings,
		runs:         deps.Runs,
		location:     deps.Location,
		pollInterval: deps.PollInterval,
		reportSvc:    deps.ReportSvc,
		dayEndSvc:    deps.DayEndSvc,
		now:          deps.Now,
	}
}

func (d *DayEndDispatcher) Run(ctx context.Context) error {
	if err := d.RunOnce(ctx); err != nil {
		return err
	}

	ticker := time.NewTicker(d.pollInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
			if err := d.RunOnce(ctx); err != nil {
				return err
			}
		}
	}
}

func (d *DayEndDispatcher) RunOnce(ctx context.Context) error {
	if d.attendance == nil || d.reports == nil || d.runs == nil {
		return nil
	}

	now := d.now().In(d.location)
	cutoff, cutoffText, err := d.loadCutoffTime(ctx, now.Location())
	if err != nil {
		return err
	}
	if now.Before(cutoff) {
		return nil
	}

	businessDate := truncateToDate(now)
	if _, err := d.runs.FindByDate(ctx, businessDate); err == nil {
		return nil
	} else if !errors.Is(err, sql.ErrNoRows) {
		return err
	}

	records, err := d.attendance.ListByDateRange(ctx, businessDate, businessDate)
	if err != nil {
		return err
	}

	for idx := range records {
		record := records[idx]
		if err := d.finalizeRecord(ctx, &record, now); err != nil {
			return err
		}
	}

	run := domain.DayEndRun{
		BusinessDate: businessDate,
		CutoffTime:   cutoffText,
		ExecutedAt:   now,
	}
	return d.saveRun(ctx, &run)
}

func (d *DayEndDispatcher) loadCutoffTime(ctx context.Context, location *time.Location) (time.Time, string, error) {
	now := d.now().In(location)
	cutoffText := defaultDayEndTime
	if d.settings != nil {
		setting, err := d.settings.GetByKey(ctx, dayEndTimeSettingKey)
		if err != nil && !errors.Is(err, sql.ErrNoRows) {
			return time.Time{}, "", err
		}
		if setting != nil && strings.TrimSpace(setting.SettingValue) != "" {
			cutoffText = strings.TrimSpace(setting.SettingValue)
		}
	}

	parsed, err := time.ParseInLocation("15:04", cutoffText, location)
	if err != nil {
		return time.Time{}, "", fmt.Errorf("invalid day_end_time %q: %w", cutoffText, err)
	}

	cutoff := time.Date(now.Year(), now.Month(), now.Day(), parsed.Hour(), parsed.Minute(), 0, 0, location)
	return cutoff, cutoffText, nil
}

func (d *DayEndDispatcher) finalizeRecord(ctx context.Context, record *domain.AttendanceRecord, now time.Time) error {
	if record == nil {
		return nil
	}

	original := *record
	finalized := d.dayEndSvc.FinalizeForDay(*record, now)
	finalized.LastCalculatedAt = timePointer(now)

	var report *domain.AttendanceReport
	if finalized.LastDisconnectAt != nil {
		idempotencyKey := d.reportSvc.BuildIdempotencyKey(finalized, "clock_out", *finalized.LastDisconnectAt)
		existing, err := d.reports.FindByIdempotencyKey(ctx, idempotencyKey)
		if err != nil && !errors.Is(err, sql.ErrNoRows) {
			return err
		}
		if existing == nil {
			generated := d.reportSvc.CreatePendingReport(finalized, "clock_out", *finalized.LastDisconnectAt)
			if err := copyLatestDeleteRecordID(ctx, d.reports, &generated); err != nil {
				return err
			}
			report = &generated
		}
	}

	if !attendanceRecordChanged(original, finalized) && report == nil {
		return nil
	}

	if d.db == nil {
		if err := d.attendance.Save(ctx, &finalized); err != nil {
			return err
		}
		if report != nil {
			report.AttendanceRecordID = finalized.ID
			if err := d.reports.Save(ctx, report); err != nil {
				return err
			}
		}
		return nil
	}

	attendanceRepo, ok := d.attendance.(attendanceTxRepository)
	if !ok {
		return errors.New("attendance repository does not support tx")
	}
	reportRepo, ok := d.reports.(reportTxRepository)
	if !ok {
		return errors.New("report repository does not support tx")
	}

	tx, err := d.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() {
		if err != nil {
			_ = tx.Rollback()
		}
	}()

	if err = attendanceRepo.WithTx(tx).Save(ctx, &finalized); err != nil {
		return err
	}
	if report != nil {
		report.AttendanceRecordID = finalized.ID
		if err = reportRepo.WithTx(tx).Save(ctx, report); err != nil {
			return err
		}
	}

	if err = tx.Commit(); err != nil {
		return err
	}
	return nil
}

func (d *DayEndDispatcher) saveRun(ctx context.Context, run *domain.DayEndRun) error {
	if run == nil {
		return nil
	}
	if d.db == nil {
		return d.runs.Save(ctx, run)
	}

	runRepo, ok := d.runs.(dayEndRunTxRepository)
	if !ok {
		return errors.New("day end run repository does not support tx")
	}

	tx, err := d.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() {
		if err != nil {
			_ = tx.Rollback()
		}
	}()

	if err = runRepo.WithTx(tx).Save(ctx, run); err != nil {
		return err
	}
	if err = tx.Commit(); err != nil {
		return err
	}
	return nil
}

func attendanceRecordChanged(before, after domain.AttendanceRecord) bool {
	return !timePointerEqual(before.FirstConnectAt, after.FirstConnectAt) ||
		!timePointerEqual(before.LastDisconnectAt, after.LastDisconnectAt) ||
		before.ClockInStatus != after.ClockInStatus ||
		before.ClockOutStatus != after.ClockOutStatus ||
		before.ExceptionStatus != after.ExceptionStatus ||
		before.SourceMode != after.SourceMode ||
		before.Version != after.Version ||
		!timePointerEqual(before.LastCalculatedAt, after.LastCalculatedAt)
}

func truncateToDate(value time.Time) time.Time {
	return time.Date(value.Year(), value.Month(), value.Day(), 0, 0, 0, 0, value.Location())
}
