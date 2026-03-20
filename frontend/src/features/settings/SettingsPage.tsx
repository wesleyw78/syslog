const settingsModules = [
  "Scanner retry thresholds",
  "Supervisor approval policy",
  "Archive retention windows",
];

export function SettingsPage() {
  return (
    <section className="page">
      <header className="page-header">
        <span className="page-header__eyebrow">Configuration</span>
        <div>
          <h2>Settings</h2>
          <p>
            Placeholder admin space for runtime controls, policy toggles, and
            audit-aware changes.
          </p>
        </div>
      </header>

      <div className="page-grid page-grid--split">
        <article className="panel">
          <div className="panel__header">
            <h3>Control Modules</h3>
            <span>Staged inputs</span>
          </div>
          <ul className="stack-list">
            {settingsModules.map((item) => (
              <li key={item}>{item}</li>
            ))}
          </ul>
        </article>

        <article className="panel panel--tall">
          <div className="panel__header">
            <h3>Audit Locks</h3>
            <span>Immutable trail</span>
          </div>
          <p className="panel__copy">
            This area will host change approval summaries, release toggles, and
            protected configuration history.
          </p>
          <div className="signal-bars" aria-hidden="true">
            <span style={{ width: "42%" }} />
            <span style={{ width: "71%" }} />
            <span style={{ width: "58%" }} />
          </div>
        </article>
      </div>
    </section>
  );
}
