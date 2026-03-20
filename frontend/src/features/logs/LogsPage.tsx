const logRows = [
  {
    channel: "AUTH_GATE",
    status: "Accepted",
    time: "02:14:22",
    message: "Badge 2117 verified for Line 4 access corridor",
  },
  {
    channel: "INGEST_PIPE",
    status: "Warning",
    time: "02:12:08",
    message: "Batch delay crossed 2200ms threshold on archive fan-out",
  },
  {
    channel: "SHIFT_SYNC",
    status: "Queued",
    time: "02:09:41",
    message: "Attendance consolidation awaiting supervisor signature",
  },
];

export function LogsPage() {
  return (
    <section className="page">
      <header className="page-header">
        <span className="page-header__eyebrow">Telemetry</span>
        <div>
          <h2>Logs</h2>
          <p>
            Operator-facing event stream placeholder with room for filters,
            severity pivots, and tail inspection.
          </p>
        </div>
      </header>

      <div className="page-grid page-grid--logs">
        <article className="panel">
          <div className="panel__header">
            <h3>Filter Deck</h3>
            <span>Pending integration</span>
          </div>
          <div className="filter-row">
            <span>Source: All Pipelines</span>
            <span>Severity: Warning+</span>
            <span>Window: Last 15m</span>
          </div>
        </article>

        <article className="panel panel--tall">
          <div className="panel__header">
            <h3>Stream Snapshot</h3>
            <span>3 latest records</span>
          </div>
          <div className="log-table" role="table" aria-label="Log stream preview">
            {logRows.map((row) => (
              <div key={`${row.channel}-${row.time}`} className="log-row" role="row">
                <span role="cell">{row.time}</span>
                <span role="cell">{row.channel}</span>
                <strong role="cell">{row.status}</strong>
                <span role="cell">{row.message}</span>
              </div>
            ))}
          </div>
        </article>
      </div>
    </section>
  );
}
