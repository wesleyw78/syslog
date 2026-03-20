import type { AttendanceRecord } from "../../../lib/api";

type AttendanceTableProps = {
  pendingId: string | null;
  records: AttendanceRecord[];
  onManualCorrection: (recordId: string) => Promise<void>;
};

const rowStyle = {
  display: "grid",
  gridTemplateColumns: "minmax(0, 1.3fr) minmax(0, 1fr) minmax(0, 1fr) 120px 130px",
  gap: "0.65rem",
  alignItems: "center",
};

const cellStyle = {
  padding: "0.75rem",
  border: "1px solid rgba(255, 184, 77, 0.1)",
  background: "rgba(6, 8, 8, 0.65)",
};

const buttonStyle = {
  padding: "0.75rem 0.8rem",
  border: "1px solid rgba(255, 184, 77, 0.24)",
  background: "rgba(19, 15, 7, 0.92)",
  color: "inherit",
  cursor: "pointer",
};

function getStatusLabel(status: AttendanceRecord["status"]): string {
  switch (status) {
    case "exception":
      return "异常";
    case "corrected":
      return "已修正";
    default:
      return "正常";
  }
}

export function AttendanceTable({
  pendingId,
  records,
  onManualCorrection,
}: AttendanceTableProps) {
  return (
    <div style={{ display: "grid", gap: "0.65rem" }}>
      <div style={{ ...rowStyle, color: "#8a928d", fontSize: "0.78rem", textTransform: "uppercase", letterSpacing: "0.08em" }}>
        <span>人员 / 点位</span>
        <span>班次</span>
        <span>状态</span>
        <span>时间</span>
        <span>动作</span>
      </div>

      {records.map((record) => {
        const isPending = pendingId === record.id;
        const isException = record.status === "exception";

        return (
          <div
            key={record.id}
            role="group"
            aria-label={`${record.employeeName} attendance row`}
            style={rowStyle}
          >
            <div style={cellStyle}>
              <strong>{record.employeeName}</strong>
              <div style={{ color: "#8a928d", marginTop: "0.2rem" }}>
                {record.badge} · {record.checkpoint}
              </div>
              <div style={{ color: "#bcc3bd", marginTop: "0.35rem" }}>{record.note}</div>
            </div>

            <div style={cellStyle}>{record.shift}</div>

            <div style={cellStyle}>{getStatusLabel(record.status)}</div>

            <div style={cellStyle}>{record.timestamp}</div>

            <div style={cellStyle}>
              {isException ? (
                <button
                  type="button"
                  onClick={() => void onManualCorrection(record.id)}
                  disabled={isPending}
                  style={buttonStyle}
                >
                  {isPending ? "提交中..." : "人工修正"}
                </button>
              ) : (
                <span>{record.status === "corrected" ? "已归档" : "无需处理"}</span>
              )}
            </div>
          </div>
        );
      })}
    </div>
  );
}
