import { useEffect, useState } from "react";

import {
  createEmployee,
  listEmployees,
  type Employee,
  type EmployeeDraft,
} from "../../lib/api";
import { EmployeeForm } from "./components/EmployeeForm";

export function EmployeesPage() {
  const [employees, setEmployees] = useState<Employee[]>([]);
  const [isLoading, setIsLoading] = useState(true);
  const [isSubmitting, setIsSubmitting] = useState(false);
  const [notice, setNotice] = useState("等待 roster feed...");

  useEffect(() => {
    let isActive = true;

    void (async () => {
      try {
        const items = await listEmployees();

        if (!isActive) {
          return;
        }

        setEmployees(items);
        setNotice(`已接入 ${items.length} 条员工档案`);
      } catch {
        if (!isActive) {
          return;
        }

        setNotice("员工档案装载失败，请稍后重试");
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

  async function handleCreateEmployee(draft: EmployeeDraft) {
    setIsSubmitting(true);

    try {
      const createdEmployee = await createEmployee(draft);
      setEmployees((current) => [createdEmployee, ...current]);
      setNotice(`已新增员工 ${createdEmployee.name}，待后续接入编辑/停用流程`);
    } catch {
      setNotice("新增员工失败，请检查输入后重试");
    } finally {
      setIsSubmitting(false);
    }
  }

  return (
    <section className="page">
      <header className="page-header">
        <span className="page-header__eyebrow">Workforce</span>
        <div>
          <h2>Employees</h2>
          <p>
            Placeholder roster view for headcount, certifications, and team
            state changes.
          </p>
        </div>
      </header>

      <div className="page-grid page-grid--split">
        <article className="panel">
          <div className="panel__header">
            <h3>Roster Intake</h3>
            <span>Mock create</span>
          </div>
          <p className="panel__copy">
            {notice}
          </p>
          <div style={{ marginTop: "1rem" }}>
            <EmployeeForm
              isSubmitting={isSubmitting}
              onSubmit={handleCreateEmployee}
            />
          </div>
        </article>

        <article className="panel panel--tall">
          <div className="panel__header">
            <h3>Roster Snapshot</h3>
            <span>{isLoading ? "Loading..." : `${employees.length} active badges`}</span>
          </div>
          <div className="employee-grid">
            {employees.map((employee) => (
              <article key={employee.badge} className="employee-card">
                <strong>{employee.name}</strong>
                <span>{employee.team}</span>
                <span>{employee.badge}</span>
                <p>{employee.status}</p>
                <div className="filter-row">
                  <span>编辑待接入</span>
                  <span>停用待接入</span>
                </div>
              </article>
            ))}
          </div>
        </article>
      </div>
    </section>
  );
}
