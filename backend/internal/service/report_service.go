package service

import (
	"encoding/json"
	"fmt"
	"time"

	"syslog/internal/domain"
)

const pendingReportStatus = "pending"

type ReportService struct{}

type attendanceReportPayload struct {
	AttendanceRecordID uint64 `json:"attendanceRecordId"`
	EmployeeID         uint64 `json:"employeeId"`
	AttendanceDate     string `json:"attendanceDate"`
	ReportType         string `json:"reportType"`
	Timestamp          string `json:"timestamp"`
	Version            uint32 `json:"version"`
}

func NewReportService() *ReportService {
	return &ReportService{}
}

func (s *ReportService) BuildIdempotencyKey(record domain.AttendanceRecord, reportType string, relevantTime time.Time) string {
	return fmt.Sprintf(
		"attendance-report/%s/%s/%s/v%d",
		attendanceIdentity(record),
		reportType,
		relevantTime.UTC().Format(time.RFC3339),
		record.Version,
	)
}

func (s *ReportService) CreatePendingReport(record domain.AttendanceRecord, reportType string, relevantTime time.Time, targetURL string) domain.AttendanceReport {
	payloadJSON, _ := json.Marshal(attendanceReportPayload{
		AttendanceRecordID: record.ID,
		EmployeeID:         record.EmployeeID,
		AttendanceDate:     record.AttendanceDate.Format("2006-01-02"),
		ReportType:         reportType,
		Timestamp:          relevantTime.UTC().Format(time.RFC3339),
		Version:            record.Version,
	})

	return domain.AttendanceReport{
		AttendanceRecordID: record.ID,
		ReportType:         reportType,
		IdempotencyKey:     s.BuildIdempotencyKey(record, reportType, relevantTime),
		PayloadJSON:        string(payloadJSON),
		TargetURL:          targetURL,
		ReportStatus:       pendingReportStatus,
	}
}

func attendanceIdentity(record domain.AttendanceRecord) string {
	if record.ID != 0 {
		return fmt.Sprintf("record-%d", record.ID)
	}

	return fmt.Sprintf(
		"employee-%d-%s",
		record.EmployeeID,
		record.AttendanceDate.Format("2006-01-02"),
	)
}
