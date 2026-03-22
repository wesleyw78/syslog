import { type FormEvent, useEffect, useMemo, useState } from "react";

import {
  getEmployees,
  getLogs,
  type LogItem,
  type PaginationMeta,
} from "../../lib/api";

const pollIntervalMs = 5000;
const emptyPagination: PaginationMeta = {
  page: 1,
  pageSize: 10,
  totalItems: 0,
  totalPages: 0,
};

type LogRow = {
  employeeName: string;
  eventType: string;
  item: LogItem;
  key: string;
  matchedRuleName: string;
  parseStatus: string;
  receivedAt: string;
  sourceIp: string;
  stationMac: string;
};

function formatReceivedAt(value?: string): string {
  if (!value) {
    return "-";
  }

  const normalizedValue = value.trim();
  if (/^\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}/.test(normalizedValue)) {
    return normalizedValue.slice(0, 19).replace("T", " ");
  }

  const parsedDate = new Date(normalizedValue);
  if (Number.isNaN(parsedDate.getTime())) {
    return normalizedValue;
  }

  return [
    parsedDate.getFullYear(),
    String(parsedDate.getMonth() + 1).padStart(2, "0"),
    String(parsedDate.getDate()).padStart(2, "0"),
  ].join("-") +
    ` ${String(parsedDate.getHours()).padStart(2, "0")}:${String(parsedDate.getMinutes()).padStart(2, "0")}:${String(parsedDate.getSeconds()).padStart(2, "0")}`;
}

function logMessageId(item?: LogItem): string {
  const value = item?.message.id;
  return typeof value === "string" && value !== "" ? value : "";
}

function normalizeMac(value?: string): string {
  return (value ?? "").trim().toLowerCase();
}

function buildLogRow(
  item: LogItem,
  employeeNamesByMac: Map<string, string>,
): LogRow {
  const stationMac = item.event?.stationMac ?? "-";
  const normalizedStationMac = normalizeMac(stationMac);

  return {
    key:
      logMessageId(item) ||
      `${item.message.receivedAt ?? ""}-${item.message.rawMessage ?? ""}`,
    employeeName:
      normalizedStationMac === ""
        ? "-"
        : (employeeNamesByMac.get(normalizedStationMac) ?? "未匹配员工"),
    item,
    matchedRuleName: item.message.matchedRuleName ?? "-",
    receivedAt: formatReceivedAt(item.message.receivedAt),
    parseStatus: item.message.parseStatus ?? "unknown",
    sourceIp: item.message.sourceIp ?? "-",
    eventType: item.event?.eventType ?? "-",
    stationMac,
  };
}

