package service

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"net"
	"strings"
	"time"

	"syslog/internal/domain"
	"syslog/internal/repository"
)

var ErrInvalidDebugInput = errors.New("invalid debug input")

var debugNow = time.Now

type DebugSyslogInjectInput struct {
	RawMessage string
	ReceivedAt string
}

type DebugSyslogInjectResult struct {
	Accepted    bool
	ReceivedAt  time.Time
	ParseStatus string
	ParseError  string
}

type DebugAttendanceDispatchInput struct {
	ReportType string
}

type DebugAttendanceDispatchResult struct {
	Record domain.AttendanceRecord
	Report domain.AttendanceReport
}

type DebugAdminService struct {
	location       *time.Location
	pipeline       *SyslogPipeline
	attendanceRepo repository.AttendanceRepository
	reportRepo     repository.ReportRepository
	reportSvc      *ReportService
	dispatcher     *AttendanceReportDispatcher
}

func NewDebugAdminService(
	location *time.Location,
	pipeline *SyslogPipeline,
	attendanceRepo repository.AttendanceRepository,
	reportRepo repository.ReportRepository,
	dispatcher *AttendanceReportDispatcher,
	reportSvc *ReportService,
) *DebugAdminService {
	if reportSvc == nil {
		reportSvc = NewReportService()
	}
	return &DebugAdminService{
		location:       location,
		pipeline:       pipeline,
		attendanceRepo: attendanceRepo,
		reportRepo:     reportRepo,
		reportSvc:      reportSvc,
		dispatcher:     dispatcher,
	}
}

func (s *DebugAdminService) InjectSyslog(ctx context.Context, input DebugSyslogInjectInput) (*DebugSyslogInjectResult, error) {
	if s.pipeline == nil {
		return nil, errors.New("syslog pipeline is required")
	}

	rawMessage := strings.TrimSpace(input.RawMessage)
	if rawMessage == "" {
		return nil, fmt.Errorf("%w: rawMessage is required", ErrInvalidDebugInput)
	}

	receivedAt, err := parseDebugReceivedAt(input.ReceivedAt, s.location)
	if err != nil {
		return nil, err
	}

	parseStatus := "parsed"
	parseError := ""
	preview, err := s.pipeline.Preview(rawMessage, receivedAt)
	if err != nil {
		parseStatus = "failed"
		parseError = err.Error()
	} else if preview == nil {
		parseStatus = "failed"
		parseError = "no matching syslog rule"
	}

	if err := s.pipeline.Handle(
		ctx,
		[]byte(rawMessage),
		&net.UDPAddr{IP: net.ParseIP("127.0.0.1"), Port: 0},
		receivedAt,
	); err != nil {
		return nil, err
	}

	return &DebugSyslogInjectResult{
		Accepted:    true,
		ReceivedAt:  receivedAt,
		ParseStatus: parseStatus,
		ParseError:  parseError,
	}, nil
}

func (s *DebugAdminService) DispatchAttendanceReport(ctx context.Context, attendanceID uint64, input DebugAttendanceDispatchInput) (*DebugAttendanceDispatchResult, error) {
	if s.attendanceRepo == nil {
		return nil, errors.New("attendance repository is required")
	}
	if s.reportRepo == nil {
		return nil, errors.New("report repository is required")
	}

	reportType := strings.TrimSpace(input.ReportType)
	if reportType != "clock_in" && reportType != "clock_out" {
		return nil, fmt.Errorf("%w: unsupported reportType: %s", ErrInvalidDebugInput, reportType)
	}

	record, err := s.attendanceRepo.FindByID(ctx, attendanceID)
	if err != nil {
		return nil, err
	}

	relevantTime, err := relevantAttendanceTime(*record, reportType)
	if err != nil {
		return nil, err
	}
	if s.dispatcher == nil {
		return nil, errors.New("attendance report dispatcher is required")
	}

	report := s.reportSvc.CreateManualPendingReport(*record, reportType, relevantTime, debugNow().UTC())
	if err := copyLatestDeleteRecordID(ctx, s.reportRepo, &report); err != nil {
		return nil, err
	}
	if err := s.reportRepo.Save(ctx, &report); err != nil {
		return nil, err
	}
	if err := s.dispatcher.DispatchReport(ctx, &report); err != nil {
		return nil, err
	}

	return &DebugAttendanceDispatchResult{
		Record: *record,
		Report: report,
	}, nil
}

func relevantAttendanceTime(record domain.AttendanceRecord, reportType string) (time.Time, error) {
	switch reportType {
	case "clock_in":
		if record.FirstConnectAt == nil {
			return time.Time{}, fmt.Errorf("%w: firstConnectAt is required for clock_in dispatch", ErrInvalidDebugInput)
		}
		return *record.FirstConnectAt, nil
	case "clock_out":
		if record.LastDisconnectAt == nil {
			return time.Time{}, fmt.Errorf("%w: lastDisconnectAt is required for clock_out dispatch", ErrInvalidDebugInput)
		}
		return *record.LastDisconnectAt, nil
	default:
		return time.Time{}, fmt.Errorf("%w: unsupported reportType: %s", ErrInvalidDebugInput, reportType)
	}
}

func parseDebugReceivedAt(value string, location *time.Location) (time.Time, error) {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return time.Time{}, fmt.Errorf("%w: receivedAt is required", ErrInvalidDebugInput)
	}

	loc := location
	if loc == nil {
		loc = time.Local
	}

	for _, layout := range []string{time.RFC3339, "2006-01-02T15:04:05", "2006-01-02T15:04"} {
		var (
			parsed time.Time
			err    error
		)
		if layout == time.RFC3339 {
			parsed, err = time.Parse(layout, trimmed)
		} else {
			parsed, err = time.ParseInLocation(layout, trimmed, loc)
		}
		if err == nil {
			return parsed.In(loc), nil
		}
	}

	return time.Time{}, fmt.Errorf("%w: invalid receivedAt", ErrInvalidDebugInput)
}

var _ = sql.ErrNoRows
