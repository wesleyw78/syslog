package handlers

import (
	"context"
	"net/http"
	"time"

	"syslog/internal/repository"
	"syslog/internal/service"
)

var asiaShanghai = time.FixedZone("Asia/Shanghai", 8*3600)
var attendanceNow = time.Now

type AttendanceCorrectionWriter interface {
	CorrectAttendance(context.Context, uint64, service.AttendanceCorrectionInput) (*service.AttendanceCorrectionResult, error)
}

func NewAttendanceHandler(repo repository.AttendanceRepository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if repo == nil {
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			return
		}

		from, to := attendanceWindow(attendanceNow())
		records, err := repo.ListByDateRange(r.Context(), from, to)
		if err != nil {
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			return
		}

		items := make([]any, 0, len(records))
		for _, record := range records {
			items = append(items, record)
		}

		writeJSON(w, http.StatusOK, listResponse{Items: items})
	}
}

func attendanceWindow(now time.Time) (time.Time, time.Time) {
	now = now.In(asiaShanghai)
	currentDayStart := startOfDay(now)
	from := currentDayStart.AddDate(0, 0, -29)
	to := endOfDay(now)
	return from, to
}

func startOfDay(value time.Time) time.Time {
	y, m, d := value.Date()
	return time.Date(y, m, d, 0, 0, 0, 0, asiaShanghai)
}

func endOfDay(value time.Time) time.Time {
	y, m, d := value.Date()
	return time.Date(y, m, d, 23, 59, 59, 999999999, asiaShanghai)
}

type attendanceCorrectionRequest struct {
	FirstConnectAt   *time.Time `json:"firstConnectAt"`
	LastDisconnectAt *time.Time `json:"lastDisconnectAt"`
}

func NewAttendanceCorrectionHandler(admin AttendanceCorrectionWriter) http.HandlerFunc {
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

		var req attendanceCorrectionRequest
		if err := decodeJSONBody(r, &req); err != nil {
			http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
			return
		}

		result, err := admin.CorrectAttendance(r.Context(), id, service.AttendanceCorrectionInput{
			FirstConnectAt:   req.FirstConnectAt,
			LastDisconnectAt: req.LastDisconnectAt,
		})
		if err != nil {
			status := statusCodeForServiceError(err)
			http.Error(w, http.StatusText(status), status)
			return
		}

		writeJSON(w, http.StatusOK, map[string]any{
			"attendance": result.Record,
			"reports":    result.Reports,
		})
	}
}
