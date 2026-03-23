package service

import (
	"context"
	"crypto/sha1"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"strconv"
	"strings"
	"time"

	"syslog/internal/domain"
	"syslog/internal/repository"
)

const (
	feishuAppIDSettingKey          = "feishu_app_id"
	feishuAppSecretSettingKey      = "feishu_app_secret"
	feishuLocationNameSettingKey   = "feishu_location_name"
	reportTimeoutSecondsSettingKey = "report_timeout_seconds"
	reportRetryLimitSettingKey     = "report_retry_limit"
	dispatchBatchLimit             = 20
	defaultRetryLimit              = 3
	defaultDispatchTimeout         = 10 * time.Second
)

type AttendanceReportDispatcherDeps struct {
	Reports      repository.ReportRepository
	Employees    repository.EmployeeRepository
	Settings     repository.SystemSettingRepository
	Client       FeishuAttendanceClient
	Location     *time.Location
	PollInterval time.Duration
}

type AttendanceReportDispatcher struct {
	reports      repository.ReportRepository
	employees    repository.EmployeeRepository
	settings     repository.SystemSettingRepository
	client       FeishuAttendanceClient
	location     *time.Location
	pollInterval time.Duration
}

type dispatchSettings struct {
	RetryLimit uint32
	Config     FeishuAttendanceConfig
}

func NewAttendanceReportDispatcher(deps AttendanceReportDispatcherDeps) *AttendanceReportDispatcher {
	client := deps.Client
	if client == nil {
		client = NewFeishuAttendanceHTTPClient(nil)
	}

	pollInterval := deps.PollInterval
	if pollInterval <= 0 {
		pollInterval = 15 * time.Second
	}

	return &AttendanceReportDispatcher{
		reports:      deps.Reports,
		employees:    deps.Employees,
		settings:     deps.Settings,
		client:       client,
		location:     deps.Location,
		pollInterval: pollInterval,
	}
}

func (d *AttendanceReportDispatcher) Run(ctx context.Context) error {
	if err := d.RunOnce(ctx); err != nil {
		return err
	}

	ticker := time.NewTicker(d.pollInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
			if err := d.RunOnce(ctx); err != nil {
				return err
			}
		}
	}
}

func (d *AttendanceReportDispatcher) RunOnce(ctx context.Context) error {
	if d.reports == nil {
		return nil
	}

	settings, err := d.loadDispatchSettings(ctx)
	if err != nil {
		return err
	}

	reports, err := d.reports.ListDispatchable(ctx, dispatchBatchLimit, settings.RetryLimit)
	if err != nil {
		return err
	}
	notificationReports, err := d.reports.ListNotificationDispatchable(ctx, dispatchBatchLimit, settings.RetryLimit)
	if err != nil {
		return err
	}

	for idx := range reports {
		report := reports[idx]
		if err := d.dispatchReport(ctx, settings.Config, &report); err != nil {
			return err
		}
	}
	for idx := range notificationReports {
		report := notificationReports[idx]
		if err := d.dispatchNotification(ctx, settings.Config, &report); err != nil {
			return err
		}
	}

	return nil
}

func (d *AttendanceReportDispatcher) DispatchReport(ctx context.Context, report *domain.AttendanceReport) error {
	if report == nil {
		return nil
	}
	settings, err := d.loadDispatchSettings(ctx)
	if err != nil {
		return err
	}
	return d.dispatchReport(ctx, settings.Config, report)
}

func (d *AttendanceReportDispatcher) loadDispatchSettings(ctx context.Context) (dispatchSettings, error) {
	result := dispatchSettings{
		RetryLimit: defaultRetryLimit,
		Config: FeishuAttendanceConfig{
			Timeout: defaultDispatchTimeout,
		},
	}
	if d.settings == nil {
		return result, nil
	}

	settings, err := d.settings.List(ctx)
	if err != nil {
		return result, err
	}
	values := make(map[string]string, len(settings))
	for _, item := range settings {
		values[item.SettingKey] = strings.TrimSpace(item.SettingValue)
	}

	result.Config.AppID = values[feishuAppIDSettingKey]
	result.Config.AppSecret = values[feishuAppSecretSettingKey]
	result.Config.LocationName = values[feishuLocationNameSettingKey]

	if timeoutSeconds, err := strconv.Atoi(values[reportTimeoutSecondsSettingKey]); err == nil && timeoutSeconds > 0 {
		result.Config.Timeout = time.Duration(timeoutSeconds) * time.Second
	}
	if retryLimit, err := strconv.Atoi(values[reportRetryLimitSettingKey]); err == nil && retryLimit > 0 {
		result.RetryLimit = uint32(retryLimit)
	}

	return result, nil
}

