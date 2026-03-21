import { useEffect, useMemo, useState } from "react";

import {
  createEmployee,
  disableEmployee,
  getEmployees,
  type Employee,
  type EmployeeUpsertInput,
  updateEmployee,
} from "../../lib/api";
import { EmployeeForm } from "./components/EmployeeForm";

const actionButtonStyle = {
  padding: "0.65rem 0.8rem",
  border: "1px solid rgba(255, 184, 77, 0.24)",
  background: "rgba(19, 15, 7, 0.92)",
  color: "inherit",
  cursor: "pointer",
};

function toFormValues(employee: Employee): EmployeeUpsertInput {
  return {
    employeeNo: employee.employeeNo,
    systemNo: employee.systemNo,
    name: employee.name,
    status: employee.status,
    devices: employee.devices.length > 0 ? employee.devices : [
      {
        macAddress: "",
        deviceLabel: "",
        status: "active",
      },
    ],
  };
}

export function EmployeesPage() {
  const [employees, setEmployees] = useState<Employee[]>([]);
  const [isLoading, setIsLoading] = useState(true);
  const [isSubmitting, setIsSubmitting] = useState(false);
  const [editingEmployeeId, setEditingEmployeeId] = useState<string | null>(null);
  const [pendingEmployeeId, setPendingEmployeeId] = useState<string | null>(null);
  const [notice, setNotice] = useState("加载员工档案...");

  useEffect(() => {
    let isActive = true;

    void (async () => {
      try {
        const items = await getEmployees();

        if (!isActive) {
          return;
        }

        setEmployees(items);
        setNotice(`已接入 ${items.length} 条员工档案`);
      } catch {
        if (isActive) {
          setNotice("员工档案加载失败，请稍后重试");
        }
      } finally {
        if (isActive) {
          setIsLoading(false);
        }
      }
    })();

    return () => {
      isActive = false;
    };
  }, []);

  const activeCount = useMemo(
    () => employees.filter((employee) => employee.status !== "disabled").length,
    [employees],
  );

  async function handleCreateEmployee(input: EmployeeUpsertInput) {
    setIsSubmitting(true);

    try {
      const createdEmployee = await createEmployee(input);
      setEmployees((current) => [createdEmployee, ...current]);
      setNotice(`已新增员工 ${createdEmployee.name}`);
    } catch {
      setNotice("新增员工失败，请检查输入后重试");
    } finally {
      setIsSubmitting(false);
    }
  }

  async function handleUpdateEmployee(employeeId: string, input: EmployeeUpsertInput) {
    setPendingEmployeeId(employeeId);

    try {
      const updatedEmployee = await updateEmployee(employeeId, input);
      setEmployees((current) =>
        current.map((employee) =>
          employee.id === updatedEmployee.id ? updatedEmployee : employee,
        ),
      );
      setEditingEmployeeId(null);
      setNotice(`已更新员工 ${updatedEmployee.name}`);
    } catch {
      setNotice("员工信息更新失败，请稍后重试");
    } finally {
      setPendingEmployeeId(null);
    }
  }

  async function handleDisableEmployee(employeeId: string) {
    setPendingEmployeeId(employeeId);

    try {
      const disabledEmployee = await disableEmployee(employeeId);
      setEmployees((current) =>
        current.map((employee) =>
          employee.id === disabledEmployee.id ? disabledEmployee : employee,
        ),
      );
      if (editingEmployeeId === employeeId) {
        setEditingEmployeeId(null);
      }
      setNotice(`已停用员工 ${disabledEmployee.name}`);
    } catch {
      setNotice("员工停用失败，请稍后重试");
    } finally {
      setPendingEmployeeId(null);
    }
  }

  return (
    <section className="page">
      <header className="page-header">
        <span className="page-header__eyebrow">Workforce</span>
        <div>
          <h2>Employees</h2>
          <p>真实员工档案管理，支持新增、编辑、停用和设备信息维护。</p>
        </div>
      </header>

      <div className="page-grid page-grid--split">
        <article className="panel">
          <div className="panel__header">
            <h3>员工录入</h3>
            <span>{isLoading ? "加载中..." : `${activeCount} active badges`}</span>
          </div>
          <p className="panel__copy">{notice}</p>
          <div style={{ marginTop: "1rem" }}>
            <EmployeeForm
              resetOnSubmit
              isSubmitting={isSubmitting}
              onSubmit={handleCreateEmployee}
            />
          </div>
        </article>

        <article className="panel panel--tall">
          <div className="panel__header">
            <h3>员工列表</h3>
            <span>{employees.length} records</span>
          </div>
          <div className="employee-grid">
            {employees.map((employee) => (
              <article key={employee.id} className="employee-card">
                <strong>{employee.name}</strong>
                <span>
                  {employee.employeeNo} / {employee.systemNo}
                </span>
                <span>{employee.status}</span>
                <p>{`${employee.devices.length} 台设备`}</p>
                <p>
                  {employee.devices
                    .map((device) => `${device.deviceLabel || device.macAddress}`)
                    .join(" / ")}
                </p>
                {editingEmployeeId === employee.id ? (
                  <EmployeeForm
                    initialValues={toFormValues(employee)}
                    isSubmitting={pendingEmployeeId === employee.id}
                    onCancel={() => setEditingEmployeeId(null)}
                    onSubmit={(input) => handleUpdateEmployee(employee.id, input)}
                    submitLabel="保存变更"
                  />
                ) : (
                  <div className="filter-row">
                    <button
                      type="button"
                      onClick={() => setEditingEmployeeId(employee.id)}
                      disabled={
                        pendingEmployeeId === employee.id || employee.status === "disabled"
                      }
                      style={actionButtonStyle}
                    >
                      编辑
                    </button>
                    <button
                      type="button"
                      onClick={() => void handleDisableEmployee(employee.id)}
                      disabled={
                        pendingEmployeeId === employee.id || employee.status === "disabled"
                      }
                      style={{
                        ...actionButtonStyle,
                        border: "1px solid rgba(116, 216, 169, 0.24)",
                        background: "rgba(11, 22, 18, 0.92)",
                      }}
                    >
                      {employee.status === "disabled" ? "已停用" : "停用"}
                    </button>
                  </div>
                )}
              </article>
            ))}
          </div>
        </article>
      </div>
    </section>
  );
}
