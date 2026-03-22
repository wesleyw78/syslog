CREATE TABLE IF NOT EXISTS employees (
    id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
    employee_no VARCHAR(64) NOT NULL,
    system_no VARCHAR(64) NOT NULL,
    feishu_employee_id VARCHAR(128) NULL,
    name VARCHAR(128) NOT NULL,
    status VARCHAR(32) NOT NULL DEFAULT 'active',
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    PRIMARY KEY (id),
    UNIQUE KEY uk_employees_employee_no (employee_no),
    UNIQUE KEY uk_employees_system_no (system_no),
    UNIQUE KEY uk_employees_feishu_employee_id (feishu_employee_id),
    KEY idx_employees_status (status)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE IF NOT EXISTS employee_devices (
    id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
    employee_id BIGINT UNSIGNED NOT NULL,
    mac_address VARCHAR(17) NOT NULL,
    device_label VARCHAR(128) NOT NULL DEFAULT '',
    status VARCHAR(32) NOT NULL DEFAULT 'active',
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    PRIMARY KEY (id),
    UNIQUE KEY uk_employee_devices_mac_address (mac_address),
    KEY idx_employee_devices_employee_id (employee_id),
    KEY idx_employee_devices_status (status),
    CONSTRAINT fk_employee_devices_employee
        FOREIGN KEY (employee_id) REFERENCES employees (id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE IF NOT EXISTS syslog_messages (
    id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
    received_at DATETIME NOT NULL,
    log_time DATETIME NULL,
    raw_message TEXT NOT NULL,
    source_ip VARCHAR(45) NOT NULL DEFAULT '',
    protocol VARCHAR(16) NOT NULL DEFAULT 'udp',
    parse_status VARCHAR(32) NOT NULL DEFAULT 'pending',
    matched_rule_id BIGINT UNSIGNED NULL,
    retention_expire_at DATETIME NOT NULL,
    PRIMARY KEY (id),
    KEY idx_syslog_messages_received_at (received_at),
    KEY idx_syslog_messages_parse_status (parse_status),
    KEY idx_syslog_messages_matched_rule_id (matched_rule_id),
    KEY idx_syslog_messages_retention_expire_at (retention_expire_at)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE IF NOT EXISTS client_events (
    id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
    syslog_message_id BIGINT UNSIGNED NOT NULL,
    event_date DATE NOT NULL,
    event_time DATETIME NOT NULL,
    event_type VARCHAR(64) NOT NULL,
    station_mac VARCHAR(17) NOT NULL,
    ap_mac VARCHAR(17) NOT NULL DEFAULT '',
    ssid VARCHAR(128) NOT NULL DEFAULT '',
    ipv4 VARCHAR(45) NOT NULL DEFAULT '',
    ipv6 VARCHAR(45) NOT NULL DEFAULT '',
    hostname VARCHAR(255) NOT NULL DEFAULT '',
    os_vendor VARCHAR(128) NOT NULL DEFAULT '',
    matched_employee_id BIGINT UNSIGNED NULL,
    match_status VARCHAR(32) NOT NULL DEFAULT 'unmatched',
    PRIMARY KEY (id),
    KEY idx_client_events_event_date_time (event_date, event_time),
    KEY idx_client_events_station_mac (station_mac),
    KEY idx_client_events_matched_employee_id (matched_employee_id),
    KEY idx_client_events_match_status (match_status),
    CONSTRAINT fk_client_events_syslog_message
        FOREIGN KEY (syslog_message_id) REFERENCES syslog_messages (id),
    CONSTRAINT fk_client_events_matched_employee
        FOREIGN KEY (matched_employee_id) REFERENCES employees (id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE IF NOT EXISTS syslog_receive_rules (
    id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
    sort_order INT UNSIGNED NOT NULL DEFAULT 0,
    name VARCHAR(191) NOT NULL,
    enabled TINYINT(1) NOT NULL DEFAULT 1,
    event_type VARCHAR(32) NOT NULL,
    message_pattern TEXT NOT NULL,
    station_mac_group VARCHAR(64) NOT NULL DEFAULT '',
    ap_mac_group VARCHAR(64) NOT NULL DEFAULT '',
    ssid_group VARCHAR(64) NOT NULL DEFAULT '',
    ipv4_group VARCHAR(64) NOT NULL DEFAULT '',
    ipv6_group VARCHAR(64) NOT NULL DEFAULT '',
    hostname_group VARCHAR(64) NOT NULL DEFAULT '',
    os_vendor_group VARCHAR(64) NOT NULL DEFAULT '',
    event_time_group VARCHAR(64) NOT NULL DEFAULT '',
    event_time_layout VARCHAR(191) NOT NULL DEFAULT '',
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    PRIMARY KEY (id),
    UNIQUE KEY uk_syslog_receive_rules_name (name),
    KEY idx_syslog_receive_rules_enabled (enabled, sort_order, id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

SET @syslog_rules_has_sort_order = (
    SELECT COUNT(*)
    FROM INFORMATION_SCHEMA.COLUMNS
    WHERE TABLE_SCHEMA = DATABASE()
      AND TABLE_NAME = 'syslog_receive_rules'
      AND COLUMN_NAME = 'sort_order'
);
SET @syslog_rules_sort_order_sql = IF(
    @syslog_rules_has_sort_order = 0,
    'ALTER TABLE syslog_receive_rules ADD COLUMN sort_order INT UNSIGNED NOT NULL DEFAULT 0 AFTER id',
    'SELECT 1'
);
PREPARE stmt FROM @syslog_rules_sort_order_sql;
EXECUTE stmt;
DEALLOCATE PREPARE stmt;

UPDATE syslog_receive_rules
SET sort_order = id
WHERE sort_order = 0;

INSERT IGNORE INTO syslog_receive_rules (
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
) VALUES
(
    1,
    '默认 connect 规则',
    1,
    'connect',
    'connect .*?Station\\[(?P<station_mac>[^\\]]+)\\](?:.*?AP\\[(?P<ap_mac>[^\\]]+)\\])?(?:.*?ssid\\[(?P<ssid>[^\\]]+)\\])?(?:.*?ipv4\\[(?P<ipv4>[^\\]]+)\\])?(?:.*?ipv6\\[(?P<ipv6>[^\\]]+)\\])?(?:.*?osvendor\\[(?P<os_vendor>[^\\]]+)\\])?(?:.*?hostname\\[(?P<hostname>[^\\]]+)\\])?',
    'station_mac',
    'ap_mac',
    'ssid',
    'ipv4',
    'ipv6',
    'hostname',
    'os_vendor',
    '',
    ''
),
(
    2,
    '默认 disconnect 规则',
    1,
    'disconnect',
    'disconnect .*?Station\\[(?P<station_mac>[^\\]]+)\\](?:.*?AP\\[(?P<ap_mac>[^\\]]+)\\])?(?:.*?ssid\\[(?P<ssid>[^\\]]+)\\])?(?:.*?ipv4\\[(?P<ipv4>[^\\]]+)\\])?(?:.*?ipv6\\[(?P<ipv6>[^\\]]+)\\])?(?:.*?osvendor\\[(?P<os_vendor>[^\\]]+)\\])?(?:.*?hostname\\[(?P<hostname>[^\\]]+)\\])?',
    'station_mac',
    'ap_mac',
    'ssid',
    'ipv4',
    'ipv6',
    'hostname',
    'os_vendor',
    '',
    ''
);

CREATE TABLE IF NOT EXISTS attendance_records (
    id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
    employee_id BIGINT UNSIGNED NOT NULL,
    attendance_date DATE NOT NULL,
    first_connect_at DATETIME NULL,
    last_disconnect_at DATETIME NULL,
    clock_in_status VARCHAR(32) NOT NULL DEFAULT 'pending',
    clock_out_status VARCHAR(32) NOT NULL DEFAULT 'pending',
    exception_status VARCHAR(32) NOT NULL DEFAULT 'none',
    source_mode VARCHAR(32) NOT NULL DEFAULT 'auto',
    version INT UNSIGNED NOT NULL DEFAULT 1,
    last_calculated_at DATETIME NULL,
    PRIMARY KEY (id),
    UNIQUE KEY uk_attendance_records_employee_date (employee_id, attendance_date),
    KEY idx_attendance_records_clock_in_status (clock_in_status),
    KEY idx_attendance_records_clock_out_status (clock_out_status),
    KEY idx_attendance_records_exception_status (exception_status),
    CONSTRAINT fk_attendance_records_employee
        FOREIGN KEY (employee_id) REFERENCES employees (id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE IF NOT EXISTS attendance_reports (
    id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
    attendance_record_id BIGINT UNSIGNED NOT NULL,
    report_type VARCHAR(32) NOT NULL,
    idempotency_key VARCHAR(191) NOT NULL,
    payload_json JSON NULL,
    target_url VARCHAR(1024) NOT NULL,
    external_record_id VARCHAR(191) NOT NULL DEFAULT '',
    delete_record_id VARCHAR(191) NOT NULL DEFAULT '',
    report_status VARCHAR(32) NOT NULL DEFAULT 'pending',
    response_code INT NULL,
    response_body TEXT NULL,
    notification_status VARCHAR(32) NOT NULL DEFAULT 'skipped',
    notification_message_id VARCHAR(191) NOT NULL DEFAULT '',
    notification_response_code INT NULL,
    notification_response_body TEXT NULL,
    notification_sent_at DATETIME NULL,
    notification_retry_count INT UNSIGNED NOT NULL DEFAULT 0,
    reported_at DATETIME NULL,
    retry_count INT UNSIGNED NOT NULL DEFAULT 0,
    PRIMARY KEY (id),
    UNIQUE KEY uk_attendance_reports_idempotency_key (idempotency_key),
    KEY idx_attendance_reports_record_type (attendance_record_id, report_type),
    KEY idx_attendance_reports_status (report_status),
    KEY idx_attendance_reports_notification_dispatch (report_status, notification_status, notification_retry_count),
    CONSTRAINT fk_attendance_reports_record
        FOREIGN KEY (attendance_record_id) REFERENCES attendance_records (id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

SET @employees_has_feishu_employee_id = (
    SELECT COUNT(*)
    FROM INFORMATION_SCHEMA.COLUMNS
    WHERE TABLE_SCHEMA = DATABASE()
      AND TABLE_NAME = 'employees'
      AND COLUMN_NAME = 'feishu_employee_id'
);
SET @employees_feishu_column_sql = IF(
    @employees_has_feishu_employee_id = 0,
    'ALTER TABLE employees ADD COLUMN feishu_employee_id VARCHAR(128) NULL AFTER system_no',
    'SELECT 1'
);
PREPARE stmt FROM @employees_feishu_column_sql;
EXECUTE stmt;
DEALLOCATE PREPARE stmt;

SET @syslog_messages_has_matched_rule_id = (
    SELECT COUNT(*)
    FROM INFORMATION_SCHEMA.COLUMNS
    WHERE TABLE_SCHEMA = DATABASE()
      AND TABLE_NAME = 'syslog_messages'
      AND COLUMN_NAME = 'matched_rule_id'
);
SET @syslog_messages_matched_rule_id_sql = IF(
    @syslog_messages_has_matched_rule_id = 0,
    'ALTER TABLE syslog_messages ADD COLUMN matched_rule_id BIGINT UNSIGNED NULL AFTER parse_status',
    'SELECT 1'
);
PREPARE stmt FROM @syslog_messages_matched_rule_id_sql;
EXECUTE stmt;
DEALLOCATE PREPARE stmt;

SET @syslog_messages_has_matched_rule_index = (
    SELECT COUNT(*)
    FROM INFORMATION_SCHEMA.STATISTICS
    WHERE TABLE_SCHEMA = DATABASE()
      AND TABLE_NAME = 'syslog_messages'
      AND INDEX_NAME = 'idx_syslog_messages_matched_rule_id'
);
SET @syslog_messages_matched_rule_index_sql = IF(
    @syslog_messages_has_matched_rule_index = 0,
    'ALTER TABLE syslog_messages ADD KEY idx_syslog_messages_matched_rule_id (matched_rule_id)',
    'SELECT 1'
);
PREPARE stmt FROM @syslog_messages_matched_rule_index_sql;
EXECUTE stmt;
DEALLOCATE PREPARE stmt;

SET @employees_has_feishu_index = (
    SELECT COUNT(*)
    FROM INFORMATION_SCHEMA.STATISTICS
    WHERE TABLE_SCHEMA = DATABASE()
      AND TABLE_NAME = 'employees'
      AND INDEX_NAME = 'uk_employees_feishu_employee_id'
);
SET @employees_feishu_index_sql = IF(
    @employees_has_feishu_index = 0,
    'ALTER TABLE employees ADD UNIQUE KEY uk_employees_feishu_employee_id (feishu_employee_id)',
    'SELECT 1'
);
PREPARE stmt FROM @employees_feishu_index_sql;
EXECUTE stmt;
DEALLOCATE PREPARE stmt;

SET @reports_has_external_record_id = (
    SELECT COUNT(*)
    FROM INFORMATION_SCHEMA.COLUMNS
    WHERE TABLE_SCHEMA = DATABASE()
      AND TABLE_NAME = 'attendance_reports'
      AND COLUMN_NAME = 'external_record_id'
);
SET @reports_external_column_sql = IF(
    @reports_has_external_record_id = 0,
    'ALTER TABLE attendance_reports ADD COLUMN external_record_id VARCHAR(191) NOT NULL DEFAULT '''' AFTER target_url',
    'SELECT 1'
);
PREPARE stmt FROM @reports_external_column_sql;
EXECUTE stmt;
DEALLOCATE PREPARE stmt;

SET @reports_has_delete_record_id = (
    SELECT COUNT(*)
    FROM INFORMATION_SCHEMA.COLUMNS
    WHERE TABLE_SCHEMA = DATABASE()
      AND TABLE_NAME = 'attendance_reports'
      AND COLUMN_NAME = 'delete_record_id'
);
SET @reports_delete_column_sql = IF(
    @reports_has_delete_record_id = 0,
    'ALTER TABLE attendance_reports ADD COLUMN delete_record_id VARCHAR(191) NOT NULL DEFAULT '''' AFTER external_record_id',
    'SELECT 1'
);
PREPARE stmt FROM @reports_delete_column_sql;
EXECUTE stmt;
DEALLOCATE PREPARE stmt;

SET @reports_has_notification_status = (
    SELECT COUNT(*)
    FROM INFORMATION_SCHEMA.COLUMNS
    WHERE TABLE_SCHEMA = DATABASE()
      AND TABLE_NAME = 'attendance_reports'
      AND COLUMN_NAME = 'notification_status'
);
SET @reports_notification_status_sql = IF(
    @reports_has_notification_status = 0,
    'ALTER TABLE attendance_reports ADD COLUMN notification_status VARCHAR(32) NOT NULL DEFAULT ''skipped'' AFTER response_body',
    'SELECT 1'
);
PREPARE stmt FROM @reports_notification_status_sql;
EXECUTE stmt;
DEALLOCATE PREPARE stmt;

SET @reports_has_notification_message_id = (
    SELECT COUNT(*)
    FROM INFORMATION_SCHEMA.COLUMNS
    WHERE TABLE_SCHEMA = DATABASE()
      AND TABLE_NAME = 'attendance_reports'
      AND COLUMN_NAME = 'notification_message_id'
);
SET @reports_notification_message_id_sql = IF(
    @reports_has_notification_message_id = 0,
    'ALTER TABLE attendance_reports ADD COLUMN notification_message_id VARCHAR(191) NOT NULL DEFAULT '''' AFTER notification_status',
    'SELECT 1'
);
PREPARE stmt FROM @reports_notification_message_id_sql;
EXECUTE stmt;
DEALLOCATE PREPARE stmt;

SET @reports_has_notification_response_code = (
    SELECT COUNT(*)
    FROM INFORMATION_SCHEMA.COLUMNS
    WHERE TABLE_SCHEMA = DATABASE()
      AND TABLE_NAME = 'attendance_reports'
      AND COLUMN_NAME = 'notification_response_code'
);
SET @reports_notification_response_code_sql = IF(
    @reports_has_notification_response_code = 0,
    'ALTER TABLE attendance_reports ADD COLUMN notification_response_code INT NULL AFTER notification_message_id',
    'SELECT 1'
);
PREPARE stmt FROM @reports_notification_response_code_sql;
EXECUTE stmt;
DEALLOCATE PREPARE stmt;

SET @reports_has_notification_response_body = (
    SELECT COUNT(*)
    FROM INFORMATION_SCHEMA.COLUMNS
    WHERE TABLE_SCHEMA = DATABASE()
      AND TABLE_NAME = 'attendance_reports'
      AND COLUMN_NAME = 'notification_response_body'
);
SET @reports_notification_response_body_sql = IF(
    @reports_has_notification_response_body = 0,
    'ALTER TABLE attendance_reports ADD COLUMN notification_response_body TEXT NULL AFTER notification_response_code',
    'SELECT 1'
);
PREPARE stmt FROM @reports_notification_response_body_sql;
EXECUTE stmt;
DEALLOCATE PREPARE stmt;

SET @reports_has_notification_sent_at = (
    SELECT COUNT(*)
    FROM INFORMATION_SCHEMA.COLUMNS
    WHERE TABLE_SCHEMA = DATABASE()
      AND TABLE_NAME = 'attendance_reports'
      AND COLUMN_NAME = 'notification_sent_at'
);
SET @reports_notification_sent_at_sql = IF(
    @reports_has_notification_sent_at = 0,
    'ALTER TABLE attendance_reports ADD COLUMN notification_sent_at DATETIME NULL AFTER notification_response_body',
    'SELECT 1'
);
PREPARE stmt FROM @reports_notification_sent_at_sql;
EXECUTE stmt;
DEALLOCATE PREPARE stmt;

SET @reports_has_notification_retry_count = (
    SELECT COUNT(*)
    FROM INFORMATION_SCHEMA.COLUMNS
    WHERE TABLE_SCHEMA = DATABASE()
      AND TABLE_NAME = 'attendance_reports'
      AND COLUMN_NAME = 'notification_retry_count'
);
SET @reports_notification_retry_count_sql = IF(
    @reports_has_notification_retry_count = 0,
    'ALTER TABLE attendance_reports ADD COLUMN notification_retry_count INT UNSIGNED NOT NULL DEFAULT 0 AFTER notification_sent_at',
    'SELECT 1'
);
PREPARE stmt FROM @reports_notification_retry_count_sql;
EXECUTE stmt;
DEALLOCATE PREPARE stmt;

SET @reports_has_notification_dispatch_index = (
    SELECT COUNT(*)
    FROM INFORMATION_SCHEMA.STATISTICS
    WHERE TABLE_SCHEMA = DATABASE()
      AND TABLE_NAME = 'attendance_reports'
      AND INDEX_NAME = 'idx_attendance_reports_notification_dispatch'
);
SET @reports_notification_dispatch_index_sql = IF(
    @reports_has_notification_dispatch_index = 0,
    'ALTER TABLE attendance_reports ADD KEY idx_attendance_reports_notification_dispatch (report_status, notification_status, notification_retry_count)',
    'SELECT 1'
);
PREPARE stmt FROM @reports_notification_dispatch_index_sql;
EXECUTE stmt;
DEALLOCATE PREPARE stmt;

CREATE TABLE IF NOT EXISTS system_settings (
    id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
    setting_key VARCHAR(128) NOT NULL,
    setting_value TEXT NOT NULL,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    PRIMARY KEY (id),
    UNIQUE KEY uk_system_settings_setting_key (setting_key)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

INSERT IGNORE INTO system_settings (setting_key, setting_value) VALUES
    ('day_end_time', '23:59'),
    ('syslog_retention_days', '30'),
    ('feishu_app_id', ''),
    ('feishu_app_secret', ''),
    ('feishu_location_name', ''),
    ('report_timeout_seconds', '10'),
    ('report_retry_limit', '3');

DELETE FROM system_settings
WHERE setting_key IN ('report_target_url', 'feishu_creator_employee_id');
