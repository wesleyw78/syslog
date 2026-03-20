const employeeCards = [
  { name: "Lena Wu", team: "Assembly", badge: "A-447", status: "On shift" },
  { name: "Arjun Patel", team: "Security", badge: "S-118", status: "Briefing" },
  { name: "Mina Torres", team: "Maintenance", badge: "M-233", status: "Standby" },
];

export function EmployeesPage() {
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
        <article className="panel panel--tall">
          <div className="panel__header">
            <h3>Roster Snapshot</h3>
            <span>Current floor leaders</span>
          </div>
          <div className="employee-grid">
            {employeeCards.map((employee) => (
              <article key={employee.badge} className="employee-card">
                <strong>{employee.name}</strong>
                <span>{employee.team}</span>
                <span>{employee.badge}</span>
                <p>{employee.status}</p>
              </article>
            ))}
          </div>
        </article>

        <article className="panel">
          <div className="panel__header">
            <h3>Certification Queue</h3>
            <span>Expiring soon</span>
          </div>
          <ul className="stack-list">
            <li>Forklift recertification: 6 staff in next 14 days</li>
            <li>Fire drill acknowledgement: 2 pending supervisors</li>
            <li>Restricted zone briefing: 1 contractor awaiting approval</li>
          </ul>
        </article>
      </div>
    </section>
  );
}
