package repository

import (
	"context"
	"database/sql"
	"strings"
	"time"

	"syslog/internal/domain"
)

type EmployeeRepository interface {
	FindByMACAddress(ctx context.Context, macAddress string) (*domain.Employee, error)
	FindByID(ctx context.Context, id uint64) (*domain.Employee, error)
	List(ctx context.Context) ([]domain.Employee, error)
	Create(ctx context.Context, employee *domain.Employee) error
	Update(ctx context.Context, employee *domain.Employee) error
	Disable(ctx context.Context, id uint64) error
	ReplaceDevices(ctx context.Context, employeeID uint64, devices []domain.EmployeeDevice) error
	DisableDevicesByEmployeeID(ctx context.Context, employeeID uint64) error
	WithTx(tx *sql.Tx) EmployeeRepository
}

type MySQLEmployeeRepository struct {
	db sqlExecutor
}

func NewMySQLEmployeeRepository(db *sql.DB) *MySQLEmployeeRepository {
	return &MySQLEmployeeRepository{db: db}
}

func (r *MySQLEmployeeRepository) WithTx(tx *sql.Tx) EmployeeRepository {
	return &MySQLEmployeeRepository{db: tx}
}

func (r *MySQLEmployeeRepository) FindByMACAddress(ctx context.Context, macAddress string) (*domain.Employee, error) {
	const query = `
SELECT e.id, e.employee_no, e.system_no, e.name, e.status, e.created_at, e.updated_at
FROM employees e
JOIN employee_devices d ON d.employee_id = e.id
WHERE d.mac_address = ?
LIMIT 1`

	row := r.db.QueryRowContext(ctx, trimSQL(query), strings.ToLower(strings.TrimSpace(macAddress)))

	var employee domain.Employee
	if err := row.Scan(
		&employee.ID,
		&employee.EmployeeNo,
		&employee.SystemNo,
		&employee.Name,
		&employee.Status,
		&employee.CreatedAt,
		&employee.UpdatedAt,
	); err != nil {
		return nil, err
	}

	return &employee, nil
}

