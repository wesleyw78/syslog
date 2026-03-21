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
  border: "1px solid rgba(255, 184, 77, 0.35)",
  background: "linear-gradient(180deg, rgba(41, 28, 8, 0.95), rgba(18, 15, 8, 0.96))",
  color: "inherit",
  cursor: "pointer",
  textTransform: "uppercase" as const,
  letterSpacing: "0.08em",
};

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
    name: initialValues.name,
    status: initialValues.status,
    devices: initialValues.devices.length > 0 ? initialValues.devices : [createEmptyDevice()],
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

  async function handleSubmit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();

    const nextDraft: EmployeeUpsertInput = {
      employeeNo: draft.employeeNo.trim(),
      systemNo: draft.systemNo.trim(),
      name: draft.name.trim(),
      status: draft.status.trim() || "active",
      devices: draft.devices.map((device) => ({
        macAddress: device.macAddress.trim().toUpperCase(),
        deviceLabel: device.deviceLabel.trim(),
        status: device.status.trim() || "active",
      })),
    };

    if (!nextDraft.employeeNo || !nextDraft.systemNo || !nextDraft.name) {
      setErrorMessage("员工编号、系统编号和姓名不能为空");
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
    <form
      noValidate
      onSubmit={handleSubmit}
      style={{ display: "grid", gap: "0.9rem" }}
    >
      <label style={fieldStyle}>
        <span>员工编号</span>
        <input
          required
          type="text"
          value={draft.employeeNo}
          onChange={(event) =>
            setDraft((current) => ({ ...current, employeeNo: event.target.value }))
          }
          onInput={() => setErrorMessage("")}
          style={inputStyle}
        />
      </label>

      <label style={fieldStyle}>
        <span>系统编号</span>
        <input
          required
          type="text"
          value={draft.systemNo}
          onChange={(event) =>
            setDraft((current) => ({ ...current, systemNo: event.target.value }))
          }
          onInput={() => setErrorMessage("")}
          style={inputStyle}
        />
      </label>

      <label style={fieldStyle}>
        <span>姓名</span>
        <input
          required
          type="text"
          value={draft.name}
          onChange={(event) =>
            setDraft((current) => ({ ...current, name: event.target.value }))
          }
          onInput={() => setErrorMessage("")}
          style={inputStyle}
        />
      </label>

      <div style={{ display: "grid", gap: "0.65rem" }}>
        <div style={{ display: "flex", justifyContent: "space-between", gap: "0.75rem" }}>
          <span>设备</span>
          <button
            type="button"
            onClick={addDevice}
            disabled={isSubmitting}
            style={{
              ...buttonStyle,
              padding: "0.55rem 0.8rem",
            }}
          >
            添加设备
          </button>
        </div>

        {draft.devices.map((device, index) => (
          <div
            key={index}
            style={{
              display: "grid",
              gap: "0.65rem",
              padding: "0.8rem",
              border: "1px solid rgba(255, 184, 77, 0.12)",
              background: "rgba(7, 9, 9, 0.56)",
            }}
          >
            <label style={fieldStyle}>
              <span>{`设备 ${index + 1} MAC`}</span>
              <input
                required
                type="text"
                value={device.macAddress}
                onChange={(event) =>
                  updateDevice(index, { macAddress: event.target.value })
                }
                onInput={() => setErrorMessage("")}
                style={inputStyle}
              />
            </label>

            <label style={fieldStyle}>
              <span>{`设备 ${index + 1} 标签`}</span>
              <input
                required
                type="text"
                value={device.deviceLabel}
                onChange={(event) =>
                  updateDevice(index, { deviceLabel: event.target.value })
                }
                onInput={() => setErrorMessage("")}
                style={inputStyle}
              />
            </label>
          </div>
        ))}
      </div>

      {errorMessage ? (
        <p
          role="alert"
          style={{ margin: 0, color: "#ffb86b", letterSpacing: "0.03em" }}
        >
          {errorMessage}
        </p>
      ) : null}

      <div style={{ display: "flex", gap: "0.65rem" }}>
        <button type="submit" disabled={isSubmitting} style={buttonStyle}>
          {isSubmitting ? "提交中..." : submitLabel}
        </button>
        {onCancel ? (
          <button
            type="button"
            onClick={onCancel}
            disabled={isSubmitting}
            style={{
              ...buttonStyle,
              border: "1px solid rgba(255, 184, 77, 0.18)",
              background: "rgba(9, 11, 12, 0.86)",
            }}
          >
            取消
          </button>
        ) : null}
      </div>
    </form>
  );
}
