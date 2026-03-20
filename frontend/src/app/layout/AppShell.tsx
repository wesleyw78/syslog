import type { ReactNode } from "react";
import { NavLink, Outlet } from "react-router-dom";

type ConsoleSection = {
  description: string;
  path: string;
  title: string;
};

export const consoleSections: ConsoleSection[] = [
  {
    title: "Dashboard",
    path: "/",
    description: "Plant pulse and active alerts",
  },
  {
    title: "Logs",
    path: "/logs",
    description: "Ingestion stream and exception tail",
  },
  {
    title: "Employees",
    path: "/employees",
    description: "Roster status and certifications",
  },
  {
    title: "Attendance",
    path: "/attendance",
    description: "Shift coverage and check-in drift",
  },
  {
    title: "Settings",
    path: "/settings",
    description: "Runtime controls and audit locks",
  },
];

const railSignals = [
  { label: "Sync", value: "Stable", tone: "good" },
  { label: "Alerts", value: "02 Open", tone: "alert" },
  { label: "Feed", value: "512/s", tone: "neutral" },
  { label: "Auth", value: "Nominal", tone: "good" },
] as const;

const quickPulse = [
  "North Gate scanners online",
  "Shift handoff in 12 minutes",
  "Archive compression window 03:00",
];

type AppShellProps = {
  children?: ReactNode;
};

export function AppShell({ children }: AppShellProps) {
  return (
    <div className="app-shell">
      <header className="status-rail">
        <div className="status-rail__identity">
          <span className="status-rail__eyebrow">SYSLOG / INDUSTRIAL OPS</span>
          <h1>Control Room Console</h1>
        </div>
        <div className="status-rail__signals" aria-label="System status rail">
          {railSignals.map((signal) => (
            <article
              key={signal.label}
              className={`signal-chip signal-chip--${signal.tone}`}
            >
              <span>{signal.label}</span>
              <strong>{signal.value}</strong>
            </article>
          ))}
        </div>
      </header>

      <div className="shell-frame">
        <aside className="console-nav" aria-label="Primary navigation">
          <div className="console-nav__panel">
            <p className="console-nav__label">Route Matrix</p>
            <nav className="console-nav__items">
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
                  <span className="nav-card__description">
                    {section.description}
                  </span>
                </NavLink>
              ))}
            </nav>
          </div>

          <div className="console-nav__panel console-nav__panel--pulse">
            <p className="console-nav__label">Pulse Notes</p>
            <ul className="pulse-list">
              {quickPulse.map((item) => (
                <li key={item}>{item}</li>
              ))}
            </ul>
          </div>
        </aside>

        <main className="console-main">
          <div className="console-main__frame">{children ?? <Outlet />}</div>
        </main>
      </div>
    </div>
  );
}
