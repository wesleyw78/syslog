const attendanceBands = [
  { label: "On Time", value: "92%" },
  { label: "Late", value: "05%" },
  { label: "Unreconciled", value: "03%" },
];

export function AttendancePage() {
  return (
    <section className="page">
      <header className="page-header">
        <span className="page-header__eyebrow">Shift Control</span>
        <div>
          <h2>Attendance</h2>
          <p>
            Placeholder coverage page for punch events, exceptions, and shift
            reconciliation.
          </p>
        </div>
      </header>

      <div className="page-grid page-grid--split">
        <article className="panel">
          <div className="panel__header">
            <h3>Coverage Bands</h3>
            <span>Current shift</span>
          </div>
          <div className="coverage-list">
            {attendanceBands.map((band) => (
              <div key={band.label} className="coverage-row">
                <span>{band.label}</span>
                <strong>{band.value}</strong>
              </div>
            ))}
          </div>
        </article>

        <article className="panel panel--tall">
          <div className="panel__header">
            <h3>Checkpoint Queue</h3>
            <span>Supervisor review</span>
          </div>
          <ul className="stack-list">
            <li>North gate missed punch review for Team C</li>
            <li>Visitor escort log awaiting sign-off</li>
            <li>Overtime carryover for packing line flagged for approval</li>
          </ul>
        </article>
      </div>
    </section>
  );
}
