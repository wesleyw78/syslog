import { useState, type FormEvent } from "react";

type SettingsFormValues = {
  dayEndTime: string;
  reportRetryLimit: string;
  reportTargetUrl: string;
  reportTimeoutSeconds: string;
  retentionDays: string;
};

type SettingsFormProps = {
  initialValues: SettingsFormValues;
  isSaving: boolean;
  onSubmit: (values: SettingsFormValues) => Promise<void>;
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

function isInteger(value: string, minimum: number): boolean {
  return /^\d+$/.test(value) && Number(value) >= minimum;
}

export function SettingsForm({
  initialValues,
  isSaving,
  onSubmit,
}: SettingsFormProps) {
  const [errorMessage, setErrorMessage] = useState("");

  async function handleSubmit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();

    const formData = new FormData(event.currentTarget);
    const values: SettingsFormValues = {
      dayEndTime: String(formData.get("dayEndTime") ?? ""),
      retentionDays: String(formData.get("retentionDays") ?? ""),
      reportTargetUrl: String(formData.get("reportTargetUrl") ?? ""),
      reportTimeoutSeconds: String(formData.get("reportTimeoutSeconds") ?? ""),
      reportRetryLimit: String(formData.get("reportRetryLimit") ?? ""),
    };

    const isInvalid =
      !/^\d{2}:\d{2}$/.test(values.dayEndTime) ||
      !values.reportTargetUrl.trim() ||
      !isInteger(values.retentionDays, 1) ||
      !isInteger(values.reportTimeoutSeconds, 1) ||
      !isInteger(values.reportRetryLimit, 0);

    if (isInvalid) {
      setErrorMessage("设置数值不合法");
      return;
    }

    setErrorMessage("");
    await onSubmit(values);
  }

  return (
    <form
      key={JSON.stringify(initialValues)}
      noValidate
      onSubmit={handleSubmit}
      style={{ display: "grid", gap: "0.9rem" }}
    >
      <label style={fieldStyle}>
        <span>日切时间</span>
        <input
          name="dayEndTime"
          type="time"
          defaultValue={initialValues.dayEndTime}
          onInput={() => setErrorMessage("")}
          style={inputStyle}
          step={60}
        />
      </label>

      <label style={fieldStyle}>
        <span>日志保留天数</span>
        <input
          name="retentionDays"
          min={1}
          type="number"
          defaultValue={initialValues.retentionDays}
          onInput={() => setErrorMessage("")}
          style={inputStyle}
        />
      </label>

      <label style={fieldStyle}>
        <span>报告目标地址</span>
        <input
          name="reportTargetUrl"
          type="text"
          defaultValue={initialValues.reportTargetUrl}
          onInput={() => setErrorMessage("")}
          style={inputStyle}
        />
      </label>

      <label style={fieldStyle}>
        <span>报告超时秒数</span>
        <input
          name="reportTimeoutSeconds"
          min={1}
          type="number"
          defaultValue={initialValues.reportTimeoutSeconds}
          onInput={() => setErrorMessage("")}
          style={inputStyle}
        />
      </label>

      <label style={fieldStyle}>
        <span>重试次数</span>
        <input
          name="reportRetryLimit"
          min={0}
          type="number"
          defaultValue={initialValues.reportRetryLimit}
          onInput={() => setErrorMessage("")}
          style={inputStyle}
        />
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

export type { SettingsFormValues };
