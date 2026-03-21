package service

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"syslog/internal/domain"
	"syslog/internal/repository"
)

var ErrInvalidEmployeeInput = errors.New("invalid employee input")
var ErrInvalidSettingsInput = errors.New("invalid settings input")
var ErrInvalidAttendanceCorrection = errors.New("invalid attendance correction")

type EmployeeDeviceInput struct {
	MacAddress  string
	DeviceLabel string
	Status      string
}

type EmployeeWriteInput struct {
	EmployeeNo string
	SystemNo   string
	Name       string
	Status     string
	Devices    []EmployeeDeviceInput
}

type EmployeeAdminService struct {
	db   *sql.DB
	repo repository.EmployeeRepository
}

func NewEmployeeAdminService(db *sql.DB, repo repository.EmployeeRepository) *EmployeeAdminService {
	return &EmployeeAdminService{db: db, repo: repo}
}

func (s *EmployeeAdminService) ListEmployees(ctx context.Context) ([]domain.Employee, error) {
	if s.repo == nil {
		return nil, errors.New("employee repository is required")
	}

	return s.repo.List(ctx)
}

func (s *EmployeeAdminService) CreateEmployee(ctx context.Context, input EmployeeWriteInput) (*domain.Employee, error) {
	return s.saveEmployee(ctx, 0, input, true)
}

func (s *EmployeeAdminService) UpdateEmployee(ctx context.Context, id uint64, input EmployeeWriteInput) (*domain.Employee, error) {
	return s.saveEmployee(ctx, id, input, false)
}

func (s *EmployeeAdminService) DisableEmployee(ctx context.Context, id uint64) (*domain.Employee, error) {
	if s.repo == nil {
		return nil, errors.New("employee repository is required")
	}
	if s.db == nil {
		return nil, errors.New("database is required")
	}

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}

	repoTx := s.repo.WithTx(tx)
	if err := repoTx.Disable(ctx, id); err != nil {
		_ = tx.Rollback()
		return nil, err
	}
	if err := repoTx.DisableDevicesByEmployeeID(ctx, id); err != nil {
		_ = tx.Rollback()
		return nil, err
	}
	if err := tx.Commit(); err != nil {
		return nil, err
	}

	return s.repo.FindByID(ctx, id)
}

func (s *EmployeeAdminService) saveEmployee(ctx context.Context, id uint64, input EmployeeWriteInput, creating bool) (*domain.Employee, error) {
	if s.repo == nil {
		return nil, errors.New("employee repository is required")
	}
	if s.db == nil {
		return nil, errors.New("database is required")
	}

	employee, devices, err := normalizeEmployeeInput(id, input)
	if err != nil {
		return nil, err
	}

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}

	repoTx := s.repo.WithTx(tx)
	if creating {
		if err := repoTx.Create(ctx, &employee); err != nil {
			_ = tx.Rollback()
			return nil, err
		}
	} else {
		if err := repoTx.Update(ctx, &employee); err != nil {
			_ = tx.Rollback()
			return nil, err
		}
	}

	if err := repoTx.ReplaceDevices(ctx, employee.ID, devices); err != nil {
		_ = tx.Rollback()
		return nil, err
	}

	if err := tx.Commit(); err != nil {
		return nil, err
	}

	return s.repo.FindByID(ctx, employee.ID)
}

func normalizeEmployeeInput(id uint64, input EmployeeWriteInput) (domain.Employee, []domain.EmployeeDevice, error) {
	employeeNo := strings.TrimSpace(input.EmployeeNo)
	systemNo := strings.TrimSpace(input.SystemNo)
	name := strings.TrimSpace(input.Name)
	status := strings.TrimSpace(input.Status)
	if status == "" {
		status = "active"
	}

	if employeeNo == "" {
		return domain.Employee{}, nil, fmt.Errorf("%w: employeeNo is required", ErrInvalidEmployeeInput)
	}
	if systemNo == "" {
		return domain.Employee{}, nil, fmt.Errorf("%w: systemNo is required", ErrInvalidEmployeeInput)
	}
	if name == "" {
		return domain.Employee{}, nil, fmt.Errorf("%w: name is required", ErrInvalidEmployeeInput)
	}

	seen := make(map[string]struct{}, len(input.Devices))
	devices := make([]domain.EmployeeDevice, 0, len(input.Devices))
	for _, item := range input.Devices {
		mac := normalizeMACAddress(item.MacAddress)
		if mac == "" {
			return domain.Employee{}, nil, fmt.Errorf("%w: device macAddress is required", ErrInvalidEmployeeInput)
		}
		if _, ok := seen[mac]; ok {
			return domain.Employee{}, nil, fmt.Errorf("%w: duplicate device macAddress: %s", ErrInvalidEmployeeInput, mac)
		}
		seen[mac] = struct{}{}

		deviceStatus := strings.TrimSpace(item.Status)
		if deviceStatus == "" {
			deviceStatus = "active"
		}

		devices = append(devices, domain.EmployeeDevice{
			EmployeeID:  id,
			MacAddress:  mac,
			DeviceLabel: strings.TrimSpace(item.DeviceLabel),
			Status:      deviceStatus,
		})
	}

	employee := domain.Employee{
		ID:         id,
		EmployeeNo: employeeNo,
		SystemNo:   systemNo,
		Name:       name,
		Status:     status,
	}

	return employee, devices, nil
}

