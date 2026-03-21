import type { AttendanceRecord } from "../../lib/api";

export function requiresAttendanceAttention(record: AttendanceRecord): boolean {
  return (
    record.exceptionStatus !== "none" ||
    record.clockInStatus === "pending" ||
    record.clockOutStatus === "pending" ||
    record.clockOutStatus === "missing"
  );
}
