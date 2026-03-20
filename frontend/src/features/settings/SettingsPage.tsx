import { useEffect, useState } from "react";

import {
  getSettings,
  saveSettings,
  type SettingsRecord,
} from "../../lib/api";
import { SettingsForm } from "./components/SettingsForm";

export function SettingsPage() {
  const [settings, setSettings] = useState<SettingsRecord>({
    scannerRetryThreshold: 1,
    lateToleranceMinutes: 0,
    archiveRetentionDays: 1,
    manualCorrectionRequiresApproval: false,
  });
  const [isSaving, setIsSaving] = useState(false);
  const [statusMessage, setStatusMessage] = useState("等待配置装载...");

  useEffect(() => {
    let isActive = true;

    void (async () => {
      try {
        const loadedSettings = await getSettings();

        if (!isActive) {
          return;
        }

        setSettings(loadedSettings);
        setStatusMessage("已装载当前运行参数");
      } catch {
        if (isActive) {
          setStatusMessage("设置装载失败，请稍后重试");
        }
      }
    })();

    return () => {
      isActive = false;
    };
  }, []);

  async function handleSave(nextSettings: SettingsRecord) {
    setIsSaving(true);

    try {
      const savedSettings = await saveSettings(nextSettings);
      setSettings(savedSettings);
      setStatusMessage("设置已保存到本地 mock 控制面");
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
          <p>
            Placeholder admin space for runtime controls, policy toggles, and
            audit-aware changes.
          </p>
        </div>
      </header>

      <div className="page-grid page-grid--split">
        <article className="panel">
          <div className="panel__header">
            <h3>Control Modules</h3>
            <span>Mock save</span>
          </div>
          <p className="panel__copy">{statusMessage}</p>
          <div style={{ marginTop: "1rem" }}>
            <SettingsForm
              initialValues={settings}
              isSaving={isSaving}
              onSubmit={handleSave}
            />
          </div>
        </article>

        <article className="panel panel--tall">
          <div className="panel__header">
            <h3>Audit Locks</h3>
            <span>Immutable trail</span>
          </div>
          <p className="panel__copy">
            人工修正审批、扫描重试和归档保留策略都先写入本地 mock
            层，后续替换成真实接口时只需要收敛到 API 抽象。
          </p>
          <ul className="stack-list">
            <li>Scanner retry thresholds</li>
            <li>Supervisor approval policy</li>
            <li>Archive retention windows</li>
          </ul>
        </article>
      </div>
    </section>
  );
}
