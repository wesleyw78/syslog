import "@testing-library/jest-dom/vitest";
import { render, screen, waitFor, within } from "@testing-library/react";
import { afterEach, describe, expect, it, vi } from "vitest";

import { DashboardPage } from "../features/dashboard/DashboardPage";
import { mockJsonFetch } from "./fetchMock";

afterEach(() => {
  vi.unstubAllGlobals();
  vi.restoreAllMocks();
});

describe("dashboard page", () => {
  it("derives summary cards and watch items from live API data", async () => {
    const { assertAllMatched, fetchMock } = mockJsonFetch([
      {
        method: "GET",
        path: "/api/employees",
        response: {
          items: [
            {
              id: "emp-1",
              employeeNo: "E-001",
              systemNo: "SYS-001",
              name: "Lena Wu",
              status: "active",
              devices: [],
              createdAt: "2026-03-01T08:00:00Z",
              updatedAt: "2026-03-01T08:00:00Z",
            },
            {
              id: "emp-2",
              employeeNo: "E-002",
              systemNo: "SYS-002",
              name: "Arjun Patel",
              status: "active",
              devices: [],
              createdAt: "2026-03-01T08:00:00Z",
              updatedAt: "2026-03-01T08:00:00Z",
            },
            {
              id: "emp-3",
              employeeNo: "E-003",
              systemNo: "SYS-003",
              name: "Mina Torres",
              status: "disabled",
              devices: [],
              createdAt: "2026-03-01T08:00:00Z",
              updatedAt: "2026-03-01T08:00:00Z",
            },
            {
              id: 4,
              employeeNo: "E-004",
              systemNo: "SYS-004",
              name: "Nora King",
              status: "active",
              devices: [],
              createdAt: "2026-03-01T08:00:00Z",
              updatedAt: "2026-03-01T08:00:00Z",
            },
            {
              id: 5,
              employeeNo: "E-005",
              systemNo: "SYS-005",
              name: "Omar Reed",
              status: "active",
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
              id: "att-1",
              employeeId: "emp-1",
              attendanceDate: "2026-03-21",
              firstConnectAt: "2026-03-21T06:02:00Z",
              lastDisconnectAt: "2026-03-21T14:00:00Z",
              clockInStatus: "done",
              clockOutStatus: "ready",
              exceptionStatus: "none",
              sourceMode: "syslog",
              version: 1,
              lastCalculatedAt: "2026-03-21T14:01:00Z",
            },
            {
              id: "att-2",
              employeeId: "emp-2",
              attendanceDate: "2026-03-21",
              firstConnectAt: "2026-03-21T06:14:00Z",
              lastDisconnectAt: null,
              clockInStatus: "done",
              clockOutStatus: "missing",
              exceptionStatus: "missing_disconnect",
              sourceMode: "syslog",
              version: 2,
              lastCalculatedAt: "2026-03-21T14:10:00Z",
            },
            {
              id: 4004,
              employeeId: 4,
              attendanceDate: "2026-03-21",
              firstConnectAt: "2026-03-21T06:30:00Z",
              lastDisconnectAt: "2026-03-21T15:40:00Z",
              clockInStatus: "done",
              clockOutStatus: "ready",
              exceptionStatus: "none",
              sourceMode: "manual",
              version: 5,
              lastCalculatedAt: "2026-03-21T15:45:00Z",
            },
            {
              id: 5005,
              employeeId: 5,
              attendanceDate: "2026-03-21",
              firstConnectAt: null,
              lastDisconnectAt: "2026-03-21T15:20:00Z",
              clockInStatus: "pending",
              clockOutStatus: "ready",
              exceptionStatus: "none",
              sourceMode: "manual",
              version: 6,
              lastCalculatedAt: "2026-03-21T15:25:00Z",
            },
          ],
        },
      },
      {
        method: "GET",
        path: "/api/logs?page=1",
        response: {
          items: [
            {
              message: {
                ID: 10,
                ReceivedAt: "2026-03-21T08:01:00Z",
                LogTime: "2026-03-21T08:01:00Z",
                ParseStatus: "parsed",
                RawMessage: "gate-1 access granted",
              },
              event: {
                ID: 20,
                EventType: "badge.scan",
                StationMac: "AA:BB:CC:DD:EE:01",
                Hostname: "gate-1",
              },
            },
            {
              message: {
                ID: 11,
                ReceivedAt: "2026-03-21T08:03:00Z",
                LogTime: "2026-03-21T08:03:00Z",
                ParseStatus: "pending",
                RawMessage: "retry after timeout",
              },
              event: {
                ID: 21,
                EventType: "attendance.sync",
                StationMac: "AA:BB:CC:DD:EE:02",
                Hostname: "packing-2",
              },
            },
          ],
          pagination: {
            page: 1,
            pageSize: 10,
            totalItems: 2,
            totalPages: 1,
          },
        },
      },
    ]);

    render(<DashboardPage />);

    await waitFor(() => {
      const totalCard = screen.getByText("员工总数").closest("article");
      expect(totalCard).not.toBeNull();
      expect(
        within(totalCard as HTMLElement).getByText("5"),
      ).toBeInTheDocument();
    });

    const metricCards = Array.from(document.querySelectorAll(".metric-card"));
    const totalCard = metricCards.find((card) => within(card as HTMLElement).queryByText("员工总数")) ?? null;
    const activeCard = metricCards.find((card) => within(card as HTMLElement).queryByText("在岗员工")) ?? null;
    const exceptionCard =
      metricCards.find((card) => within(card as HTMLElement).queryByText("待处理考勤")) ?? null;
    const logCard = metricCards.find((card) => within(card as HTMLElement).queryByText("日志接入")) ?? null;

    expect(totalCard).not.toBeNull();
    expect(activeCard).not.toBeNull();
    expect(exceptionCard).not.toBeNull();
    expect(logCard).not.toBeNull();
    expect(within(totalCard as HTMLElement).getByText("5")).toBeInTheDocument();
    expect(
      within(activeCard as HTMLElement).getByText("4"),
    ).toBeInTheDocument();
    expect(
      within(exceptionCard as HTMLElement).getByText("2"),
    ).toBeInTheDocument();
    expect(within(logCard as HTMLElement).getByText("2")).toBeInTheDocument();
    expect(screen.getByText(/attendance\.sync · AA:BB:CC:DD:EE:02/)).toBeInTheDocument();
    expect(screen.getByText(/retry after timeout/)).toBeInTheDocument();
    const attentionPanel = screen.getByRole("heading", { name: "待处理考勤" }).closest("article");
    expect(attentionPanel).not.toBeNull();
    expect(
      within(attentionPanel as HTMLElement).getByText(/Arjun Patel 2026-03-21/),
    ).toBeInTheDocument();
    expect(
      within(attentionPanel as HTMLElement).getByText(/Omar Reed 2026-03-21/),
    ).toBeInTheDocument();
    expect(attentionPanel).toHaveTextContent("等待确认上班记录");
    expect(
      screen.queryByText(/Nora King 2026-03-21 none/),
    ).not.toBeInTheDocument();
    expect(fetchMock.mock.calls).toHaveLength(3);
    assertAllMatched();
  });
});
