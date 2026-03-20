package domain

import "time"

type Employee struct {
	ID         uint64
	EmployeeNo string
	SystemNo   string
	Name       string
	Status     string
	CreatedAt  time.Time
	UpdatedAt  time.Time
}

type EmployeeDevice struct {
	ID          uint64
	EmployeeID  uint64
	MacAddress  string
	DeviceLabel string
	Status      string
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

type AttendanceRecord struct {
	ID               uint64
	EmployeeID       uint64
	AttendanceDate   time.Time
	FirstConnectAt   *time.Time
	LastDisconnectAt *time.Time
	ClockInStatus    string
	ClockOutStatus   string
	ExceptionStatus  string
	SourceMode       string
	Version          uint32
	LastCalculatedAt *time.Time
}
