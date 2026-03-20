import { useEffect, useState, type FormEvent } from "react";

import type { EmployeeDraft } from "../../../lib/api";

type EmployeeFormProps = {
  initialValues?: EmployeeDraft;
  isSubmitting: boolean;
  onCancel?: () => void;
  resetOnSubmit?: boolean;
  submitLabel?: string;
  onSubmit: (draft: EmployeeDraft) => Promise<void>;
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
  border: "1px solid rgba(255, 184, 77, 0.35)",
  background: "linear-gradient(180deg, rgba(41, 28, 8, 0.95), rgba(18, 15, 8, 0.96))",
  color: "inherit",
  cursor: "pointer",
  textTransform: "uppercase" as const,
  letterSpacing: "0.08em",
};

export function EmployeeForm({
  initialValues,
  isSubmitting,
  onCancel,
  onSubmit,
  resetOnSubmit = false,
  submitLabel = "新增员工",
}: EmployeeFormProps) {
  const emptyDraft = {
    name: "",
    team: "",
    badge: "",
  };
  const [draft, setDraft] = useState<EmployeeDraft>({
    ...emptyDraft,
    ...initialValues,
  });

  useEffect(() => {
    setDraft({
      ...emptyDraft,
      ...initialValues,
    });
  }, [initialValues]);

  async function handleSubmit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    await onSubmit(draft);

    if (resetOnSubmit) {
      setDraft(emptyDraft);
    }
  }

  return (
    <form onSubmit={handleSubmit} style={{ display: "grid", gap: "0.9rem" }}>
      <label style={fieldStyle}>
        <span>员工姓名</span>
        <input
          required
          type="text"
          value={draft.name}
          onChange={(event) =>
            setDraft((current) => ({ ...current, name: event.target.value }))
          }
          style={inputStyle}
        />
      </label>

      <label style={fieldStyle}>
        <span>班组</span>
        <input
          required
          type="text"
          value={draft.team}
          onChange={(event) =>
            setDraft((current) => ({ ...current, team: event.target.value }))
          }
          style={inputStyle}
        />
      </label>

      <label style={fieldStyle}>
        <span>工牌号</span>
        <input
          required
          type="text"
          value={draft.badge}
          onChange={(event) =>
            setDraft((current) => ({ ...current, badge: event.target.value }))
          }
          style={inputStyle}
        />
      </label>

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
