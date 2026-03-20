package httpapi

import (
	"net/http"

	"syslog/internal/http/handlers"
)

// Dependencies reserves a thin seam for future HTTP wiring.
type Dependencies struct{}

func NewRouter(_ Dependencies) http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/attendance", handlers.NewAttendanceHandler())
	mux.HandleFunc("GET /api/employees", handlers.NewEmployeesHandler())
	mux.HandleFunc("GET /api/logs", handlers.NewLogsHandler())
	mux.HandleFunc("GET /api/settings", handlers.NewSettingsHandler())

	return mux
}
