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
  it("loads employees, allows removing draft devices, and sends create, update, and disable requests to the API", async () => {
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
              feishuEmployeeId: "fs_emp_001",
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
            feishuEmployeeId: "fs_emp_102",
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
            feishuEmployeeId: "fs_emp_102",
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
            feishuEmployeeId: "fs_emp_102",
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
            feishuEmployeeId: "fs_emp_102",
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
            feishuEmployeeId: "fs_emp_102",
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

    expect(screen.getByRole("heading", { name: "员工档案" })).toBeInTheDocument();
    expect(screen.getByRole("heading", { name: "新增员工" })).toBeInTheDocument();
    expect(screen.getByText("加载员工档案...")).toBeInTheDocument();
    expect(await screen.findByText("Lena Wu")).toBeInTheDocument();

    fireEvent.change(screen.getByLabelText("员工编号"), {
      target: { value: "E-102" },
    });
    fireEvent.change(screen.getByLabelText("系统编号"), {
      target: { value: "SYS-102" },
    });
    fireEvent.change(screen.getByLabelText("飞书员工 ID"), {
      target: { value: "fs_emp_102" },
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
    fireEvent.click(screen.getByRole("button", { name: "移除设备 2" }));
    expect(screen.queryByLabelText("设备 2 MAC")).not.toBeInTheDocument();
    fireEvent.click(screen.getByRole("button", { name: "添加设备" }));
    fireEvent.change(screen.getByLabelText("设备 2 MAC"), {
      target: { value: "AA:BB:CC:DD:EE:03" },
    });
    fireEvent.change(screen.getByLabelText("设备 2 标签"), {
      target: { value: "Packing Line" },
    });
    fireEvent.click(screen.getByRole("button", { name: "新增员工" }));

    const createdRowLabel = await screen.findByText("Chen Li");
    const createdRow = createdRowLabel.closest('[role="row"]');

    expect(createdRow).not.toBeNull();
    expect(within(createdRow as HTMLElement).getByText(/E-102/)).toBeInTheDocument();
    expect(within(createdRow as HTMLElement).getByText("fs_emp_102")).toBeInTheDocument();
    expect(within(createdRow as HTMLElement).getByText("Front Gate")).toBeInTheDocument();
    expect(within(createdRow as HTMLElement).getByText("Packing Line")).toBeInTheDocument();

    fireEvent.click(
      within(createdRow as HTMLElement).getByRole("button", { name: "编辑" }),
    );

    expect(screen.getByRole("heading", { name: "编辑员工" })).toBeInTheDocument();
    fireEvent.change(screen.getByLabelText("姓名"), {
      target: { value: "Chen Li Updated" },
    });
    fireEvent.click(screen.getByRole("button", { name: "保存变更" }));

    expect(
      await within(createdRow as HTMLElement).findByText("Chen Li Updated"),
    ).toBeInTheDocument();

    fireEvent.click(
      within(createdRow as HTMLElement).getByRole("button", { name: "停用" }),
    );

    expect(await within(createdRow as HTMLElement).findByText("已停用")).toBeInTheDocument();

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

    expect(screen.getByRole("heading", { name: "员工档案" })).toBeInTheDocument();
    expect(
      await screen.findByText("员工档案加载失败，请稍后重试"),
    ).toBeInTheDocument();
  });
});
