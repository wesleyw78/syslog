import {
  type ReactNode,
  useEffect,
  useMemo,
  useState,
} from "react";
import { NavLink, Outlet } from "react-router-dom";

import {
  getAttendanceRecords,
  getEmployees,
  getLogs,
  getSettings,
  type AttendanceRecord,
  type Employee,
  type SystemSetting,
} from "../../lib/api";
import { requiresAttendanceAttention } from "../../features/attendance/attention";

type ThemeMode = "system" | "light" | "dark";
type ResolvedTheme = "light" | "dark";

type ConsoleSection = {
  description: string;
  path: string;
  title: string;
};

type ShellSummary = {
  activeEmployees: number;
  attentionCount: number;
  dayEndTime: string;
  employeeTotal: number;
  feishuReady: boolean;
  hasLoadError: boolean;
  logsTotal: number;
};

const THEME_STORAGE_KEY = "syslog-console-theme";

export const consoleSections: ConsoleSection[] = [
  {
    title: "指挥台",
    path: "/",
    description: "实时总览与关键告警",
  },
  {
    title: "日志流",
    path: "/logs",
    description: "接入状态与实时检索",
  },
  {
    title: "考勤复核",
    path: "/attendance",
    description: "异常队列与人工修正",
  },
  {
    title: "员工档案",
    path: "/employees",
    description: "人员与设备映射维护",
  },
  {
    title: "系统设置",
    path: "/settings",
    description: "日切规则与上报链路",
  },
  {
    title: "调试工具",
    path: "/debug",
    description: "手工注入与飞书重发",
  },
];

const shellQuickActions = [
  { label: "前往考勤复核", path: "/attendance" },
  { label: "查看日志流", path: "/logs" },
  { label: "检查系统设置", path: "/settings" },
] as const;

type AppShellProps = {
  children?: ReactNode;
};

function readStoredThemeMode(): ThemeMode {
  if (typeof window === "undefined") {
    return "system";
  }

  const storedValue = window.localStorage.getItem(THEME_STORAGE_KEY);
  return storedValue === "light" || storedValue === "dark" ? storedValue : "system";
}

function getSystemTheme(): ResolvedTheme {
  if (typeof window === "undefined" || typeof window.matchMedia !== "function") {
    return "light";
  }

  return window.matchMedia("(prefers-color-scheme: dark)").matches ? "dark" : "light";
}

function resolveTheme(mode: ThemeMode, systemTheme: ResolvedTheme): ResolvedTheme {
  return mode === "system" ? systemTheme : mode;
}

function createShellSummary(
  employees: Employee[],
  attendance: AttendanceRecord[],
  logsTotal: number,
  settings: SystemSetting[],
  hasLoadError: boolean,
): ShellSummary {
  const settingsMap = new Map(
    settings.map((item) => [item.settingKey, item.settingValue.trim()]),
  );

  return {
    activeEmployees: employees.filter((employee) => employee.status !== "disabled")
      .length,
    attentionCount: attendance.filter(requiresAttendanceAttention).length,
    dayEndTime: settingsMap.get("day_end_time") ?? "",
    employeeTotal: employees.length,
    feishuReady: Boolean(
      settingsMap.get("feishu_app_id") &&
        settingsMap.get("feishu_app_secret") &&
        settingsMap.get("feishu_location_name"),
    ),
    hasLoadError,
    logsTotal,
  };
}

function buildShellHeadline(summary: ShellSummary): string {
  if (summary.hasLoadError) {
    return "部分运行摘要加载失败，请优先检查接口与后端链路";
  }
  if (summary.attentionCount > 0) {
    return `当前有 ${summary.attentionCount} 条待处理考勤，建议优先进入复核工作区`;
  }
  if (!summary.feishuReady) {
    return "飞书上报配置未完成，建议先补齐关键设置";
  }
  return "系统摘要已同步，可继续观察日志与考勤链路";
}

function buildShellDescription(summary: ShellSummary): string {
  if (summary.hasLoadError) {
    return "壳层摘要已经改为真实数据驱动；当后端或接口异常时，这里会直接暴露问题，而不是继续显示伪造的稳定状态。";
  }

  return `当前共 ${summary.employeeTotal} 条员工档案、${summary.logsTotal} 条日志记录；日切时间 ${
    summary.dayEndTime || "未配置"
  }。`;
}

