package handlers

import (
	"net/http"

	"syslog/internal/repository"
)

func NewEmployeesHandler(repo repository.EmployeeRepository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if repo == nil {
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			return
		}

		employees, err := repo.List(r.Context())
		if err != nil {
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			return
		}

		items := make([]any, 0, len(employees))
		for _, employee := range employees {
			items = append(items, employee)
		}

		writeJSON(w, http.StatusOK, listResponse{Items: items})
	}
}
