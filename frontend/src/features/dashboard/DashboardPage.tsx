import { useEffect, useMemo, useState } from "react";

import {
  getAttendanceRecords,
  getEmployees,
  getLogs,
  type AttendanceRecord,
  type Employee,
  type LogItem,
} from "../../lib/api";
import {
  getAttendanceAttentionReason,
  requiresAttendanceAttention,
} from "../attendance/attention";

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

type DashboardFocusItem = {
  title: string;
  detail: string;
};

type DashboardFocusState = {
  attendanceItems: DashboardFocusItem[];
  logItems: DashboardFocusItem[];
};

function toAttendanceAttentionLabel(reason: string): string {
  switch (reason) {
    case "missing_disconnect":
    case "clock_out_missing":
      return "缺少下班记录";
    case "clock_in_pending":
      return "等待确认上班记录";
    case "clock_out_pending":
      return "等待确认下班记录";
    default:
      return reason;
  }
}

function buildFocusState(state: DashboardState): DashboardFocusState {
  const employeeMap = new Map(
    state.employees.map((employee) => [employee.id, employee]),
  );
  const attendanceItems: DashboardFocusItem[] = state.attendance
    .filter(requiresAttendanceAttention)
    .map((record) => {
      const employeeName =
        employeeMap.get(record.employeeId)?.name ?? record.employeeId;
      const attentionReason = getAttendanceAttentionReason(record) ?? "unknown";
      return {
        title: `${employeeName} ${record.attendanceDate}`,
        detail: toAttendanceAttentionLabel(attentionReason),
      };
    });

  const logItems: DashboardFocusItem[] = state.logs.slice(0, 4).map((item) => {
    const eventType = item.event?.eventType?.trim() || "未识别事件";
    const stationMac = item.event?.stationMac?.trim() || "未识别设备";
    const rawMessage = item.message.rawMessage?.trim() || "";
    const hostname = item.event?.hostname?.trim() || "";
    return {
      title: `${eventType} · ${stationMac}`,
      detail: rawMessage || hostname || "缺少可读线索",
    };
  });

  return { attendanceItems, logItems };
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
      const [employeesResult, attendanceResult, logsResult] =
        await Promise.allSettled([
          getEmployees(),
          getAttendanceRecords(),
          getLogs({ page: 1 }),
        ]);

      if (!isActive) {
        return;
      }

      const nextState: DashboardState = {
        employees:
          employeesResult.status === "fulfilled" ? employeesResult.value : [],
        attendance:
          attendanceResult.status === "fulfilled" ? attendanceResult.value : [],
        logs: logsResult.status === "fulfilled" ? logsResult.value.items : [],
      };

      setState(nextState);
      setIsLoading(false);

      const failedRequests = [
        employeesResult,
        attendanceResult,
        logsResult,
      ].filter((result) => result.status === "rejected").length;

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
      requiresAttendanceAttention,
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
        status:
          totalEmployees > 0
            ? `${Math.round((activeEmployees / totalEmployees) * 100)}% 在线`
            : "暂无员工",
      },
      {
        label: "待处理考勤",
        value: String(attentionCount),
        status: attentionCount > 0 ? "需要人工复核" : "当前无待处理记录",
      },
      {
        label: "日志接入",
        value: String(recentLogs),
        status: recentLogs > 0 ? "当前页存在最新日志" : "暂无日志",
      },
    ];
  }, [isLoading, state.attendance, state.employees, state.logs]);

  const focusState = useMemo(() => buildFocusState(state), [state]);

  const signalWidths = useMemo(() => {
    const totalEmployees = state.employees.length || 1;
    const activeEmployees = state.employees.filter(
      (employee) => employee.status !== "disabled",
    ).length;
    const attendanceHealth = state.attendance.length
      ? 1 -
        state.attendance.filter(requiresAttendanceAttention).length /
          state.attendance.length
      : 0.5;
    const parsedLogs = state.logs.filter(
      (item) =>
        item.message.parseStatus === "parsed" ||
        item.message.parseStatus === "ok",
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
        <span className="page-header__eyebrow">总览态势</span>
        <div>
          <h2>指挥台</h2>
          <p>聚合员工、考勤与日志状态，优先展示当前值班最需要处理的异常与信号。</p>
        </div>
      </header>

      <section className="dashboard-hero">
        <div className="dashboard-hero__content">
          <span className="dashboard-hero__label">值班摘要</span>
          <h3>
            {errorMessage
              ? "部分上游数据未完成同步，需要优先确认链路状态"
              : focusState.attendanceItems.length > 0
                ? `当前有 ${focusState.attendanceItems.length} 条考勤待复核`
                : focusState.logItems.length > 0
                  ? "当前无阻塞性考勤异常，可继续观察最新日志线索"
                : "当前无阻塞性异常，可继续观察实时信号"}
          </h3>
          <p>
            首页承担先判断后处理的职责，按“值班状态、待处理动作、链路观察”三层组织内容。
          </p>
        </div>
        <div className="dashboard-hero__aside">
          <span className="dashboard-hero__chip">
            {errorMessage ? "同步异常，请检查后端" : "摘要已同步"}
          </span>
          <span className="dashboard-hero__chip dashboard-hero__chip--accent">
            待处理考勤 {metrics[2]?.value ?? "0"}
          </span>
          <span className="dashboard-hero__chip">
            当前页日志 {metrics[3]?.value ?? "0"}
          </span>
        </div>
      </section>

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
            <h3>待处理考勤</h3>
            <span>
              {isLoading ? "同步中..." : `${focusState.attendanceItems.length} 条`}
            </span>
          </div>
          {errorMessage ? <p className="panel__copy">{errorMessage}</p> : null}
          <ul className="stack-list">
            {focusState.attendanceItems.length > 0 ? (
              focusState.attendanceItems.map((item) => (
                <li key={`${item.title}-${item.detail}`}>
                  <strong>{item.title}</strong>
                  {` · ${item.detail}`}
                </li>
              ))
            ) : (
              <li>{isLoading ? "等待数据加载..." : "暂无待处理考勤"}</li>
            )}
          </ul>
        </article>

        <article className="panel">
          <div className="panel__header">
            <h3>链路观察</h3>
            <span>{`${focusState.logItems.length} 条日志线索`}</span>
          </div>
          <div className="signal-bars" aria-hidden="true">
            {signalWidths.map((width, index) => (
              <span key={`${index}-${width}`} style={{ width }} />
            ))}
          </div>
          <p className="panel__copy">
            {isLoading
              ? "正在接入后端员工、考勤和日志数据..."
              : "条带基于员工在岗率、考勤异常率和日志解析率计算，用于快速判断是否需要切页深入处理。"}
          </p>
          <ul className="stack-list">
            {focusState.logItems.length > 0 ? (
              focusState.logItems.map((item) => (
                <li key={`${item.title}-${item.detail}`}>
                  <strong>{item.title}</strong>
                  {` · ${item.detail}`}
                </li>
              ))
            ) : (
              <li>{isLoading ? "等待日志同步..." : "当前没有最新日志线索"}</li>
            )}
          </ul>
        </article>
      </div>
    </section>
  );
}
