package handlers

import (
	"net/http"

	"syslog/internal/repository"
)

func NewSettingsHandler(repo repository.SystemSettingRepository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if repo == nil {
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			return
		}

		settings, err := repo.List(r.Context())
		if err != nil {
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			return
		}

		items := make([]any, 0, len(settings))
		for _, setting := range settings {
			items = append(items, setting)
		}

		writeJSON(w, http.StatusOK, listResponse{Items: items})
	}
}