func (d *AttendanceReportDispatcher) dispatchReport(ctx context.Context, config FeishuAttendanceConfig, report *domain.AttendanceReport) error {
	if report == nil {
		return nil
	}

	log.Printf(
		"attendance report dispatch start: report_id=%d attendance_record_id=%d report_type=%s status=%s retry_count=%d delete_record_id=%s",
		report.ID,
		report.AttendanceRecordID,
		report.ReportType,
		report.ReportStatus,
		report.RetryCount,
		report.DeleteRecordID,
	)

	payload, err := d.parsePayload(report.PayloadJSON)
	if err != nil {
		return d.markFailed(ctx, report, nil, err)
	}
	if err := validateFeishuConfig(config); err != nil {
		return d.markFailed(ctx, report, nil, err)
	}
	if d.employees == nil {
		return d.markFailed(ctx, report, nil, errors.New("employee repository is required"))
	}

	employee, err := d.employees.FindByID(ctx, payload.EmployeeID)
	if err != nil {
		return d.markFailed(ctx, report, nil, fmt.Errorf("load employee: %w", err))
	}
	if strings.TrimSpace(employee.FeishuEmployeeID) == "" {
		return d.markFailed(ctx, report, nil, fmt.Errorf("employee %d missing feishu_employee_id", employee.ID))
	}

	if strings.TrimSpace(report.DeleteRecordID) != "" {
		log.Printf(
			"attendance report dispatch delete start: report_id=%d attendance_record_id=%d report_type=%s delete_record_id=%s",
			report.ID,
			report.AttendanceRecordID,
			report.ReportType,
			report.DeleteRecordID,
		)
		deleteResult, deleteErr := d.client.DeleteFlows(ctx, config, []string{report.DeleteRecordID})
		if deleteErr != nil {
			return d.markFailed(ctx, report, deleteResult, deleteErr)
		}
		report.DeleteRecordID = ""
		report.ResponseCode = intPointer(deleteResult.StatusCode)
		report.ResponseBody = deleteResult.ResponseBody

		if payload.Action == "clear" {
			now := time.Now().UTC()
			report.ReportStatus = "success"
			report.NotificationStatus = skippedNotificationStatus
			report.ReportedAt = &now
			log.Printf(
				"attendance report dispatch clear succeeded: report_id=%d attendance_record_id=%d report_type=%s status=%s",
				report.ID,
				report.AttendanceRecordID,
				report.ReportType,
				report.ReportStatus,
			)
			return d.reports.Save(ctx, report)
		}

		if err := d.reports.Save(ctx, report); err != nil {
			return err
		}
	}

	if payload.Action == "clear" {
		now := time.Now().UTC()
		report.ReportStatus = "success"
		report.NotificationStatus = skippedNotificationStatus
		report.ReportedAt = &now
		log.Printf(
			"attendance report dispatch clear succeeded without create: report_id=%d attendance_record_id=%d report_type=%s status=%s",
			report.ID,
			report.AttendanceRecordID,
			report.ReportType,
			report.ReportStatus,
		)
		return d.reports.Save(ctx, report)
	}

	checkTime, err := unixSecondsString(payload.Timestamp)
	if err != nil {
		return d.markFailed(ctx, report, nil, err)
	}

	createInput := FeishuAttendanceCreateInput{
		EmployeeType: "employee_id",
		UserID:       employee.FeishuEmployeeID,
		CreatorID:    employee.FeishuEmployeeID,
		LocationName: config.LocationName,
		CheckTime:    checkTime,
		Comment:      commentForReportType(payload.ReportType),
		ExternalID:   report.IdempotencyKey,
		IdempotentID: report.IdempotencyKey,
		Type:         7,
	}

	log.Printf(
		"attendance report dispatch create start: report_id=%d attendance_record_id=%d report_type=%s employee_id=%d feishu_employee_id=%s",
		report.ID,
		report.AttendanceRecordID,
		report.ReportType,
		employee.ID,
		employee.FeishuEmployeeID,
	)
	createResult, createErr := d.client.CreateFlow(ctx, config, createInput)
	if createErr != nil {
		return d.markFailed(ctx, report, createResult, createErr)
	}

	now := time.Now().UTC()
	report.ExternalRecordID = createResult.RecordID
	report.ReportStatus = "success"
	report.ResponseCode = intPointer(createResult.StatusCode)
	report.ResponseBody = createResult.ResponseBody
	report.ReportedAt = &now
	if report.NotificationStatus == "" {
		report.NotificationStatus = pendingNotificationStatus
	}

	log.Printf(
		"attendance report dispatch succeeded: report_id=%d attendance_record_id=%d report_type=%s external_record_id=%s status=%s response_code=%d",
		report.ID,
		report.AttendanceRecordID,
		report.ReportType,
		report.ExternalRecordID,
		report.ReportStatus,
		createResult.StatusCode,
	)
	if err := d.dispatchNotificationForReport(ctx, config, report, payload, employee); err != nil {
		return err
	}
	return d.reports.Save(ctx, report)
}

