package domain

import "time"

type Employee struct {
	ID         uint64
	EmployeeNo string
	Name       string
	Timezone   string
	Active     bool
	CreatedAt  time.Time
	UpdatedAt  time.Time
}

type EmployeeDevice struct {
	ID               uint64
	EmployeeID       uint64
	DeviceIdentifier string
	DeviceName       string
	IsPrimary        bool
	CreatedAt        time.Time
	UpdatedAt        time.Time
}

type AttendanceRecord struct {
	ID               uint64
	EmployeeID       uint64
	WorkDate         time.Time
	FirstCheckInAt   *time.Time
	LastCheckOutAt   *time.Time
	Status           string
	SourceEventCount uint32
	CreatedAt        time.Time
	UpdatedAt        time.Time
}
