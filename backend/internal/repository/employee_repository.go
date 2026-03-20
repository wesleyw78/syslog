package repository

import (
	"context"
	"database/sql"
	"strings"

	"syslog/internal/domain"
)

type EmployeeRepository interface {
	FindByMACAddress(ctx context.Context, macAddress string) (*domain.Employee, error)
	List(ctx context.Context) ([]domain.Employee, error)
}

type MySQLEmployeeRepository struct {
	db *sql.DB
}

func NewMySQLEmployeeRepository(db *sql.DB) *MySQLEmployeeRepository {
	return &MySQLEmployeeRepository{db: db}
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

func (r *MySQLEmployeeRepository) List(ctx context.Context) ([]domain.Employee, error) {
	const query = `
SELECT id, employee_no, system_no, name, status, created_at, updated_at
FROM employees
ORDER BY id ASC`

	rows, err := r.db.QueryContext(ctx, trimSQL(query))
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	employees := make([]domain.Employee, 0)
	for rows.Next() {
		var employee domain.Employee
		if err := rows.Scan(
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

		employees = append(employees, employee)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return employees, nil
}
