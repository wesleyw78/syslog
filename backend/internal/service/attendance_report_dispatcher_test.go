package service

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"syslog/internal/domain"
	"syslog/internal/repository"
)

type dispatcherEmployeeRepo struct {
	employee *domain.Employee
}

func (r *dispatcherEmployeeRepo) FindByMACAddress(context.Context, string) (*domain.Employee, error) {
	return nil, sql.ErrNoRows
}

func (r *dispatcherEmployeeRepo) FindByID(context.Context, uint64) (*domain.Employee, error) {
	if r.employee == nil {
		return nil, sql.ErrNoRows
	}
	copied := *r.employee
	return &copied, nil
}

func (r *dispatcherEmployeeRepo) List(context.Context) ([]domain.Employee, error) { return nil, nil }
func (r *dispatcherEmployeeRepo) Create(context.Context, *domain.Employee) error  { return nil }
func (r *dispatcherEmployeeRepo) Update(context.Context, *domain.Employee) error  { return nil }
func (r *dispatcherEmployeeRepo) Disable(context.Context, uint64) error           { return nil }
func (r *dispatcherEmployeeRepo) ReplaceDevices(context.Context, uint64, []domain.EmployeeDevice) error {
	return nil
}
func (r *dispatcherEmployeeRepo) DisableDevicesByEmployeeID(context.Context, uint64) error {
	return nil
}
func (r *dispatcherEmployeeRepo) WithTx(*sql.Tx) repository.EmployeeRepository { return r }

type dispatcherSettingsRepo struct {
	values map[string]string
}

func (r *dispatcherSettingsRepo) GetByKey(_ context.Context, key string) (*domain.SystemSetting, error) {
	value, ok := r.values[key]
	if !ok {
		return nil, sql.ErrNoRows
	}
	return &domain.SystemSetting{SettingKey: key, SettingValue: value}, nil
}

func (r *dispatcherSettingsRepo) List(context.Context) ([]domain.SystemSetting, error) {
	items := make([]domain.SystemSetting, 0, len(r.values))
	for key, value := range r.values {
		items = append(items, domain.SystemSetting{SettingKey: key, SettingValue: value})
	}
	return items, nil
}
func (r *dispatcherSettingsRepo) Save(context.Context, *domain.SystemSetting) error { return nil }
func (r *dispatcherSettingsRepo) WithTx(*sql.Tx) repository.SystemSettingRepository { return r }

type dispatcherReportRepo struct {
	dispatchable             []domain.AttendanceReport
	notificationDispatchable []domain.AttendanceReport
	saved                    []*domain.AttendanceReport
}

func (r *dispatcherReportRepo) FindByIdempotencyKey(context.Context, string) (*domain.AttendanceReport, error) {
	return nil, sql.ErrNoRows
}

func (r *dispatcherReportRepo) Save(_ context.Context, report *domain.AttendanceReport) error {
	copied := *report
	r.saved = append(r.saved, &copied)
	return nil
}

func (r *dispatcherReportRepo) ListByAttendanceRecordID(context.Context, uint64) ([]domain.AttendanceReport, error) {
	return nil, nil
}

func (r *dispatcherReportRepo) FindLatestSuccessfulByAttendanceRecordAndType(context.Context, uint64, string) (*domain.AttendanceReport, error) {
	return nil, sql.ErrNoRows
}

func (r *dispatcherReportRepo) ListDispatchable(context.Context, int, uint32) ([]domain.AttendanceReport, error) {
	items := make([]domain.AttendanceReport, len(r.dispatchable))
	copy(items, r.dispatchable)
	return items, nil
}

func (r *dispatcherReportRepo) ListNotificationDispatchable(context.Context, int, uint32) ([]domain.AttendanceReport, error) {
	items := make([]domain.AttendanceReport, len(r.notificationDispatchable))
	copy(items, r.notificationDispatchable)
	return items, nil
}

type fakeFeishuAttendanceClient struct {
	createRequests      []FeishuAttendanceCreateInput
	deleteRequests      [][]string
	sendMessageRequests []FeishuSendMessageInput
	recordID            string
	messageID           string
	sendMessageErr      error
}

