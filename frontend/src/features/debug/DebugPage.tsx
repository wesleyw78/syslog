import { useEffect, useMemo, useState } from "react";

import {
  dispatchAttendanceReport,
  getAttendanceRecords,
  getEmployees,
  injectDebugSyslog,
  type DebugAttendanceDispatchResult,
  type DebugSyslogInjectResult,
  type AttendanceRecord,
  type Employee,
} from "../../lib/api";

type DebugAttendanceRow = AttendanceRecord & {
  employeeName: string;
};

function createDefaultDateTimeLocalValue(): string {
  const current = new Date();
  const year = current.getFullYear();
  const month = String(current.getMonth() + 1).padStart(2, "0");
  const day = String(current.getDate()).padStart(2, "0");
  const hours = String(current.getHours()).padStart(2, "0");
  const minutes = String(current.getMinutes()).padStart(2, "0");

  return `${year}-${month}-${day}T${hours}:${minutes}`;
}

function toApiDateTime(value: string): string {
  const normalized = value.trim();
  if (normalized === "") {
    return "";
  }
  if (/([zZ]|[+-]\d{2}:\d{2})$/.test(normalized)) {
    return normalized;
  }

  const parsed = new Date(normalized);
  if (Number.isNaN(parsed.getTime())) {
    return normalized;
  }

  const offsetMinutes = -parsed.getTimezoneOffset();
  const sign = offsetMinutes >= 0 ? "+" : "-";
  const absoluteOffsetMinutes = Math.abs(offsetMinutes);
  const offsetHours = String(Math.floor(absoluteOffsetMinutes / 60)).padStart(2, "0");
  const offsetRemainingMinutes = String(absoluteOffsetMinutes % 60).padStart(2, "0");

  return `${parsed.getFullYear()}-${String(parsed.getMonth() + 1).padStart(2, "0")}-${String(parsed.getDate()).padStart(2, "0")}T${String(parsed.getHours()).padStart(2, "0")}:${String(parsed.getMinutes()).padStart(2, "0")}:${String(parsed.getSeconds()).padStart(2, "0")}${sign}${offsetHours}:${offsetRemainingMinutes}`;
}

function formatDisplayDateTime(value?: string | null): string {
  if (!value) {
    return "-";
  }
  const parsed = new Date(value);
  if (Number.isNaN(parsed.getTime())) {
    return value;
  }

  return `${parsed.getFullYear()}-${String(parsed.getMonth() + 1).padStart(2, "0")}-${String(parsed.getDate()).padStart(2, "0")} ${String(parsed.getHours()).padStart(2, "0")}:${String(parsed.getMinutes()).padStart(2, "0")}:${String(parsed.getSeconds()).padStart(2, "0")}`;
}

function toAttendanceRows(attendance: AttendanceRecord[], employees: Employee[]): DebugAttendanceRow[] {
  const employeeMap = new Map(
    employees.map((employee) => [employee.id, employee.name]),
  );

  return attendance.map((record) => ({
    ...record,
    employeeName: employeeMap.get(record.employeeId) ?? `员工 ${record.employeeId}`,
  }));
}

