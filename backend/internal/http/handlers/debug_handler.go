package handlers

import (
	"context"
	"net/http"
	"time"

	"syslog/internal/service"
)

type DebugAdminWriter interface {
	InjectSyslog(context.Context, service.DebugSyslogInjectInput) (*service.DebugSyslogInjectResult, error)
	DispatchAttendanceReport(context.Context, uint64, service.DebugAttendanceDispatchInput) (*service.DebugAttendanceDispatchResult, error)
}

type debugSyslogRequest struct {
	RawMessage string `json:"rawMessage"`
	ReceivedAt string `json:"receivedAt"`
}

type debugAttendanceDispatchRequest struct {
	ReportType string `json:"reportType"`
}

func NewDebugSyslogHandler(admin DebugAdminWriter) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if admin == nil {
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			return
		}

		var req debugSyslogRequest
		if err := decodeJSONBody(r, &req); err != nil {
			http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
			return
		}

		result, err := admin.InjectSyslog(r.Context(), service.DebugSyslogInjectInput{
			RawMessage: req.RawMessage,
			ReceivedAt: req.ReceivedAt,
		})
		if err != nil {
			status := statusCodeForServiceError(err)
			http.Error(w, http.StatusText(status), status)
			return
		}

		writeJSON(w, http.StatusOK, map[string]any{
			"accepted":    result.Accepted,
			"receivedAt":  result.ReceivedAt.Format(time.RFC3339),
			"parseStatus": result.ParseStatus,
			"parseError":  result.ParseError,
		})
	}
}

func NewDebugAttendanceDispatchHandler(admin DebugAdminWriter) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if admin == nil {
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			return
		}

		id, err := parseUint64PathValue(r, "id")
		if err != nil {
			http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
			return
		}

		var req debugAttendanceDispatchRequest
		if err := decodeJSONBody(r, &req); err != nil {
			http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
			return
		}

		result, err := admin.DispatchAttendanceReport(r.Context(), id, service.DebugAttendanceDispatchInput{
			ReportType: req.ReportType,
		})
		if err != nil {
			status := statusCodeForServiceError(err)
			http.Error(w, http.StatusText(status), status)
			return
		}

		writeJSON(w, http.StatusOK, map[string]any{
			"attendance": result.Record,
			"report":     result.Report,
		})
	}
}
