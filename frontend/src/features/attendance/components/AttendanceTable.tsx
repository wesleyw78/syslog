import type { AttendanceRecord } from "../../../lib/api";

type AttendanceDraft = {
  firstConnectAt: string;
  lastDisconnectAt: string;
};

type AttendanceRow = AttendanceRecord & {
  employeeName: string;
};

type AttendanceTableProps = {
  drafts: Record<string, AttendanceDraft>;
  pendingId: string | null;
  records: AttendanceRow[];
  onDraftChange: (
    recordId: string,
    field: keyof AttendanceDraft,
    value: string,
  ) => void;
  onManualCorrection: (recordId: string) => Promise<void>;
};

const rowStyle = {
  display: "grid",
  gridTemplateColumns:
    "minmax(0, 1.1fr) minmax(0, 1.1fr) minmax(0, 1fr) minmax(0, 1fr) minmax(0, 1fr) 140px 140px",
  gap: "0.65rem",
  alignItems: "stretch",
};

const cellStyle = {
  padding: "0.75rem",
  border: "1px solid rgba(255, 184, 77, 0.1)",
  background: "rgba(6, 8, 8, 0.65)",
};

const inputStyle = {
  width: "100%",
  padding: "0.55rem 0.65rem",
  border: "1px solid rgba(255, 184, 77, 0.14)",
  background: "rgba(7, 9, 9, 0.8)",
  color: "inherit",
};

const buttonStyle = {
  padding: "0.75rem 0.8rem",
  border: "1px solid rgba(255, 184, 77, 0.24)",
  background: "rgba(19, 15, 7, 0.92)",
  color: "inherit",
  cursor: "pointer",
};

function getStatusLabel(record: AttendanceRecord): string {
  return `${record.clockInStatus} / ${record.clockOutStatus}`;
}

function requiresAttention(record: AttendanceRecord): boolean {
  return (
    record.exceptionStatus !== "none" ||
    record.clockOutStatus === "pending" ||
    record.clockOutStatus === "missing" ||
    record.sourceMode === "manual"
  );
}

export function AttendanceTable({
  drafts,
  pendingId,
  records,
  onDraftChange,
  onManualCorrection,
}: AttendanceTableProps) {
  return (
    <div style={{ display: "grid", gap: "0.65rem" }}>
      <div
        style={{
          ...rowStyle,
          color: "#8a928d",
          fontSize: "0.78rem",
          textTransform: "uppercase",
          letterSpacing: "0.08em",
        }}
      >
        <span>日期 / 员工</span>
        <span>首次接入</span>
        <span>最后断开</span>
        <span>签到 / 签退</span>
        <span>异常</span>
        <span>人工修正</span>
        <span>动作</span>
      </div>

      {records.map((record) => {
        const isPending = pendingId === record.id;
        const isActionable = requiresAttention(record);
        const draft = drafts[record.id] ?? {
          firstConnectAt: record.firstConnectAt ?? "",
          lastDisconnectAt: record.lastDisconnectAt ?? "",
        };

        return (
          <div
            key={record.id}
            role="group"
            aria-label={`${record.employeeName} 考勤记录`}
            style={rowStyle}
          >
            <div style={cellStyle}>
              <strong>{record.attendanceDate}</strong>
              <div style={{ color: "#8a928d", marginTop: "0.2rem" }}>
                {record.employeeName}
              </div>
            </div>

            <div style={cellStyle}>
              {isActionable ? (
                <label style={{ display: "grid", gap: "0.25rem" }}>
                  <span>{`${record.employeeName} 首次接入`}</span>
                  <input
                    aria-label="首次接入"
                    type="text"
                    value={draft.firstConnectAt}
                    onChange={(event) =>
                      onDraftChange(record.id, "firstConnectAt", event.target.value)
                    }
                    style={inputStyle}
                  />
                </label>
              ) : (
                <span>{record.firstConnectAt ?? "-"}</span>
              )}
            </div>

            <div style={cellStyle}>
              {isActionable ? (
                <label style={{ display: "grid", gap: "0.25rem" }}>
                  <span>{`${record.employeeName} 最后断开`}</span>
                  <input
                    aria-label="最后断开"
                    type="text"
                    value={draft.lastDisconnectAt}
                    onChange={(event) =>
                      onDraftChange(record.id, "lastDisconnectAt", event.target.value)
                    }
                    style={inputStyle}
                  />
                </label>
              ) : (
                <span>{record.lastDisconnectAt ?? "-"}</span>
              )}
            </div>

            <div style={cellStyle}>{getStatusLabel(record)}</div>

            <div style={cellStyle}>
              {record.exceptionStatus === "none" ? "无异常" : record.exceptionStatus}
            </div>

            <div style={cellStyle}>{record.sourceMode}</div>

            <div style={cellStyle}>
              {isActionable ? (
                <button
                  type="button"
                  onClick={() => void onManualCorrection(record.id)}
                  disabled={isPending}
                  style={buttonStyle}
                >
                  {isPending ? "提交中..." : "提交修正"}
                </button>
              ) : (
                <span>无需处理</span>
              )}
            </div>
          </div>
        );
      })}
    </div>
  );
}
