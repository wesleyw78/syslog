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

function toFormValues(employee: Employee): EmployeeUpsertInput {
  return {
    employeeNo: employee.employeeNo,
    systemNo: employee.systemNo,
    feishuEmployeeId: employee.feishuEmployeeId,
    name: employee.name,
    status: employee.status,
    devices:
      employee.devices.length > 0
        ? employee.devices
        : [
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
  const totalDevices = useMemo(
    () =>
      employees.reduce((count, employee) => count + employee.devices.length, 0),
    [employees],
  );
  const editingEmployee = useMemo(
    () => employees.find((employee) => employee.id === editingEmployeeId) ?? null,
    [editingEmployeeId, employees],
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
        <span className="page-header__eyebrow">人员映射</span>
        <div>
          <h2>员工档案</h2>
          <p>用固定编辑区维护档案，用紧凑名册查看人员、设备和飞书映射，减少重复信息与扫描成本。</p>
        </div>
      </header>

      <div className="metric-strip">
        <article className="panel metric-card">
          <span>员工档案</span>
          <strong>{employees.length}</strong>
          <p>当前已登记的人员总数</p>
        </article>
        <article className="panel metric-card">
          <span>在岗人员</span>
          <strong>{activeCount}</strong>
          <p>状态不为 disabled 的员工</p>
        </article>
        <article className="panel metric-card">
          <span>绑定设备</span>
          <strong>{totalDevices}</strong>
          <p>已登记的设备映射总数</p>
        </article>
        <article className="panel metric-card">
          <span>当前模式</span>
          <strong>{editingEmployee ? "编辑中" : "新建中"}</strong>
          <p>{editingEmployee ? editingEmployee.name : "准备新增员工档案"}</p>
        </article>
      </div>

      <div className="page-grid page-grid--split">
        <article className="panel employee-editor">
          <div className="panel__header">
            <h3>{editingEmployee ? "编辑员工" : "新增员工"}</h3>
            <span>{isLoading ? "加载中..." : `${activeCount} 名在岗`}</span>
          </div>
          <p className="panel__copy">{notice}</p>
          {editingEmployee ? (
            <button
              type="button"
              className="button button--ghost button--small"
              onClick={() => setEditingEmployeeId(null)}
            >
              返回新增
            </button>
          ) : null}
          <div className="employee-editor__form">
            <EmployeeForm
              key={editingEmployee?.id ?? "create"}
              initialValues={editingEmployee ? toFormValues(editingEmployee) : undefined}
              resetOnSubmit={!editingEmployee}
              isSubmitting={
                editingEmployee ? pendingEmployeeId === editingEmployee.id : isSubmitting
              }
              onCancel={editingEmployee ? () => setEditingEmployeeId(null) : undefined}
              onSubmit={(input) =>
                editingEmployee
                  ? handleUpdateEmployee(editingEmployee.id, input)
                  : handleCreateEmployee(input)
              }
              submitLabel={editingEmployee ? "保存变更" : "新增员工"}
            />
          </div>
        </article>

        <article className="panel panel--tall">
          <div className="panel__header">
            <h3>人员名册</h3>
            <span>{employees.length} 条档案</span>
          </div>
          <div className="employee-roster" role="table" aria-label="员工名册">
            <div className="employee-roster__head" role="row">
              <span>员工</span>
              <span>编号</span>
              <span>飞书</span>
              <span>设备</span>
              <span>状态</span>
              <span>动作</span>
            </div>
            {employees.map((employee) => (
              <div key={employee.id} className="employee-roster__row" role="row">
                <div className="employee-roster__identity" role="cell">
                  <strong>{employee.name}</strong>
                  <span>{employee.systemNo}</span>
                </div>
                <span role="cell">{employee.employeeNo}</span>
                <span role="cell">{employee.feishuEmployeeId || "未配置"}</span>
                <div role="cell" className="employee-device-tags">
                  {employee.devices.length > 0 ? (
                    employee.devices.map((device) => (
                      <span key={`${employee.id}-${device.macAddress}`} className="employee-device-tag">
                        {device.deviceLabel || device.macAddress}
                      </span>
                    ))
                  ) : (
                    <span className="employee-device-tag">无设备</span>
                  )}
                </div>
                <span role="cell">{employee.status === "disabled" ? "已停用" : "启用中"}</span>
                <div role="cell" className="employee-roster__actions">
                  <button
                    type="button"
                    onClick={() => setEditingEmployeeId(employee.id)}
                    disabled={
                      pendingEmployeeId === employee.id || employee.status === "disabled"
                    }
                    className="button button--ghost button--small"
                  >
                    编辑
                  </button>
                  <button
                    type="button"
                    onClick={() => void handleDisableEmployee(employee.id)}
                    disabled={
                      pendingEmployeeId === employee.id || employee.status === "disabled"
                    }
                    className="button button--danger button--small"
                  >
                    停用
                  </button>
                </div>
              </div>
            ))}
          </div>
        </article>
      </div>
    </section>
  );
}
