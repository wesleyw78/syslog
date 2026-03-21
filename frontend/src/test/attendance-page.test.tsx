import "@testing-library/jest-dom/vitest";
import { fireEvent, render, screen, within } from "@testing-library/react";
import { afterEach, describe, expect, it, vi } from "vitest";

import { AttendancePage } from "../features/attendance/AttendancePage";
import { mockJsonFetch } from "./fetchMock";

afterEach(() => {
  vi.unstubAllGlobals();
  vi.restoreAllMocks();
});

describe("attendance page", () => {
  it("joins attendance records with employees and posts correction payloads", async () => {
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
              id: 3,
              employeeNo: "E-003",
              systemNo: "SYS-003",
              name: "Mina Torres",
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
              id: 3003,
              employeeId: 3,
              attendanceDate: "2026-03-21",
              firstConnectAt: "2026-03-21T06:20:00Z",
              lastDisconnectAt: "2026-03-21T15:00:00Z",
              clockInStatus: "done",
              clockOutStatus: "ready",
              exceptionStatus: "none",
              sourceMode: "manual",
              version: 4,
              lastCalculatedAt: "2026-03-21T15:10:00Z",
            },
          ],
        },
      },
      {
        method: "POST",
        path: "/api/attendance/att-2/correction",
        response: {
          attendance: {
            id: "att-2",
            employeeId: "emp-2",
            attendanceDate: "2026-03-21",
            firstConnectAt: "2026-03-21T06:18:00Z",
            lastDisconnectAt: null,
            clockInStatus: "done",
            clockOutStatus: "missing",
            exceptionStatus: "missing_disconnect",
            sourceMode: "manual",
            version: 3,
            lastCalculatedAt: "2026-03-21T14:20:00Z",
          },
          reports: [],
        },
        assertBody: (body) => {
          expect(body).toEqual({
            firstConnectAt: "2026-03-21T06:18:00Z",
            lastDisconnectAt: null,
          });
        },
      },
    ]);

    render(<AttendancePage />);

    const correctedRow = await screen.findByRole("group", {
      name: /Arjun Patel 考勤记录/i,
    });

    expect(within(correctedRow).getByText("Arjun Patel")).toBeInTheDocument();
    expect(within(correctedRow).getByText(/done \/ missing/)).toBeInTheDocument();
    expect(within(correctedRow).getByText("missing_disconnect")).toBeInTheDocument();
    expect(within(correctedRow).getByRole("button", { name: "提交修正" })).toBeInTheDocument();

    const resolvedManualRow = await screen.findByRole("group", {
      name: /Mina Torres 考勤记录/i,
    });
    expect(within(resolvedManualRow).getByText("manual")).toBeInTheDocument();
    expect(within(resolvedManualRow).getByText("无异常")).toBeInTheDocument();
    expect(
      within(resolvedManualRow).queryByRole("button", { name: "提交修正" }),
    ).not.toBeInTheDocument();
    expect(within(resolvedManualRow).getByText("无需处理")).toBeInTheDocument();

    fireEvent.change(within(correctedRow).getByLabelText("首次接入"), {
      target: { value: "2026-03-21T06:18:00Z" },
    });
    fireEvent.change(within(correctedRow).getByLabelText("最后断开"), {
      target: { value: "" },
    });
    fireEvent.click(within(correctedRow).getByRole("button", { name: "提交修正" }));

    expect(await within(correctedRow).findByText("manual")).toBeInTheDocument();
    expect(
      await screen.findByText("已提交 Arjun Patel 的人工修正"),
    ).toBeInTheDocument();
    expect(fetchMock.mock.calls).toHaveLength(3);
    assertAllMatched();
  });

  it("shows a loading error when attendance cannot be fetched", async () => {
    mockJsonFetch([
      {
        method: "GET",
        path: "/api/employees",
        response: { items: [] },
      },
      {
        method: "GET",
        path: "/api/attendance",
        response: { message: "boom" },
        status: 500,
      },
    ]);

    render(<AttendancePage />);

    expect(
      await screen.findByText("考勤异常队列加载失败，请稍后重试"),
    ).toBeInTheDocument();
  });
});
