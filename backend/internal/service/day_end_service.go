package service

import (
	"time"

	"syslog/internal/domain"
)

const (
	clockOutStatusPending = "pending"
	clockOutStatusMissing = "missing"
	clockOutStatusReady   = "ready"

	exceptionStatusNone              = "none"
	exceptionStatusMissingDisconnect = "missing_disconnect"
)

type DayEndService struct{}

func NewDayEndService() *DayEndService {
	return &DayEndService{}
}

func FinalizeForDay(record domain.AttendanceRecord, now time.Time) domain.AttendanceRecord {
	return NewDayEndService().FinalizeForDay(record, now)
}

func (s *DayEndService) FinalizeForDay(record domain.AttendanceRecord, now time.Time) domain.AttendanceRecord {
	result := record
	result.LastCalculatedAt = timePointer(now)

	if result.LastDisconnectAt == nil {
		result.ClockOutStatus = clockOutStatusMissing
		result.ExceptionStatus = exceptionStatusMissingDisconnect
		return result
	}

	if result.ClockOutStatus == "" || result.ClockOutStatus == clockOutStatusPending || result.ClockOutStatus == clockOutStatusMissing {
		result.ClockOutStatus = clockOutStatusReady
	}

	if result.ExceptionStatus == "" || result.ExceptionStatus == exceptionStatusMissingDisconnect {
		result.ExceptionStatus = exceptionStatusNone
	}

	return result
}
