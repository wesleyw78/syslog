import { useEffect, useMemo, useState } from "react";

import {
  correctAttendanceRecord,
  listAttendanceRecords,
  type AttendanceRecord,
} from "../../lib/api";
import { AttendanceTable } from "./components/AttendanceTable";

export function AttendancePage() {
  const [records, setRecords] = useState<AttendanceRecord[]>([]);
  const [pendingId, setPendingId] = useState<string | null>(null);
  const [queueMessage, setQueueMessage] = useState("等待考勤采集链路...");

  useEffect(() => {
    let isActive = true;

    void (async () => {
      try {
        const items = await listAttendanceRecords();

        if (!isActive) {
          return;
        }

        setRecords(items);
        setQueueMessage("已同步异常考勤与人工修正队列");
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
    const total = records.length || 1;
    const normalCount = records.filter((record) => record.status === "normal").length;
    const exceptionCount = records.filter(
      (record) => record.status === "exception",
    ).length;
    const correctedCount = records.filter(
      (record) => record.status === "corrected",
    ).length;

    return [
      { label: "On Time", value: `${Math.round((normalCount / total) * 100)}%` },
      { label: "Exceptions", value: `${exceptionCount}` },
      { label: "Corrected", value: `${correctedCount}` },
    ];
  }, [records]);

  async function handleManualCorrection(recordId: string) {
    setPendingId(recordId);

    try {
      const correctedRecord = await correctAttendanceRecord(recordId);
      setRecords((current) =>
        current.map((record) =>
          record.id === correctedRecord.id ? correctedRecord : record,
        ),
      );
      setQueueMessage(`已提交 ${correctedRecord.employeeName} 的人工修正`);
    } catch {
      setQueueMessage("人工修正提交失败，请稍后重试");
    } finally {
      setPendingId(null);
    }
  }

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
          <p className="panel__copy">{queueMessage}</p>
          <div style={{ marginTop: "1rem" }}>
            <AttendanceTable
              pendingId={pendingId}
              records={records}
              onManualCorrection={handleManualCorrection}
            />
          </div>
        </article>
      </div>
    </section>
  );
}
