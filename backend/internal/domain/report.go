package domain

import "time"

type AttendanceReport struct {
	ID                 uint64
	AttendanceRecordID uint64
	ReportType         string
	IdempotencyKey     string
	PayloadJSON        string
	TargetURL          string
	ReportStatus       string
	ResponseCode       *int
	ResponseBody       string
	ReportedAt         *time.Time
	RetryCount         uint32
}

type SystemSetting struct {
	ID           uint64
	SettingKey   string
	SettingValue string
	UpdatedAt    time.Time
}
