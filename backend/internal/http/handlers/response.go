package handlers

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"

	"github.com/go-sql-driver/mysql"

	"syslog/internal/service"
)

type listResponse struct {
	Items []any `json:"items"`
}

func writeJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)

	if err := json.NewEncoder(w).Encode(payload); err != nil {
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
	}
}

func decodeJSONBody(r *http.Request, dst any) error {
	if r.Body == nil {
		return errors.New("request body is required")
	}

	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(dst); err != nil {
		return err
	}

	var trailing json.RawMessage
	if err := decoder.Decode(&trailing); err != io.EOF {
		if err == nil {
			return errors.New("unexpected trailing JSON")
		}
		return err
	}

	return nil
}

func parseUint64PathValue(r *http.Request, key string) (uint64, error) {
	value := r.PathValue(key)
	if value == "" {
		return 0, fmt.Errorf("missing %s", key)
	}

	parsed, err := strconv.ParseUint(value, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("invalid %s: %w", key, err)
	}

	return parsed, nil
}

func statusCodeForServiceError(err error) int {
	switch {
	case errors.Is(err, service.ErrInvalidEmployeeInput):
		return http.StatusBadRequest
	case errors.Is(err, service.ErrInvalidSettingsInput):
		return http.StatusBadRequest
	case errors.Is(err, service.ErrInvalidAttendanceCorrection):
		return http.StatusBadRequest
	case errors.Is(err, sql.ErrNoRows):
		return http.StatusNotFound
	case isDuplicateKeyError(err):
		return http.StatusConflict
	default:
		return http.StatusInternalServerError
	}
}

func isDuplicateKeyError(err error) bool {
	if err == nil {
		return false
	}

	var mysqlErr *mysql.MySQLError
	return errors.As(err, &mysqlErr) && mysqlErr.Number == 1062
}
