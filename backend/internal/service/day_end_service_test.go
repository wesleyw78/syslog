package service

import (
	"testing"
	"time"

	"syslog/internal/domain"
)

func TestFinalizeMarksMissingDisconnect(t *testing.T) {
	now := time.Date(2026, 3, 21, 23, 59, 0, 0, time.FixedZone("CST", 8*3600))

	result := FinalizeForDay(domain.AttendanceRecord{
		ClockOutStatus: "pending",
	}, now)

	if result.ExceptionStatus != exceptionStatusMissingDisconnect {
		t.Fatalf("expected missing disconnect exception, got %s", result.ExceptionStatus)
	}
	if result.ClockOutStatus != clockOutStatusMissing {
		t.Fatalf("expected missing clock-out status, got %s", result.ClockOutStatus)
	}
}

func TestFinalizeMarksDisconnectAsDone(t *testing.T) {
	now := time.Date(2026, 3, 21, 23, 59, 0, 0, time.FixedZone("CST", 8*3600))
	disconnectAt := now.Add(-2 * time.Hour)

	result := FinalizeForDay(domain.AttendanceRecord{
		LastDisconnectAt: &disconnectAt,
		ClockOutStatus:   "pending",
	}, now)

	if result.ExceptionStatus == exceptionStatusMissingDisconnect {
		t.Fatalf("expected non-missing exception status, got %s", result.ExceptionStatus)
	}
	if result.ClockOutStatus != clockOutStatusDone {
		t.Fatalf("expected done clock-out status, got %s", result.ClockOutStatus)
	}
}
