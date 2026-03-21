package handlers

import (
	"net/http"
	"time"

	"syslog/internal/repository"
)

var asiaShanghai = time.FixedZone("Asia/Shanghai", 8*3600)
var attendanceNow = time.Now

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
