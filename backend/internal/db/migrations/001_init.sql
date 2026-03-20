CREATE TABLE employees (
    id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
    employee_no VARCHAR(64) NOT NULL,
    name VARCHAR(128) NOT NULL,
    timezone VARCHAR(64) NOT NULL DEFAULT 'Asia/Shanghai',
    is_active TINYINT(1) NOT NULL DEFAULT 1,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    PRIMARY KEY (id),
    UNIQUE KEY uk_employees_employee_no (employee_no)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE employee_devices (
    id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
    employee_id BIGINT UNSIGNED NOT NULL,
    device_identifier VARCHAR(128) NOT NULL,
    device_name VARCHAR(128) NOT NULL DEFAULT '',
    is_primary TINYINT(1) NOT NULL DEFAULT 0,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    PRIMARY KEY (id),
    UNIQUE KEY uk_employee_devices_identifier (device_identifier),
    KEY idx_employee_devices_employee_id (employee_id),
    CONSTRAINT fk_employee_devices_employee
        FOREIGN KEY (employee_id) REFERENCES employees (id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE syslog_messages (
    id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
    device_identifier VARCHAR(128) NOT NULL,
    raw_message TEXT NOT NULL,
    source_ip VARCHAR(45) NOT NULL DEFAULT '',
    received_at DATETIME NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (id),
    KEY idx_syslog_messages_device_received_at (device_identifier, received_at)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE client_events (
    id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
    syslog_message_id BIGINT UNSIGNED NULL,
    employee_id BIGINT UNSIGNED NULL,
    employee_device_id BIGINT UNSIGNED NULL,
    event_type VARCHAR(64) NOT NULL,
    event_time DATETIME NOT NULL,
    payload JSON NULL,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (id),
    KEY idx_client_events_event_time (event_time),
    KEY idx_client_events_employee_id (employee_id),
    KEY idx_client_events_employee_device_id (employee_device_id),
    CONSTRAINT fk_client_events_syslog_message
        FOREIGN KEY (syslog_message_id) REFERENCES syslog_messages (id),
    CONSTRAINT fk_client_events_employee
        FOREIGN KEY (employee_id) REFERENCES employees (id),
    CONSTRAINT fk_client_events_employee_device
        FOREIGN KEY (employee_device_id) REFERENCES employee_devices (id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE attendance_records (
    id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
    employee_id BIGINT UNSIGNED NOT NULL,
    work_date DATE NOT NULL,
    first_check_in_at DATETIME NULL,
    last_check_out_at DATETIME NULL,
    status VARCHAR(32) NOT NULL DEFAULT 'pending',
    source_event_count INT UNSIGNED NOT NULL DEFAULT 0,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    PRIMARY KEY (id),
    UNIQUE KEY uk_attendance_records_employee_date (employee_id, work_date),
    KEY idx_attendance_records_status (status),
    CONSTRAINT fk_attendance_records_employee
        FOREIGN KEY (employee_id) REFERENCES employees (id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE attendance_reports (
    id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
    report_date DATE NOT NULL,
    employee_id BIGINT UNSIGNED NOT NULL,
    status VARCHAR(32) NOT NULL DEFAULT 'pending',
    summary JSON NULL,
    generated_at DATETIME NULL,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (id),
    UNIQUE KEY uk_attendance_reports_date_employee (report_date, employee_id),
    KEY idx_attendance_reports_status (status),
    CONSTRAINT fk_attendance_reports_employee
        FOREIGN KEY (employee_id) REFERENCES employees (id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE system_settings (
    setting_key VARCHAR(128) NOT NULL,
    setting_value TEXT NOT NULL,
    description VARCHAR(255) NOT NULL DEFAULT '',
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    PRIMARY KEY (setting_key)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
