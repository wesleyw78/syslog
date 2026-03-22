package service

import (
	"context"
	"database/sql"
	"errors"
	"net"
	"strings"
	"time"

	"syslog/internal/domain"
	"syslog/internal/parser"
	"syslog/internal/repository"
)

type SyslogPipelineDeps struct {
	DB             *sql.DB
	Messages       repository.SyslogMessageRepository
	Events         repository.ClientEventRepository
	Employees      repository.EmployeeRepository
	Attendance     repository.AttendanceRepository
	Reports        repository.ReportRepository
	Settings       repository.SystemSettingRepository
	Rules          repository.SyslogReceiveRuleRepository
	RetentionDays  int
	AttendanceProc *AttendanceProcessor
	ReportSvc      *ReportService
}

type SyslogPipeline struct {
	db             *sql.DB
	messages       repository.SyslogMessageRepository
	events         repository.ClientEventRepository
	employees      repository.EmployeeRepository
	attendance     repository.AttendanceRepository
	reports        repository.ReportRepository
	settings       repository.SystemSettingRepository
	rules          repository.SyslogReceiveRuleRepository
	retentionDays  int
	attendanceProc *AttendanceProcessor
	reportSvc      *ReportService
}

func NewSyslogPipeline(deps SyslogPipelineDeps) *SyslogPipeline {
	if deps.AttendanceProc == nil {
		deps.AttendanceProc = NewAttendanceProcessor()
	}
	if deps.ReportSvc == nil {
		deps.ReportSvc = NewReportService()
	}

	return &SyslogPipeline{
		db:             deps.DB,
		messages:       deps.Messages,
		events:         deps.Events,
		employees:      deps.Employees,
		attendance:     deps.Attendance,
		reports:        deps.Reports,
		settings:       deps.Settings,
		rules:          deps.Rules,
		retentionDays:  deps.RetentionDays,
		attendanceProc: deps.AttendanceProc,
		reportSvc:      deps.ReportSvc,
	}
}

func (p *SyslogPipeline) Handle(ctx context.Context, payload []byte, addr net.Addr, receivedAt time.Time) error {
	raw := string(payload)
	match, err := p.parseSyslog(ctx, raw, receivedAt)
	if err != nil {
		return err
	}

	var (
		event         *domain.ClientEvent
		parseStatus   = "ignored"
		logTime       *time.Time
		matchedRuleID *uint64
	)
	if match != nil && match.Event != nil {
		event = match.Event
		parseStatus = "parsed"
		logTime = timePointer(event.EventTime)
		if match.Rule != nil {
			matchedRuleID = &match.Rule.ID
		}
	}

	message := domain.SyslogMessage{
		ReceivedAt:        receivedAt,
		LogTime:           logTime,
		RawMessage:        raw,
		SourceIP:          sourceIPFromAddr(addr),
		Protocol:          "udp",
		ParseStatus:       parseStatus,
		MatchedRuleID:     matchedRuleID,
		RetentionExpireAt: receivedAt.Add(time.Duration(p.retentionDays) * 24 * time.Hour),
	}

	if p.messages != nil {
		if err := p.messages.Save(ctx, &message); err != nil {
			return err
		}
	}
	if event == nil {
		return nil
	}
	event.SyslogMessageID = message.ID
	event.MatchStatus = "unmatched"

	employee, err := p.matchEmployee(ctx, event.StationMac)
	if err != nil {
		return err
	}
	if employee != nil {
		event.MatchStatus = "matched"
		event.MatchedEmployeeID = &employee.ID
	}

	if p.events != nil {
		if err := p.events.Save(ctx, event); err != nil {
			return err
		}
	}

	if employee == nil {
		return nil
	}

	record, err := p.loadAttendanceRecord(ctx, employee.ID, event.EventDate)
	if err != nil {
		return err
	}

	result := p.attendanceProc.ApplyEvent(*record, *employee, *event)
	if !result.ClockInNeedsReport {
		return p.saveAttendanceOnly(ctx, result.Record)
	}

	reportType := "clock_in"
	reportTime := event.EventTime
	idempotencyKey := p.reportSvc.BuildIdempotencyKey(result.Record, reportType, reportTime)
	existing, err := p.reports.FindByIdempotencyKey(ctx, idempotencyKey)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return err
	}
	if existing != nil {
		return p.saveAttendanceOnly(ctx, result.Record)
	}

	if p.db != nil {
		return p.saveAttendanceAndReportWithTx(ctx, result.Record, reportType, reportTime)
	}

	if p.attendance != nil {
		if err := p.attendance.Save(ctx, &result.Record); err != nil {
			return err
		}
	}

	report := p.reportSvc.CreatePendingReport(result.Record, reportType, reportTime)
	if p.reports != nil {
		if err := p.reports.Save(ctx, &report); err != nil {
			return err
		}
	}

	return nil
}

