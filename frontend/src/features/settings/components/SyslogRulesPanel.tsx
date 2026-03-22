import { useEffect, useMemo, useState } from "react";

import {
  createSyslogRule,
  deleteSyslogRule,
  getSyslogRules,
  moveSyslogRule,
  previewSyslogRule,
  updateSyslogRule,
  type SyslogReceiveRule,
  type SyslogReceiveRuleInput,
  type SyslogRulePreviewResult,
} from "../../../lib/api";

const EMPTY_RULE: SyslogReceiveRuleInput = {
  name: "",
  enabled: true,
  eventType: "connect",
  messagePattern: "",
  stationMacGroup: "",
  apMacGroup: "",
  ssidGroup: "",
  ipv4Group: "",
  ipv6Group: "",
  hostnameGroup: "",
  osVendorGroup: "",
  eventTimeGroup: "",
  eventTimeLayout: "",
};

function toInput(rule: SyslogReceiveRule): SyslogReceiveRuleInput {
  return {
    name: rule.name,
    enabled: rule.enabled,
    eventType: rule.eventType,
    messagePattern: rule.messagePattern,
    stationMacGroup: rule.stationMacGroup,
    apMacGroup: rule.apMacGroup,
    ssidGroup: rule.ssidGroup,
    ipv4Group: rule.ipv4Group,
    ipv6Group: rule.ipv6Group,
    hostnameGroup: rule.hostnameGroup,
    osVendorGroup: rule.osVendorGroup,
    eventTimeGroup: rule.eventTimeGroup,
    eventTimeLayout: rule.eventTimeLayout,
  };
}

function validateRule(input: SyslogReceiveRuleInput): string {
  if (!input.name.trim()) {
    return "规则字段不合法";
  }
  if (!input.messagePattern.trim()) {
    return "规则字段不合法";
  }
  if (!input.stationMacGroup.trim()) {
    return "规则字段不合法";
  }
  if (input.eventTimeGroup.trim() && !input.eventTimeLayout.trim()) {
    return "规则字段不合法";
  }
  return "";
}

function nowPreviewValue(): string {
  return new Date().toISOString().slice(0, 16);
}

function sortRules(items: SyslogReceiveRule[]): SyslogReceiveRule[] {
  return [...items].sort((left, right) => {
    if (left.sortOrder !== right.sortOrder) {
      return left.sortOrder - right.sortOrder;
    }
    return left.id.localeCompare(right.id);
  });
}

function moveRuleInList(
  items: SyslogReceiveRule[],
  ruleId: string,
  direction: "up" | "down",
): SyslogReceiveRule[] {
  const sorted = sortRules(items);
  const index = sorted.findIndex((rule) => rule.id === ruleId);
  if (index === -1) {
    return sorted;
  }

  const targetIndex = direction === "up" ? index - 1 : index + 1;
  if (targetIndex < 0 || targetIndex >= sorted.length) {
    return sorted;
  }

  const next = [...sorted];
  const currentRule = next[index];
  const targetRule = next[targetIndex];
  next[index] = { ...targetRule, sortOrder: currentRule.sortOrder };
  next[targetIndex] = { ...currentRule, sortOrder: targetRule.sortOrder };
  return next;
}

