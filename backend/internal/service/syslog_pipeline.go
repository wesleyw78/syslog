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

const reportTargetURLSettingKey = "report_target_url"

type SyslogPipelineDeps struct {
	Messages       repository.SyslogMessageRepository
	Events         repository.ClientEventRepository
	Employees      repository.EmployeeRepository
	Attendance     repository.AttendanceRepository
	Reports        repository.ReportRepository
	Settings       repository.SystemSettingRepository
	RetentionDays  int
	AttendanceProc *AttendanceProcessor
	ReportSvc      *ReportService
}

type SyslogPipeline struct {
	messages       repository.SyslogMessageRepository
	events         repository.ClientEventRepository
	employees      repository.EmployeeRepository
	attendance     repository.AttendanceRepository
	reports        repository.ReportRepository
	settings       repository.SystemSettingRepository
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
		messages:       deps.Messages,
		events:         deps.Events,
		employees:      deps.Employees,
		attendance:     deps.Attendance,
		reports:        deps.Reports,
		settings:       deps.Settings,
		retentionDays:  deps.RetentionDays,
		attendanceProc: deps.AttendanceProc,
		reportSvc:      deps.ReportSvc,
	}
}

func (p *SyslogPipeline) Handle(ctx context.Context, payload []byte, addr net.Addr, receivedAt time.Time) error {
	raw := string(payload)
	event, parseErr := parser.ParseAPSyslog(raw, receivedAt)
	message := domain.SyslogMessage{
		ReceivedAt:        receivedAt,
		LogTime:           timePointer(receivedAt),
		RawMessage:        raw,
		SourceIP:          sourceIPFromAddr(addr),
		Protocol:          "udp",
		ParseStatus:       "parsed",
		RetentionExpireAt: receivedAt.Add(time.Duration(p.retentionDays) * 24 * time.Hour),
	}

	if parseErr != nil {
		message.ParseStatus = "failed"
		if p.messages != nil {
			if err := p.messages.Save(ctx, &message); err != nil {
				return err
			}
		}
		return nil
	}

	if p.messages != nil {
		if err := p.messages.Save(ctx, &message); err != nil {
			return err
		}
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
		if err := p.events.Save(ctx, &event); err != nil {
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

	result := p.attendanceProc.ApplyEvent(*record, *employee, event)
	if p.attendance != nil {
		if err := p.attendance.Save(ctx, &result.Record); err != nil {
			return err
		}
	}

	if !result.ClockInNeedsReport {
		return nil
	}

	reportType := "clock_in"
	reportTime := event.EventTime
	idempotencyKey := p.reportSvc.BuildIdempotencyKey(result.Record, reportType, reportTime)
	existing, err := p.reports.FindByIdempotencyKey(ctx, idempotencyKey)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return err
	}
	if existing != nil {
		return nil
	}

	targetURL, err := p.reportTargetURL(ctx)
	if err != nil {
		return err
	}

	report := p.reportSvc.CreatePendingReport(result.Record, reportType, reportTime, targetURL)
	if p.reports != nil {
		if err := p.reports.Save(ctx, &report); err != nil {
			return err
		}
	}

	return nil
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

func (p *SyslogPipeline) reportTargetURL(ctx context.Context) (string, error) {
	if p.settings == nil {
		return "", nil
	}

	setting, err := p.settings.GetByKey(ctx, reportTargetURLSettingKey)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return "", nil
		}
		return "", err
	}
	if setting == nil {
		return "", nil
	}

	return strings.TrimSpace(setting.SettingValue), nil
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