func (p *SyslogPipeline) Preview(raw string, receivedAt time.Time) (*domain.ClientEvent, error) {
	match, err := p.parseSyslog(context.Background(), raw, receivedAt)
	if err != nil || match == nil {
		return nil, err
	}
	return match.Event, nil
}

func (p *SyslogPipeline) parseSyslog(ctx context.Context, raw string, receivedAt time.Time) (*SyslogRuleMatchResult, error) {
	if p.rules == nil {
		event, err := parser.ParseAPSyslog(raw, receivedAt)
		if err != nil {
			return nil, nil
		}
		return &SyslogRuleMatchResult{Event: &event}, nil
	}

	rules, err := p.rules.ListEnabled(ctx)
	if err != nil {
		return nil, err
	}
	if len(rules) == 0 {
		return nil, nil
	}

	return matchSyslogRule(raw, receivedAt, rules)
}

func (p *SyslogPipeline) saveAttendanceOnly(ctx context.Context, record domain.AttendanceRecord) error {
	if p.attendance == nil {
		return nil
	}

	return p.attendance.Save(ctx, &record)
}

func (p *SyslogPipeline) matchEmployee(ctx context.Context, stationMac string) (*domain.Employee, error) {
	if p.employees == nil {
		return nil, nil
	}

	employee, err := p.employees.FindByMACAddress(ctx, stationMac)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}

	return employee, nil
}

func (p *SyslogPipeline) loadAttendanceRecord(ctx context.Context, employeeID uint64, attendanceDate time.Time) (*domain.AttendanceRecord, error) {
	if p.attendance == nil {
		return &domain.AttendanceRecord{
			EmployeeID:      employeeID,
			AttendanceDate:  attendanceDate,
			ClockInStatus:   "pending",
			ClockOutStatus:  "pending",
			ExceptionStatus: "none",
			SourceMode:      "syslog",
			Version:         1,
		}, nil
	}

	record, err := p.attendance.FindByEmployeeAndDate(ctx, employeeID, attendanceDate)
	if err != nil {
		if !errors.Is(err, sql.ErrNoRows) {
			return nil, err
		}

		return &domain.AttendanceRecord{
			EmployeeID:      employeeID,
			AttendanceDate:  attendanceDate,
			ClockInStatus:   "pending",
			ClockOutStatus:  "pending",
			ExceptionStatus: "none",
			SourceMode:      "syslog",
			Version:         1,
		}, nil
	}

	return record, nil
}

type attendanceTxRepository interface {
	WithTx(*sql.Tx) repository.AttendanceRepository
}

type reportTxRepository interface {
	WithTx(*sql.Tx) repository.ReportRepository
}

func (p *SyslogPipeline) saveAttendanceAndReportWithTx(ctx context.Context, record domain.AttendanceRecord, reportType string, reportTime time.Time) (err error) {
	tx, err := p.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}

	defer func() {
		if err != nil {
			_ = tx.Rollback()
		}
	}()

	attendanceRepo, ok := p.attendance.(attendanceTxRepository)
	if !ok {
		return errors.New("attendance repository does not support tx")
	}
	reportRepo, ok := p.reports.(reportTxRepository)
	if !ok {
		return errors.New("report repository does not support tx")
	}

	if err = attendanceRepo.WithTx(tx).Save(ctx, &record); err != nil {
		return err
	}

	report := p.reportSvc.CreatePendingReport(record, reportType, reportTime)
	if err = reportRepo.WithTx(tx).Save(ctx, &report); err != nil {
		return err
	}

	if err = tx.Commit(); err != nil {
		return err
	}

	return nil
}

func sourceIPFromAddr(addr net.Addr) string {
	if addr == nil {
		return ""
	}

	if udpAddr, ok := addr.(*net.UDPAddr); ok {
		if udpAddr.IP != nil {
			return udpAddr.IP.String()
		}
	}

	host, _, err := net.SplitHostPort(addr.String())
	if err == nil {
		return strings.Trim(host, "[]")
	}

	return addr.String()
}