export function AppShell({ children }: AppShellProps) {
  const [themeMode, setThemeMode] = useState<ThemeMode>(() => readStoredThemeMode());
  const [systemTheme, setSystemTheme] = useState<ResolvedTheme>(() => getSystemTheme());
  const [shellSummary, setShellSummary] = useState<ShellSummary>({
    activeEmployees: 0,
    attentionCount: 0,
    dayEndTime: "",
    employeeTotal: 0,
    feishuReady: false,
    hasLoadError: false,
    logsTotal: 0,
  });

  useEffect(() => {
    if (typeof window === "undefined" || typeof window.matchMedia !== "function") {
      return;
    }

    const mediaQueryList = window.matchMedia("(prefers-color-scheme: dark)");
    const handleChange = (event: MediaQueryListEvent) => {
      setSystemTheme(event.matches ? "dark" : "light");
    };

    setSystemTheme(mediaQueryList.matches ? "dark" : "light");
    mediaQueryList.addEventListener?.("change", handleChange);

    return () => {
      mediaQueryList.removeEventListener?.("change", handleChange);
    };
  }, []);

  const resolvedTheme = useMemo(
    () => resolveTheme(themeMode, systemTheme),
    [systemTheme, themeMode],
  );

  useEffect(() => {
    document.documentElement.dataset.theme = resolvedTheme;

    if (themeMode === "system") {
      window.localStorage.removeItem(THEME_STORAGE_KEY);
      return;
    }

    window.localStorage.setItem(THEME_STORAGE_KEY, themeMode);
  }, [resolvedTheme, themeMode]);

  useEffect(() => {
    let isActive = true;

    void (async () => {
      const [employeesResult, attendanceResult, logsResult, settingsResult] =
        await Promise.allSettled([
          getEmployees(),
          getAttendanceRecords(),
          getLogs({ page: 1 }),
          getSettings(),
        ]);

      if (!isActive) {
        return;
      }

      const employees =
        employeesResult.status === "fulfilled" ? employeesResult.value : [];
      const attendance =
        attendanceResult.status === "fulfilled" ? attendanceResult.value : [];
      const logsTotal =
        logsResult.status === "fulfilled"
          ? logsResult.value.pagination.totalItems
          : 0;
      const settings =
        settingsResult.status === "fulfilled" ? settingsResult.value : [];
      const hasLoadError = [
        employeesResult,
        attendanceResult,
        logsResult,
        settingsResult,
      ].some((result) => result.status === "rejected");

      setShellSummary(
        createShellSummary(
          employees,
          attendance,
          logsTotal,
          settings,
          hasLoadError,
        ),
      );
    })();

    return () => {
      isActive = false;
    };
  }, []);

  const commandSignals = useMemo(
    () => [
      {
        label: "同步状态",
        value: shellSummary.hasLoadError ? "部分异常" : "已同步",
        tone: shellSummary.hasLoadError ? "warn" : "good",
      },
      {
        label: "待处理项",
        value: `${shellSummary.attentionCount} 条待处理`,
        tone: shellSummary.attentionCount > 0 ? "warn" : "good",
      },
      {
        label: "日志总量",
        value: `${shellSummary.logsTotal} 条`,
        tone: "neutral",
      },
      {
        label: "飞书配置",
        value: shellSummary.feishuReady ? "已配置" : "待完善",
        tone: shellSummary.feishuReady ? "good" : "warn",
      },
    ],
    [shellSummary],
  );

  const pulseNotes = useMemo(
    () => [
      `员工档案 ${shellSummary.employeeTotal} 人，在岗 ${shellSummary.activeEmployees} 人`,
      `日切时间 ${shellSummary.dayEndTime || "未配置"}`,
      shellSummary.hasLoadError
        ? "当前摘要存在加载失败项"
        : shellSummary.attentionCount > 0
          ? `当前 ${shellSummary.attentionCount} 条考勤待复核`
          : "当前无待处理考勤",
    ],
    [shellSummary],
  );

  return (
    <div className="app-shell">
      <aside className="shell-rail" aria-label="主导航">
        <div className="shell-rail__brand">
          <span className="shell-rail__eyebrow">SYSLOG CONSOLE</span>
          <h1>值班指挥台</h1>
          <p>面向日志接入、考勤异常和配置联动的运维控制中心。</p>
        </div>

        <nav className="shell-rail__nav" aria-label="主导航">
          {consoleSections.map((section) => (
            <NavLink
              key={section.path}
              end={section.path === "/"}
              to={section.path}
              className={({ isActive }) =>
                isActive ? "nav-card nav-card--active" : "nav-card"
              }
            >
              <span className="nav-card__title">{section.title}</span>
              <span className="nav-card__description">{section.description}</span>
            </NavLink>
          ))}
        </nav>

        <section className="shell-rail__footer">
          <div className="rail-panel">
            <p className="rail-panel__label">班次提醒</p>
            <ul className="pulse-list">
              {pulseNotes.map((item) => (
                <li key={item}>{item}</li>
              ))}
            </ul>
          </div>
        </section>
      </aside>

      <div className="shell-stage">
        <header className="command-bar">
          <div className="command-bar__summary">
            <span className="command-bar__eyebrow">Command Deck</span>
            <h2>{buildShellHeadline(shellSummary)}</h2>
            <p>{buildShellDescription(shellSummary)}</p>
          </div>

          <div className="command-bar__actions">
            <div className="theme-toggle" role="group" aria-label="主题切换">
              <button
                type="button"
                aria-label="跟随系统主题"
                aria-pressed={themeMode === "system"}
                className={themeMode === "system" ? "theme-toggle__button theme-toggle__button--active" : "theme-toggle__button"}
                onClick={() => setThemeMode("system")}
              >
                跟随系统
              </button>
              <button
                type="button"
                aria-label="浅色主题"
                aria-pressed={themeMode === "light"}
                className={themeMode === "light" ? "theme-toggle__button theme-toggle__button--active" : "theme-toggle__button"}
                onClick={() => setThemeMode("light")}
              >
                浅色
              </button>
              <button
                type="button"
                aria-label="深色主题"
                aria-pressed={themeMode === "dark"}
                className={themeMode === "dark" ? "theme-toggle__button theme-toggle__button--active" : "theme-toggle__button"}
                onClick={() => setThemeMode("dark")}
              >
                深色
              </button>
            </div>

            <div className="quick-actions" aria-label="快捷动作">
              {shellQuickActions.map((action) => (
                <NavLink
                  key={action.path}
                  to={action.path}
                  className="quick-actions__button quick-actions__link"
                >
                  {action.label}
                </NavLink>
              ))}
            </div>
          </div>
        </header>

        <section className="signal-grid" aria-label="系统摘要">
          {commandSignals.map((signal) => (
            <article
              key={signal.label}
              className={`signal-chip signal-chip--${signal.tone}`}
            >
              <span>{signal.label}</span>
              <strong>{signal.value}</strong>
            </article>
          ))}
        </section>

        <main className="console-main">
          <div className="console-main__frame">{children ?? <Outlet />}</div>
        </main>
      </div>
    </div>
  );
}
