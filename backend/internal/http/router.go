package httpapi

import (
	"net/http"

	"syslog/internal/http/handlers"
	"syslog/internal/repository"
)

type Dependencies struct {
	Employees       repository.EmployeeRepository
	EmployeeAdmin   handlers.EmployeeAdminWriter
	SyslogMessages  repository.SyslogMessageRepository
	ClientEvents    repository.ClientEventRepository
	Logs            repository.LogQueryRepository
	Attendance      repository.AttendanceRepository
	AttendanceAdmin handlers.AttendanceCorrectionWriter
	DebugAdmin      handlers.DebugAdminWriter
	Settings        repository.SystemSettingRepository
	SettingsAdmin   handlers.SettingsAdminWriter
	SyslogRules     repository.SyslogReceiveRuleRepository
	SyslogRuleAdmin handlers.SyslogRuleAdminWriter
}

func NewRouter(deps Dependencies) http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/attendance", handlers.NewAttendanceHandler(deps.Attendance))
	mux.HandleFunc("POST /api/attendance/{id}/correction", handlers.NewAttendanceCorrectionHandler(deps.AttendanceAdmin))
	mux.HandleFunc("GET /api/employees", handlers.NewEmployeesHandler(deps.Employees))
	mux.HandleFunc("POST /api/employees", handlers.NewEmployeeCreateHandler(deps.EmployeeAdmin))
	mux.HandleFunc("POST /api/employees/{id}/disable", handlers.NewEmployeeDisableHandler(deps.EmployeeAdmin))
	mux.HandleFunc("PUT /api/employees/{id}", handlers.NewEmployeeUpdateHandler(deps.EmployeeAdmin))
	mux.HandleFunc("GET /api/logs", handlers.NewLogsHandler(deps.Logs))
	mux.HandleFunc("GET /api/syslog-rules", handlers.NewSyslogRulesHandler(deps.SyslogRules))
	mux.HandleFunc("POST /api/syslog-rules", handlers.NewSyslogRuleCreateHandler(deps.SyslogRuleAdmin))
	mux.HandleFunc("POST /api/syslog-rules/preview", handlers.NewSyslogRulePreviewHandler(deps.SyslogRuleAdmin))
	mux.HandleFunc("POST /api/syslog-rules/{id}/move", handlers.NewSyslogRuleMoveHandler(deps.SyslogRuleAdmin))
	mux.HandleFunc("PUT /api/syslog-rules/{id}", handlers.NewSyslogRuleUpdateHandler(deps.SyslogRuleAdmin))
	mux.HandleFunc("DELETE /api/syslog-rules/{id}", handlers.NewSyslogRuleDeleteHandler(deps.SyslogRuleAdmin))
	mux.HandleFunc("POST /api/debug/syslog", handlers.NewDebugSyslogHandler(deps.DebugAdmin))
	mux.HandleFunc("POST /api/debug/attendance/{id}/dispatch", handlers.NewDebugAttendanceDispatchHandler(deps.DebugAdmin))
	mux.HandleFunc("GET /api/settings", handlers.NewSettingsHandler(deps.Settings))
	mux.HandleFunc("PUT /api/settings", handlers.NewSettingsUpdateHandler(deps.SettingsAdmin))

	return mux
}

func NewServer(addr string, deps Dependencies) *http.Server {
	return &http.Server{
		Addr:    addr,
		Handler: NewRouter(deps),
	}
}
