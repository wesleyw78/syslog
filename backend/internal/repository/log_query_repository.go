package repository

import (
	"context"
	"database/sql"
	"math"
	"strings"
	"time"

	"syslog/internal/domain"
)

const logsPageSize = 10

type LogListParams struct {
	FromDate string
	Page     int
	PageSize int
	Query    string
	Scope    string
	ToDate   string
}

type LogListItem struct {
	Message domain.SyslogMessage
	Event   *domain.ClientEvent
}

type LogListResult struct {
	Items      []LogListItem
	Page       int
	PageSize   int
	TotalItems int
	TotalPages int
}

type LogQueryRepository interface {
	ListPage(ctx context.Context, params LogListParams) (LogListResult, error)
}

type MySQLLogQueryRepository struct {
	db *sql.DB
}

func NewMySQLLogQueryRepository(db *sql.DB) *MySQLLogQueryRepository {
	return &MySQLLogQueryRepository{db: db}
}

func (r *MySQLLogQueryRepository) ListPage(ctx context.Context, params LogListParams) (LogListResult, error) {
	page := normalizeLogsPage(params.Page)
	pageSize := normalizeLogsPageSize(params.PageSize)
	fromDate := normalizeLogFilterDate(params.FromDate)
	query := strings.TrimSpace(params.Query)
	scope := normalizeLogScope(params.Scope)
	toDate := normalizeLogFilterDate(params.ToDate)

	filterClause, filterArgs := buildLogListFilter(query, fromDate, toDate, scope)

	countQuery := `
SELECT COUNT(*)
FROM syslog_messages AS sm
LEFT JOIN client_events AS ce ON ce.syslog_message_id = sm.id
` + filterClause

	var totalItems int
	if err := r.db.QueryRowContext(ctx, trimSQL(countQuery), filterArgs...).Scan(&totalItems); err != nil {
		return LogListResult{}, err
	}

	totalPages := totalPagesForLogs(totalItems, pageSize)
	if totalPages > 0 && page > totalPages {
		page = totalPages
	}

	listQuery := `
SELECT
	sm.id,
	sm.received_at,
	sm.log_time,
	sm.raw_message,
	sm.source_ip,
	sm.protocol,
	sm.parse_status,
	sm.matched_rule_id,
	sm.retention_expire_at,
	sr.name,
	ce.id,
	ce.syslog_message_id,
	ce.event_date,
	ce.event_time,
	ce.event_type,
	ce.station_mac,
	ce.ap_mac,
	ce.ssid,
	ce.ipv4,
	ce.ipv6,
	ce.hostname,
	ce.os_vendor,
	ce.matched_employee_id,
	ce.match_status
FROM syslog_messages AS sm
LEFT JOIN syslog_receive_rules AS sr ON sr.id = sm.matched_rule_id
LEFT JOIN client_events AS ce ON ce.syslog_message_id = sm.id
` + filterClause + `
ORDER BY sm.received_at DESC, sm.id DESC
LIMIT ? OFFSET ?`

	args := append([]any{}, filterArgs...)
	args = append(args, pageSize, logsOffset(page, pageSize))
	rows, err := r.db.QueryContext(ctx, trimSQL(listQuery), args...)
	if err != nil {
		return LogListResult{}, err
	}
	defer rows.Close()

	items := make([]LogListItem, 0)
	for rows.Next() {
		item, err := scanLogListItem(rows)
		if err != nil {
			return LogListResult{}, err
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return LogListResult{}, err
	}

	return LogListResult{
		Items:      items,
		Page:       page,
		PageSize:   pageSize,
		TotalItems: totalItems,
		TotalPages: totalPages,
	}, nil
}

func buildLogListFilter(query string, fromDate string, toDate string, scope string) (string, []any) {
	conditions := make([]string, 0, 4)
	args := make([]any, 0, 16)

	if scope == "matched" {
		conditions = append(conditions, "ce.id IS NOT NULL")
	}

	if query != "" {
		like := "%" + query + "%"
		args = append(args,
			like, like, like, like,
			like, like, like, like,
			like, like, like, like, like,
		)
		conditions = append(conditions, `(
	sm.raw_message LIKE ?
	OR sm.parse_status LIKE ?
	OR sm.source_ip LIKE ?
	OR sm.protocol LIKE ?
	OR ce.event_type LIKE ?
	OR ce.station_mac LIKE ?
	OR ce.hostname LIKE ?
	OR ce.ap_mac LIKE ?
	OR ce.ssid LIKE ?
	OR ce.ipv4 LIKE ?
	OR ce.ipv6 LIKE ?
	OR ce.os_vendor LIKE ?
	OR ce.match_status LIKE ?
)`)
	}

	if fromDate != "" {
		conditions = append(conditions, "sm.received_at >= ?")
		args = append(args, fromDate+" 00:00:00")
	}

	if toDate != "" {
		nextDay := nextLogFilterDay(toDate)
		if nextDay != "" {
			conditions = append(conditions, "sm.received_at < ?")
			args = append(args, nextDay+" 00:00:00")
		}
	}

	if len(conditions) == 0 {
		return "", nil
	}

	return "\nWHERE " + strings.Join(conditions, "\nAND "), args
}

func normalizeLogFilterDate(value string) string {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return ""
	}

	if _, err := time.Parse("2006-01-02", trimmed); err != nil {
		return ""
	}

	return trimmed
}

