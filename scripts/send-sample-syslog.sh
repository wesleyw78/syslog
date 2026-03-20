#!/usr/bin/env bash

set -euo pipefail

MODE="${1:-both}"
HOST="${SYSLOG_HOST:-127.0.0.1}"
PORT="${SYSLOG_PORT:-5514}"

CONNECT_SAMPLE='Mar 21 08:01:00 stamgr: Mef85d2S4D0 client_footprints connect Station[94:89:78:55:9a:f3] AP[28:b3:71:25:ae:a0] ssid[FactoryOps] osvendor[Unknown] hostname[scanner-01]'
DISCONNECT_SAMPLE='Mar 21 18:05:00 stamgr: Mef85d2S4D0 client_footprints disconnect Station[94:89:78:55:9a:f3] AP[28:b3:71:25:ae:a0] ssid[FactoryOps] osvendor[Unknown] hostname[scanner-01]'

send_udp() {
  local payload="$1"
  printf '%s\n' "$payload" | nc -u -w 1 "$HOST" "$PORT"
  printf 'sent to %s:%s/udp -> %s\n' "$HOST" "$PORT" "$payload"
}

case "$MODE" in
  connect)
    send_udp "$CONNECT_SAMPLE"
    ;;
  disconnect)
    send_udp "$DISCONNECT_SAMPLE"
    ;;
  both)
    send_udp "$CONNECT_SAMPLE"
    send_udp "$DISCONNECT_SAMPLE"
    ;;
  *)
    echo "usage: $0 [connect|disconnect|both]" >&2
    exit 1
    ;;
esac