func (d *AttendanceReportDispatcher) dispatchNotification(ctx context.Context, config FeishuAttendanceConfig, report *domain.AttendanceReport) error {
	if report == nil || d.employees == nil {
		return nil
	}
	if report.ReportStatus != "success" {
		return nil
	}
	if report.NotificationStatus != pendingNotificationStatus && report.NotificationStatus != "failed" {
		return nil
	}

	payload, err := d.parsePayload(report.PayloadJSON)
	if err != nil {
		return err
	}
	if !shouldNotifyForReport(payload) {
		report.NotificationStatus = skippedNotificationStatus
		return d.reports.Save(ctx, report)
	}

	employee, err := d.employees.FindByID(ctx, payload.EmployeeID)
	if err != nil {
		return err
	}

	if err := d.dispatchNotificationForReport(ctx, config, report, payload, employee); err != nil {
		return err
	}
	return d.reports.Save(ctx, report)
}

func (d *AttendanceReportDispatcher) dispatchNotificationForReport(ctx context.Context, config FeishuAttendanceConfig, report *domain.AttendanceReport, payload attendanceReportPayload, employee *domain.Employee) error {
	if report == nil || employee == nil {
		return nil
	}
	if !shouldNotifyForReport(payload) {
		report.NotificationStatus = skippedNotificationStatus
		return nil
	}

	text, err := buildAttendanceNotificationText(*employee, payload, config.LocationName, d.location)
	if err != nil {
		return err
	}
	result, sendErr := d.client.SendTextMessage(ctx, config, FeishuSendMessageInput{
		ReceiveIDType: "user_id",
		ReceiveID:     employee.FeishuEmployeeID,
		Text:          text,
		UUID:          notificationUUID(report.IdempotencyKey),
	})
	if sendErr != nil {
		report.NotificationStatus = "failed"
		report.NotificationRetryCount++
		now := time.Now().UTC()
		report.NotificationSentAt = &now
		if result != nil {
			report.NotificationResponseCode = intPointer(result.StatusCode)
			report.NotificationResponseBody = result.ResponseBody
			report.NotificationMessageID = result.MessageID
		} else {
			report.NotificationResponseBody = sendErr.Error()
		}
		log.Printf(
			"attendance report notification failed: report_id=%d attendance_record_id=%d report_type=%s retry_count=%d err=%v response_code=%s response=%s",
			report.ID,
			report.AttendanceRecordID,
			report.ReportType,
			report.NotificationRetryCount,
			sendErr,
			formatOptionalInt(report.NotificationResponseCode),
			sanitizeFeishuResponseBody(report.NotificationResponseBody),
		)
		return nil
	}

	now := time.Now().UTC()
	report.NotificationStatus = "success"
	report.NotificationSentAt = &now
	report.NotificationRetryCount = 0
	report.NotificationResponseCode = intPointer(result.StatusCode)
	report.NotificationResponseBody = result.ResponseBody
	report.NotificationMessageID = result.MessageID
	log.Printf(
		"attendance report notification succeeded: report_id=%d attendance_record_id=%d report_type=%s message_id=%s response_code=%d",
		report.ID,
		report.AttendanceRecordID,
		report.ReportType,
		report.NotificationMessageID,
		result.StatusCode,
	)
	return nil
}

func (d *AttendanceReportDispatcher) parsePayload(raw string) (attendanceReportPayload, error) {
	var payload attendanceReportPayload
	if err := json.Unmarshal([]byte(raw), &payload); err != nil {
		return attendanceReportPayload{}, fmt.Errorf("decode attendance report payload: %w", err)
	}
	return payload, nil
}

