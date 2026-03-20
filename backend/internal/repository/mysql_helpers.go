package repository

import (
	"database/sql"
	"strings"
	"time"
)

const defaultRecentLimit = 50

func trimSQL(query string) string {
	return strings.TrimSpace(query)
}

func nullableTime(value *time.Time) any {
	if value == nil {
		return nil
	}

	return *value
}

func nullableUint64(value *uint64) any {
	if value == nil {
		return nil
	}

	return int64(*value)
}

func nullableIntArg(value *int) any {
	if value == nil {
		return nil
	}

	return int64(*value)
}

func timeFromNullTime(value sql.NullTime) *time.Time {
	if !value.Valid {
		return nil
	}

	copied := value.Time
	return &copied
}

func uint64FromNullInt64(value sql.NullInt64) *uint64 {
	if !value.Valid || value.Int64 < 0 {
		return nil
	}

	result := uint64(value.Int64)
	return &result
}

func intFromNullInt64(value sql.NullInt64) *int {
	if !value.Valid {
		return nil
	}

	result := int(value.Int64)
	return &result
}

func limitOrDefault(limit int) int {
	if limit <= 0 {
		return defaultRecentLimit
	}

	return limit
}

func parseInsertedID(result sql.Result) (uint64, error) {
	id, err := result.LastInsertId()
	if err != nil {
		return 0, err
	}

	if id < 0 {
		return 0, nil
	}

	return uint64(id), nil
}
