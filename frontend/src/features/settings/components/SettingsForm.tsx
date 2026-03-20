import { useEffect, useState, type FormEvent } from "react";

import type { SettingsRecord } from "../../../lib/api";

type SettingsFormProps = {
  initialValues: SettingsRecord;
  isSaving: boolean;
  onSubmit: (values: SettingsRecord) => Promise<void>;
};

const fieldStyle = {
  display: "grid",
  gap: "0.35rem",
};

const inputStyle = {
  width: "100%",
  padding: "0.75rem 0.85rem",
  border: "1px solid rgba(255, 184, 77, 0.18)",
  background: "rgba(7, 9, 9, 0.8)",
  color: "inherit",
};

const buttonStyle = {
  padding: "0.8rem 1rem",
  border: "1px solid rgba(116, 216, 169, 0.25)",
  background: "linear-gradient(180deg, rgba(13, 33, 22, 0.95), rgba(10, 18, 14, 0.96))",
  color: "inherit",
  cursor: "pointer",
  textTransform: "uppercase" as const,
  letterSpacing: "0.08em",
};

export function SettingsForm({
  initialValues,
  isSaving,
  onSubmit,
}: SettingsFormProps) {
  const [errorMessage, setErrorMessage] = useState("");
  const [values, setValues] = useState(initialValues);

  useEffect(() => {
    setErrorMessage("");
    setValues(initialValues);
  }, [initialValues]);

  async function handleSubmit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    const isInvalid =
      !Number.isInteger(values.scannerRetryThreshold) ||
      values.scannerRetryThreshold < 1 ||
      !Number.isInteger(values.lateToleranceMinutes) ||
      values.lateToleranceMinutes < 0 ||
      !Number.isInteger(values.archiveRetentionDays) ||
      values.archiveRetentionDays < 1;

    if (isInvalid) {
      setErrorMessage("设置数值不合法");
      return;
    }

    setErrorMessage("");
    await onSubmit(values);
  }

  return (
    <form
      noValidate
      onSubmit={handleSubmit}
      style={{ display: "grid", gap: "0.9rem" }}
    >
      <label style={fieldStyle}>
        <span>扫描重试阈值</span>
        <input
          min={1}
          type="number"
          value={values.scannerRetryThreshold}
          onChange={(event) =>
            setValues((current) => ({
              ...current,
              scannerRetryThreshold: Number(event.target.value),
            }))
          }
          onInput={() => setErrorMessage("")}
          style={inputStyle}
        />
      </label>

      <label style={fieldStyle}>
        <span>迟到容忍分钟</span>
        <input
          min={0}
          type="number"
          value={values.lateToleranceMinutes}
          onChange={(event) =>
            setValues((current) => ({
              ...current,
              lateToleranceMinutes: Number(event.target.value),
            }))
          }
          onInput={() => setErrorMessage("")}
          style={inputStyle}
        />
      </label>

      <label style={fieldStyle}>
        <span>归档保留天数</span>
        <input
          min={1}
          type="number"
          value={values.archiveRetentionDays}
          onChange={(event) =>
            setValues((current) => ({
              ...current,
              archiveRetentionDays: Number(event.target.value),
            }))
          }
          onInput={() => setErrorMessage("")}
          style={inputStyle}
        />
      </label>

      <label
        style={{
          display: "flex",
          alignItems: "center",
          gap: "0.65rem",
          padding: "0.75rem 0.85rem",
          border: "1px solid rgba(255, 184, 77, 0.12)",
          background: "rgba(7, 9, 9, 0.56)",
        }}
      >
        <input
          checked={values.manualCorrectionRequiresApproval}
          type="checkbox"
          onChange={(event) =>
            setValues((current) => ({
              ...current,
              manualCorrectionRequiresApproval: event.target.checked,
            }))
          }
          onInput={() => setErrorMessage("")}
        />
        <span>人工修正需要主管审批</span>
      </label>

      {errorMessage ? (
        <p
          role="alert"
          style={{ margin: 0, color: "#ffb86b", letterSpacing: "0.03em" }}
        >
          {errorMessage}
        </p>
      ) : null}

      <button type="submit" disabled={isSaving} style={buttonStyle}>
        {isSaving ? "保存中..." : "保存设置"}
      </button>
    </form>
  );
}