func (c *fakeFeishuAttendanceClient) CreateFlow(_ context.Context, _ FeishuAttendanceConfig, input FeishuAttendanceCreateInput) (*FeishuCreateFlowResult, error) {
	c.createRequests = append(c.createRequests, input)
	return &FeishuCreateFlowResult{
		RecordID:     c.recordID,
		StatusCode:   200,
		ResponseBody: `{"code":0}`,
	}, nil
}

func (c *fakeFeishuAttendanceClient) DeleteFlows(_ context.Context, _ FeishuAttendanceConfig, recordIDs []string) (*FeishuDeleteFlowsResult, error) {
	copied := append([]string(nil), recordIDs...)
	c.deleteRequests = append(c.deleteRequests, copied)
	return &FeishuDeleteFlowsResult{
		SuccessRecordIDs: copied,
		StatusCode:       200,
		ResponseBody:     `{"code":0}`,
	}, nil
}

func (c *fakeFeishuAttendanceClient) SendTextMessage(_ context.Context, _ FeishuAttendanceConfig, input FeishuSendMessageInput) (*FeishuSendMessageResult, error) {
	c.sendMessageRequests = append(c.sendMessageRequests, input)
	result := &FeishuSendMessageResult{
		MessageID:    c.messageID,
		StatusCode:   200,
		ResponseBody: `{"code":0}`,
	}
	if c.sendMessageErr != nil {
		result.ResponseBody = `{"code":230013,"msg":"Bot has NO availability to this user"}`
		return result, c.sendMessageErr
	}
	return result, nil
}

func TestAttendanceAdminServiceCorrectionCopiesLatestFeishuRecordForDeletion(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("expected sqlmock db, got %v", err)
	}
	defer db.Close()

	recordDate := time.Date(2026, 3, 21, 0, 0, 0, 0, time.UTC)
	existingFirst := time.Date(2026, 3, 21, 8, 0, 0, 0, time.UTC)
	newFirst := time.Date(2026, 3, 21, 8, 10, 0, 0, time.UTC)

	attendanceRepo := &testAttendanceRepo{
		record: &domain.AttendanceRecord{
			ID:             41,
			EmployeeID:     7,
			AttendanceDate: recordDate,
			FirstConnectAt: &existingFirst,
			ClockInStatus:  "done",
			Version:        2,
		},
	}
	reportRepo := &testReportRepo{
		latestClockIn: &domain.AttendanceReport{
			ID:                 91,
			AttendanceRecordID: 41,
			ReportType:         "clock_in",
			ReportStatus:       "success",
			ExternalRecordID:   "flow_prev_001",
		},
	}
	service := NewAttendanceAdminService(db, attendanceRepo, reportRepo, nil, NewReportService())

	mock.ExpectBegin()
	mock.ExpectCommit()

	result, err := service.CorrectAttendance(context.Background(), 41, AttendanceCorrectionInput{
		FirstConnectAt: OptionalTimeField{Provided: true, Valid: true, Value: &newFirst},
	})
	if err != nil {
		t.Fatalf("expected correction to succeed, got %v", err)
	}
	if len(result.Reports) != 1 {
		t.Fatalf("expected one report, got %d", len(result.Reports))
	}
	if len(reportRepo.saved) != 1 {
		t.Fatalf("expected one saved report, got %d", len(reportRepo.saved))
	}
	if reportRepo.saved[0].DeleteRecordID != "flow_prev_001" {
		t.Fatalf("expected delete record id to copy previous flow, got %q", reportRepo.saved[0].DeleteRecordID)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("expected transaction expectations to be met, got %v", err)
	}
}

