const criticalMetrics = [
  { label: "Workers Onsite", value: "148", status: "Shift A settled" },
  { label: "Event Throughput", value: "31.2k", status: "Last 60 minutes" },
  { label: "Open Exceptions", value: "04", status: "2 require review" },
];

const watchList = [
  "East dock camera handshake latency trending up",
  "Badge printer 02 entered maintenance bypass mode",
  "Attendance reconciliation window scheduled for 04:30",
];

export function DashboardPage() {
  return (
    <section className="page">
      <header className="page-header">
        <span className="page-header__eyebrow">Overview</span>
        <div>
          <h2>Dashboard</h2>
          <p>
            Live operating picture for attendance, logging, and workforce
            readiness.
          </p>
        </div>
      </header>

      <div className="metric-strip">
        {criticalMetrics.map((metric) => (
          <article key={metric.label} className="panel metric-card">
            <span>{metric.label}</span>
            <strong>{metric.value}</strong>
            <p>{metric.status}</p>
          </article>
        ))}
      </div>

      <div className="page-grid page-grid--dashboard">
        <article className="panel panel--tall">
          <div className="panel__header">
            <h3>Incident Watch</h3>
            <span>Next audit sweep 11m</span>
          </div>
          <ul className="stack-list">
            {watchList.map((item) => (
              <li key={item}>{item}</li>
            ))}
          </ul>
        </article>

        <article className="panel">
          <div className="panel__header">
            <h3>Shift Stability</h3>
            <span>Zone spread</span>
          </div>
          <div className="signal-bars" aria-hidden="true">
            <span style={{ width: "88%" }} />
            <span style={{ width: "62%" }} />
            <span style={{ width: "74%" }} />
            <span style={{ width: "91%" }} />
          </div>
          <p className="panel__copy">
            Control-room summary space reserved for upcoming staffing and
            anomaly widgets.
          </p>
        </article>
      </div>
    </section>
  );
}