func nextLogFilterDay(value string) string {
	parsed, err := time.Parse("2006-01-02", value)
	if err != nil {
		return ""
	}

	return parsed.AddDate(0, 0, 1).Format("2006-01-02")
}

func scanLogListItem(rows *sql.Rows) (LogListItem, error) {
	var (
		item              LogListItem
		logTime           sql.NullTime
		matchedRuleID     sql.NullInt64
		matchedRuleName   sql.NullString
		eventID           sql.NullInt64
		eventDate         sql.NullTime
		eventTime         sql.NullTime
		eventType         sql.NullString
		stationMAC        sql.NullString
		apMAC             sql.NullString
		ssid              sql.NullString
		ipv4              sql.NullString
		ipv6              sql.NullString
		hostname          sql.NullString
		osVendor          sql.NullString
		matchedEmployeeID sql.NullInt64
		matchStatus       sql.NullString
		eventMessageID    sql.NullInt64
	)

	if err := rows.Scan(
		&item.Message.ID,
		&item.Message.ReceivedAt,
		&logTime,
		&item.Message.RawMessage,
		&item.Message.SourceIP,
		&item.Message.Protocol,
		&item.Message.ParseStatus,
		&matchedRuleID,
		&item.Message.RetentionExpireAt,
		&matchedRuleName,
		&eventID,
		&eventMessageID,
		&eventDate,
		&eventTime,
		&eventType,
		&stationMAC,
		&apMAC,
		&ssid,
		&ipv4,
		&ipv6,
		&hostname,
		&osVendor,
		&matchedEmployeeID,
		&matchStatus,
	); err != nil {
		return LogListItem{}, err
	}

	item.Message.LogTime = timeFromNullTime(logTime)
	item.Message.MatchedRuleID = uint64FromNullInt64(matchedRuleID)
	item.Message.MatchedRuleName = stringFromNullString(matchedRuleName)

	if eventID.Valid {
		event := domain.ClientEvent{
			ID:                uint64(eventID.Int64),
			EventType:         stringFromNullString(eventType),
			StationMac:        stringFromNullString(stationMAC),
			APMac:             stringFromNullString(apMAC),
			SSID:              stringFromNullString(ssid),
			IPv4:              stringFromNullString(ipv4),
			IPv6:              stringFromNullString(ipv6),
			Hostname:          stringFromNullString(hostname),
			OSVendor:          stringFromNullString(osVendor),
			MatchedEmployeeID: uint64FromNullInt64(matchedEmployeeID),
			MatchStatus:       stringFromNullString(matchStatus),
		}
		if eventMessageID.Valid && eventMessageID.Int64 >= 0 {
			event.SyslogMessageID = uint64(eventMessageID.Int64)
		}
		if date := timeFromNullTime(eventDate); date != nil {
			event.EventDate = *date
		}
		if ts := timeFromNullTime(eventTime); ts != nil {
			event.EventTime = *ts
		}
		item.Event = &event
	}

	return item, nil
}

func normalizeLogsPage(page int) int {
	if page < 1 {
		return 1
	}
	return page
}

func normalizeLogScope(scope string) string {
	if strings.EqualFold(strings.TrimSpace(scope), "all") {
		return "all"
	}
	return "matched"
}

func normalizeLogsPageSize(pageSize int) int {
	if pageSize != logsPageSize {
		return logsPageSize
	}
	return pageSize
}

func logsOffset(page int, pageSize int) int {
	return (page - 1) * pageSize
}

func totalPagesForLogs(totalItems int, pageSize int) int {
	if totalItems <= 0 {
		return 0
	}
	return int(math.Ceil(float64(totalItems) / float64(pageSize)))
}
