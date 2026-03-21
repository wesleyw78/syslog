package handlers

import (
	"net/http"
	"time"

	"syslog/internal/repository"
)

var asiaShanghai = time.FixedZone("Asia/Shanghai", 8*3600)

func NewAttendanceHandler(repo repository.AttendanceRepository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if repo == nil {
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			return
		}

		now := time.Now().In(asiaShanghai)
		records, err := repo.ListByDateRange(r.Context(), now.AddDate(0, 0, -30), now)
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
