package domain

import "time"

type Employee struct {
	ID               uint64           `json:"id"`
	EmployeeNo       string           `json:"employeeNo"`
	SystemNo         string           `json:"systemNo"`
	FeishuEmployeeID string           `json:"feishuEmployeeId"`
	Name             string           `json:"name"`
	Status           string           `json:"status"`
	Devices          []EmployeeDevice `json:"devices,omitempty"`
	CreatedAt        time.Time        `json:"createdAt"`
	UpdatedAt        time.Time        `json:"updatedAt"`
}

type EmployeeDevice struct {
	ID          uint64    `json:"id"`
	EmployeeID  uint64    `json:"employeeId"`
	MacAddress  string    `json:"macAddress"`
	DeviceLabel string    `json:"deviceLabel"`
	Status      string    `json:"status"`
	CreatedAt   time.Time `json:"createdAt"`
	UpdatedAt   time.Time `json:"updatedAt"`
}

type AttendanceRecord struct {
	ID               uint64     `json:"id"`
	EmployeeID       uint64     `json:"employeeId"`
	AttendanceDate   time.Time  `json:"attendanceDate"`
	FirstConnectAt   *time.Time `json:"firstConnectAt,omitempty"`
	LastDisconnectAt *time.Time `json:"lastDisconnectAt,omitempty"`
	ClockInStatus    string     `json:"clockInStatus"`
	ClockOutStatus   string     `json:"clockOutStatus"`
	ExceptionStatus  string     `json:"exceptionStatus"`
	SourceMode       string     `json:"sourceMode"`
	Version          uint32     `json:"version"`
	LastCalculatedAt *time.Time `json:"lastCalculatedAt,omitempty"`
}
