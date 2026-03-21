import { useEffect, useMemo, useState } from "react";

import { getLogs, type LogItem } from "../../lib/api";

function formatTime(value?: string): string {
  if (!value) {
    return "-";
  }

  const parsedDate = new Date(value);
  if (Number.isNaN(parsedDate.getTime())) {
    return value;
  }

  return parsedDate.toLocaleTimeString("en-GB", { hour12: false });
}

function buildLogSummary(item: LogItem): string {
  const message = item.message;
  const eventType = item.event?.eventType ?? "-";
  const stationMac = item.event?.stationMac ?? "-";
  const hostname = item.event?.hostname ?? "-";
  const rawMessage = message.rawMessage ?? "-";

  return `${eventType} · ${stationMac} · ${hostname} · ${rawMessage}`;
}

export function LogsPage() {
  const [logs, setLogs] = useState<LogItem[]>([]);
  const [isLoading, setIsLoading] = useState(true);
  const [errorMessage, setErrorMessage] = useState("");

  useEffect(() => {
    let isActive = true;

    void (async () => {
      try {
        const items = await getLogs();

        if (!isActive) {
          return;
        }

        setLogs(items);
      } catch {
        if (isActive) {
          setErrorMessage("日志加载失败，请稍后重试");
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

  const logRows = useMemo(
    () =>
      logs.map((item) => ({
        key: buildLogSummary(item),
        time: formatTime(item.message.logTime ?? item.message.receivedAt),
        parseStatus: item.message.parseStatus ?? "unknown",
        eventType: item.event?.eventType ?? "-",
        stationMac: item.event?.stationMac ?? "-",
        hostname: item.event?.hostname ?? "-",
        rawMessage: item.message.rawMessage ?? "-",
      })),
    [logs],
  );

  return (
    <section className="page">
      <header className="page-header">
        <span className="page-header__eyebrow">Telemetry</span>
        <div>
          <h2>Logs</h2>
          <p>真实日志流，展示 message 与 event 的关键信息。</p>
        </div>
      </header>

      <div className="page-grid page-grid--logs">
        <article className="panel">
          <div className="panel__header">
            <h3>过滤条件</h3>
            <span>{isLoading ? "加载中..." : `${logRows.length} 条记录`}</span>
          </div>
          <div className="filter-row">
            <span>来源：全部</span>
            <span>解析状态：全部</span>
            <span>窗口：最近 15m</span>
          </div>
          {errorMessage ? <p className="panel__copy">{errorMessage}</p> : null}
        </article>

        <article className="panel panel--tall">
          <div className="panel__header">
            <h3>日志明细</h3>
            <span>{logRows.length > 0 ? `${logRows.length} 条最新记录` : "暂无记录"}</span>
          </div>
          <div className="log-table" role="table" aria-label="Log stream preview">
            <div
              style={{
                display: "grid",
                gridTemplateColumns: "90px 110px 110px 110px 110px minmax(0, 1fr)",
                gap: "0.65rem",
                color: "#8a928d",
                fontSize: "0.78rem",
                textTransform: "uppercase",
                letterSpacing: "0.08em",
                marginBottom: "0.65rem",
              }}
            >
              <span>时间</span>
              <span>解析状态</span>
              <span>事件类型</span>
              <span>站点 MAC</span>
              <span>主机</span>
              <span>原始消息</span>
            </div>
            {logRows.length > 0 ? (
              logRows.map((row) => (
                <div
                  key={row.key}
                  className="log-row"
                  role="row"
                  style={{
                    display: "grid",
                    gridTemplateColumns: "90px 110px 110px 110px 110px minmax(0, 1fr)",
                    gap: "0.65rem",
                  }}
                >
                  <span role="cell">{row.time}</span>
                  <span role="cell">{row.parseStatus}</span>
                  <strong role="cell">{row.eventType}</strong>
                  <span role="cell">{row.stationMac}</span>
                  <span role="cell">{row.hostname}</span>
                  <span role="cell">{row.rawMessage}</span>
                </div>
              ))
            ) : (
              <p className="panel__copy">{isLoading ? "日志加载中..." : "暂无日志"}</p>
            )}
          </div>
        </article>
      </div>
    </section>
  );
}
