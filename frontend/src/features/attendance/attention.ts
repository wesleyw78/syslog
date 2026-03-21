import type { AttendanceRecord } from "../../lib/api";

export function getAttendanceAttentionReason(
  record: AttendanceRecord,
): string | null {
  if (record.exceptionStatus !== "none") {
    return record.exceptionStatus;
  }
  if (record.clockInStatus === "pending") {
    return "clock_in_pending";
  }
  if (record.clockOutStatus === "missing") {
    return "clock_out_missing";
  }
  if (record.clockOutStatus === "pending") {
    return "clock_out_pending";
  }
  return null;
}

export function requiresAttendanceAttention(record: AttendanceRecord): boolean {
  return getAttendanceAttentionReason(record) !== null;
}
