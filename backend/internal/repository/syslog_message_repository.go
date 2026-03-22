package repository

import (
	"context"
	"database/sql"

	"syslog/internal/domain"
)

type SyslogMessageRepository interface {
	Save(ctx context.Context, message *domain.SyslogMessage) error
	ListRecent(ctx context.Context, limit int) ([]domain.SyslogMessage, error)
}

type MySQLSyslogMessageRepository struct {
	db *sql.DB
}

func NewMySQLSyslogMessageRepository(db *sql.DB) *MySQLSyslogMessageRepository {
	return &MySQLSyslogMessageRepository{db: db}
}

func (r *MySQLSyslogMessageRepository) Save(ctx context.Context, message *domain.SyslogMessage) error {
	const query = `
INSERT INTO syslog_messages (
	received_at,
	log_time,
	raw_message,
	source_ip,
	protocol,
	parse_status,
	matched_rule_id,
	retention_expire_at
) VALUES (?, ?, ?, ?, ?, ?, ?, ?)`

	result, err := r.db.ExecContext(
		ctx,
		trimSQL(query),
		message.ReceivedAt,
		nullableTime(message.LogTime),
		message.RawMessage,
		message.SourceIP,
		message.Protocol,
		message.ParseStatus,
		nullableUint64(message.MatchedRuleID),
		message.RetentionExpireAt,
	)
	if err != nil {
		return err
	}

	id, err := parseInsertedID(result)
	if err != nil {
		return err
	}

	message.ID = id
	return nil
}

func (r *MySQLSyslogMessageRepository) ListRecent(ctx context.Context, limit int) ([]domain.SyslogMessage, error) {
	const query = `
SELECT id, received_at, log_time, raw_message, source_ip, protocol, parse_status, matched_rule_id, retention_expire_at
FROM syslog_messages
ORDER BY received_at DESC, id DESC
LIMIT ?`

	rows, err := r.db.QueryContext(ctx, trimSQL(query), limitOrDefault(limit))
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	messages := make([]domain.SyslogMessage, 0)
	for rows.Next() {
		var message domain.SyslogMessage
		var logTime sql.NullTime
		var matchedRuleID sql.NullInt64
		if err := rows.Scan(
			&message.ID,
			&message.ReceivedAt,
			&logTime,
			&message.RawMessage,
			&message.SourceIP,
			&message.Protocol,
			&message.ParseStatus,
			&matchedRuleID,
			&message.RetentionExpireAt,
		); err != nil {
			return nil, err
		}

		message.LogTime = timeFromNullTime(logTime)
		message.MatchedRuleID = uint64FromNullInt64(matchedRuleID)
		messages = append(messages, message)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return messages, nil
}
