import "@testing-library/jest-dom/vitest";
import { fireEvent, render, screen, waitFor, within } from "@testing-library/react";
import { createMemoryRouter, RouterProvider } from "react-router-dom";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

import { appRoutes } from "../app/router";
import { mockJsonFetch } from "./fetchMock";

describe("debug page", () => {
  beforeEach(() => {
    vi.stubGlobal("confirm", vi.fn(() => true));
  });

  afterEach(() => {
    vi.unstubAllGlobals();
    vi.restoreAllMocks();
  });

  it("injects syslog and dispatches the selected attendance report after confirmation", async () => {
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
      {
        method: "POST",
        path: "/api/debug/syslog",
        response: {
          accepted: true,
          receivedAt: "2026-03-21T08:01:00+08:00",
          parseStatus: "parsed",
          parseError: "",
        },
        assertBody: (body) => {
          expect(body).toEqual({
            rawMessage:
              "Mar 21 08:01:00 stamgr: client_footprints connect Station[94:89:78:55:9a:f3] AP[28:b3:71:25:ae:a0] ssid[FactoryOps] osvendor[Unknown] hostname[scanner-01]",
            receivedAt: "2026-03-21T08:01:00+08:00",
          });
        },
      },
      {
        method: "POST",
        path: "/api/debug/attendance/101/dispatch",
        response: {
          attendance: {
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
          report: {
            id: 501,
            attendanceRecordId: 101,
            reportType: "clock_in",
            reportStatus: "success",
            notificationStatus: "success",
            notificationMessageId: "om_msg_001",
            notificationResponseCode: 200,
            notificationResponseBody: "{\"code\":0}",
            notificationRetryCount: 0,
            responseCode: 200,
            responseBody: "{\"code\":0}",
            externalRecordId: "flow_new_001",
          },
        },
        assertBody: (body) => {
          expect(body).toEqual({ reportType: "clock_in" });
        },
      },
    ]);

    const router = createMemoryRouter(appRoutes, {
      initialEntries: ["/debug"],
    });

    render(<RouterProvider router={router} />);

    const rawInput = await screen.findByLabelText("原始 syslog");
    fireEvent.change(rawInput, {
      target: {
        value:
          "Mar 21 08:01:00 stamgr: client_footprints connect Station[94:89:78:55:9a:f3] AP[28:b3:71:25:ae:a0] ssid[FactoryOps] osvendor[Unknown] hostname[scanner-01]",
      },
    });
    fireEvent.change(screen.getByLabelText("接收日期时间"), {
      target: { value: "2026-03-21T08:01" },
    });
    fireEvent.click(screen.getByRole("button", { name: "注入 syslog" }));

    expect(window.confirm).toHaveBeenCalled();
    expect(await screen.findByText("已注入，当前解析状态：parsed")).toBeInTheDocument();

    const row = screen.getByRole("group", { name: /Lena Wu 调试记录/i });
    fireEvent.click(within(row).getByRole("button", { name: "发送上班到飞书" }));

    expect(
      await screen.findByText("clock_in 已发送，结果：success，通知：success（消息 ID: om_msg_001）"),
    ).toBeInTheDocument();
    expect(screen.getByRole("heading", { name: "最近一次注入结果" })).toBeInTheDocument();
    expect(screen.getByRole("heading", { name: "最近一次发送结果" })).toBeInTheDocument();
    assertAllMatched();
  });

  it("does not fire debug requests when the user cancels confirmation", async () => {
    vi.stubGlobal("confirm", vi.fn(() => false));

    const { assertAllMatched, requests } = mockJsonFetch([
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
    ]);

    const router = createMemoryRouter(appRoutes, {
      initialEntries: ["/debug"],
    });

    render(<RouterProvider router={router} />);

    fireEvent.change(await screen.findByLabelText("原始 syslog"), {
      target: { value: "invalid syslog" },
    });
    fireEvent.change(screen.getByLabelText("接收日期时间"), {
      target: { value: "2026-03-21T08:01" },
    });
    fireEvent.click(screen.getByRole("button", { name: "注入 syslog" }));

    expect(window.confirm).toHaveBeenCalled();
    await waitFor(() => {
      expect(requests.filter((request) => request.method === "POST")).toHaveLength(0);
    });
    assertAllMatched();
  });
});
