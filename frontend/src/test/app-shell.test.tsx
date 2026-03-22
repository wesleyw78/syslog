import "@testing-library/jest-dom/vitest";
import { fireEvent, render, screen } from "@testing-library/react";
import { MemoryRouter } from "react-router-dom";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

import { AppShell } from "../app/layout/AppShell";
import { mockJsonFetch } from "./fetchMock";

describe("app shell theme controls", () => {
  beforeEach(() => {
    document.documentElement.removeAttribute("data-theme");
    window.localStorage.clear();
    vi.stubGlobal(
      "matchMedia",
      vi.fn().mockImplementation((query: string) => ({
        media: query,
        matches: query.includes("dark"),
        addEventListener: vi.fn(),
        removeEventListener: vi.fn(),
      })),
    );
  });

  afterEach(() => {
    vi.unstubAllGlobals();
  });

  it("switches between system, light, and dark themes and persists manual choice", async () => {
    mockJsonFetch([
      {
        method: "GET",
        path: "/api/employees",
        response: { items: [] },
      },
      {
        method: "GET",
        path: "/api/attendance",
        response: { items: [] },
      },
      {
        method: "GET",
        path: "/api/logs?page=1",
        response: {
          items: [],
          pagination: {
            page: 1,
            pageSize: 10,
            totalItems: 0,
            totalPages: 0,
          },
        },
      },
      {
        method: "GET",
        path: "/api/settings",
        response: { items: [] },
      },
    ]);

    render(
      <MemoryRouter>
        <AppShell>
          <section>
            <h2>占位内容</h2>
          </section>
        </AppShell>
      </MemoryRouter>,
    );

    expect(
      await screen.findByText(/当前共 0 条员工档案、0 条日志记录；日切时间 未配置。/),
    ).toBeInTheDocument();
    expect(document.documentElement).toHaveAttribute("data-theme", "dark");

    fireEvent.click(screen.getByRole("button", { name: "浅色主题" }));
    expect(document.documentElement).toHaveAttribute("data-theme", "light");
    expect(window.localStorage.getItem("syslog-console-theme")).toBe("light");

    fireEvent.click(screen.getByRole("button", { name: "深色主题" }));
    expect(document.documentElement).toHaveAttribute("data-theme", "dark");
    expect(window.localStorage.getItem("syslog-console-theme")).toBe("dark");

    fireEvent.click(screen.getByRole("button", { name: "跟随系统主题" }));
    expect(document.documentElement).toHaveAttribute("data-theme", "dark");
    expect(window.localStorage.getItem("syslog-console-theme")).toBeNull();
  });

  it("renders shell summary from live api data instead of hard-coded signal values", async () => {
    const { assertAllMatched } = mockJsonFetch([
      {
        method: "GET",
        path: "/api/employees",
        response: {
          items: [
            {
              id: 1,
              employeeNo: "E-001",
              systemNo: "SYS-001",
              feishuEmployeeId: "fs-001",
              name: "Lena Wu",
              status: "active",
              devices: [],
              createdAt: "2026-03-01T08:00:00Z",
              updatedAt: "2026-03-01T08:00:00Z",
            },
            {
              id: 2,
              employeeNo: "E-002",
              systemNo: "SYS-002",
              feishuEmployeeId: "",
              name: "Kai Sun",
              status: "disabled",
              devices: [],
              createdAt: "2026-03-01T08:00:00Z",
              updatedAt: "2026-03-01T08:00:00Z",
            },
          ],
        },
      },
      {
        method: "GET",
        path: "/api/attendance",
        response: {
          items: [
            {
              id: 101,
              employeeId: 1,
              attendanceDate: "2026-03-21",
              firstConnectAt: "2026-03-21T08:01:00+08:00",
              lastDisconnectAt: null,
              clockInStatus: "done",
              clockOutStatus: "missing",
              exceptionStatus: "missing_disconnect",
              sourceMode: "syslog",
              version: 1,
              lastCalculatedAt: "2026-03-21T18:05:01+08:00",
            },
          ],
        },
      },
      {
        method: "GET",
        path: "/api/logs?page=1",
        response: {
          items: [],
          pagination: {
            page: 1,
            pageSize: 10,
            totalItems: 23,
            totalPages: 3,
          },
        },
      },
      {
        method: "GET",
        path: "/api/settings",
        response: {
          items: [
            { SettingKey: "day_end_time", SettingValue: "18:30" },
            { SettingKey: "feishu_app_id", SettingValue: "cli_xxx" },
            { SettingKey: "feishu_app_secret", SettingValue: "secret_xxx" },
            { SettingKey: "feishu_location_name", SettingValue: "总部办公区" },
          ],
        },
      },
    ]);

    render(
      <MemoryRouter>
        <AppShell>
          <section>
            <h2>占位内容</h2>
          </section>
        </AppShell>
      </MemoryRouter>,
    );

    expect(await screen.findByText("1 条待处理")).toBeInTheDocument();
    expect(screen.getByText("23 条")).toBeInTheDocument();
    expect(screen.getByText("已配置")).toBeInTheDocument();
    expect(screen.getByRole("link", { name: "前往考勤复核" })).toBeInTheDocument();
    expect(screen.queryByText("512/s")).not.toBeInTheDocument();
    expect(screen.queryByText("运行中")).not.toBeInTheDocument();
    assertAllMatched();
  });
});