func TestAttendanceReportDispatcherRunOnceCreatesFeishuFlowAndStoresRecordID(t *testing.T) {
	reportTime := "2026-03-21T00:10:00Z"
	payload, _ := json.Marshal(map[string]any{
		"attendanceRecordId": uint64(41),
		"employeeId":         uint64(7),
		"attendanceDate":     "2026-03-21",
		"reportType":         "clock_in",
		"timestamp":          reportTime,
		"version":            uint32(3),
	})

	reportRepo := &dispatcherReportRepo{
		dispatchable: []domain.AttendanceReport{
			{
				ID:                 81,
				AttendanceRecordID: 41,
				ReportType:         "clock_in",
				IdempotencyKey:     "attendance-report/employee-7-2026-03-21/clock_in/2026-03-21T00:10:00Z/v3",
				PayloadJSON:        string(payload),
				ReportStatus:       "pending",
			},
		},
	}
	employeeRepo := &dispatcherEmployeeRepo{
		employee: &domain.Employee{
			ID:               7,
			Name:             "Alice",
			FeishuEmployeeID: "fs_emp_007",
		},
	}
	settingsRepo := &dispatcherSettingsRepo{
		values: map[string]string{
			"feishu_app_id":          "cli_123",
			"feishu_app_secret":      "secret_456",
			"feishu_location_name":   "总部办公区",
			"report_timeout_seconds": "15",
			"report_retry_limit":     "3",
		},
	}
	client := &fakeFeishuAttendanceClient{recordID: "flow_new_001", messageID: "om_msg_001"}
	dispatcher := NewAttendanceReportDispatcher(AttendanceReportDispatcherDeps{
		Reports:   reportRepo,
		Employees: employeeRepo,
		Settings:  settingsRepo,
		Client:    client,
	})

	if err := dispatcher.RunOnce(context.Background()); err != nil {
		t.Fatalf("expected dispatcher run to succeed, got %v", err)
	}
	if len(client.createRequests) != 1 {
		t.Fatalf("expected one create request, got %d", len(client.createRequests))
	}
	if client.createRequests[0].UserID != "fs_emp_007" {
		t.Fatalf("expected feishu user id fs_emp_007, got %q", client.createRequests[0].UserID)
	}
	if client.createRequests[0].CreatorID != "fs_emp_007" {
		t.Fatalf("expected creator id to match user id fs_emp_007, got %q", client.createRequests[0].CreatorID)
	}
	if client.createRequests[0].LocationName != "总部办公区" {
		t.Fatalf("expected location name to propagate, got %q", client.createRequests[0].LocationName)
	}
	if len(client.sendMessageRequests) != 1 {
		t.Fatalf("expected one message request, got %d", len(client.sendMessageRequests))
	}
	if client.sendMessageRequests[0].ReceiveIDType != "user_id" {
		t.Fatalf("expected receive id type user_id, got %q", client.sendMessageRequests[0].ReceiveIDType)
	}
	if client.sendMessageRequests[0].ReceiveID != "fs_emp_007" {
		t.Fatalf("expected receive id fs_emp_007, got %q", client.sendMessageRequests[0].ReceiveID)
	}
	if len(reportRepo.saved) != 1 {
		t.Fatalf("expected one saved report update, got %d", len(reportRepo.saved))
	}
	if reportRepo.saved[0].ReportStatus != "success" {
		t.Fatalf("expected report to become success, got %q", reportRepo.saved[0].ReportStatus)
	}
	if reportRepo.saved[0].ExternalRecordID != "flow_new_001" {
		t.Fatalf("expected external record id flow_new_001, got %q", reportRepo.saved[0].ExternalRecordID)
	}
	if reportRepo.saved[0].NotificationStatus != "success" {
		t.Fatalf("expected notification status success, got %q", reportRepo.saved[0].NotificationStatus)
	}
	if reportRepo.saved[0].NotificationMessageID != "om_msg_001" {
		t.Fatalf("expected notification message id om_msg_001, got %q", reportRepo.saved[0].NotificationMessageID)
	}
}

