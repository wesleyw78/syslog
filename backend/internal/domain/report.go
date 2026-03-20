package domain

import "time"

type AttendanceReport struct {
	ID          uint64
	ReportDate  time.Time
	EmployeeID  uint64
	Status      string
	Summary     string
	GeneratedAt *time.Time
	CreatedAt   time.Time
}

type SystemSetting struct {
	Key         string
	Value       string
	Description string
	UpdatedAt   time.Time
}
