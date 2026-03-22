package repository

import (
	"context"
	"database/sql"

	"syslog/internal/domain"
)

type SyslogReceiveRuleRepository interface {
	List(ctx context.Context) ([]domain.SyslogReceiveRule, error)
	ListEnabled(ctx context.Context) ([]domain.SyslogReceiveRule, error)
	FindByID(ctx context.Context, id uint64) (*domain.SyslogReceiveRule, error)
	Create(ctx context.Context, rule *domain.SyslogReceiveRule) error
	Update(ctx context.Context, rule *domain.SyslogReceiveRule) error
	Delete(ctx context.Context, id uint64) error
	Move(ctx context.Context, id uint64, direction string) error
}

type MySQLSyslogReceiveRuleRepository struct {
	db *sql.DB
}

func NewMySQLSyslogReceiveRuleRepository(db *sql.DB) *MySQLSyslogReceiveRuleRepository {
	return &MySQLSyslogReceiveRuleRepository{db: db}
}

func (r *MySQLSyslogReceiveRuleRepository) List(ctx context.Context) ([]domain.SyslogReceiveRule, error) {
	const query = `
SELECT id, sort_order, name, enabled, event_type, message_pattern, station_mac_group, ap_mac_group, ssid_group, ipv4_group, ipv6_group, hostname_group, os_vendor_group, event_time_group, event_time_layout, created_at, updated_at
FROM syslog_receive_rules
ORDER BY sort_order ASC, id ASC`

	rows, err := r.db.QueryContext(ctx, trimSQL(query))
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	rules := make([]domain.SyslogReceiveRule, 0)
	for rows.Next() {
		rule, err := scanSyslogReceiveRule(rows)
		if err != nil {
			return nil, err
		}
		rules = append(rules, rule)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return rules, nil
}

func (r *MySQLSyslogReceiveRuleRepository) ListEnabled(ctx context.Context) ([]domain.SyslogReceiveRule, error) {
	const query = `
SELECT id, sort_order, name, enabled, event_type, message_pattern, station_mac_group, ap_mac_group, ssid_group, ipv4_group, ipv6_group, hostname_group, os_vendor_group, event_time_group, event_time_layout, created_at, updated_at
FROM syslog_receive_rules
WHERE enabled = 1
ORDER BY sort_order ASC, id ASC`

	rows, err := r.db.QueryContext(ctx, trimSQL(query))
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	rules := make([]domain.SyslogReceiveRule, 0)
	for rows.Next() {
		rule, err := scanSyslogReceiveRule(rows)
		if err != nil {
			return nil, err
		}
		rules = append(rules, rule)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return rules, nil
}

func (r *MySQLSyslogReceiveRuleRepository) FindByID(ctx context.Context, id uint64) (*domain.SyslogReceiveRule, error) {
	const query = `
SELECT id, sort_order, name, enabled, event_type, message_pattern, station_mac_group, ap_mac_group, ssid_group, ipv4_group, ipv6_group, hostname_group, os_vendor_group, event_time_group, event_time_layout, created_at, updated_at
FROM syslog_receive_rules
WHERE id = ?
LIMIT 1`

	row := r.db.QueryRowContext(ctx, trimSQL(query), id)
	rule, err := scanSyslogReceiveRule(row)
	if err != nil {
		return nil, err
	}

	return &rule, nil
}

func (r *MySQLSyslogReceiveRuleRepository) Create(ctx context.Context, rule *domain.SyslogReceiveRule) error {
	const query = `
INSERT INTO syslog_receive_rules (
	sort_order,
	name,
	enabled,
	event_type,
	message_pattern,
	station_mac_group,
	ap_mac_group,
	ssid_group,
	ipv4_group,
	ipv6_group,
	hostname_group,
	os_vendor_group,
	event_time_group,
	event_time_layout
) VALUES (
	(SELECT COALESCE(MAX(sort_order), 0) + 1 FROM syslog_receive_rules AS ordered_rules),
	?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?
)`

	result, err := r.db.ExecContext(
		ctx,
		trimSQL(query),
		rule.Name,
		rule.Enabled,
		rule.EventType,
		rule.MessagePattern,
		rule.StationMacGroup,
		rule.APMacGroup,
		rule.SSIDGroup,
		rule.IPv4Group,
		rule.IPv6Group,
		rule.HostnameGroup,
		rule.OSVendorGroup,
		rule.EventTimeGroup,
		rule.EventTimeLayout,
	)
	if err != nil {
		return err
	}

	id, err := parseInsertedID(result)
	if err != nil {
		return err
	}
	rule.ID = id
	return nil
}

func (r *MySQLSyslogReceiveRuleRepository) Update(ctx context.Context, rule *domain.SyslogReceiveRule) error {
	const query = `
UPDATE syslog_receive_rules
SET
	name = ?,
	enabled = ?,
	event_type = ?,
	message_pattern = ?,
	station_mac_group = ?,
	ap_mac_group = ?,
	ssid_group = ?,
	ipv4_group = ?,
	ipv6_group = ?,
	hostname_group = ?,
	os_vendor_group = ?,
	event_time_group = ?,
	event_time_layout = ?
WHERE id = ?`

	_, err := r.db.ExecContext(
		ctx,
		trimSQL(query),
		rule.Name,
		rule.Enabled,
		rule.EventType,
		rule.MessagePattern,
		rule.StationMacGroup,
		rule.APMacGroup,
		rule.SSIDGroup,
		rule.IPv4Group,
		rule.IPv6Group,
		rule.HostnameGroup,
		rule.OSVendorGroup,
		rule.EventTimeGroup,
		rule.EventTimeLayout,
		rule.ID,
	)
	return err
}

func (r *MySQLSyslogReceiveRuleRepository) Delete(ctx context.Context, id uint64) error {
	const query = `DELETE FROM syslog_receive_rules WHERE id = ?`
	_, err := r.db.ExecContext(ctx, trimSQL(query), id)
	return err
}

func (r *MySQLSyslogReceiveRuleRepository) Move(ctx context.Context, id uint64, direction string) error {
	const currentQuery = `
SELECT id, sort_order
FROM syslog_receive_rules
WHERE id = ?
LIMIT 1`

	var currentID uint64
	var currentSortOrder uint32
	if err := r.db.QueryRowContext(ctx, trimSQL(currentQuery), id).Scan(&currentID, &currentSortOrder); err != nil {
		return err
	}

	comparator := "<"
	orderDirection := "DESC"
	if direction == "down" {
		comparator = ">"
		orderDirection = "ASC"
	}

	neighborQuery := `
SELECT id, sort_order
FROM syslog_receive_rules
WHERE sort_order ` + comparator + ` ?
ORDER BY sort_order ` + orderDirection + `, id ` + orderDirection + `
LIMIT 1`

	var neighborID uint64
	var neighborSortOrder uint32
	if err := r.db.QueryRowContext(ctx, trimSQL(neighborQuery), currentSortOrder).Scan(&neighborID, &neighborSortOrder); err != nil {
		if err == sql.ErrNoRows {
			return nil
		}
		return err
	}

	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() {
		if err != nil {
			_ = tx.Rollback()
		}
	}()

	const swapQuery = `
UPDATE syslog_receive_rules
SET sort_order = ?
WHERE id = ?`

	if _, err = tx.ExecContext(ctx, trimSQL(swapQuery), neighborSortOrder, currentID); err != nil {
		return err
	}
	if _, err = tx.ExecContext(ctx, trimSQL(swapQuery), currentSortOrder, neighborID); err != nil {
		return err
	}

	return tx.Commit()
}

type syslogRuleScanner interface {
	Scan(dest ...any) error
}

func scanSyslogReceiveRule(scanner syslogRuleScanner) (domain.SyslogReceiveRule, error) {
	var (
		rule    domain.SyslogReceiveRule
		enabled bool
	)
	if err := scanner.Scan(
		&rule.ID,
		&rule.SortOrder,
		&rule.Name,
		&enabled,
		&rule.EventType,
		&rule.MessagePattern,
		&rule.StationMacGroup,
		&rule.APMacGroup,
		&rule.SSIDGroup,
		&rule.IPv4Group,
		&rule.IPv6Group,
		&rule.HostnameGroup,
		&rule.OSVendorGroup,
		&rule.EventTimeGroup,
		&rule.EventTimeLayout,
		&rule.CreatedAt,
		&rule.UpdatedAt,
	); err != nil {
		return domain.SyslogReceiveRule{}, err
	}
	rule.Enabled = enabled
	return rule, nil
}
