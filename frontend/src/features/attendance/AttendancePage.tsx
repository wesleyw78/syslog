import { useEffect, useMemo, useState } from "react";

import {
  correctAttendanceRecord,
  getAttendanceRecords,
  getEmployees,
  type AttendanceRecord,
  type Employee,
} from "../../lib/api";
import { AttendanceTable } from "./components/AttendanceTable";

type AttendanceDraft = {
  firstConnectAt: string;
  lastDisconnectAt: string;
};

type AttendanceRow = AttendanceRecord & {
  employeeName: string;
};

function createDraft(record: AttendanceRecord): AttendanceDraft {
  return {
    firstConnectAt: record.firstConnectAt ?? "",
    lastDisconnectAt: record.lastDisconnectAt ?? "",
  };
}

export function AttendancePage() {
  const [records, setRecords] = useState<AttendanceRow[]>([]);
  const [drafts, setDrafts] = useState<Record<string, AttendanceDraft>>({});
  const [pendingId, setPendingId] = useState<string | null>(null);
  const [queueMessage, setQueueMessage] = useState("加载考勤记录...");

  useEffect(() => {
    let isActive = true;

    void (async () => {
      try {
        const [employees, attendance] = await Promise.all([
          getEmployees(),
          getAttendanceRecords(),
        ]);

        if (!isActive) {
          return;
        }

        const employeeMap = new Map(
          employees.map((employee) => [employee.id, employee]),
        );
        const nextRecords = attendance.map((record) =>
          toAttendanceRow(record, employeeMap),
        );

        setRecords(nextRecords);
        setDrafts(
          Object.fromEntries(
            nextRecords.map((record) => [record.id, createDraft(record)]),
          ),
        );
        setQueueMessage(`已同步 ${attendance.length} 条考勤记录`);
      } catch {
        if (isActive) {
          setQueueMessage("考勤异常队列加载失败，请稍后重试");
        }
      }
    })();

    return () => {
      isActive = false;
    };
  }, []);

  const attendanceBands = useMemo(() => {
    const clockInCount = records.filter((record) => record.firstConnectAt).length;
    const pendingClockOutCount = records.filter(
      (record) =>
        record.clockOutStatus === "pending" || record.clockOutStatus === "missing",
    ).length;
    const manualCount = records.filter(
      (record) => record.sourceMode === "manual",
    ).length;

    return [
      { label: "已记录上班", value: `${clockInCount}` },
      { label: "待确认下班", value: `${pendingClockOutCount}` },
      { label: "人工修正", value: `${manualCount}` },
    ];
  }, [records]);
  const attentionRecords = useMemo(
    () =>
      records.filter(
        (record) =>
          record.exceptionStatus !== "none" ||
          record.clockInStatus === "pending" ||
          record.clockOutStatus === "pending" ||
          record.clockOutStatus === "missing",
      ),
    [records],
  );
  const normalRecords = useMemo(
    () => records.filter((record) => !attentionRecords.some((item) => item.id === record.id)),
    [attentionRecords, records],
  );

  function updateDraft(
    recordId: string,
    field: keyof AttendanceDraft,
    value: string,
  ) {
    setDrafts((current) => ({
      ...current,
      [recordId]: {
        ...(current[recordId] ?? { firstConnectAt: "", lastDisconnectAt: "" }),
        [field]: value,
      },
    }));
  }

  async function handleManualCorrection(recordId: string) {
    setPendingId(recordId);

    const draft = drafts[recordId] ?? {
      firstConnectAt: "",
      lastDisconnectAt: "",
    };

    try {
      const result = await correctAttendanceRecord(recordId, {
        firstConnectAt: draft.firstConnectAt.trim() || null,
        lastDisconnectAt: draft.lastDisconnectAt.trim() || null,
      });

      setRecords((current) =>
        current.map((record) =>
          record.id === result.attendance.id
            ? toAttendanceRow(
                result.attendance,
                currentEmployeeMapFromRows(current),
              )
            : record,
        ),
      );
      setDrafts((current) => ({
        ...current,
        [result.attendance.id]: createDraft(result.attendance),
      }));
      setQueueMessage(`已提交 ${recordNameById(records, recordId)} 的人工修正`);
    } catch {
      setQueueMessage("人工修正提交失败，请稍后重试");
    } finally {
      setPendingId(null);
    }
  }

  return (
    <section className="page">
      <header className="page-header">
        <span className="page-header__eyebrow">异常复核</span>
        <div>
          <h2>考勤复核</h2>
          <p>聚焦待处理考勤与人工修正，保持异常优先、动作明确的复核工作流。</p>
        </div>
      </header>

      <div className="page-grid page-grid--split">
        <article className="panel">
          <div className="panel__header">
            <h3>覆盖指标</h3>
            <span>当前班次</span>
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
            <h3>待处理队列</h3>
            <span>{attentionRecords.length} 条</span>
          </div>
          <p className="panel__copy">{queueMessage}</p>
          <div className="panel__body-offset">
            {attentionRecords.length > 0 ? (
              <AttendanceTable
                drafts={drafts}
                pendingId={pendingId}
                records={attentionRecords}
                onDraftChange={updateDraft}
                onManualCorrection={handleManualCorrection}
              />
            ) : (
              <p className="panel__copy">当前没有待处理考勤，下面仍保留完整记录以便核对。</p>
            )}
          </div>
        </article>
      </div>

      <article className="panel panel--tall">
        <div className="panel__header">
          <h3>全部记录</h3>
          <span>{records.length} 条</span>
        </div>
        <p className="panel__copy">
          保留完整记录区，便于主管在处理待办后继续核对已经稳定的考勤结果。
        </p>
        <div className="panel__body-offset">
          <AttendanceTable
            drafts={drafts}
            pendingId={pendingId}
            records={normalRecords.length > 0 ? normalRecords : records}
            onDraftChange={updateDraft}
            onManualCorrection={handleManualCorrection}
          />
        </div>
      </article>
    </section>
  );
}

function currentEmployeeMapFromRows(records: AttendanceRow[]): Map<string, Employee> {
  return new Map(
    records.map((record) => [
      record.employeeId,
      {
        id: record.employeeId,
        employeeNo: "",
        systemNo: "",
        feishuEmployeeId: "",
        name: record.employeeName,
        status: "active",
        devices: [],
        createdAt: "",
        updatedAt: "",
      },
    ]),
  );
}

function toAttendanceRow(
  record: AttendanceRecord,
  employees: Map<string, Employee>,
): AttendanceRow {
  const employee = employees.get(record.employeeId);

  return {
    ...record,
    employeeName: employee?.name ?? `员工 ${record.employeeId}`,
  };
}

function recordNameById(records: AttendanceRow[], recordId: string): string {
  return records.find((record) => record.id === recordId)?.employeeName ?? recordId;
}