export function LogsPage() {
  const [scope, setScope] = useState<"matched" | "all">("matched");
  const [logs, setLogs] = useState<LogItem[]>([]);
  const [employeeNamesByMac, setEmployeeNamesByMac] = useState<Map<string, string>>(
    () => new Map(),
  );
  const [pagination, setPagination] = useState<PaginationMeta>(emptyPagination);
  const [currentPage, setCurrentPage] = useState(1);
  const [inputQuery, setInputQuery] = useState("");
  const [inputFromDate, setInputFromDate] = useState("");
  const [inputToDate, setInputToDate] = useState("");
  const [submittedQuery, setSubmittedQuery] = useState("");
  const [submittedFromDate, setSubmittedFromDate] = useState("");
  const [submittedToDate, setSubmittedToDate] = useState("");
  const [firstPageLatestId, setFirstPageLatestId] = useState("");
  const [hasNewerResults, setHasNewerResults] = useState(false);
  const [isLoading, setIsLoading] = useState(true);
  const [errorMessage, setErrorMessage] = useState("");
  const [selectedLog, setSelectedLog] = useState<LogRow | null>(null);

  useEffect(() => {
    let isActive = true;

    void (async () => {
      try {
        const employees = await getEmployees();

        if (!isActive) {
          return;
        }

        const nextMap = new Map<string, string>();
        employees.forEach((employee) => {
          employee.devices.forEach((device) => {
            const normalizedMac = normalizeMac(device.macAddress);
            if (normalizedMac !== "" && !nextMap.has(normalizedMac)) {
              nextMap.set(normalizedMac, employee.name);
            }
          });
        });
        setEmployeeNamesByMac(nextMap);
      } catch {
        if (isActive) {
          setEmployeeNamesByMac(new Map());
        }
      }
    })();

    return () => {
      isActive = false;
    };
  }, []);

  useEffect(() => {
    let isActive = true;
    setIsLoading(true);

    void (async () => {
      try {
        const response = await getLogs({
          fromDate: submittedFromDate,
          page: currentPage,
          query: submittedQuery,
          scope: scope === "all" ? "all" : undefined,
          toDate: submittedToDate,
        });

        if (!isActive) {
          return;
        }

        setLogs(response.items);
        setPagination(response.pagination);
        setErrorMessage("");
        setSelectedLog(null);

        if (response.pagination.page !== currentPage) {
          setCurrentPage(response.pagination.page);
        }

        if (response.pagination.page === 1) {
          setFirstPageLatestId(logMessageId(response.items[0]));
          setHasNewerResults(false);
        }
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
  }, [currentPage, scope, submittedFromDate, submittedQuery, submittedToDate]);

  useEffect(() => {
    const timer = window.setInterval(() => {
      void (async () => {
        try {
          const response = await getLogs({
            fromDate: submittedFromDate,
            page: 1,
            query: submittedQuery,
            scope: scope === "all" ? "all" : undefined,
            toDate: submittedToDate,
          });
          const latestId = logMessageId(response.items[0]);

          setFirstPageLatestId((previousId) => {
            const baselineId = previousId || latestId;

            if (latestId === "" || latestId === baselineId) {
              return baselineId;
            }

            if (currentPage === 1) {
              setLogs(response.items);
              setPagination(response.pagination);
              setHasNewerResults(false);
              setSelectedLog(null);
            } else {
              setHasNewerResults(true);
            }

            return latestId;
          });
        } catch {
          // Ignore polling failures and keep the current page stable.
        }
      })();
    }, pollIntervalMs);

    return () => {
      window.clearInterval(timer);
    };
  }, [currentPage, scope, submittedFromDate, submittedQuery, submittedToDate]);

  const logRows = useMemo(
    () => logs.map((item) => buildLogRow(item, employeeNamesByMac)),
    [employeeNamesByMac, logs],
  );
  const displayTotalPages = Math.max(1, pagination.totalPages);
  const logSummary = useMemo(
    () => [
      {
        label: "当前页记录",
        value: `${logRows.length}`,
      },
      {
        label: "解析失败",
        value: `${logRows.filter((row) => row.parseStatus !== "parsed").length}`,
      },
      {
        label: scope === "matched" ? "已匹配员工" : "命中规则",
        value:
          scope === "matched"
            ? `${logRows.filter((row) => row.employeeName !== "未匹配员工" && row.employeeName !== "-").length}`
            : `${logRows.filter((row) => row.parseStatus === "parsed").length}`,
      },
      {
        label: "筛选范围",
        value:
          submittedFromDate || submittedToDate
            ? `${submittedFromDate || "不限"} 至 ${submittedToDate || "不限"}`
            : "全部日期",
      },
    ],
    [logRows, scope, submittedFromDate, submittedToDate],
  );

  function handleSearchSubmit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    setHasNewerResults(false);
    setSubmittedFromDate(inputFromDate.trim());
    setSubmittedQuery(inputQuery.trim());
    setSubmittedToDate(inputToDate.trim());
    setCurrentPage(1);
  }

  function handleScopeChange(nextScope: "matched" | "all") {
    setScope(nextScope);
    setHasNewerResults(false);
    setCurrentPage(1);
    setSelectedLog(null);
  }

  return (
    <section className="page">
      <header className="page-header">
        <span className="page-header__eyebrow">实时接入</span>
        <div>
          <h2>日志流</h2>
          <p>面向实时检索和问题定位的日志工作台，保留轮询、分页和新消息提醒能力。</p>
        </div>
      </header>

      <div className="page-grid page-grid--logs">
        <article className="panel">
          <div className="panel__header">
            <h3>过滤条件</h3>
            <span>
              {isLoading ? "加载中..." : `${pagination.totalItems} 条记录`}
            </span>
          </div>

          <form className="log-search-form" onSubmit={handleSearchSubmit}>
            <label
              className="log-search-form__label"
              htmlFor="logs-search-input"
            >
              模糊搜索
            </label>
            <div className="log-search-form__controls">
              <input
                id="logs-search-input"
                name="logs-search"
                type="search"
                value={inputQuery}
                onChange={(event) => setInputQuery(event.target.value)}
                placeholder="搜索原始消息、MAC、主机名等"
              />
              <button type="submit">搜索</button>
            </div>

            <div className="log-search-form__date-range">
              <label className="form-field">
                <span className="form-field__label">开始日期</span>
                <input
                  className="form-field__control"
                  type="date"
                  value={inputFromDate}
                  onChange={(event) => setInputFromDate(event.target.value)}
                />
              </label>
              <label className="form-field">
                <span className="form-field__label">结束日期</span>
                <input
                  className="form-field__control"
                  type="date"
                  value={inputToDate}
                  onChange={(event) => setInputToDate(event.target.value)}
                />
              </label>
            </div>
          </form>

          <div className="filter-row">
            <span>{`视图：${scope === "matched" ? "有效事件" : "全部接收"}`}</span>
            <span>排序：按接收时间倒序</span>
            <span>分页：每页 10 条</span>
            <span>
              {submittedQuery ? `关键字：${submittedQuery}` : "关键字：全部"}
            </span>
            <span>
              {submittedFromDate || submittedToDate
                ? `日期：${submittedFromDate || "不限"} 至 ${submittedToDate || "不限"}`
                : "日期：全部"}
            </span>
          </div>

          <div className="log-summary-strip">
            {logSummary.map((item) => (
              <article key={item.label} className="log-summary-card">
                <span>{item.label}</span>
                <strong>{item.value}</strong>
              </article>
            ))}
          </div>

          {hasNewerResults ? (
            <div className="log-refresh-banner" role="status">
              <span>有新消息，返回第一页查看</span>
              <button
                type="button"
                onClick={() => {
                  setHasNewerResults(false);
                  setCurrentPage(1);
                }}
              >
                返回第一页查看
              </button>
            </div>
          ) : null}

          {errorMessage ? <p className="panel__copy">{errorMessage}</p> : null}

          <div className="scope-switch">
            <button
              type="button"
              className={scope === "matched" ? "button button--primary" : "button button--secondary"}
              onClick={() => handleScopeChange("matched")}
            >
              有效事件
            </button>
            <button
              type="button"
              className={scope === "all" ? "button button--primary" : "button button--secondary"}
              onClick={() => handleScopeChange("all")}
            >
              全部接收
            </button>
          </div>
        </article>

        <article className="panel panel--tall">
          <div className="panel__header">
            <h3>日志明细</h3>
            <span>
              {logRows.length > 0
                ? `第 ${pagination.page} / ${displayTotalPages} 页`
                : "暂无记录"}
            </span>
          </div>

          <div
            className="log-table"
            role="table"
            aria-label="日志流预览"
          >
            <div className="log-table__head">
              <span>接收日期时间</span>
              <span>解析状态</span>
              {scope === "matched" ? (
                <>
                  <span>事件类型</span>
                  <span>站点 MAC</span>
                  <span>员工</span>
                  <span>详情</span>
                </>
              ) : (
                <>
                  <span>命中规则</span>
                  <span>来源</span>
                  <span>原始消息</span>
                  <span>详情</span>
                </>
              )}
            </div>

            {logRows.length > 0 ? (
              logRows.map((row) => (
                <div
                  key={row.key}
                  className="log-row"
                  role="row"
                >
                  <span role="cell" className="log-row__datetime">{row.receivedAt}</span>
                  <span role="cell">{row.parseStatus}</span>
                  {scope === "matched" ? (
                    <>
                      <strong role="cell">{row.eventType}</strong>
                      <span role="cell" className="log-row__mac">{row.stationMac}</span>
                      <span role="cell">{row.employeeName}</span>
                    </>
                  ) : (
                    <>
                      <span role="cell">{row.matchedRuleName || "-"}</span>
                      <span role="cell">{row.sourceIp}</span>
                      <span role="cell">{row.item.message.rawMessage ?? "-"}</span>
                    </>
                  )}
                  <span role="cell">
                    <button
                      type="button"
                      className="button button--ghost button--small"
                      onClick={() => setSelectedLog(row)}
                    >
                      详情
                    </button>
                  </span>
                </div>
              ))
            ) : (
              <p className="panel__copy">
                {isLoading ? "日志加载中..." : "暂无日志"}
              </p>
            )}
          </div>

          <div className="log-pagination">
            <button
              type="button"
              onClick={() => setCurrentPage((page) => Math.max(1, page - 1))}
              disabled={currentPage <= 1 || isLoading}
            >
              上一页
            </button>
            <span>
              第 {pagination.page} / {displayTotalPages} 页
            </span>
            <button
              type="button"
              onClick={() =>
                setCurrentPage((page) => Math.min(displayTotalPages, page + 1))
              }
              disabled={
                currentPage >= displayTotalPages ||
                isLoading ||
                pagination.totalItems === 0
              }
            >
              下一页
            </button>
          </div>
        </article>
      </div>

      {selectedLog ? (
        <div
          className="detail-modal"
          role="presentation"
          onClick={() => setSelectedLog(null)}
        >
          <section
            className="detail-modal__content"
            role="dialog"
            aria-modal="true"
            aria-labelledby="log-detail-title"
            onClick={(event) => event.stopPropagation()}
          >
            <div className="panel__header">
              <h3 id="log-detail-title">日志详情</h3>
              <button
                type="button"
                className="button button--ghost button--small"
                onClick={() => setSelectedLog(null)}
              >
                关闭详情
              </button>
            </div>

            <div className="detail-modal__grid">
              <div className="detail-modal__field">
                <span className="detail-modal__label">接收日期时间</span>
                <strong>{selectedLog.receivedAt}</strong>
              </div>
              <div className="detail-modal__field">
                <span className="detail-modal__label">解析状态</span>
                <strong>{selectedLog.parseStatus}</strong>
              </div>
              <div className="detail-modal__field">
                <span className="detail-modal__label">事件类型</span>
                <strong>{selectedLog.eventType}</strong>
              </div>
              <div className="detail-modal__field">
                <span className="detail-modal__label">站点 MAC</span>
                <strong>{selectedLog.stationMac}</strong>
              </div>
              <div className="detail-modal__field">
                <span className="detail-modal__label">员工</span>
                <strong>{selectedLog.employeeName}</strong>
              </div>
              <div className="detail-modal__field">
                <span className="detail-modal__label">命中规则</span>
                <strong>{selectedLog.matchedRuleName || "-"}</strong>
              </div>
              <div className="detail-modal__field">
                <span className="detail-modal__label">来源地址</span>
                <strong>{selectedLog.sourceIp}</strong>
              </div>
            </div>

            <div className="detail-modal__message">
              <span className="detail-modal__label">原始消息</span>
              <p>{selectedLog.item.message.rawMessage ?? "-"}</p>
            </div>
          </section>
        </div>
      ) : null}
    </section>
  );
}