type SettingWriteInput struct {
	SettingKey   string
	SettingValue string
}

type SettingsAdminService struct {
	db   *sql.DB
	repo repository.SystemSettingRepository
}

func NewSettingsAdminService(db *sql.DB, repo repository.SystemSettingRepository) *SettingsAdminService {
	return &SettingsAdminService{db: db, repo: repo}
}

func (s *SettingsAdminService) UpdateSettings(ctx context.Context, items []SettingWriteInput) ([]domain.SystemSetting, error) {
	if s.repo == nil {
		return nil, errors.New("settings repository is required")
	}
	current, err := s.repo.List(ctx)
	if err != nil {
		return nil, err
	}

	currentByKey := make(map[string]domain.SystemSetting, len(current))
	for _, setting := range current {
		currentByKey[setting.SettingKey] = setting
	}

	seen := make(map[string]struct{}, len(items))
	for _, item := range items {
		if _, ok := currentByKey[item.SettingKey]; !ok {
			return nil, fmt.Errorf("%w: unknown setting key: %s", ErrInvalidSettingsInput, item.SettingKey)
		}
		if _, ok := seen[item.SettingKey]; ok {
			return nil, fmt.Errorf("%w: duplicate setting key: %s", ErrInvalidSettingsInput, item.SettingKey)
		}
		seen[item.SettingKey] = struct{}{}
	}

	if len(items) == 0 {
		return current, nil
	}
	if s.db == nil {
		return nil, errors.New("database is required")
	}

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}

	repoTx := s.repo.WithTx(tx)
	for _, item := range items {
		setting := currentByKey[item.SettingKey]
		setting.SettingValue = item.SettingValue
		if err := repoTx.Save(ctx, &setting); err != nil {
			_ = tx.Rollback()
			return nil, err
		}
		currentByKey[item.SettingKey] = setting
	}

	if err := tx.Commit(); err != nil {
		return nil, err
	}

	for idx, setting := range current {
		current[idx] = currentByKey[setting.SettingKey]
	}

	return current, nil
}

type OptionalTimeField struct {
	Provided bool
	Valid    bool
	Value    *time.Time
}

func (f *OptionalTimeField) UnmarshalJSON(data []byte) error {
	f.Provided = true
	if bytes.Equal(bytes.TrimSpace(data), []byte("null")) {
		f.Valid = false
		f.Value = nil
		return nil
	}

	var parsed time.Time
	if err := json.Unmarshal(data, &parsed); err != nil {
		return err
	}

	f.Valid = true
	f.Value = &parsed
	return nil
}

func (f OptionalTimeField) Apply(existing *time.Time) *time.Time {
	if !f.Provided {
		return existing
	}
	if !f.Valid {
		return nil
	}

	if f.Value == nil {
		return nil
	}

	copied := *f.Value
	return &copied
}

func (f OptionalTimeField) ShouldGenerateReport() bool {
	return f.Provided && f.Valid && f.Value != nil
}

type AttendanceCorrectionInput struct {
	FirstConnectAt   OptionalTimeField `json:"firstConnectAt"`
	LastDisconnectAt OptionalTimeField `json:"lastDisconnectAt"`
}

type AttendanceCorrectionResult struct {
	Record  domain.AttendanceRecord
	Reports []domain.AttendanceReport
}

type AttendanceAdminService struct {
	db             *sql.DB
	attendanceRepo repository.AttendanceRepository
	reportRepo     repository.ReportRepository
	settingsRepo   repository.SystemSettingRepository
	reportSvc      *ReportService
}

func NewAttendanceAdminService(db *sql.DB, attendanceRepo repository.AttendanceRepository, reportRepo repository.ReportRepository, settingsRepo repository.SystemSettingRepository, reportSvc *ReportService) *AttendanceAdminService {
	if reportSvc == nil {
		reportSvc = NewReportService()
	}

	return &AttendanceAdminService{
		db:             db,
		attendanceRepo: attendanceRepo,
		reportRepo:     reportRepo,
		settingsRepo:   settingsRepo,
		reportSvc:      reportSvc,
	}
}

