import { useEffect, useState } from "react";

import {
  getSettings,
  saveSettings,
  type SystemSetting,
} from "../../lib/api";
import { SettingsForm, type SettingsFormValues } from "./components/SettingsForm";
import { SyslogRulesPanel } from "./components/SyslogRulesPanel";

const DEFAULT_VALUES: SettingsFormValues = {
  dayEndTime: "18:30",
  retentionDays: "45",
  feishuAppId: "",
  feishuAppSecret: "",
  feishuLocationName: "",
  reportTimeoutSeconds: "30",
  reportRetryLimit: "5",
};

function toValues(items: SystemSetting[]): SettingsFormValues {
  const settingsMap = new Map(
    items.map((item) => [item.settingKey, item.settingValue]),
  );

  return {
    dayEndTime: settingsMap.get("day_end_time") ?? DEFAULT_VALUES.dayEndTime,
    retentionDays:
      settingsMap.get("syslog_retention_days") ?? DEFAULT_VALUES.retentionDays,
    feishuAppId:
      settingsMap.get("feishu_app_id") ?? DEFAULT_VALUES.feishuAppId,
    feishuAppSecret:
      settingsMap.get("feishu_app_secret") ?? DEFAULT_VALUES.feishuAppSecret,
    feishuLocationName:
      settingsMap.get("feishu_location_name") ??
      DEFAULT_VALUES.feishuLocationName,
    reportTimeoutSeconds:
      settingsMap.get("report_timeout_seconds") ??
      DEFAULT_VALUES.reportTimeoutSeconds,
    reportRetryLimit:
      settingsMap.get("report_retry_limit") ?? DEFAULT_VALUES.reportRetryLimit,
  };
}

function toItems(values: SettingsFormValues): SystemSetting[] {
  return [
    { settingKey: "day_end_time", settingValue: values.dayEndTime },
    { settingKey: "syslog_retention_days", settingValue: values.retentionDays },
    { settingKey: "feishu_app_id", settingValue: values.feishuAppId },
    { settingKey: "feishu_app_secret", settingValue: values.feishuAppSecret },
    {
      settingKey: "feishu_location_name",
      settingValue: values.feishuLocationName,
    },
    {
      settingKey: "report_timeout_seconds",
      settingValue: values.reportTimeoutSeconds,
    },
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

  const configurationSummary = [
    {
      label: "日切时间",
      value: settings.dayEndTime || "未配置",
      detail: "用于日终归档与跨日计算",
    },
    {
      label: "日志保留",
      value: `${settings.retentionDays || "0"} 天`,
      detail: "控制原始日志清理窗口",
    },
    {
      label: "飞书集成",
      value:
        settings.feishuAppId && settings.feishuAppSecret && settings.feishuLocationName
          ? "已完成"
          : "待完善",
      detail: settings.feishuLocationName || "缺少应用凭据或打卡地点",
    },
    {
      label: "投递策略",
      value: `${settings.reportTimeoutSeconds || "0"}s / ${settings.reportRetryLimit || "0"} 次`,
      detail: "超时与失败重试上限",
    },
  ];

  return (
    <section className="page">
      <header className="page-header">
        <span className="page-header__eyebrow">运行参数</span>
        <div>
          <h2>系统设置</h2>
          <p>按分组方式维护系统运行参数，让日切、保留与上报链路具备更稳定的操作语义。</p>
        </div>
      </header>

      <div className="page-grid page-grid--split">
        <article className="panel">
          <div className="panel__header">
            <h3>运行参数</h3>
            <span>{isSaving ? "保存中..." : "后端同步"}</span>
          </div>
          <p className="panel__copy">{statusMessage}</p>
          <div className="panel__body-offset">
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
            <h3>当前配置摘要</h3>
            <span>实时映射</span>
          </div>
          <div className="settings-summary-list">
            {configurationSummary.map((item) => (
              <div key={item.label} className="settings-summary-card">
                <span>{item.label}</span>
                <strong>{item.value}</strong>
                <p>{item.detail}</p>
              </div>
            ))}
          </div>
          <div className="settings-note">
            <h4>字段说明</h4>
            <ul className="stack-list">
              <li>`day_end_time` 决定跨日归档窗口</li>
              <li>`syslog_retention_days` 控制日志保留天数</li>
              <li>`feishu_app_id`、`feishu_app_secret` 和 `feishu_location_name` 共同决定飞书同步是否可用</li>
              <li>`report_timeout_seconds` 与 `report_retry_limit` 控制投递超时和补偿重试</li>
            </ul>
          </div>
        </article>
      </div>

      <SyslogRulesPanel />
    </section>
  );
}
