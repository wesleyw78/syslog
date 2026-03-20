package domain

import "time"

type SyslogMessage struct {
	ID               uint64
	DeviceIdentifier string
	RawMessage       string
	SourceIP         string
	ReceivedAt       time.Time
	CreatedAt        time.Time
}

type ClientEvent struct {
	ID               uint64
	SyslogMessageID  *uint64
	EmployeeID       *uint64
	EmployeeDeviceID *uint64
	EventType        string
	EventTime        time.Time
	Payload          string
	CreatedAt        time.Time
}
