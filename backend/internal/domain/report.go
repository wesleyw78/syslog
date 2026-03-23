package domain

import "time"

type AttendanceReport struct {
	ID                       uint64
	AttendanceRecordID       uint64
	ReportType               string
	IdempotencyKey           string
	PayloadJSON              string
	TargetURL                string
	ExternalRecordID         string
	DeleteRecordID           string
	ReportStatus             string
	ResponseCode             *int
	ResponseBody             string
	NotificationStatus       string
	NotificationMessageID    string
	NotificationResponseCode *int
	NotificationResponseBody string
	NotificationSentAt       *time.Time
	NotificationRetryCount   uint32
	ReportedAt               *time.Time
	RetryCount               uint32
}

type SystemSetting struct {
	ID           uint64
	SettingKey   string
	SettingValue string
	UpdatedAt    time.Time
}

type DayEndRun struct {
	ID           uint64
	BusinessDate time.Time
	CutoffTime   string
	ExecutedAt   time.Time
}