func (d *AttendanceReportDispatcher) markFailed(ctx context.Context, report *domain.AttendanceReport, result interface{}, dispatchErr error) error {
	if report == nil {
		return dispatchErr
	}

	report.ReportStatus = "failed"
	report.RetryCount++
	now := time.Now().UTC()
	report.ReportedAt = &now
	report.ResponseBody = dispatchErr.Error()

	switch typed := result.(type) {
	case *FeishuCreateFlowResult:
		if typed != nil {
			report.ResponseCode = intPointer(typed.StatusCode)
			if strings.TrimSpace(typed.ResponseBody) != "" {
				report.ResponseBody = typed.ResponseBody
			}
		}
	case *FeishuDeleteFlowsResult:
		if typed != nil {
			report.ResponseCode = intPointer(typed.StatusCode)
			if strings.TrimSpace(typed.ResponseBody) != "" {
				report.ResponseBody = typed.ResponseBody
			}
		}
	}

	if d.reports == nil {
		return dispatchErr
	}
	log.Printf(
		"attendance report dispatch failed: report_id=%d attendance_record_id=%d report_type=%s retry_count=%d err=%v response_code=%s response=%s",
		report.ID,
		report.AttendanceRecordID,
		report.ReportType,
		report.RetryCount,
		dispatchErr,
		formatOptionalInt(report.ResponseCode),
		sanitizeFeishuResponseBody(report.ResponseBody),
	)
	if err := d.reports.Save(ctx, report); err != nil {
		return err
	}

	return nil
}

func validateFeishuConfig(config FeishuAttendanceConfig) error {
	if strings.TrimSpace(config.AppID) == "" {
		return errors.New("missing feishu_app_id")
	}
	if strings.TrimSpace(config.AppSecret) == "" {
		return errors.New("missing feishu_app_secret")
	}
	if strings.TrimSpace(config.LocationName) == "" {
		return errors.New("missing feishu_location_name")
	}
	return nil
}

func unixSecondsString(value *string) (string, error) {
	if value == nil || strings.TrimSpace(*value) == "" {
		return "", errors.New("attendance report timestamp is required")
	}

	parsed, err := time.Parse(time.RFC3339, *value)
	if err != nil {
		return "", fmt.Errorf("parse attendance report timestamp: %w", err)
	}

	return strconv.FormatInt(parsed.UTC().Unix(), 10), nil
}

func commentForReportType(reportType string) string {
	switch reportType {
	case "clock_out":
		return "Syslog 导入下班打卡"
	default:
		return "Syslog 导入上班打卡"
	}
}

func shouldNotifyForReport(payload attendanceReportPayload) bool {
	return payload.Action != "clear" && (payload.ReportType == "clock_in" || payload.ReportType == "clock_out")
}

func buildAttendanceNotificationText(employee domain.Employee, payload attendanceReportPayload, locationName string, location *time.Location) (string, error) {
	reportTypeLabel := "上班打卡"
	if payload.ReportType == "clock_out" {
		reportTypeLabel = "下班打卡"
	}

	if payload.Timestamp == nil || strings.TrimSpace(*payload.Timestamp) == "" {
		return "", errors.New("attendance notification timestamp is required")
	}
	parsed, err := time.Parse(time.RFC3339, *payload.Timestamp)
	if err != nil {
		return "", fmt.Errorf("parse attendance notification timestamp: %w", err)
	}
	if location == nil {
		location = time.Local
	}
	localTime := parsed.In(location)

	return fmt.Sprintf(
		"【打卡成功通知】\n姓名：%s\n类型：%s\n日期：%s\n时间：%s\n地点：%s\n状态：已成功同步到飞书考勤",
		employee.Name,
		reportTypeLabel,
		localTime.Format("2006-01-02"),
		localTime.Format("15:04:05"),
		locationName,
	), nil
}

func notificationUUID(idempotencyKey string) string {
	digest := sha1.Sum([]byte(strings.TrimSpace(idempotencyKey)))
	return "notify-" + hex.EncodeToString(digest[:])
}

func intPointer(value int) *int {
	return &value
}

func formatOptionalInt(value *int) string {
	if value == nil {
		return "nil"
	}
	return strconv.Itoa(*value)
}

var _ = sql.ErrNoRows