func TestAttendanceReportDispatcherKeepsAttendanceSuccessWhenNotificationFails(t *testing.T) {
	reportTime := "2026-03-21T10:10:00Z"
	payload, _ := json.Marshal(map[string]any{
		"attendanceRecordId": uint64(52),
		"employeeId":         uint64(8),
		"attendanceDate":     "2026-03-21",
		"reportType":         "clock_out",
		"timestamp":          reportTime,
		"version":            uint32(2),
	})

	reportRepo := &dispatcherReportRepo{
		dispatchable: []domain.AttendanceReport{
			{
				ID:                 82,
				AttendanceRecordID: 52,
				ReportType:         "clock_out",
				IdempotencyKey:     "attendance-report/employee-8-2026-03-21/clock_out/2026-03-21T10:10:00Z/v2",
				PayloadJSON:        string(payload),
				ReportStatus:       "pending",
			},
		},
	}
	employeeRepo := &dispatcherEmployeeRepo{
		employee: &domain.Employee{
			ID:               8,
			Name:             "Bob",
			FeishuEmployeeID: "fs_emp_008",
		},
	}
	settingsRepo := &dispatcherSettingsRepo{
		values: map[string]string{
			"feishu_app_id":          "cli_123",
			"feishu_app_secret":      "secret_456",
			"feishu_location_name":   "总部办公区",
			"report_timeout_seconds": "15",
			"report_retry_limit":     "3",
		},
	}
	client := &fakeFeishuAttendanceClient{
		recordID:       "flow_new_002",
		sendMessageErr: errors.New("feishu send message failed: code=230013 msg=Bot has NO availability to this user"),
	}
	dispatcher := NewAttendanceReportDispatcher(AttendanceReportDispatcherDeps{
		Reports:   reportRepo,
		Employees: employeeRepo,
		Settings:  settingsRepo,
		Client:    client,
	})

	if err := dispatcher.RunOnce(context.Background()); err != nil {
		t.Fatalf("expected dispatcher run to keep attendance success despite notification failure, got %v", err)
	}
	if len(client.createRequests) != 1 {
		t.Fatalf("expected one create request, got %d", len(client.createRequests))
	}
	if len(client.sendMessageRequests) != 1 {
		t.Fatalf("expected one message request, got %d", len(client.sendMessageRequests))
	}
	if len(reportRepo.saved) != 1 {
		t.Fatalf("expected one saved report update, got %d", len(reportRepo.saved))
	}
	if reportRepo.saved[0].ReportStatus != "success" {
		t.Fatalf("expected report status success, got %q", reportRepo.saved[0].ReportStatus)
	}
	if reportRepo.saved[0].NotificationStatus != "failed" {
		t.Fatalf("expected notification status failed, got %q", reportRepo.saved[0].NotificationStatus)
	}
	if reportRepo.saved[0].NotificationRetryCount != 1 {
		t.Fatalf("expected notification retry count 1, got %d", reportRepo.saved[0].NotificationRetryCount)
	}
}

