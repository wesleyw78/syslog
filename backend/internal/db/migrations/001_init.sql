CREATE TABLE employees (
    id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
    employee_no VARCHAR(64) NOT NULL,
    system_no VARCHAR(64) NOT NULL,
    name VARCHAR(128) NOT NULL,
    status VARCHAR(32) NOT NULL DEFAULT 'active',
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    PRIMARY KEY (id),
    UNIQUE KEY uk_employees_employee_no (employee_no),
    UNIQUE KEY uk_employees_system_no (system_no),
    KEY idx_employees_status (status)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE employee_devices (
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

CREATE TABLE syslog_messages (
    id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
    received_at DATETIME NOT NULL,
    log_time DATETIME NULL,
    raw_message TEXT NOT NULL,
    source_ip VARCHAR(45) NOT NULL DEFAULT '',
    protocol VARCHAR(16) NOT NULL DEFAULT 'udp',
    parse_status VARCHAR(32) NOT NULL DEFAULT 'pending',
    retention_expire_at DATETIME NOT NULL,
    PRIMARY KEY (id),
    KEY idx_syslog_messages_received_at (received_at),
    KEY idx_syslog_messages_parse_status (parse_status),
    KEY idx_syslog_messages_retention_expire_at (retention_expire_at)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE client_events (
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

CREATE TABLE attendance_records (
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

CREATE TABLE attendance_reports (
    id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
    attendance_record_id BIGINT UNSIGNED NOT NULL,
    report_type VARCHAR(32) NOT NULL,
    idempotency_key VARCHAR(191) NOT NULL,
    payload_json JSON NULL,
    target_url VARCHAR(1024) NOT NULL,
    report_status VARCHAR(32) NOT NULL DEFAULT 'pending',
    response_code INT NULL,
    response_body TEXT NULL,
    reported_at DATETIME NULL,
    retry_count INT UNSIGNED NOT NULL DEFAULT 0,
    PRIMARY KEY (id),
    UNIQUE KEY uk_attendance_reports_idempotency_key (idempotency_key),
    KEY idx_attendance_reports_record_type (attendance_record_id, report_type),
    KEY idx_attendance_reports_status (report_status),
    CONSTRAINT fk_attendance_reports_record
        FOREIGN KEY (attendance_record_id) REFERENCES attendance_records (id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE system_settings (
    id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
    setting_key VARCHAR(128) NOT NULL,
    setting_value TEXT NOT NULL,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    PRIMARY KEY (id),
    UNIQUE KEY uk_system_settings_setting_key (setting_key)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

INSERT INTO system_settings (setting_key, setting_value) VALUES
    ('day_end_time', '23:59'),
    ('syslog_retention_days', '30'),
    ('report_target_url', ''),
    ('report_timeout_seconds', '10'),
    ('report_retry_limit', '3');
