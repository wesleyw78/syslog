import { useState, type FormEvent } from "react";

type SettingsFormValues = {
  dayEndTime: string;
  feishuAppId: string;
  feishuAppSecret: string;
  feishuLocationName: string;
  reportRetryLimit: string;
  reportTimeoutSeconds: string;
  retentionDays: string;
};

type SettingsFormProps = {
  isDisabled: boolean;
  initialValues: SettingsFormValues;
  isSaving: boolean;
  onSubmit: (values: SettingsFormValues) => Promise<void>;
};

function isInteger(value: string, minimum: number): boolean {
  return /^\d+$/.test(value) && Number(value) >= minimum;
}

function isValidTime(value: string): boolean {
  const matched = /^(\d{2}):(\d{2})$/.exec(value.trim());
  if (!matched) {
    return false;
  }

  const hours = Number(matched[1]);
  const minutes = Number(matched[2]);
  return hours >= 0 && hours <= 23 && minutes >= 0 && minutes <= 59;
}

export function SettingsForm({
  isDisabled,
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
      feishuAppId: String(formData.get("feishuAppId") ?? ""),
      feishuAppSecret: String(formData.get("feishuAppSecret") ?? ""),
      feishuLocationName: String(formData.get("feishuLocationName") ?? ""),
      reportTimeoutSeconds: String(formData.get("reportTimeoutSeconds") ?? ""),
      reportRetryLimit: String(formData.get("reportRetryLimit") ?? ""),
    };

    const isInvalid =
      !isValidTime(values.dayEndTime) ||
      !values.feishuAppId.trim() ||
      !values.feishuAppSecret.trim() ||
      !values.feishuLocationName.trim() ||
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
      className="console-form"
    >
      <section className="form-section">
        <div className="form-section__header">
          <div>
            <h4>日切与保留</h4>
            <p>控制归档窗口、保留周期与跨日处理规则。</p>
          </div>
        </div>

        <label className="form-field">
          <span className="form-field__label">日切时间</span>
          <input
            className="form-field__control"
            name="dayEndTime"
            type="time"
            defaultValue={initialValues.dayEndTime}
            disabled={isDisabled}
            onInput={() => setErrorMessage("")}
            step={60}
          />
        </label>

        <label className="form-field">
          <span className="form-field__label">日志保留天数</span>
          <input
            className="form-field__control"
            name="retentionDays"
            min={1}
            type="number"
            defaultValue={initialValues.retentionDays}
            disabled={isDisabled}
            onInput={() => setErrorMessage("")}
          />
        </label>
      </section>

      <section className="form-section">
        <div className="form-section__header">
          <div>
            <h4>飞书上报</h4>
            <p>维护飞书应用凭据和打卡地点；创建人默认与员工本人保持一致，并统一控制超时重试。</p>
          </div>
        </div>

        <label className="form-field">
          <span className="form-field__label">Feishu App ID</span>
          <input
            className="form-field__control"
            name="feishuAppId"
            type="text"
            defaultValue={initialValues.feishuAppId}
            disabled={isDisabled}
            onInput={() => setErrorMessage("")}
          />
        </label>

        <label className="form-field">
          <span className="form-field__label">Feishu App Secret</span>
          <input
            className="form-field__control"
            name="feishuAppSecret"
            type="password"
            defaultValue={initialValues.feishuAppSecret}
            disabled={isDisabled}
            onInput={() => setErrorMessage("")}
          />
        </label>

        <label className="form-field">
          <span className="form-field__label">打卡地点名称</span>
          <input
            className="form-field__control"
            name="feishuLocationName"
            type="text"
            defaultValue={initialValues.feishuLocationName}
            disabled={isDisabled}
            onInput={() => setErrorMessage("")}
          />
        </label>

        <label className="form-field">
          <span className="form-field__label">报告超时秒数</span>
          <input
            className="form-field__control"
            name="reportTimeoutSeconds"
            min={1}
            type="number"
            defaultValue={initialValues.reportTimeoutSeconds}
            disabled={isDisabled}
            onInput={() => setErrorMessage("")}
          />
        </label>

        <label className="form-field">
          <span className="form-field__label">重试次数</span>
          <input
            className="form-field__control"
            name="reportRetryLimit"
            min={0}
            type="number"
            defaultValue={initialValues.reportRetryLimit}
            disabled={isDisabled}
            onInput={() => setErrorMessage("")}
          />
        </label>
      </section>

      {errorMessage ? <p role="alert" className="form-error">{errorMessage}</p> : null}

      <button
        type="submit"
        disabled={isSaving || isDisabled}
        className="button button--primary"
      >
        {isSaving ? "保存中..." : "保存设置"}
      </button>
    </form>
  );
}

export type { SettingsFormValues };
