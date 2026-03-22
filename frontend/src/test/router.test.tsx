import "@testing-library/jest-dom/vitest";
import { render, screen } from "@testing-library/react";
import { createMemoryRouter, RouterProvider } from "react-router-dom";
import { afterEach, describe, expect, it, vi } from "vitest";

import { appRoutes } from "../app/router";
import { mockJsonFetch } from "./fetchMock";

afterEach(() => {
  vi.unstubAllGlobals();
  vi.restoreAllMocks();
});

describe("console router", () => {
  it("renders all six navigation items in Chinese on the command deck route", async () => {
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
          pagination: { page: 1, pageSize: 10, totalItems: 0, totalPages: 0 },
        },
      },
      {
        method: "GET",
        path: "/api/settings",
        response: [],
      },
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
          pagination: { page: 1, pageSize: 10, totalItems: 0, totalPages: 0 },
        },
      },
    ]);

    const router = createMemoryRouter(appRoutes, {
      initialEntries: ["/"],
    });

    render(<RouterProvider router={router} />);

    expect(
      screen.getByRole("link", { name: /指挥台 实时总览与关键告警/i }),
    ).toBeInTheDocument();
    expect(
      screen.getByRole("link", { name: /日志流 接入状态与实时检索/i }),
    ).toBeInTheDocument();
    expect(
      screen.getByRole("link", { name: /员工档案 人员与设备映射维护/i }),
    ).toBeInTheDocument();
    expect(
      screen.getByRole("link", { name: /考勤复核 异常队列与人工修正/i }),
    ).toBeInTheDocument();
    expect(
      screen.getByRole("link", { name: /系统设置 日切规则与上报链路/i }),
    ).toBeInTheDocument();
    expect(
      screen.getByRole("link", { name: /调试工具 手工注入与飞书重发/i }),
    ).toBeInTheDocument();
    expect(await screen.findByRole("heading", { name: "指挥台" })).toBeInTheDocument();
  });

  it("renders the logs page on a non-default route", async () => {
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
          pagination: { page: 1, pageSize: 10, totalItems: 0, totalPages: 0 },
        },
      },
      {
        method: "GET",
        path: "/api/settings",
        response: [],
      },
      {
        method: "GET",
        path: "/api/employees",
        response: { items: [] },
      },
      {
        method: "GET",
        path: /\/api\/logs\?page=1(?:&.*)?$/,
        response: {
          items: [],
          pagination: { page: 1, pageSize: 10, totalItems: 0, totalPages: 0 },
        },
      },
    ]);

    const router = createMemoryRouter(appRoutes, {
      initialEntries: ["/logs"],
    });

    render(<RouterProvider router={router} />);

    expect(await screen.findByRole("heading", { name: "日志流" })).toBeInTheDocument();
    expect(
      screen.getByRole("link", { name: /日志流 接入状态与实时检索/i }),
    ).toHaveAttribute("aria-current", "page");
  });

  it("renders the debug page on the debug route", async () => {
    mockJsonFetch([
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
              devices: [
                {
                  macAddress: "94:89:78:55:9a:f3",
                  deviceLabel: "Scanner",
                  status: "active",
                },
              ],
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
              lastDisconnectAt: "2026-03-21T18:05:00+08:00",
              clockInStatus: "done",
              clockOutStatus: "done",
              exceptionStatus: "none",
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
          pagination: { page: 1, pageSize: 10, totalItems: 0, totalPages: 0 },
        },
      },
      {
        method: "GET",
        path: "/api/settings",
        response: [],
      },
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
              devices: [
                {
                  macAddress: "94:89:78:55:9a:f3",
                  deviceLabel: "Scanner",
                  status: "active",
                },
              ],
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
              lastDisconnectAt: "2026-03-21T18:05:00+08:00",
              clockInStatus: "done",
              clockOutStatus: "done",
              exceptionStatus: "none",
              sourceMode: "syslog",
              version: 1,
              lastCalculatedAt: "2026-03-21T18:05:01+08:00",
            },
          ],
        },
      },
    ]);

    const router = createMemoryRouter(appRoutes, {
      initialEntries: ["/debug"],
    });

    render(<RouterProvider router={router} />);

    expect(await screen.findByRole("heading", { name: "调试工具" })).toBeInTheDocument();
    expect(screen.getByLabelText("原始 syslog")).toBeInTheDocument();
    expect(screen.getByLabelText("接收日期时间")).toBeInTheDocument();
    expect(screen.getByRole("button", { name: "注入 syslog" })).toBeInTheDocument();
    expect(screen.getByRole("button", { name: "发送上班到飞书" })).toBeInTheDocument();
    expect(screen.getByRole("button", { name: "发送下班到飞书" })).toBeInTheDocument();
  });
});
