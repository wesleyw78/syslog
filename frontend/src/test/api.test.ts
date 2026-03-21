import "@testing-library/jest-dom/vitest";
import { afterEach, describe, expect, it, vi } from "vitest";

import { getAttendanceRecords, getEmployees } from "../lib/api";
import { mockJsonFetch } from "./fetchMock";

afterEach(() => {
  vi.unstubAllGlobals();
  vi.restoreAllMocks();
});

describe("api normalization", () => {
  it("normalizes numeric employee and attendance ids into strings", async () => {
    const { assertAllMatched } = mockJsonFetch([
      {
        method: "GET",
        path: "/api/employees",
        response: {
          items: [
            {
              id: 101,
              employeeNo: "E-101",
              systemNo: "SYS-101",
              name: "Lena Wu",
              status: "active",
              devices: [
                {
                  macAddress: "AA:BB:CC:DD:EE:01",
                  deviceLabel: "North Gate",
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
              id: 9001,
              employeeId: 101,
              attendanceDate: "2026-03-21",
              firstConnectAt: "2026-03-21T06:02:00Z",
              lastDisconnectAt: null,
              clockInStatus: "done",
              clockOutStatus: "missing",
              exceptionStatus: "missing_disconnect",
              sourceMode: "syslog",
              version: 2,
              lastCalculatedAt: "2026-03-21T14:10:00Z",
            },
          ],
        },
      },
    ]);

    const employees = await getEmployees();
    const attendance = await getAttendanceRecords();

    expect(employees[0]?.id).toBe("101");
    expect(attendance[0]?.id).toBe("9001");
    expect(attendance[0]?.employeeId).toBe("101");
    assertAllMatched();
  });
});