export function SyslogRulesPanel() {
  const [rules, setRules] = useState<SyslogReceiveRule[]>([]);
  const [selectedRuleId, setSelectedRuleId] = useState<string | null>(null);
  const [draft, setDraft] = useState<SyslogReceiveRuleInput>(EMPTY_RULE);
  const [isLoading, setIsLoading] = useState(true);
  const [isSaving, setIsSaving] = useState(false);
  const [statusMessage, setStatusMessage] = useState("加载 Syslog 接收规则...");
  const [errorMessage, setErrorMessage] = useState("");
  const [previewReceivedAt, setPreviewReceivedAt] = useState(nowPreviewValue());
  const [previewRawMessage, setPreviewRawMessage] = useState("");
  const [previewResult, setPreviewResult] = useState<SyslogRulePreviewResult | null>(null);

  useEffect(() => {
    let active = true;

    void (async () => {
      try {
        const loadedRules = sortRules(await getSyslogRules());
        if (!active) {
          return;
        }

        setRules(loadedRules);
        if (loadedRules.length > 0) {
          setSelectedRuleId(loadedRules[0].id);
          setDraft(toInput(loadedRules[0]));
        }
        setStatusMessage("未命中规则会进入“全部接收”视图，但不会参与结构化事件");
      } catch {
        if (active) {
          setStatusMessage("规则装载失败，请稍后重试");
        }
      } finally {
        if (active) {
          setIsLoading(false);
        }
      }
    })();

    return () => {
      active = false;
    };
  }, []);

  const enabledRuleCount = useMemo(
    () => rules.filter((rule) => rule.enabled).length,
    [rules],
  );

  const selectedRule = useMemo(
    () => rules.find((rule) => rule.id === selectedRuleId) ?? null,
    [rules, selectedRuleId],
  );

  function handleNewRule() {
    setSelectedRuleId(null);
    setDraft(EMPTY_RULE);
    setErrorMessage("");
    setPreviewResult(null);
    setStatusMessage("正在创建新规则");
  }

  function handleSelectRule(rule: SyslogReceiveRule) {
    setSelectedRuleId(rule.id);
    setDraft(toInput(rule));
    setErrorMessage("");
    setPreviewResult(null);
    setStatusMessage("规则已载入编辑区");
  }

  function updateDraft<K extends keyof SyslogReceiveRuleInput>(
    key: K,
    value: SyslogReceiveRuleInput[K],
  ) {
    setDraft((current) => ({
      ...current,
      [key]: value,
    }));
    setErrorMessage("");
    setPreviewResult(null);
  }

  async function handleSave() {
    const validationMessage = validateRule(draft);
    if (validationMessage) {
      setErrorMessage(validationMessage);
      return;
    }

    setIsSaving(true);
    try {
      const savedRule = selectedRule
        ? await updateSyslogRule(selectedRule.id, draft)
        : await createSyslogRule(draft);

      setRules((current) => {
        const next = selectedRule
          ? current.map((item) => (item.id === savedRule.id ? savedRule : item))
          : [...current, savedRule];
        return sortRules(next);
      });
      setSelectedRuleId(savedRule.id);
      setDraft(toInput(savedRule));
      setStatusMessage("规则已保存");
    } catch {
      setStatusMessage("规则保存失败，请稍后重试");
    } finally {
      setIsSaving(false);
    }
  }

  async function handleDelete() {
    if (!selectedRule) {
      return;
    }
    if (!window.confirm(`确认删除规则“${selectedRule.name}”？`)) {
      return;
    }

    setIsSaving(true);
    try {
      await deleteSyslogRule(selectedRule.id);
      const remaining = rules.filter((rule) => rule.id !== selectedRule.id);
      setRules(remaining);
      if (remaining.length > 0) {
        setSelectedRuleId(remaining[0].id);
        setDraft(toInput(remaining[0]));
      } else {
        setSelectedRuleId(null);
        setDraft(EMPTY_RULE);
      }
      setPreviewResult(null);
      setStatusMessage("规则已删除");
    } catch {
      setStatusMessage("规则删除失败，请稍后重试");
    } finally {
      setIsSaving(false);
    }
  }

  async function handleMove(direction: "up" | "down") {
    if (!selectedRule) {
      return;
    }

    setIsSaving(true);
    try {
      await moveSyslogRule(selectedRule.id, direction);
      setRules((current) => moveRuleInList(current, selectedRule.id, direction));
      setStatusMessage("规则顺序已更新");
    } catch {
      setStatusMessage("规则顺序更新失败，请稍后重试");
    } finally {
      setIsSaving(false);
    }
  }

  async function handlePreview() {
    const validationMessage = validateRule(draft);
    if (validationMessage) {
      setErrorMessage(validationMessage);
      return;
    }
    if (!previewRawMessage.trim()) {
      setErrorMessage("规则字段不合法");
      return;
    }

    setIsSaving(true);
    try {
      const result = await previewSyslogRule({
        receivedAt: new Date(previewReceivedAt).toISOString(),
        rawMessage: previewRawMessage.trim(),
        rule: draft,
      });
      setPreviewResult(result);
      setStatusMessage(result.matched ? "规则预览已完成" : "规则预览已完成，当前未命中");
    } catch {
      setStatusMessage("规则预览失败，请稍后重试");
    } finally {
      setIsSaving(false);
    }
  }

  return (
    <article className="panel panel--full">
      <div className="panel__header">
        <h3>Syslog 接收规则</h3>
        <span>{isSaving ? "保存中..." : "仅接收 connect / disconnect"}</span>
      </div>
      <p className="panel__copy">{statusMessage}</p>

      <div className="syslog-rules-layout">
        <div className="syslog-rules-list">
          <div className="syslog-rules-list__header">
            <strong>{`启用规则 ${enabledRuleCount} 条`}</strong>
            <button
              type="button"
              className="button button--secondary"
              onClick={handleNewRule}
              disabled={isLoading || isSaving}
            >
              新建规则
            </button>
          </div>
          <div className="syslog-rules-list__items">
            {rules.map((rule) => (
              <button
                key={rule.id}
                type="button"
                className={`syslog-rule-card${selectedRuleId === rule.id ? " syslog-rule-card--active" : ""}`}
                onClick={() => handleSelectRule(rule)}
                disabled={isLoading || isSaving}
              >
                <strong>{rule.name}</strong>
                <span>{`${rule.enabled ? "启用" : "停用"} / ${rule.eventType}`}</span>
              </button>
            ))}
          </div>
        </div>

        <div className="syslog-rules-editor">
          <div className="syslog-rules-editor__grid">
            <label className="form-field">
              <span className="form-field__label">规则名称</span>
              <input
                className="form-field__control"
                value={draft.name}
                onChange={(event) => updateDraft("name", event.target.value)}
                disabled={isLoading || isSaving}
              />
            </label>

            <label className="form-field">
              <span className="form-field__label">事件类型</span>
              <select
                className="form-field__control"
                value={draft.eventType}
                onChange={(event) =>
                  updateDraft("eventType", event.target.value as "connect" | "disconnect")
                }
                disabled={isLoading || isSaving}
              >
                <option value="connect">connect</option>
                <option value="disconnect">disconnect</option>
              </select>
            </label>

            <label className="form-field form-field--checkbox">
              <input
                type="checkbox"
                checked={draft.enabled}
                onChange={(event) => updateDraft("enabled", event.target.checked)}
                disabled={isLoading || isSaving}
              />
              <span className="form-field__label">启用规则</span>
            </label>

            <label className="form-field form-field--wide">
              <span className="form-field__label">原始消息匹配正则</span>
              <textarea
                className="form-field__control form-field__control--multiline"
                value={draft.messagePattern}
                onChange={(event) => updateDraft("messagePattern", event.target.value)}
                disabled={isLoading || isSaving}
                rows={3}
              />
            </label>

            {[
              ["stationMacGroup", "站点 MAC 分组"],
              ["apMacGroup", "AP MAC 分组"],
              ["ssidGroup", "SSID 分组"],
              ["ipv4Group", "IPv4 分组"],
              ["ipv6Group", "IPv6 分组"],
              ["hostnameGroup", "主机名分组"],
              ["osVendorGroup", "OS 厂商分组"],
              ["eventTimeGroup", "事件时间分组"],
              ["eventTimeLayout", "事件时间格式"],
            ].map(([key, label]) => (
              <label key={key} className="form-field">
                <span className="form-field__label">{label}</span>
                <input
                  className="form-field__control"
                  value={draft[key as keyof SyslogReceiveRuleInput] as string}
                  onChange={(event) =>
                    updateDraft(
                      key as keyof SyslogReceiveRuleInput,
                      event.target.value,
                    )
                  }
                  disabled={isLoading || isSaving}
                />
              </label>
            ))}
          </div>

          {errorMessage ? <p role="alert" className="form-error">{errorMessage}</p> : null}

          <div className="syslog-rules-editor__actions">
            <button
              type="button"
              className="button button--secondary"
              onClick={() => void handleMove("up")}
              disabled={isLoading || isSaving || !selectedRule}
            >
              上移规则
            </button>
            <button
              type="button"
              className="button button--secondary"
              onClick={() => void handleMove("down")}
              disabled={isLoading || isSaving || !selectedRule}
            >
              下移规则
            </button>
            <button
              type="button"
              className="button button--primary"
              onClick={() => void handleSave()}
              disabled={isLoading || isSaving}
            >
              保存规则
            </button>
            {selectedRule ? (
              <button
                type="button"
                className="button button--secondary"
                onClick={() => void handleDelete()}
                disabled={isLoading || isSaving}
              >
                删除规则
              </button>
            ) : null}
          </div>

          <div className="syslog-rule-preview">
            <div className="syslog-rule-preview__header">
              <strong>规则命中预览</strong>
              <span>直接复用后端 matcher，不走前端假解析</span>
            </div>
            <label className="form-field">
              <span className="form-field__label">预览接收时间</span>
              <input
                className="form-field__control"
                type="datetime-local"
                value={previewReceivedAt}
                onChange={(event) => setPreviewReceivedAt(event.target.value)}
                disabled={isLoading || isSaving}
              />
            </label>
            <label className="form-field">
              <span className="form-field__label">预览原始消息</span>
              <textarea
                className="form-field__control form-field__control--multiline"
                rows={3}
                value={previewRawMessage}
                onChange={(event) => setPreviewRawMessage(event.target.value)}
                disabled={isLoading || isSaving}
              />
            </label>
            <div className="syslog-rules-editor__actions">
              <button
                type="button"
                className="button button--secondary"
                onClick={() => void handlePreview()}
                disabled={isLoading || isSaving}
              >
                预览命中
              </button>
            </div>
            {previewResult?.matched && previewResult.event ? (
              <div className="syslog-rule-preview__result">
                <strong>命中规则，已提取结构化字段</strong>
                <span>{previewResult.event.stationMac || "-"}</span>
                <span>{previewResult.event.eventType || "-"}</span>
                <span>{previewResult.event.hostname || "-"}</span>
              </div>
            ) : null}
            {previewResult && !previewResult.matched ? (
              <p className="panel__copy">未命中当前规则，请检查正则和分组映射</p>
            ) : null}
          </div>
        </div>
      </div>
    </article>
  );
}
