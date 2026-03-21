package httpapi

import (
	"net/http"

	"syslog/internal/http/handlers"
	"syslog/internal/repository"
)

type Dependencies struct {
	Employees      repository.EmployeeRepository
	SyslogMessages repository.SyslogMessageRepository
	ClientEvents   repository.ClientEventRepository
	Attendance     repository.AttendanceRepository
	Settings       repository.SystemSettingRepository
}

func NewRouter(deps Dependencies) http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/attendance", handlers.NewAttendanceHandler(deps.Attendance))
	mux.HandleFunc("GET /api/employees", handlers.NewEmployeesHandler(deps.Employees))
	mux.HandleFunc("GET /api/logs", handlers.NewLogsHandler(deps.SyslogMessages, deps.ClientEvents))
	mux.HandleFunc("GET /api/settings", handlers.NewSettingsHandler(deps.Settings))

	return mux
}

func NewServer(addr string, deps Dependencies) *http.Server {
	return &http.Server{
		Addr:    addr,
		Handler: NewRouter(deps),
	}
}
