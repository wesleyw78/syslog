package service

import (
	"strings"
	"time"

	"syslog/internal/domain"
)

type AttendanceProcessResult struct {
	Record             domain.AttendanceRecord
	ClockInNeedsReport bool
}

type AttendanceProcessor struct{}

func NewAttendanceProcessor() *AttendanceProcessor {
	return &AttendanceProcessor{}
}

func (p *AttendanceProcessor) ApplyEvent(record domain.AttendanceRecord, employee domain.Employee, event domain.ClientEvent) AttendanceProcessResult {
	result := AttendanceProcessResult{Record: record}

	if result.Record.EmployeeID == 0 && employee.ID != 0 {
		result.Record.EmployeeID = employee.ID
	}

	switch strings.ToLower(event.EventType) {
	case "connect":
		if result.Record.FirstConnectAt == nil {
			result.Record.FirstConnectAt = timePointer(event.EventTime)
			result.ClockInNeedsReport = true
			return result
		}

		if event.EventTime.Before(*result.Record.FirstConnectAt) {
			result.Record.FirstConnectAt = timePointer(event.EventTime)
			result.ClockInNeedsReport = true
		}
	case "disconnect":
		if result.Record.LastDisconnectAt == nil || event.EventTime.After(*result.Record.LastDisconnectAt) {
			result.Record.LastDisconnectAt = timePointer(event.EventTime)
		}
	}

	return result
}

func timePointer(value time.Time) *time.Time {
	copied := value
	return &copied
}
