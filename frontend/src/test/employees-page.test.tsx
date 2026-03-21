import "@testing-library/jest-dom/vitest";
import { fireEvent, render, screen, within } from "@testing-library/react";
import { afterEach, describe, expect, it, vi } from "vitest";

import { EmployeesPage } from "../features/employees/EmployeesPage";
import { mockJsonFetch } from "./fetchMock";

afterEach(() => {
  vi.unstubAllGlobals();
  vi.restoreAllMocks();
});

describe("employees page", () => {
  it("loads employees and sends create, update, and disable requests to the API", async () => {
    const { fetchMock } = mockJsonFetch([
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
        method: "POST",
        path: "/api/employees",
        response: {
          employee: {
            id: "emp-2",
            employeeNo: "E-102",
            systemNo: "SYS-102",
            name: "Chen Li",
            status: "active",
            devices: [
              {
                macAddress: "AA:BB:CC:DD:EE:02",
                deviceLabel: "Front Gate",
                status: "active",
              },
              {
                macAddress: "AA:BB:CC:DD:EE:03",
                deviceLabel: "Packing Line",
                status: "active",
              },
            ],
            createdAt: "2026-03-21T00:00:00Z",
            updatedAt: "2026-03-21T00:00:00Z",
          },
        },
        assertBody: (body) => {
          expect(body).toEqual({
            employeeNo: "E-102",
            systemNo: "SYS-102",
            name: "Chen Li",
            status: "active",
            devices: [
              {
                macAddress: "AA:BB:CC:DD:EE:02",
                deviceLabel: "Front Gate",
                status: "active",
              },
              {
                macAddress: "AA:BB:CC:DD:EE:03",
                deviceLabel: "Packing Line",
                status: "active",
              },
            ],
          });
        },
      },
      {
        method: "PUT",
        path: "/api/employees/emp-2",
        response: {
          employee: {
            id: "emp-2",
            employeeNo: "E-102",
            systemNo: "SYS-102",
            name: "Chen Li Updated",
            status: "active",
            devices: [
              {
                macAddress: "AA:BB:CC:DD:EE:02",
                deviceLabel: "Front Gate",
                status: "active",
              },
              {
                macAddress: "AA:BB:CC:DD:EE:03",
                deviceLabel: "Packing Line",
                status: "active",
              },
            ],
            createdAt: "2026-03-21T00:00:00Z",
            updatedAt: "2026-03-21T00:05:00Z",
          },
        },
        assertBody: (body) => {
          expect(body).toEqual({
            employeeNo: "E-102",
            systemNo: "SYS-102",
            name: "Chen Li Updated",
            status: "active",
            devices: [
              {
                macAddress: "AA:BB:CC:DD:EE:02",
                deviceLabel: "Front Gate",
                status: "active",
              },
              {
                macAddress: "AA:BB:CC:DD:EE:03",
                deviceLabel: "Packing Line",
                status: "active",
              },
            ],
          });
        },
      },
      {
        method: "POST",
        path: "/api/employees/emp-2/disable",
        response: {
          employee: {
            id: "emp-2",
            employeeNo: "E-102",
            systemNo: "SYS-102",
            name: "Chen Li Updated",
            status: "disabled",
            devices: [
              {
                macAddress: "AA:BB:CC:DD:EE:02",
                deviceLabel: "Front Gate",
                status: "active",
              },
              {
                macAddress: "AA:BB:CC:DD:EE:03",
                deviceLabel: "Packing Line",
                status: "active",
              },
            ],
            createdAt: "2026-03-21T00:00:00Z",
            updatedAt: "2026-03-21T00:10:00Z",
          },
        },
      },
    ]);

    render(<EmployeesPage />);

    expect(screen.getByText("加载员工档案...")).toBeInTheDocument();
    expect(await screen.findByText("Lena Wu")).toBeInTheDocument();

    fireEvent.change(screen.getByLabelText("员工编号"), {
      target: { value: "E-102" },
    });
    fireEvent.change(screen.getByLabelText("系统编号"), {
      target: { value: "SYS-102" },
    });
    fireEvent.change(screen.getByLabelText("姓名"), {
      target: { value: "Chen Li" },
    });
    fireEvent.change(screen.getByLabelText("设备 1 MAC"), {
      target: { value: "AA:BB:CC:DD:EE:02" },
    });
    fireEvent.change(screen.getByLabelText("设备 1 标签"), {
      target: { value: "Front Gate" },
    });
    fireEvent.click(screen.getByRole("button", { name: "添加设备" }));
    fireEvent.change(screen.getByLabelText("设备 2 MAC"), {
      target: { value: "AA:BB:CC:DD:EE:03" },
    });
    fireEvent.change(screen.getByLabelText("设备 2 标签"), {
      target: { value: "Packing Line" },
    });
    fireEvent.click(screen.getByRole("button", { name: "新增员工" }));

    const createdCard = await screen.findByText("Chen Li");
    const createdArticle = createdCard.closest("article");

    expect(createdArticle).not.toBeNull();
    expect(within(createdArticle as HTMLElement).getByText(/E-102/)).toBeInTheDocument();
    expect(within(createdArticle as HTMLElement).getByText("2 台设备")).toBeInTheDocument();

    fireEvent.click(
      within(createdArticle as HTMLElement).getByRole("button", { name: "编辑" }),
    );

    fireEvent.change(within(createdArticle as HTMLElement).getByLabelText("姓名"), {
      target: { value: "Chen Li Updated" },
    });
    fireEvent.click(
      within(createdArticle as HTMLElement).getByRole("button", { name: "保存变更" }),
    );

    expect(
      await within(createdArticle as HTMLElement).findByText("Chen Li Updated"),
    ).toBeInTheDocument();

    fireEvent.click(
      within(createdArticle as HTMLElement).getByRole("button", { name: "停用" }),
    );

    expect(await within(createdArticle as HTMLElement).findByText("已停用")).toBeInTheDocument();

    expect(fetchMock).toHaveBeenCalledWith("/api/employees", expect.any(Object));
    expect(fetchMock).toHaveBeenCalledWith(
      "/api/employees",
      expect.objectContaining({ method: "POST" }),
    );
    expect(fetchMock).toHaveBeenCalledWith(
      "/api/employees/emp-2",
      expect.objectContaining({ method: "PUT" }),
    );
    expect(fetchMock).toHaveBeenCalledWith(
      "/api/employees/emp-2/disable",
      expect.objectContaining({ method: "POST" }),
    );

    expect(fetchMock.mock.calls).toHaveLength(4);
  });

  it("shows an error when the employees endpoint fails", async () => {
    mockJsonFetch([
      {
        method: "GET",
        path: "/api/employees",
        response: { message: "boom" },
        status: 500,
      },
    ]);

    render(<EmployeesPage />);

    expect(
      await screen.findByText("员工档案加载失败，请稍后重试"),
    ).toBeInTheDocument();
  });
});
