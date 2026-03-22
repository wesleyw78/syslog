#!/usr/bin/env bash

set -euo pipefail

MYSQL_HOST="${MYSQL_HOST:-127.0.0.1}"
MYSQL_PORT="${MYSQL_PORT:-3306}"
MYSQL_USER="${MYSQL_USER:-syslog}"
MYSQL_PASSWORD="${MYSQL_PASSWORD:-syslog}"
MYSQL_DATABASE="${MYSQL_DATABASE:-syslog}"

INCLUDE_EMPLOYEES=0
INCLUDE_SETTINGS=0
SKIP_CONFIRM=0

usage() {
  cat <<'EOF'
usage: ./scripts/reset-dev-data.sh [--with-employees] [--with-settings] [--yes]

默认清理运行期数据：
  - syslog_messages
  - client_events
  - attendance_records
  - attendance_reports

可选参数：
  --with-employees  额外清理 employees / employee_devices
  --with-settings   额外清理 system_settings
  --yes             跳过交互确认

环境变量：
  MYSQL_HOST MYSQL_PORT MYSQL_USER MYSQL_PASSWORD MYSQL_DATABASE
EOF
}

while (($# > 0)); do
  case "$1" in
    --with-employees)
      INCLUDE_EMPLOYEES=1
      ;;
    --with-settings)
      INCLUDE_SETTINGS=1
      ;;
    --yes)
      SKIP_CONFIRM=1
      ;;
    -h|--help)
      usage
      exit 0
      ;;
    *)
      echo "unknown option: $1" >&2
      usage >&2
      exit 1
      ;;
  esac
  shift
done

if ! command -v mysql >/dev/null 2>&1; then
  echo "mysql client is required but was not found in PATH" >&2
  exit 1
fi

tables=(
  attendance_reports
  attendance_records
  client_events
  syslog_messages
)

if [[ "$INCLUDE_EMPLOYEES" == "1" ]]; then
  tables=(
    employee_devices
    employees
    "${tables[@]}"
  )
fi

if [[ "$INCLUDE_SETTINGS" == "1" ]]; then
  tables+=("system_settings")
fi

echo "目标数据库: ${MYSQL_USER}@${MYSQL_HOST}:${MYSQL_PORT}/${MYSQL_DATABASE}"
echo "即将清理以下表:"
for table in "${tables[@]}"; do
  echo "  - ${table}"
done

if [[ "$SKIP_CONFIRM" != "1" ]]; then
  printf '继续执行数据清理? [y/N] '
  read -r answer
  if [[ ! "$answer" =~ ^[Yy]$ ]]; then
    echo "已取消"
    exit 0
  fi
fi

sql="SET FOREIGN_KEY_CHECKS=0;"
for table in "${tables[@]}"; do
  sql+="TRUNCATE TABLE ${table};"
done
sql+="SET FOREIGN_KEY_CHECKS=1;"

mysql \
  --host="${MYSQL_HOST}" \
  --port="${MYSQL_PORT}" \
  --user="${MYSQL_USER}" \
  --password="${MYSQL_PASSWORD}" \
  "${MYSQL_DATABASE}" \
  --execute="${sql}"

echo "数据清理完成"
