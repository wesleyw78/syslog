import { useEffect, useState } from "react";

import {
  getSettings,
  saveSettings,
  type SystemSetting,
} from "../../lib/api";
import { SettingsForm, type SettingsFormValues } from "./components/SettingsForm";

const DEFAULT_VALUES: SettingsFormValues = {
  dayEndTime: "18:30",
  retentionDays: "45",
  reportTargetUrl: "",
  reportTimeoutSeconds: "30",
  reportRetryLimit: "5",
};

function toValues(items: SystemSetting[]): SettingsFormValues {
  const settingsMap = new Map(items.map((item) => [item.settingKey, item.settingValue]));

  return {
    dayEndTime: settingsMap.get("day_end_time") ?? DEFAULT_VALUES.dayEndTime,
    retentionDays:
      settingsMap.get("syslog_retention_days") ?? DEFAULT_VALUES.retentionDays,
    reportTargetUrl:
      settingsMap.get("report_target_url") ?? DEFAULT_VALUES.reportTargetUrl,
    reportTimeoutSeconds:
      settingsMap.get("report_timeout_seconds") ?? DEFAULT_VALUES.reportTimeoutSeconds,
    reportRetryLimit:
      settingsMap.get("report_retry_limit") ?? DEFAULT_VALUES.reportRetryLimit,
  };
}

function toItems(values: SettingsFormValues): SystemSetting[] {
  return [
    { settingKey: "day_end_time", settingValue: values.dayEndTime },
    { settingKey: "syslog_retention_days", settingValue: values.retentionDays },
    { settingKey: "report_target_url", settingValue: values.reportTargetUrl },
    { settingKey: "report_timeout_seconds", settingValue: values.reportTimeoutSeconds },
    { settingKey: "report_retry_limit", settingValue: values.reportRetryLimit },
  ];
}

export function SettingsPage() {
  const [settings, setSettings] = useState<SettingsFormValues>(DEFAULT_VALUES);
  const [hasLoadedSettings, setHasLoadedSettings] = useState(false);
  const [isLoading, setIsLoading] = useState(true);
  const [isSaving, setIsSaving] = useState(false);
  const [statusMessage, setStatusMessage] = useState("加载系统设置...");

  useEffect(() => {
    let isActive = true;

    void (async () => {
      try {
        const loadedSettings = await getSettings();

        if (!isActive) {
          return;
        }

        setSettings(toValues(loadedSettings));
        setHasLoadedSettings(true);
        setStatusMessage("已装载当前运行参数");
      } catch {
        if (isActive) {
          setHasLoadedSettings(false);
          setStatusMessage("设置装载失败，请稍后重试");
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
  }, []);

  async function handleSave(nextSettings: SettingsFormValues) {
    setIsSaving(true);

    try {
      const savedSettings = await saveSettings(toItems(nextSettings));
      setSettings(toValues(savedSettings));
      setStatusMessage("设置已保存到后端");
    } catch {
      setStatusMessage("设置保存失败，请稍后重试");
    } finally {
      setIsSaving(false);
    }
  }

  return (
    <section className="page">
      <header className="page-header">
        <span className="page-header__eyebrow">Configuration</span>
        <div>
          <h2>Settings</h2>
          <p>真实系统配置管理，覆盖日切、保留和报告投递参数。</p>
        </div>
      </header>

      <div className="page-grid page-grid--split">
        <article className="panel">
          <div className="panel__header">
            <h3>系统参数</h3>
            <span>{isSaving ? "保存中..." : "后端同步"}</span>
          </div>
          <p className="panel__copy">{statusMessage}</p>
          <div style={{ marginTop: "1rem" }}>
            <SettingsForm
              initialValues={settings}
              isDisabled={isLoading || !hasLoadedSettings}
              isSaving={isSaving}
              onSubmit={handleSave}
            />
          </div>
        </article>

        <article className="panel panel--tall">
          <div className="panel__header">
            <h3>配置说明</h3>
            <span>当前 keys</span>
          </div>
          <p className="panel__copy">
            `day_end_time`、`syslog_retention_days`、`report_target_url`、
            `report_timeout_seconds` 和 `report_retry_limit` 已对齐真实后端字段。
          </p>
          <ul className="stack-list">
            <li>日切时间用于跨日归档和报表分段</li>
            <li>保留天数控制日志与报表清理窗口</li>
            <li>报告配置控制告警投递链路</li>
          </ul>
        </article>
      </div>
    </section>
  );
}
