package domain

import "time"

type AttendanceReport struct {
	ID                 uint64     `json:"id"`
	AttendanceRecordID uint64     `json:"attendanceRecordId"`
	ReportType         string     `json:"reportType"`
	IdempotencyKey     string     `json:"idempotencyKey"`
	PayloadJSON        string     `json:"payloadJson"`
	TargetURL          string     `json:"targetUrl"`
	ReportStatus       string     `json:"reportStatus"`
	ResponseCode       *int       `json:"responseCode,omitempty"`
	ResponseBody       string     `json:"responseBody"`
	ReportedAt         *time.Time `json:"reportedAt,omitempty"`
	RetryCount         uint32     `json:"retryCount"`
}

type SystemSetting struct {
	ID           uint64    `json:"id"`
	SettingKey   string    `json:"settingKey"`
	SettingValue string    `json:"settingValue"`
	UpdatedAt    time.Time `json:"updatedAt"`
}
