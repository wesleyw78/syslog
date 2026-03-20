package repository

import (
	"context"
	"database/sql"

	"syslog/internal/domain"
)

type ClientEventRepository interface {
	Save(ctx context.Context, event *domain.ClientEvent) error
	ListRecent(ctx context.Context, limit int) ([]domain.ClientEvent, error)
}

type MySQLClientEventRepository struct {
	db *sql.DB
}

func NewMySQLClientEventRepository(db *sql.DB) *MySQLClientEventRepository {
	return &MySQLClientEventRepository{db: db}
}

func (r *MySQLClientEventRepository) Save(ctx context.Context, event *domain.ClientEvent) error {
	const query = `
INSERT INTO client_events (
	syslog_message_id,
	event_date,
	event_time,
	event_type,
	station_mac,
	ap_mac,
	ssid,
	ipv4,
	ipv6,
	hostname,
	os_vendor,
	matched_employee_id,
	match_status
) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`

	result, err := r.db.ExecContext(
		ctx,
		trimSQL(query),
		event.SyslogMessageID,
		event.EventDate,
		event.EventTime,
		event.EventType,
		event.StationMac,
		event.APMac,
		event.SSID,
		event.IPv4,
		event.IPv6,
		event.Hostname,
		event.OSVendor,
		nullableUint64(event.MatchedEmployeeID),
		event.MatchStatus,
	)
	if err != nil {
		return err
	}

	id, err := parseInsertedID(result)
	if err != nil {
		return err
	}

	event.ID = id
	return nil
}

func (r *MySQLClientEventRepository) ListRecent(ctx context.Context, limit int) ([]domain.ClientEvent, error) {
	const query = `
SELECT id, syslog_message_id, event_date, event_time, event_type, station_mac, ap_mac, ssid, ipv4, ipv6, hostname, os_vendor, matched_employee_id, match_status
FROM client_events
ORDER BY event_time DESC, id DESC
LIMIT ?`

	rows, err := r.db.QueryContext(ctx, trimSQL(query), limitOrDefault(limit))
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	events := make([]domain.ClientEvent, 0)
	for rows.Next() {
		var event domain.ClientEvent
		var matchedEmployeeID sql.NullInt64
		if err := rows.Scan(
			&event.ID,
			&event.SyslogMessageID,
			&event.EventDate,
			&event.EventTime,
			&event.EventType,
			&event.StationMac,
			&event.APMac,
			&event.SSID,
			&event.IPv4,
			&event.IPv6,
			&event.Hostname,
			&event.OSVendor,
			&matchedEmployeeID,
			&event.MatchStatus,
		); err != nil {
			return nil, err
		}

		event.MatchedEmployeeID = uint64FromNullInt64(matchedEmployeeID)
		events = append(events, event)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return events, nil
}
