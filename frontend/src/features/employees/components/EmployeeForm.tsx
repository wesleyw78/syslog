import { useEffect, useMemo, useState, type FormEvent } from "react";

import type { EmployeeUpsertInput } from "../../../lib/api";

type EmployeeFormProps = {
  initialValues?: EmployeeUpsertInput;
  isSubmitting: boolean;
  onCancel?: () => void;
  resetOnSubmit?: boolean;
  submitLabel?: string;
  onSubmit: (input: EmployeeUpsertInput) => Promise<void>;
};

type DeviceDraft = EmployeeUpsertInput["devices"][number];

function createEmptyDevice(): DeviceDraft {
  return {
    macAddress: "",
    deviceLabel: "",
    status: "active",
  };
}

function createEmptyDraft(): EmployeeUpsertInput {
  return {
    employeeNo: "",
    systemNo: "",
    feishuEmployeeId: "",
    name: "",
    status: "active",
    devices: [createEmptyDevice()],
  };
}

function toDraft(initialValues?: EmployeeUpsertInput): EmployeeUpsertInput {
  if (!initialValues) {
    return createEmptyDraft();
  }

  return {
    employeeNo: initialValues.employeeNo,
    systemNo: initialValues.systemNo,
    feishuEmployeeId: initialValues.feishuEmployeeId,
    name: initialValues.name,
    status: initialValues.status,
    devices:
      initialValues.devices.length > 0
        ? initialValues.devices
        : [createEmptyDevice()],
  };
}

export function EmployeeForm({
  initialValues,
  isSubmitting,
  onCancel,
  onSubmit,
  resetOnSubmit = false,
  submitLabel = "新增员工",
}: EmployeeFormProps) {
  const [errorMessage, setErrorMessage] = useState("");
  const defaultDraft = useMemo(() => createEmptyDraft(), []);
  const [draft, setDraft] = useState<EmployeeUpsertInput>(toDraft(initialValues));

  useEffect(() => {
    setErrorMessage("");
    setDraft(toDraft(initialValues));
  }, [initialValues]);

  function updateDevice(index: number, patch: Partial<DeviceDraft>) {
    setDraft((current) => ({
      ...current,
      devices: current.devices.map((device, deviceIndex) =>
        deviceIndex === index ? { ...device, ...patch } : device,
      ),
    }));
  }

  function addDevice() {
    setDraft((current) => ({
      ...current,
      devices: [...current.devices, createEmptyDevice()],
    }));
  }

  function removeDevice(index: number) {
    setDraft((current) => {
      if (current.devices.length <= 1) {
        return current;
      }

      return {
        ...current,
        devices: current.devices.filter((_, deviceIndex) => deviceIndex !== index),
      };
    });
  }

  async function handleSubmit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();

    const nextDraft: EmployeeUpsertInput = {
      employeeNo: draft.employeeNo.trim(),
      systemNo: draft.systemNo.trim(),
      feishuEmployeeId: draft.feishuEmployeeId.trim(),
      name: draft.name.trim(),
      status: draft.status.trim() || "active",
      devices: draft.devices.map((device) => ({
        macAddress: device.macAddress.trim().toUpperCase(),
        deviceLabel: device.deviceLabel.trim(),
        status: device.status.trim() || "active",
      })),
    };

    if (
      !nextDraft.employeeNo ||
      !nextDraft.systemNo ||
      !nextDraft.feishuEmployeeId ||
      !nextDraft.name
    ) {
      setErrorMessage("员工编号、系统编号、飞书员工 ID 和姓名不能为空");
      return;
    }

    if (nextDraft.devices.length === 0) {
      setErrorMessage("请至少添加一个设备");
      return;
    }

    const hasEmptyDevice = nextDraft.devices.some(
      (device) => !device.macAddress || !device.deviceLabel,
    );

    if (hasEmptyDevice) {
      setErrorMessage("设备信息不能为空");
      return;
    }

    setErrorMessage("");
    await onSubmit(nextDraft);

    if (resetOnSubmit) {
      setDraft(defaultDraft);
    }
  }

  return (
    <form noValidate onSubmit={handleSubmit} className="console-form">
      <label className="form-field">
        <span className="form-field__label">员工编号</span>
        <input
          className="form-field__control"
          required
          type="text"
          value={draft.employeeNo}
          onChange={(event) =>
            setDraft((current) => ({ ...current, employeeNo: event.target.value }))
          }
          onInput={() => setErrorMessage("")}
        />
      </label>

      <label className="form-field">
        <span className="form-field__label">系统编号</span>
        <input
          className="form-field__control"
          required
          type="text"
          value={draft.systemNo}
          onChange={(event) =>
            setDraft((current) => ({ ...current, systemNo: event.target.value }))
          }
          onInput={() => setErrorMessage("")}
        />
      </label>

      <label className="form-field">
        <span className="form-field__label">姓名</span>
        <input
          className="form-field__control"
          required
          type="text"
          value={draft.name}
          onChange={(event) =>
            setDraft((current) => ({ ...current, name: event.target.value }))
          }
          onInput={() => setErrorMessage("")}
        />
      </label>

      <label className="form-field">
        <span className="form-field__label">飞书员工 ID</span>
        <input
          className="form-field__control"
          required
          type="text"
          value={draft.feishuEmployeeId}
          onChange={(event) =>
            setDraft((current) => ({
              ...current,
              feishuEmployeeId: event.target.value,
            }))
          }
          onInput={() => setErrorMessage("")}
        />
      </label>

      <section className="form-section">
        <div className="form-section__header">
          <div>
            <h4>绑定设备</h4>
            <p>维护员工可识别的终端 MAC 与标签。</p>
          </div>
          <button
            type="button"
            onClick={addDevice}
            disabled={isSubmitting}
            className="button button--ghost button--small"
          >
            添加设备
          </button>
        </div>

        {draft.devices.map((device, index) => (
          <div key={index} className="device-card">
            <div className="device-card__header">
              <strong>{`设备 ${index + 1}`}</strong>
              {draft.devices.length > 1 ? (
                <button
                  type="button"
                  onClick={() => removeDevice(index)}
                  disabled={isSubmitting}
                  className="button button--ghost button--small"
                >
                  {`移除设备 ${index + 1}`}
                </button>
              ) : null}
            </div>

            <label className="form-field">
              <span className="form-field__label">{`设备 ${index + 1} MAC`}</span>
              <input
                className="form-field__control"
                required
                type="text"
                value={device.macAddress}
                onChange={(event) =>
                  updateDevice(index, { macAddress: event.target.value })
                }
                onInput={() => setErrorMessage("")}
              />
            </label>

            <label className="form-field">
              <span className="form-field__label">{`设备 ${index + 1} 标签`}</span>
              <input
                className="form-field__control"
                required
                type="text"
                value={device.deviceLabel}
                onChange={(event) =>
                  updateDevice(index, { deviceLabel: event.target.value })
                }
                onInput={() => setErrorMessage("")}
              />
            </label>
          </div>
        ))}
      </section>

      {errorMessage ? <p role="alert" className="form-error">{errorMessage}</p> : null}

      <div className="form-actions">
        <button type="submit" disabled={isSubmitting} className="button button--primary">
          {isSubmitting ? "提交中..." : submitLabel}
        </button>
        {onCancel ? (
          <button
            type="button"
            onClick={onCancel}
            disabled={isSubmitting}
            className="button button--ghost"
          >
            取消
          </button>
        ) : null}
      </div>
    </form>
  );
}
