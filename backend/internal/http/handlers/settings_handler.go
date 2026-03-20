package handlers

import "net/http"

func NewSettingsHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, listResponse{
			Items: make([]any, 0),
		})
	}
}