func (s *AttendanceAdminService) CorrectAttendance(ctx context.Context, attendanceID uint64, input AttendanceCorrectionInput) (*AttendanceCorrectionResult, error) {
	if s.attendanceRepo == nil || s.reportRepo == nil {
		return nil, errors.New("attendance and report repositories are required")
	}
	if !input.FirstConnectAt.Provided && !input.LastDisconnectAt.Provided {
		return nil, fmt.Errorf("%w: at least one timestamp is required", ErrInvalidAttendanceCorrection)
	}

	record, err := s.attendanceRepo.FindByID(ctx, attendanceID)
	if err != nil {
		return nil, err
	}

	originalFirst := record.FirstConnectAt
	originalLast := record.LastDisconnectAt
	nextFirst := input.FirstConnectAt.Apply(originalFirst)
	nextLast := input.LastDisconnectAt.Apply(originalLast)
	if timePointerEqual(originalFirst, nextFirst) && timePointerEqual(originalLast, nextLast) {
		return &AttendanceCorrectionResult{Record: *record}, nil
	}

	targetURL, err := s.reportTargetURL(ctx)
	if err != nil {
		return nil, err
	}

	if s.db == nil {
		return nil, errors.New("database is required")
	}

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}

	attendanceTx, ok := s.attendanceRepo.(interface {
		WithTx(*sql.Tx) repository.AttendanceRepository
	})
	if !ok {
		_ = tx.Rollback()
		return nil, errors.New("attendance repository does not support tx")
	}
	reportTx, ok := s.reportRepo.(interface {
		WithTx(*sql.Tx) repository.ReportRepository
	})
	if !ok {
		_ = tx.Rollback()
		return nil, errors.New("report repository does not support tx")
	}

	record.FirstConnectAt = nextFirst
	record.LastDisconnectAt = nextLast
	record.SourceMode = "manual"
	record.Version++
	now := time.Now()
	record.LastCalculatedAt = &now
	record.ClockInStatus = "pending"
	if record.FirstConnectAt != nil {
		record.ClockInStatus = "done"
	}
	record.ClockOutStatus = "missing"
	record.ExceptionStatus = "missing_disconnect"
	if record.LastDisconnectAt != nil {
		record.ClockOutStatus = "done"
		record.ExceptionStatus = "none"
	}

	if err := attendanceTx.WithTx(tx).Save(ctx, record); err != nil {
		_ = tx.Rollback()
		return nil, err
	}

	reports := make([]domain.AttendanceReport, 0, 2)
	if report, ok := s.correctionReportForField(*record, "clock_in", originalFirst, input.FirstConnectAt, targetURL); ok {
		reports = append(reports, *report)
		if err := reportTx.WithTx(tx).Save(ctx, report); err != nil {
			_ = tx.Rollback()
			return nil, err
		}
	}
	if report, ok := s.correctionReportForField(*record, "clock_out", originalLast, input.LastDisconnectAt, targetURL); ok {
		reports = append(reports, *report)
		if err := reportTx.WithTx(tx).Save(ctx, report); err != nil {
			_ = tx.Rollback()
			return nil, err
		}
	}

	if err := tx.Commit(); err != nil {
		return nil, err
	}

	return &AttendanceCorrectionResult{Record: *record, Reports: reports}, nil
}

func (s *AttendanceAdminService) reportTargetURL(ctx context.Context) (string, error) {
	if s.settingsRepo == nil {
		return "", nil
	}

	setting, err := s.settingsRepo.GetByKey(ctx, reportTargetURLSettingKey)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return "", nil
		}
		return "", err
	}
	if setting == nil {
		return "", nil
	}

	return strings.TrimSpace(setting.SettingValue), nil
}

func normalizeMACAddress(value string) string {
	return strings.ToLower(strings.TrimSpace(value))
}

func timePointerEqual(a, b *time.Time) bool {
	if a == nil || b == nil {
		return a == nil && b == nil
	}

	return a.Equal(*b)
}

func (s *AttendanceAdminService) correctionReportForField(record domain.AttendanceRecord, reportType string, original *time.Time, input OptionalTimeField, targetURL string) (*domain.AttendanceReport, bool) {
	if !input.Provided {
		return nil, false
	}

	if input.Valid {
		if input.Value == nil || timePointerEqual(original, input.Value) {
			return nil, false
		}

		report := s.reportSvc.CreatePendingReport(record, reportType, *input.Value, targetURL)
		return &report, true
	}

	if original == nil {
		return nil, false
	}

	report := s.reportSvc.CreateClearReport(record, reportType, targetURL)
	return &report, true
}
