import "@testing-library/jest-dom/vitest";
import {
  act,
  fireEvent,
  render,
  screen,
  waitFor,
  within,
} from "@testing-library/react";
import { afterEach, describe, expect, it, vi } from "vitest";

import { LogsPage } from "../features/logs/LogsPage";
import { mockJsonFetch } from "./fetchMock";

afterEach(() => {
  vi.unstubAllGlobals();
  vi.restoreAllMocks();
});

describe("logs page", () => {
  function buildLogResponse(
    ids: number[],
    page = 1,
    totalItems = ids.length,
    totalPages = Math.max(1, Math.ceil(totalItems / 10)),
  ) {
    return {
      items: ids.map((id) => ({
        message: {
          ID: id,
          ReceivedAt: `2026-03-21T08:${String(id).padStart(2, "0")}:00Z`,
          LogTime: `2026-03-21T07:${String(id).padStart(2, "0")}:00Z`,
          ParseStatus: id % 2 === 0 ? "parsed" : "failed",
          RawMessage: `device-${id} connected`,
        },
        event: {
          ID: 100 + id,
          EventType: id % 2 === 0 ? "connect" : "disconnect",
          StationMac: `AA:BB:CC:DD:EE:${String(id).padStart(2, "0")}`,
          Hostname: `host-${id}`,
        },
      })),
      pagination: {
        page,
        pageSize: 10,
        totalItems,
        totalPages,
      },
    };
  }

  it("loads the first page, paginates, and searches on submit", async () => {
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
              name: "Wesley Wang",
              status: "active",
              devices: [
                {
                  macAddress: "AA:BB:CC:DD:EE:10",
                  deviceLabel: "Office iPad",
                  status: "active",
                },
                {
                  macAddress: "AA:BB:CC:DD:EE:12",
                  deviceLabel: "Home Device",
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
        path: "/api/logs?page=1",
        response: buildLogResponse([10, 9, 8, 7, 6, 5, 4, 3, 2, 1], 1, 12, 2),
      },
      {
        method: "GET",
        path: "/api/logs?page=2",
        response: buildLogResponse([12, 11], 2, 12, 2),
      },
      {
        method: "GET",
        path: "/api/logs?page=1&query=device-12&fromDate=2026-03-20&toDate=2026-03-21",
        response: buildLogResponse([12], 1, 1, 1),
      },
    ]);

    render(<LogsPage />);

    expect(await screen.findByText("2026-03-21 08:10:00")).toBeInTheDocument();
    expect(screen.getByText("Wesley Wang")).toBeInTheDocument();
    expect(screen.getAllByText("第 1 / 2 页")).toHaveLength(2);
    expect(screen.getAllByRole("row")).toHaveLength(10);

    fireEvent.click(
      within(screen.getAllByRole("row")[0] as HTMLElement).getByRole("button", {
        name: "详情",
      }),
    );

    const detailDialog = await screen.findByRole("dialog", { name: "日志详情" });
    expect(detailDialog).toBeInTheDocument();
    expect(within(detailDialog).getByText("员工")).toBeInTheDocument();
    expect(within(detailDialog).getByText("Wesley Wang")).toBeInTheDocument();
    expect(within(detailDialog).getByText("device-10 connected")).toBeInTheDocument();
    fireEvent.click(screen.getByRole("button", { name: "关闭详情" }));

    fireEvent.change(screen.getByLabelText("开始日期"), {
      target: { value: "2026-03-20" },
    });
    fireEvent.change(screen.getByLabelText("结束日期"), {
      target: { value: "2026-03-21" },
    });
    fireEvent.click(screen.getByRole("button", { name: "下一页" }));

    expect(await screen.findByText("2026-03-21 08:12:00")).toBeInTheDocument();
    expect(screen.getAllByText("第 2 / 2 页")).toHaveLength(2);

    fireEvent.change(screen.getByLabelText("模糊搜索"), {
      target: { value: "device-12" },
    });
    fireEvent.click(screen.getByRole("button", { name: "搜索" }));

    await waitFor(() => {
      expect(screen.getAllByText("第 1 / 1 页")).toHaveLength(2);
    });
    expect(
      within(screen.getByRole("table", { name: "日志流预览" })).getAllByRole("button", {
        name: "详情",
      }),
    ).toHaveLength(1);
    expect(screen.getByText("日期：2026-03-20 至 2026-03-21")).toBeInTheDocument();
    expect(fetchMock.mock.calls).toHaveLength(4);
    assertAllMatched();
  });

  it("polls the first page and auto-refreshes when a newer log arrives", async () => {
    vi.useFakeTimers({ shouldAdvanceTime: true });

    const { assertAllMatched } = mockJsonFetch([
      {
        method: "GET",
        path: "/api/employees",
        response: { items: [] },
      },
      {
        method: "GET",
        path: "/api/logs?page=1",
        response: buildLogResponse([10, 9, 8, 7, 6, 5, 4, 3, 2, 1], 1, 10, 1),
      },
      {
        method: "GET",
        path: "/api/logs?page=1",
        response: buildLogResponse([11, 10, 9, 8, 7, 6, 5, 4, 3, 2], 1, 11, 2),
      },
    ]);

    render(<LogsPage />);

    expect(await screen.findByText("2026-03-21 08:10:00")).toBeInTheDocument();

    await act(async () => {
      await vi.advanceTimersByTimeAsync(5000);
    });

    expect(await screen.findByText("2026-03-21 08:11:00")).toBeInTheDocument();
    assertAllMatched();
  });

  it("shows a new-log notice on later pages without replacing current rows", async () => {
    vi.useFakeTimers({ shouldAdvanceTime: true });

    const { assertAllMatched } = mockJsonFetch([
      {
        method: "GET",
        path: "/api/employees",
        response: { items: [] },
      },
      {
        method: "GET",
        path: "/api/logs?page=1",
        response: buildLogResponse([10, 9, 8, 7, 6, 5, 4, 3, 2, 1], 1, 12, 2),
      },
      {
        method: "GET",
        path: "/api/logs?page=2",
        response: buildLogResponse([12, 11], 2, 12, 2),
      },
      {
        method: "GET",
        path: "/api/logs?page=1",
        response: buildLogResponse([13, 10, 9, 8, 7, 6, 5, 4, 3, 2], 1, 13, 2),
      },
      {
        method: "GET",
        path: "/api/logs?page=1",
        response: buildLogResponse([13, 10, 9, 8, 7, 6, 5, 4, 3, 2], 1, 13, 2),
      },
    ]);

    render(<LogsPage />);

    expect(await screen.findByText("2026-03-21 08:10:00")).toBeInTheDocument();

    fireEvent.click(screen.getByRole("button", { name: "下一页" }));
    expect(await screen.findByText("2026-03-21 08:12:00")).toBeInTheDocument();

    await act(async () => {
      await vi.advanceTimersByTimeAsync(5000);
    });

    expect(screen.getByText("有新消息，返回第一页查看")).toBeInTheDocument();
    expect(
      screen.getByRole("button", { name: "返回第一页查看" }),
    ).toBeInTheDocument();
    expect(screen.getByText("2026-03-21 08:12:00")).toBeInTheDocument();
    expect(screen.queryByText("2026-03-21 08:13:00")).not.toBeInTheDocument();

    fireEvent.click(screen.getByRole("button", { name: "返回第一页查看" }));

    expect(await screen.findByText("2026-03-21 08:13:00")).toBeInTheDocument();
    assertAllMatched();
  });

  it("switches to all-received scope and shows unmatched raw inbox rows", async () => {
    const { assertAllMatched } = mockJsonFetch([
      {
        method: "GET",
        path: "/api/employees",
        response: { items: [] },
      },
      {
        method: "GET",
        path: "/api/logs?page=1",
        response: buildLogResponse([10, 9], 1, 2, 1),
      },
      {
        method: "GET",
        path: "/api/logs?page=1&scope=all",
        response: {
          items: [
            {
              message: {
                id: "501",
                receivedAt: "2026-03-21T09:00:00Z",
                parseStatus: "ignored",
                rawMessage: "unrelated syslog noise",
                sourceIp: "10.0.0.15",
                protocol: "udp",
                matchedRuleName: "",
              },
            },
          ],
          pagination: {
            page: 1,
            pageSize: 10,
            totalItems: 1,
            totalPages: 1,
          },
        },
      },
    ]);

    render(<LogsPage />);

    fireEvent.click(screen.getByRole("button", { name: "全部接收" }));

    expect(await screen.findByText("unrelated syslog noise")).toBeInTheDocument();
    expect(screen.getByText("ignored")).toBeInTheDocument();
    assertAllMatched();
  });
});
