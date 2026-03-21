package handlers

import (
	"context"
	"net/http"

	"syslog/internal/domain"
	"syslog/internal/repository"
	"syslog/internal/service"
)

type SettingsAdminWriter interface {
	UpdateSettings(context.Context, []service.SettingWriteInput) ([]domain.SystemSetting, error)
}

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

type settingsWriteRequest struct {
	Items []settingWriteRequest `json:"items"`
}

type settingWriteRequest struct {
	SettingKey   string `json:"settingKey"`
	SettingValue string `json:"settingValue"`
}

func NewSettingsUpdateHandler(admin SettingsAdminWriter) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if admin == nil {
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			return
		}

		var req settingsWriteRequest
		if err := decodeJSONBody(r, &req); err != nil {
			http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
			return
		}

		items := make([]service.SettingWriteInput, 0, len(req.Items))
		for _, item := range req.Items {
			items = append(items, service.SettingWriteInput{
				SettingKey:   item.SettingKey,
				SettingValue: item.SettingValue,
			})
		}

		settings, err := admin.UpdateSettings(r.Context(), items)
		if err != nil {
			status := statusCodeForServiceError(err)
			http.Error(w, http.StatusText(status), status)
			return
		}

		values := make([]any, 0, len(settings))
		for _, setting := range settings {
			values = append(values, setting)
		}

		writeJSON(w, http.StatusOK, listResponse{Items: values})
	}
}
