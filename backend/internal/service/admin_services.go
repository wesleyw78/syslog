package service

import (
	"context"
	"database/sql"
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

	return &domain.Employee{ID: id, Status: "disabled"}, nil
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

	for i := range devices {
		devices[i].EmployeeID = employee.ID
	}
	employee.Devices = devices
	return &employee, nil
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

	for _, item := range items {
		if _, ok := currentByKey[item.SettingKey]; !ok {
			return nil, fmt.Errorf("%w: unknown setting key: %s", ErrInvalidSettingsInput, item.SettingKey)
		}
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

type AttendanceCorrectionInput struct {
	FirstConnectAt   *time.Time
	LastDisconnectAt *time.Time
}

type AttendanceCorrectionResult struct {
	Record  domain.AttendanceRecord
	Reports []domain.AttendanceReport
}

type AttendanceAdminService struct {
	db             *sql.DB
	attendanceRepo repository.AttendanceRepository
	reportRepo     repository.ReportRepository
	reportSvc      *ReportService
}

func NewAttendanceAdminService(db *sql.DB, attendanceRepo repository.AttendanceRepository, reportRepo repository.ReportRepository, reportSvc *ReportService) *AttendanceAdminService {
	if reportSvc == nil {
		reportSvc = NewReportService()
	}

	return &AttendanceAdminService{
		db:             db,
		attendanceRepo: attendanceRepo,
		reportRepo:     reportRepo,
		reportSvc:      reportSvc,
	}
}

func (s *AttendanceAdminService) CorrectAttendance(ctx context.Context, attendanceID uint64, input AttendanceCorrectionInput) (*AttendanceCorrectionResult, error) {
	if s.attendanceRepo == nil || s.reportRepo == nil {
		return nil, errors.New("attendance and report repositories are required")
	}
	if input.FirstConnectAt == nil && input.LastDisconnectAt == nil {
		return nil, fmt.Errorf("%w: at least one timestamp is required", ErrInvalidAttendanceCorrection)
	}

	record, err := s.attendanceRepo.FindByID(ctx, attendanceID)
	if err != nil {
		return nil, err
	}

	record.FirstConnectAt = input.FirstConnectAt
	record.LastDisconnectAt = input.LastDisconnectAt
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

	if err := attendanceTx.WithTx(tx).Save(ctx, record); err != nil {
		_ = tx.Rollback()
		return nil, err
	}

	reports := make([]domain.AttendanceReport, 0, 2)
	if record.FirstConnectAt != nil {
		report := s.reportSvc.CreatePendingReport(*record, "clock_in", *record.FirstConnectAt, "")
		reports = append(reports, report)
		if err := reportTx.WithTx(tx).Save(ctx, &report); err != nil {
			_ = tx.Rollback()
			return nil, err
		}
	}
	if record.LastDisconnectAt != nil {
		report := s.reportSvc.CreatePendingReport(*record, "clock_out", *record.LastDisconnectAt, "")
		reports = append(reports, report)
		if err := reportTx.WithTx(tx).Save(ctx, &report); err != nil {
			_ = tx.Rollback()
			return nil, err
		}
	}

	if err := tx.Commit(); err != nil {
		return nil, err
	}

	return &AttendanceCorrectionResult{Record: *record, Reports: reports}, nil
}

func normalizeMACAddress(value string) string {
	return strings.ToLower(strings.TrimSpace(value))
}