export function DebugPage() {
  const [rawMessage, setRawMessage] = useState("");
  const [receivedAt, setReceivedAt] = useState(createDefaultDateTimeLocalValue);
  const [syslogStatus, setSyslogStatus] = useState("");
  const [syslogPending, setSyslogPending] = useState(false);
  const [lastSyslogResult, setLastSyslogResult] = useState<DebugSyslogInjectResult | null>(null);

  const [rows, setRows] = useState<DebugAttendanceRow[]>([]);
  const [loadingMessage, setLoadingMessage] = useState("加载调试数据...");
  const [dispatchStatus, setDispatchStatus] = useState("");
  const [dispatchPendingKey, setDispatchPendingKey] = useState("");
  const [lastDispatchResult, setLastDispatchResult] =
    useState<DebugAttendanceDispatchResult | null>(null);

  useEffect(() => {
    let isActive = true;

    void (async () => {
      try {
        const [employees, attendance] = await Promise.all([
          getEmployees(),
          getAttendanceRecords(),
        ]);

        if (!isActive) {
          return;
        }

        setRows(toAttendanceRows(attendance, employees));
        setLoadingMessage(`已载入 ${attendance.length} 条考勤记录`);
      } catch {
        if (isActive) {
          setLoadingMessage("调试数据加载失败，请稍后重试");
        }
      }
    })();

    return () => {
      isActive = false;
    };
  }, []);

  const sortedRows = useMemo(
    () =>
      [...rows].sort((left, right) => {
        if (left.attendanceDate === right.attendanceDate) {
          return left.employeeName.localeCompare(right.employeeName, "zh-CN");
        }
        return right.attendanceDate.localeCompare(left.attendanceDate);
      }),
    [rows],
  );

  async function handleInjectSyslog() {
    const trimmedRawMessage = rawMessage.trim();
    if (trimmedRawMessage === "") {
      setSyslogStatus("请输入原始 syslog");
      return;
    }

    if (!window.confirm("确认注入这条 syslog 调试报文？")) {
      return;
    }

    setSyslogPending(true);

    try {
      const result = await injectDebugSyslog({
        rawMessage: trimmedRawMessage,
        receivedAt: toApiDateTime(receivedAt),
      });
      setLastSyslogResult(result);
      const parseErrorSuffix = result.parseError ? `，错误：${result.parseError}` : "";
      setSyslogStatus(`已注入，当前解析状态：${result.parseStatus}${parseErrorSuffix}`);
    } catch {
      setSyslogStatus("syslog 注入失败，请稍后重试");
    } finally {
      setSyslogPending(false);
    }
  }

  async function handleDispatchAttendance(
    recordId: string,
    reportType: "clock_in" | "clock_out",
    employeeName: string,
  ) {
    if (!window.confirm(`确认发送 ${employeeName} 的 ${reportType} 到飞书？`)) {
      return;
    }

    setDispatchPendingKey(`${recordId}:${reportType}`);

    try {
      const result = await dispatchAttendanceReport(recordId, { reportType });
      setLastDispatchResult(result);
      const notificationSuffix =
        result.report.notificationStatus === "success"
          ? `，通知：success（消息 ID: ${result.report.notificationMessageId || "unknown"}）`
          : result.report.notificationStatus === "failed"
            ? `，通知：failed${result.report.notificationResponseBody ? `（${result.report.notificationResponseBody}）` : ""}`
            : `，通知：${result.report.notificationStatus || "unknown"}`;
      setDispatchStatus(
        `${result.report.reportType} 已发送，结果：${result.report.reportStatus}${notificationSuffix}`,
      );
    } catch {
      setDispatchStatus(`${reportType} 发送失败，请稍后重试`);
    } finally {
      setDispatchPendingKey("");
    }
  }

  return (
    <section className="page">
      <header className="page-header">
        <span className="page-header__eyebrow">受控调试</span>
        <div>
          <h2>调试工具</h2>
          <p>手工注入原始 syslog，或按上班/下班维度重发单条考勤到飞书，全部动作都复用正式链路。</p>
        </div>
      </header>

      <div className="page-grid page-grid--debug">
        <article className="panel">
          <div className="panel__header">
            <h3>Syslog 注入</h3>
            <span>手工接入</span>
          </div>

          <div className="debug-form">
            <label className="form-field">
              <span className="form-field__label">原始 syslog</span>
              <textarea
                aria-label="原始 syslog"
                className="debug-form__textarea"
                value={rawMessage}
                onChange={(event) => setRawMessage(event.target.value)}
              />
            </label>

            <label className="form-field">
              <span className="form-field__label">接收日期时间</span>
              <input
                aria-label="接收日期时间"
                type="datetime-local"
                value={receivedAt}
                onChange={(event) => setReceivedAt(event.target.value)}
              />
            </label>

            <div className="form-actions">
              <button
                type="button"
                className="button button--primary"
                disabled={syslogPending || rawMessage.trim() === ""}
                onClick={() => void handleInjectSyslog()}
              >
                {syslogPending ? "注入中..." : "注入 syslog"}
              </button>
            </div>

            {syslogStatus ? <p className="panel__copy">{syslogStatus}</p> : null}
          </div>

          <div className="debug-result-card">
            <div className="panel__header">
              <h3>最近一次注入结果</h3>
              <span>{lastSyslogResult ? "已记录" : "暂无"}</span>
            </div>
            {lastSyslogResult ? (
              <div className="debug-result-grid">
                <div className="debug-result-grid__item">
                  <span>接收状态</span>
                  <strong>{lastSyslogResult.accepted ? "accepted" : "rejected"}</strong>
                </div>
                <div className="debug-result-grid__item">
                  <span>解析状态</span>
                  <strong>{lastSyslogResult.parseStatus}</strong>
                </div>
                <div className="debug-result-grid__item debug-result-grid__item--wide">
                  <span>接收时间</span>
                  <strong>{formatDisplayDateTime(lastSyslogResult.receivedAt)}</strong>
                </div>
                <div className="debug-result-grid__item debug-result-grid__item--wide">
                  <span>解析错误</span>
                  <strong>{lastSyslogResult.parseError || "无"}</strong>
                </div>
              </div>
            ) : (
              <p className="panel__copy">尚未执行 syslog 注入。</p>
            )}
          </div>
        </article>

        <article className="panel panel--tall">
          <div className="panel__header">
            <h3>飞书单条上报</h3>
            <span>立即调度</span>
          </div>
          <p className="panel__copy">{loadingMessage}</p>
          {dispatchStatus ? <p className="panel__copy">{dispatchStatus}</p> : null}

          <div className="debug-attendance-table">
            <div className="debug-attendance-table__head">
              <span>日期 / 员工</span>
              <span>上班时间</span>
              <span>下班时间</span>
              <span>状态</span>
              <span>动作</span>
            </div>

            {sortedRows.map((row) => {
              const clockInPendingKey = `${row.id}:clock_in`;
              const clockOutPendingKey = `${row.id}:clock_out`;

              return (
                <div
                  key={row.id}
                  className="debug-attendance-row"
                  role="group"
                  aria-label={`${row.employeeName} 调试记录`}
                >
                  <div className="debug-attendance-row__identity">
                    <strong>{row.attendanceDate}</strong>
                    <span>{row.employeeName}</span>
                  </div>
                  <span>{formatDisplayDateTime(row.firstConnectAt)}</span>
                  <span>{formatDisplayDateTime(row.lastDisconnectAt)}</span>
                  <span>{`${row.clockInStatus} / ${row.clockOutStatus}`}</span>
                  <div className="debug-attendance-row__actions">
                    <button
                      type="button"
                      className="button button--ghost button--small"
                      disabled={
                        !row.firstConnectAt ||
                        (dispatchPendingKey !== "" && dispatchPendingKey !== clockInPendingKey)
                      }
                      onClick={() => void handleDispatchAttendance(row.id, "clock_in", row.employeeName)}
                    >
                      {dispatchPendingKey === clockInPendingKey ? "发送中..." : "发送上班到飞书"}
                    </button>
                    <button
                      type="button"
                      className="button button--ghost button--small"
                      disabled={
                        !row.lastDisconnectAt ||
                        (dispatchPendingKey !== "" && dispatchPendingKey !== clockOutPendingKey)
                      }
                      onClick={() => void handleDispatchAttendance(row.id, "clock_out", row.employeeName)}
                    >
                      {dispatchPendingKey === clockOutPendingKey ? "发送中..." : "发送下班到飞书"}
                    </button>
                  </div>
                </div>
              );
            })}
          </div>

          <div className="debug-result-card">
            <div className="panel__header">
              <h3>最近一次发送结果</h3>
              <span>{lastDispatchResult ? "已记录" : "暂无"}</span>
            </div>
            {lastDispatchResult ? (
              <div className="debug-result-grid">
                <div className="debug-result-grid__item">
                  <span>发送类型</span>
                  <strong>{lastDispatchResult.report.reportType}</strong>
                </div>
                <div className="debug-result-grid__item">
                  <span>打卡结果</span>
                  <strong>{lastDispatchResult.report.reportStatus}</strong>
                </div>
                <div className="debug-result-grid__item">
                  <span>通知状态</span>
                  <strong>{lastDispatchResult.report.notificationStatus || "unknown"}</strong>
                </div>
                <div className="debug-result-grid__item">
                  <span>消息 ID</span>
                  <strong>{lastDispatchResult.report.notificationMessageId || "-"}</strong>
                </div>
                <div className="debug-result-grid__item debug-result-grid__item--wide">
                  <span>飞书流水</span>
                  <strong>{lastDispatchResult.report.externalRecordId || "-"}</strong>
                </div>
                <div className="debug-result-grid__item debug-result-grid__item--wide">
                  <span>通知响应</span>
                  <strong>{lastDispatchResult.report.notificationResponseBody || "无"}</strong>
                </div>
              </div>
            ) : (
              <p className="panel__copy">尚未执行飞书发送。</p>
            )}
          </div>
        </article>
      </div>
    </section>
  );
}
