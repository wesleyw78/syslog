package service

import (
	"testing"
	"time"

	"syslog/internal/domain"
)

func TestFirstConnectCreatesClockIn(t *testing.T) {
	processor := NewAttendanceProcessor()
	eventTime := time.Date(2026, 3, 21, 8, 1, 0, 0, time.FixedZone("CST", 8*3600))

	record := domain.AttendanceRecord{
		AttendanceDate: time.Date(2026, 3, 21, 0, 0, 0, 0, time.FixedZone("CST", 8*3600)),
		Version:        1,
	}
	employee := domain.Employee{ID: 42}
	event := domain.ClientEvent{
		EventType:  "connect",
		StationMac: "94:89:78:55:9a:f3",
		EventTime:  eventTime,
	}

	result := processor.ApplyEvent(record, employee, event)

	if result.Record.FirstConnectAt == nil {
		t.Fatal("expected clock-in time to be set")
	}
	if !result.Record.FirstConnectAt.Equal(eventTime) {
		t.Fatalf("expected first connect %s, got %s", eventTime, result.Record.FirstConnectAt)
	}
	if !result.ClockInNeedsReport {
		t.Fatal("expected immediate clock-in report")
	}
	if result.Record.EmployeeID != employee.ID {
		t.Fatalf("expected employee id %d, got %d", employee.ID, result.Record.EmployeeID)
	}
	if result.Record.LastCalculatedAt != nil {
		t.Fatalf("expected LastCalculatedAt to remain nil, got %v", result.Record.LastCalculatedAt)
	}
}

func TestLaterConnectDoesNotOverwriteEarlierFirstConnect(t *testing.T) {
	processor := NewAttendanceProcessor()
	firstConnect := time.Date(2026, 3, 21, 8, 1, 0, 0, time.FixedZone("CST", 8*3600))
	laterConnect := firstConnect.Add(15 * time.Minute)

	record := domain.AttendanceRecord{
		AttendanceDate: time.Date(2026, 3, 21, 0, 0, 0, 0, time.FixedZone("CST", 8*3600)),
		FirstConnectAt: &firstConnect,
		Version:        2,
	}

	result := processor.ApplyEvent(record, domain.Employee{ID: 42}, domain.ClientEvent{
		EventType: "connect",
		EventTime: laterConnect,
	})

	if result.Record.FirstConnectAt == nil {
		t.Fatal("expected first connect to remain set")
	}
	if !result.Record.FirstConnectAt.Equal(firstConnect) {
		t.Fatalf("expected first connect to remain %s, got %s", firstConnect, result.Record.FirstConnectAt)
	}
	if result.ClockInNeedsReport {
		t.Fatal("expected no new clock-in report for a later connect")
	}
}

func TestDisconnectKeepsLatestDisconnect(t *testing.T) {
	processor := NewAttendanceProcessor()
	existingDisconnect := time.Date(2026, 3, 21, 18, 10, 0, 0, time.FixedZone("CST", 8*3600))
	latestDisconnect := existingDisconnect.Add(20 * time.Minute)

	record := domain.AttendanceRecord{
		AttendanceDate:   time.Date(2026, 3, 21, 0, 0, 0, 0, time.FixedZone("CST", 8*3600)),
		LastDisconnectAt: &existingDisconnect,
		Version:          3,
	}

	result := processor.ApplyEvent(record, domain.Employee{ID: 42}, domain.ClientEvent{
		EventType: "disconnect",
		EventTime: latestDisconnect,
	})

	if result.Record.LastDisconnectAt == nil {
		t.Fatal("expected last disconnect to remain set")
	}
	if !result.Record.LastDisconnectAt.Equal(latestDisconnect) {
		t.Fatalf("expected last disconnect %s, got %s", latestDisconnect, result.Record.LastDisconnectAt)
	}
}

func TestApplyEventDoesNotChangeLastCalculatedAt(t *testing.T) {
	processor := NewAttendanceProcessor()
	existingCalculatedAt := time.Date(2026, 3, 21, 7, 59, 0, 0, time.FixedZone("CST", 8*3600))
	eventTime := time.Date(2026, 3, 21, 8, 1, 0, 0, time.FixedZone("CST", 8*3600))

	record := domain.AttendanceRecord{
		AttendanceDate:   time.Date(2026, 3, 21, 0, 0, 0, 0, time.FixedZone("CST", 8*3600)),
		LastCalculatedAt: &existingCalculatedAt,
	}

	result := processor.ApplyEvent(record, domain.Employee{ID: 42}, domain.ClientEvent{
		EventType: "connect",
		EventTime: eventTime,
	})

	if result.Record.LastCalculatedAt == nil {
		t.Fatal("expected LastCalculatedAt to remain set")
	}
	if !result.Record.LastCalculatedAt.Equal(existingCalculatedAt) {
		t.Fatalf("expected LastCalculatedAt to remain %s, got %s", existingCalculatedAt, result.Record.LastCalculatedAt)
	}
}
