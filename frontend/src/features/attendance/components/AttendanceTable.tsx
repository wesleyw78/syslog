import type { AttendanceRecord } from "../../../lib/api";
import {
  getAttendanceAttentionReason,
  requiresAttendanceAttention,
} from "../attention";

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

function getStatusLabel(record: AttendanceRecord): string {
  return `${record.clockInStatus} / ${record.clockOutStatus}`;
}

export function AttendanceTable({
  drafts,
  pendingId,
  records,
  onDraftChange,
  onManualCorrection,
}: AttendanceTableProps) {
  return (
    <div className="attendance-table">
      <div className="attendance-table__head">
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
        const isActionable = requiresAttendanceAttention(record);
        const attentionReason = getAttendanceAttentionReason(record);
        const draft = drafts[record.id] ?? {
          firstConnectAt: record.firstConnectAt ?? "",
          lastDisconnectAt: record.lastDisconnectAt ?? "",
        };

        return (
          <div
            key={record.id}
            className={`attendance-row${isActionable ? " attendance-row--attention" : ""}`}
            role="group"
            aria-label={`${record.employeeName} 考勤记录`}
          >
            <div className="attendance-cell attendance-cell--identity">
              <strong>{record.attendanceDate}</strong>
              <div className="attendance-cell__subtext">{record.employeeName}</div>
            </div>

            <div className="attendance-cell">
              {isActionable ? (
                <label className="form-field">
                  <span className="form-field__label">{`${record.employeeName} 首次接入`}</span>
                  <input
                    className="form-field__control form-field__control--compact"
                    aria-label="首次接入"
                    type="text"
                    value={draft.firstConnectAt}
                    onChange={(event) =>
                      onDraftChange(record.id, "firstConnectAt", event.target.value)
                    }
                  />
                </label>
              ) : (
                <span>{record.firstConnectAt ?? "-"}</span>
              )}
            </div>

            <div className="attendance-cell">
              {isActionable ? (
                <label className="form-field">
                  <span className="form-field__label">{`${record.employeeName} 最后断开`}</span>
                  <input
                    className="form-field__control form-field__control--compact"
                    aria-label="最后断开"
                    type="text"
                    value={draft.lastDisconnectAt}
                    onChange={(event) =>
                      onDraftChange(record.id, "lastDisconnectAt", event.target.value)
                    }
                  />
                </label>
              ) : (
                <span>{record.lastDisconnectAt ?? "-"}</span>
              )}
            </div>

            <div className="attendance-cell">{getStatusLabel(record)}</div>
            <div className="attendance-cell">{attentionReason ?? "无异常"}</div>
            <div className="attendance-cell">{record.sourceMode}</div>

            <div className="attendance-cell">
              {isActionable ? (
                <button
                  type="button"
                  onClick={() => void onManualCorrection(record.id)}
                  disabled={isPending}
                  className="button button--primary button--small"
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