func TestAttendanceReportDispatcherRunOnceRetriesNotificationWithoutRecreatingAttendanceFlow(t *testing.T) {
	reportTime := "2026-03-21T10:10:00Z"
	payload, _ := json.Marshal(map[string]any{
		"attendanceRecordId": uint64(63),
		"employeeId":         uint64(9),
		"attendanceDate":     "2026-03-21",
		"reportType":         "clock_in",
		"timestamp":          reportTime,
		"version":            uint32(2),
	})

	reportRepo := &dispatcherReportRepo{
		notificationDispatchable: []domain.AttendanceReport{
			{
				ID:                       93,
				AttendanceRecordID:       63,
				ReportType:               "clock_in",
				IdempotencyKey:           "attendance-report/employee-9-2026-03-21/clock_in/2026-03-21T10:10:00Z/v2",
				PayloadJSON:              string(payload),
				ReportStatus:             "success",
				ExternalRecordID:         "flow_existing_001",
				NotificationStatus:       "failed",
				NotificationRetryCount:   1,
				NotificationResponseBody: `{"code":230013}`,
			},
		},
	}
	employeeRepo := &dispatcherEmployeeRepo{
		employee: &domain.Employee{
			ID:               9,
			Name:             "Nina",
			FeishuEmployeeID: "fs_emp_009",
		},
	}
	settingsRepo := &dispatcherSettingsRepo{
		values: map[string]string{
			"feishu_app_id":          "cli_123",
			"feishu_app_secret":      "secret_456",
			"feishu_location_name":   "总部办公区",
			"report_timeout_seconds": "15",
			"report_retry_limit":     "3",
		},
	}
	client := &fakeFeishuAttendanceClient{messageID: "om_msg_retry_001"}
	dispatcher := NewAttendanceReportDispatcher(AttendanceReportDispatcherDeps{
		Reports:   reportRepo,
		Employees: employeeRepo,
		Settings:  settingsRepo,
		Client:    client,
	})

	if err := dispatcher.RunOnce(context.Background()); err != nil {
		t.Fatalf("expected dispatcher run to retry notification, got %v", err)
	}
	if len(client.createRequests) != 0 {
		t.Fatalf("expected notification retry to skip create flow, got %d create requests", len(client.createRequests))
	}
	if len(client.sendMessageRequests) != 1 {
		t.Fatalf("expected one message retry, got %d", len(client.sendMessageRequests))
	}
	if len(reportRepo.saved) != 1 {
		t.Fatalf("expected one saved notification update, got %d", len(reportRepo.saved))
	}
	if reportRepo.saved[0].NotificationStatus != "success" {
		t.Fatalf("expected notification status success after retry, got %q", reportRepo.saved[0].NotificationStatus)
	}
	if reportRepo.saved[0].NotificationMessageID != "om_msg_retry_001" {
		t.Fatalf("expected notification message id om_msg_retry_001, got %q", reportRepo.saved[0].NotificationMessageID)
	}
	if reportRepo.saved[0].NotificationRetryCount != 0 {
		t.Fatalf("expected notification retry count reset to 0, got %d", reportRepo.saved[0].NotificationRetryCount)
	}
}

func TestBuildAttendanceNotificationTextUsesConfiguredLocation(t *testing.T) {
	employee := domain.Employee{
		ID:   11,
		Name: "Wesley",
	}
	timestamp := "2026-03-23T03:04:05Z"
	payload := attendanceReportPayload{
		ReportType: "clock_in",
		Timestamp:  &timestamp,
	}
	location, err := time.LoadLocation("Asia/Shanghai")
	if err != nil {
		t.Fatalf("load location: %v", err)
	}

	originalLocal := time.Local
	time.Local = time.UTC
	defer func() {
		time.Local = originalLocal
	}()

	text, err := buildAttendanceNotificationText(employee, payload, "办公室", location)
	if err != nil {
		t.Fatalf("expected notification text build to succeed, got %v", err)
	}
	if !strings.Contains(text, "日期：2026-03-23") {
		t.Fatalf("expected local date in notification, got %q", text)
	}
	if !strings.Contains(text, "时间：11:04:05") {
		t.Fatalf("expected Asia/Shanghai time in notification, got %q", text)
	}
}

