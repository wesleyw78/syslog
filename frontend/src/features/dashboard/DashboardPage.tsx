import { useEffect, useMemo, useState } from "react";

import {
  getAttendanceRecords,
  getEmployees,
  getLogs,
  type AttendanceRecord,
  type Employee,
  type LogItem,
} from "../../lib/api";

type DashboardState = {
  employees: Employee[];
  attendance: AttendanceRecord[];
  logs: LogItem[];
};

type DashboardMetric = {
  label: string;
  value: string;
  status: string;
};

function buildAttentionItems(state: DashboardState): string[] {
  const employeeMap = new Map(state.employees.map((employee) => [employee.id, employee]));
  const attendanceItems = state.attendance
    .filter(
      (record) =>
        record.exceptionStatus !== "none" ||
        record.clockOutStatus === "pending" ||
        record.clockOutStatus === "missing" ||
        record.sourceMode === "manual",
    )
    .map((record) => {
      const employeeName = employeeMap.get(record.employeeId)?.name ?? record.employeeId;
      return `${employeeName} ${record.attendanceDate} ${record.exceptionStatus}`;
    });

  const logItems = state.logs.map((item) => {
    const message = item.message;
    const eventType = item.event?.eventType ?? "unknown-event";
    const stationMac = item.event?.stationMac ?? "unknown-mac";
    const hostname = item.event?.hostname ?? "unknown-host";
    const rawMessage = message.rawMessage ?? "raw log unavailable";
    return `${eventType} · ${stationMac} · ${hostname} · ${rawMessage}`;
  });

  return [...attendanceItems, ...logItems].slice(0, 5);
}

export function DashboardPage() {
  const [state, setState] = useState<DashboardState>({
    employees: [],
    attendance: [],
    logs: [],
  });
  const [isLoading, setIsLoading] = useState(true);
  const [errorMessage, setErrorMessage] = useState("");

  useEffect(() => {
    let isActive = true;

    void (async () => {
      const [employeesResult, attendanceResult, logsResult] = await Promise.allSettled([
        getEmployees(),
        getAttendanceRecords(),
        getLogs(),
      ]);

      if (!isActive) {
        return;
      }

      const nextState: DashboardState = {
        employees:
          employeesResult.status === "fulfilled" ? employeesResult.value : [],
        attendance:
          attendanceResult.status === "fulfilled" ? attendanceResult.value : [],
        logs: logsResult.status === "fulfilled" ? logsResult.value : [],
      };

      setState(nextState);
      setIsLoading(false);

      const failedRequests = [employeesResult, attendanceResult, logsResult].filter(
        (result) => result.status === "rejected",
      ).length;

      setErrorMessage(
        failedRequests > 0 ? "总览数据加载不完整，请稍后重试" : "",
      );
    })();

    return () => {
      isActive = false;
    };
  }, []);

  const metrics = useMemo<DashboardMetric[]>(() => {
    const totalEmployees = state.employees.length;
    const activeEmployees = state.employees.filter(
      (employee) => employee.status !== "disabled",
    ).length;
    const attentionCount = state.attendance.filter(
      (record) =>
        record.exceptionStatus !== "none" ||
        record.clockOutStatus === "pending" ||
        record.clockOutStatus === "missing" ||
        record.sourceMode === "manual",
    ).length;
    const recentLogs = state.logs.length;

    return [
      {
        label: "员工总数",
        value: String(totalEmployees),
        status: isLoading ? "等待同步" : "已加载员工档案",
      },
      {
        label: "在岗员工",
        value: String(activeEmployees),
        status: totalEmployees > 0 ? `${Math.round((activeEmployees / totalEmployees) * 100)}% 在线` : "暂无员工",
      },
      {
        label: "待处理考勤",
        value: String(attentionCount),
        status: attentionCount > 0 ? "需要人工复核" : "当前无待处理记录",
      },
      {
        label: "最近日志",
        value: String(recentLogs),
        status: recentLogs > 0 ? "已接入日志流" : "暂无日志",
      },
    ];
  }, [isLoading, state.attendance, state.employees, state.logs]);

  const watchItems = useMemo(() => buildAttentionItems(state), [state]);

  const signalWidths = useMemo(() => {
    const totalEmployees = state.employees.length || 1;
    const activeEmployees = state.employees.filter(
      (employee) => employee.status !== "disabled",
    ).length;
    const attendanceHealth = state.attendance.length
      ? 1 -
        state.attendance.filter(
          (record) =>
            record.exceptionStatus !== "none" ||
            record.clockOutStatus === "pending" ||
            record.clockOutStatus === "missing" ||
            record.sourceMode === "manual",
        ).length /
          state.attendance.length
      : 0.5;
    const parsedLogs = state.logs.filter(
      (item) => item.message.parseStatus === "parsed" || item.message.parseStatus === "ok",
    ).length;

    return [
      `${Math.max(20, Math.round((activeEmployees / totalEmployees) * 100))}%`,
      `${Math.max(20, Math.round(attendanceHealth * 100))}%`,
      `${Math.max(20, Math.round((state.logs.length ? parsedLogs / state.logs.length : 0.5) * 100))}%`,
      `${Math.max(20, Math.round(Math.min(1, state.logs.length / 10) * 100))}%`,
    ];
  }, [state.attendance, state.employees, state.logs]);

  return (
    <section className="page">
      <header className="page-header">
        <span className="page-header__eyebrow">Overview</span>
        <div>
          <h2>Dashboard</h2>
          <p>真实运行总览，汇聚员工、考勤与日志的最新状态。</p>
        </div>
      </header>

      <div className="metric-strip">
        {metrics.map((metric) => (
          <article key={metric.label} className="panel metric-card">
            <span>{metric.label}</span>
            <strong>{metric.value}</strong>
            <p>{metric.status}</p>
          </article>
        ))}
      </div>

      <div className="page-grid page-grid--dashboard">
        <article className="panel panel--tall">
          <div className="panel__header">
            <h3>近期关注</h3>
            <span>{isLoading ? "同步中..." : `${watchItems.length} 条`}</span>
          </div>
          {errorMessage ? <p className="panel__copy">{errorMessage}</p> : null}
          <ul className="stack-list">
            {watchItems.length > 0 ? (
              watchItems.map((item, index) => <li key={`${index}-${item}`}>{item}</li>)
            ) : (
              <li>{isLoading ? "等待数据加载..." : "暂无关注项"}</li>
            )}
          </ul>
        </article>

        <article className="panel">
          <div className="panel__header">
            <h3>同步健康度</h3>
            <span>实时计算</span>
          </div>
          <div className="signal-bars" aria-hidden="true">
            {signalWidths.map((width, index) => (
              <span key={`${index}-${width}`} style={{ width }} />
            ))}
          </div>
          <p className="panel__copy">
            {isLoading
              ? "正在接入后端员工、考勤和日志数据..."
              : "健康度条带基于当前员工在岗率、考勤异常率和日志解析率计算。"}
          </p>
        </article>
      </div>
    </section>
  );
}
