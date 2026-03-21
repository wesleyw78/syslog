import "@testing-library/jest-dom/vitest";
import { fireEvent, render, screen } from "@testing-library/react";
import { afterEach, describe, expect, it, vi } from "vitest";

import { SettingsPage } from "../features/settings/SettingsPage";
import { mockJsonFetch } from "./fetchMock";

afterEach(() => {
  vi.unstubAllGlobals();
  vi.restoreAllMocks();
});

describe("settings page", () => {
  it("keeps the form disabled until settings are loaded", async () => {
    mockJsonFetch([
      {
        method: "GET",
        path: "/api/settings",
        response: {
          items: [
            { ID: 1, SettingKey: "day_end_time", SettingValue: "18:30" },
            { ID: 2, SettingKey: "syslog_retention_days", SettingValue: "45" },
            { ID: 3, SettingKey: "report_target_url", SettingValue: "https://reports.example.com/inbox" },
            { ID: 4, SettingKey: "report_timeout_seconds", SettingValue: "30" },
            { ID: 5, SettingKey: "report_retry_limit", SettingValue: "5" },
          ],
        },
      },
    ]);

    render(<SettingsPage />);

    expect(
      screen.getByRole("button", { name: "保存设置" }),
    ).toBeDisabled();

    expect(await screen.findByText("已装载当前运行参数")).toBeInTheDocument();
    expect(
      screen.getByRole("button", { name: "保存设置" }),
    ).not.toBeDisabled();
  });

  it("loads settings and saves the mapped API payload", async () => {
    const { fetchMock, requests } = mockJsonFetch([
      {
        method: "GET",
        path: "/api/settings",
        response: {
          items: [
            { ID: 1, SettingKey: "day_end_time", SettingValue: "18:30" },
            { ID: 2, SettingKey: "syslog_retention_days", SettingValue: "45" },
            { ID: 3, SettingKey: "report_target_url", SettingValue: "https://reports.example.com/inbox" },
            { ID: 4, SettingKey: "report_timeout_seconds", SettingValue: "30" },
            { ID: 5, SettingKey: "report_retry_limit", SettingValue: "5" },
          ],
        },
      },
      {
        method: "PUT",
        path: "/api/settings",
        response: {
          items: [
            { ID: 1, SettingKey: "day_end_time", SettingValue: "19:00" },
            { ID: 2, SettingKey: "syslog_retention_days", SettingValue: "60" },
            { ID: 3, SettingKey: "report_target_url", SettingValue: "https://reports.example.com/alerts" },
            { ID: 4, SettingKey: "report_timeout_seconds", SettingValue: "45" },
            { ID: 5, SettingKey: "report_retry_limit", SettingValue: "7" },
          ],
        },
        assertBody: (body) => {
          expect(body).toEqual({
            items: [
              { settingKey: "day_end_time", settingValue: "19:00" },
              { settingKey: "syslog_retention_days", settingValue: "60" },
              { settingKey: "report_target_url", settingValue: "https://reports.example.com/alerts" },
              { settingKey: "report_timeout_seconds", settingValue: "45" },
              { settingKey: "report_retry_limit", settingValue: "7" },
            ],
          });
        },
      },
    ]);

    render(<SettingsPage />);

    expect(await screen.findByText("已装载当前运行参数")).toBeInTheDocument();
    expect(screen.getByLabelText("日切时间")).toHaveValue("18:30");

    fireEvent.input(screen.getByLabelText("日切时间"), {
      target: { value: "19:00" },
    });
    fireEvent.change(screen.getByLabelText("日志保留天数"), {
      target: { value: "60" },
    });
    fireEvent.change(screen.getByLabelText("报告目标地址"), {
      target: { value: "https://reports.example.com/alerts" },
    });
    fireEvent.change(screen.getByLabelText("报告超时秒数"), {
      target: { value: "45" },
    });
    fireEvent.change(screen.getByLabelText("重试次数"), {
      target: { value: "7" },
    });
    fireEvent.click(screen.getByRole("button", { name: "保存设置" }));

    expect(await screen.findByText("设置已保存到后端")).toBeInTheDocument();
    expect(fetchMock.mock.calls).toHaveLength(2);
    expect(requests[1]?.body).toEqual({
      items: [
        { settingKey: "day_end_time", settingValue: "19:00" },
        { settingKey: "syslog_retention_days", settingValue: "60" },
        { settingKey: "report_target_url", settingValue: "https://reports.example.com/alerts" },
        { settingKey: "report_timeout_seconds", settingValue: "45" },
        { settingKey: "report_retry_limit", settingValue: "7" },
      ],
    });
  });

  it("rejects invalid setting values", async () => {
    mockJsonFetch([
      {
        method: "GET",
        path: "/api/settings",
        response: {
          items: [
            { ID: 1, SettingKey: "day_end_time", SettingValue: "18:30" },
            { ID: 2, SettingKey: "syslog_retention_days", SettingValue: "45" },
            { ID: 3, SettingKey: "report_target_url", SettingValue: "https://reports.example.com/inbox" },
            { ID: 4, SettingKey: "report_timeout_seconds", SettingValue: "30" },
            { ID: 5, SettingKey: "report_retry_limit", SettingValue: "5" },
          ],
        },
      },
    ]);

    render(<SettingsPage />);

    await screen.findByText("已装载当前运行参数");

    fireEvent.change(screen.getByLabelText("日切时间"), {
      target: { value: "25:99" },
    });
    fireEvent.change(screen.getByLabelText("日志保留天数"), {
      target: { value: "0" },
    });
    fireEvent.click(screen.getByRole("button", { name: "保存设置" }));

    expect(await screen.findByText("设置数值不合法")).toBeInTheDocument();
  });

  it("keeps save disabled when the initial settings load fails", async () => {
    mockJsonFetch([
      {
        method: "GET",
        path: "/api/settings",
        response: { message: "boom" },
        status: 500,
      },
    ]);

    render(<SettingsPage />);

    expect(await screen.findByText("设置装载失败，请稍后重试")).toBeInTheDocument();
    expect(
      screen.getByRole("button", { name: "保存设置" }),
    ).toBeDisabled();
  });
});