func TestAttendanceReportDispatcherRunOnceDeletesOldFlowForClearReport(t *testing.T) {
	payload, _ := json.Marshal(map[string]any{
		"attendanceRecordId": uint64(41),
		"employeeId":         uint64(7),
		"attendanceDate":     "2026-03-21",
		"reportType":         "clock_out",
		"action":             "clear",
		"timestamp":          nil,
		"version":            uint32(4),
	})

	reportRepo := &dispatcherReportRepo{
		dispatchable: []domain.AttendanceReport{
			{
				ID:                 82,
				AttendanceRecordID: 41,
				ReportType:         "clock_out",
				IdempotencyKey:     "attendance-report/employee-7-2026-03-21/clock_out/clear/v4",
				PayloadJSON:        string(payload),
				ReportStatus:       "pending",
				DeleteRecordID:     "flow_old_002",
			},
		},
	}
	employeeRepo := &dispatcherEmployeeRepo{
		employee: &domain.Employee{
			ID:               7,
			Name:             "Alice",
			FeishuEmployeeID: "fs_emp_007",
		},
	}
	settingsRepo := &dispatcherSettingsRepo{
		values: map[string]string{
			"feishu_app_id":          "cli_123",
			"feishu_app_secret":      "secret_456",
			"feishu_location_name":   "总部办公区",
			"report_timeout_seconds": "15",
			"report_retry_limit":     "3",
		},
	}
	client := &fakeFeishuAttendanceClient{}
	dispatcher := NewAttendanceReportDispatcher(AttendanceReportDispatcherDeps{
		Reports:   reportRepo,
		Employees: employeeRepo,
		Settings:  settingsRepo,
		Client:    client,
	})

	if err := dispatcher.RunOnce(context.Background()); err != nil {
		t.Fatalf("expected clear dispatch to succeed, got %v", err)
	}
	if len(client.deleteRequests) != 1 {
		t.Fatalf("expected one delete request, got %d", len(client.deleteRequests))
	}
	if matched, _ := regexp.MatchString(`flow_old_002`, client.deleteRequests[0][0]); !matched {
		t.Fatalf("expected delete request to contain flow_old_002, got %+v", client.deleteRequests[0])
	}
	if len(client.createRequests) != 0 {
		t.Fatalf("expected clear report to skip create, got %d create requests", len(client.createRequests))
	}
	if len(reportRepo.saved) != 1 {
		t.Fatalf("expected one saved report update, got %d", len(reportRepo.saved))
	}
	if reportRepo.saved[0].DeleteRecordID != "" {
		t.Fatalf("expected delete record id to be cleared after success, got %q", reportRepo.saved[0].DeleteRecordID)
	}
	if reportRepo.saved[0].ReportStatus != "success" {
		t.Fatalf("expected clear report to be marked success, got %q", reportRepo.saved[0].ReportStatus)
	}
}

type testAttendanceRepo struct {
	record *domain.AttendanceRecord
	saved  []*domain.AttendanceRecord
}

func (r *testAttendanceRepo) FindByID(context.Context, uint64) (*domain.AttendanceRecord, error) {
	copied := *r.record
	return &copied, nil
}

func (r *testAttendanceRepo) FindByEmployeeAndDate(context.Context, uint64, time.Time) (*domain.AttendanceRecord, error) {
	return nil, sql.ErrNoRows
}

func (r *testAttendanceRepo) Save(_ context.Context, record *domain.AttendanceRecord) error {
	copied := *record
	r.saved = append(r.saved, &copied)
	return nil
}

func (r *testAttendanceRepo) ListByDateRange(context.Context, time.Time, time.Time) ([]domain.AttendanceRecord, error) {
	return nil, nil
}

func (r *testAttendanceRepo) WithTx(*sql.Tx) repository.AttendanceRepository { return r }

type testReportRepo struct {
	saved         []*domain.AttendanceReport
	latestClockIn *domain.AttendanceReport
}

func (r *testReportRepo) FindByIdempotencyKey(context.Context, string) (*domain.AttendanceReport, error) {
	return nil, sql.ErrNoRows
}

func (r *testReportRepo) Save(_ context.Context, report *domain.AttendanceReport) error {
	copied := *report
	r.saved = append(r.saved, &copied)
	return nil
}

func (r *testReportRepo) ListByAttendanceRecordID(context.Context, uint64) ([]domain.AttendanceReport, error) {
	return nil, nil
}

func (r *testReportRepo) FindLatestSuccessfulByAttendanceRecordAndType(_ context.Context, _ uint64, reportType string) (*domain.AttendanceReport, error) {
	if reportType != "clock_in" || r.latestClockIn == nil {
		return nil, sql.ErrNoRows
	}
	copied := *r.latestClockIn
	return &copied, nil
}

func (r *testReportRepo) ListDispatchable(context.Context, int, uint32) ([]domain.AttendanceReport, error) {
	return nil, nil
}

func (r *testReportRepo) ListNotificationDispatchable(context.Context, int, uint32) ([]domain.AttendanceReport, error) {
	return nil, nil
}

func (r *testReportRepo) WithTx(*sql.Tx) repository.ReportRepository { return r }