func (r *MySQLEmployeeRepository) FindByID(ctx context.Context, id uint64) (*domain.Employee, error) {
	const query = `
SELECT e.id, e.employee_no, e.system_no, e.name, e.status, e.created_at, e.updated_at,
       d.id, d.mac_address, d.device_label, d.status, d.created_at, d.updated_at
FROM employees e
LEFT JOIN employee_devices d ON d.employee_id = e.id
WHERE e.id = ?
ORDER BY d.id ASC`

	rows, err := r.db.QueryContext(ctx, trimSQL(query), id)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var employee *domain.Employee
	for rows.Next() {
		var (
			employeeID      uint64
			employeeNo      string
			systemNo        string
			name            string
			status          string
			createdAt       time.Time
			updatedAt       time.Time
			deviceID        sql.NullInt64
			macAddress      sql.NullString
			deviceLabel     sql.NullString
			deviceStatus    sql.NullString
			deviceCreatedAt sql.NullTime
			deviceUpdatedAt sql.NullTime
		)

		if err := rows.Scan(
			&employeeID,
			&employeeNo,
			&systemNo,
			&name,
			&status,
			&createdAt,
			&updatedAt,
			&deviceID,
			&macAddress,
			&deviceLabel,
			&deviceStatus,
			&deviceCreatedAt,
			&deviceUpdatedAt,
		); err != nil {
			return nil, err
		}

		if employee == nil {
			employee = &domain.Employee{
				ID:         employeeID,
				EmployeeNo: employeeNo,
				SystemNo:   systemNo,
				Name:       name,
				Status:     status,
				CreatedAt:  createdAt,
				UpdatedAt:  updatedAt,
				Devices:    make([]domain.EmployeeDevice, 0),
			}
		}

		if deviceID.Valid {
			employee.Devices = append(employee.Devices, domain.EmployeeDevice{
				ID:          uint64(deviceID.Int64),
				EmployeeID:  employeeID,
				MacAddress:  stringFromNullString(macAddress),
				DeviceLabel: stringFromNullString(deviceLabel),
				Status:      stringFromNullString(deviceStatus),
				CreatedAt:   timeValueOrZero(deviceCreatedAt),
				UpdatedAt:   timeValueOrZero(deviceUpdatedAt),
			})
		}
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	if employee == nil {
		return nil, sql.ErrNoRows
	}

	return employee, nil
}

func (r *MySQLEmployeeRepository) List(ctx context.Context) ([]domain.Employee, error) {
	const query = `
SELECT e.id, e.employee_no, e.system_no, e.name, e.status, e.created_at, e.updated_at,
       d.id, d.mac_address, d.device_label, d.status, d.created_at, d.updated_at
FROM employees e
LEFT JOIN employee_devices d ON d.employee_id = e.id
ORDER BY e.id ASC, d.id ASC`

	rows, err := r.db.QueryContext(ctx, trimSQL(query))
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	employeesByID := make(map[uint64]*domain.Employee)
	order := make([]uint64, 0)
	for rows.Next() {
		var (
			employeeID      uint64
			employeeNo      string
			systemNo        string
			name            string
			status          string
			createdAt       time.Time
			updatedAt       time.Time
			deviceID        sql.NullInt64
			macAddress      sql.NullString
			deviceLabel     sql.NullString
			deviceStatus    sql.NullString
			deviceCreatedAt sql.NullTime
			deviceUpdatedAt sql.NullTime
		)

		if err := rows.Scan(
			&employeeID,
			&employeeNo,
			&systemNo,
			&name,
			&status,
			&createdAt,
			&updatedAt,
			&deviceID,
			&macAddress,
			&deviceLabel,
			&deviceStatus,
			&deviceCreatedAt,
			&deviceUpdatedAt,
		); err != nil {
			return nil, err
		}

		employee, ok := employeesByID[employeeID]
		if !ok {
			employee = &domain.Employee{
				ID:         employeeID,
				EmployeeNo: employeeNo,
				SystemNo:   systemNo,
				Name:       name,
				Status:     status,
				CreatedAt:  createdAt,
				UpdatedAt:  updatedAt,
				Devices:    make([]domain.EmployeeDevice, 0),
			}
			employeesByID[employeeID] = employee
			order = append(order, employeeID)
		}

		if deviceID.Valid {
			employee.Devices = append(employee.Devices, domain.EmployeeDevice{
				ID:          uint64(deviceID.Int64),
				EmployeeID:  employeeID,
				MacAddress:  stringFromNullString(macAddress),
				DeviceLabel: stringFromNullString(deviceLabel),
				Status:      stringFromNullString(deviceStatus),
				CreatedAt:   timeValueOrZero(deviceCreatedAt),
				UpdatedAt:   timeValueOrZero(deviceUpdatedAt),
			})
		}
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	employees := make([]domain.Employee, 0, len(order))
	for _, id := range order {
		employees = append(employees, *employeesByID[id])
	}

	return employees, nil
}

func timeValueOrZero(value sql.NullTime) time.Time {
	if !value.Valid {
		return time.Time{}
	}

	return value.Time
}

func (r *MySQLEmployeeRepository) Create(ctx context.Context, employee *domain.Employee) error {
	const query = `
INSERT INTO employees (
	employee_no,
	system_no,
	name,
	status
) VALUES (?, ?, ?, ?)`

	result, err := r.db.ExecContext(
		ctx,
		trimSQL(query),
		employee.EmployeeNo,
		employee.SystemNo,
		employee.Name,
		employee.Status,
	)
	if err != nil {
		return err
	}

	id, err := parseInsertedID(result)
	if err != nil {
		return err
	}

	employee.ID = id
	return nil
}

func (r *MySQLEmployeeRepository) Update(ctx context.Context, employee *domain.Employee) error {
	const query = `
UPDATE employees
SET employee_no = ?, system_no = ?, name = ?, status = ?
WHERE id = ?`

	result, err := r.db.ExecContext(
		ctx,
		trimSQL(query),
		employee.EmployeeNo,
		employee.SystemNo,
		employee.Name,
		employee.Status,
		employee.ID,
	)
	if err != nil {
		return err
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rowsAffected == 0 {
		return sql.ErrNoRows
	}

	return nil
}

func (r *MySQLEmployeeRepository) Disable(ctx context.Context, id uint64) error {
	const query = `
UPDATE employees
SET status = ?
WHERE id = ?`

	result, err := r.db.ExecContext(ctx, trimSQL(query), "disabled", id)
	if err != nil {
		return err
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rowsAffected == 0 {
		return sql.ErrNoRows
	}

	return nil
}

func (r *MySQLEmployeeRepository) ReplaceDevices(ctx context.Context, employeeID uint64, devices []domain.EmployeeDevice) error {
	const deleteQuery = `
DELETE FROM employee_devices
WHERE employee_id = ?`

	if _, err := r.db.ExecContext(ctx, trimSQL(deleteQuery), employeeID); err != nil {
		return err
	}

	const insertQuery = `
INSERT INTO employee_devices (
	employee_id,
	mac_address,
	device_label,
	status
) VALUES (?, ?, ?, ?)`

	for _, device := range devices {
		_, err := r.db.ExecContext(
			ctx,
			trimSQL(insertQuery),
			employeeID,
			device.MacAddress,
			device.DeviceLabel,
			device.Status,
		)
		if err != nil {
			return err
		}
	}

	return nil
}

func (r *MySQLEmployeeRepository) DisableDevicesByEmployeeID(ctx context.Context, employeeID uint64) error {
	const query = `
UPDATE employee_devices
SET status = ?
WHERE employee_id = ?`

	_, err := r.db.ExecContext(ctx, trimSQL(query), "disabled", employeeID)
	return err
}
