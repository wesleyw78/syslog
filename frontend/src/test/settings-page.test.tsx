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
            { ID: 3, SettingKey: "feishu_app_id", SettingValue: "cli_xxx" },
            { ID: 4, SettingKey: "feishu_app_secret", SettingValue: "secret_xxx" },
            { ID: 6, SettingKey: "feishu_location_name", SettingValue: "总部办公区" },
            { ID: 7, SettingKey: "report_timeout_seconds", SettingValue: "30" },
            { ID: 8, SettingKey: "report_retry_limit", SettingValue: "5" },
          ],
        },
      },
      {
        method: "GET",
        path: "/api/syslog-rules",
        response: { items: [] },
      },
    ]);

    render(<SettingsPage />);

    expect(screen.getByRole("heading", { name: "系统设置" })).toBeInTheDocument();
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
            { ID: 3, SettingKey: "feishu_app_id", SettingValue: "cli_xxx" },
            { ID: 4, SettingKey: "feishu_app_secret", SettingValue: "secret_xxx" },
            { ID: 6, SettingKey: "feishu_location_name", SettingValue: "总部办公区" },
            { ID: 7, SettingKey: "report_timeout_seconds", SettingValue: "30" },
            { ID: 8, SettingKey: "report_retry_limit", SettingValue: "5" },
          ],
        },
      },
      {
        method: "GET",
        path: "/api/syslog-rules",
        response: { items: [] },
      },
      {
        method: "PUT",
        path: "/api/settings",
        response: {
          items: [
            { ID: 1, SettingKey: "day_end_time", SettingValue: "19:00" },
            { ID: 2, SettingKey: "syslog_retention_days", SettingValue: "60" },
            { ID: 3, SettingKey: "feishu_app_id", SettingValue: "cli_yyy" },
            { ID: 4, SettingKey: "feishu_app_secret", SettingValue: "secret_yyy" },
            { ID: 6, SettingKey: "feishu_location_name", SettingValue: "一号门考勤点" },
            { ID: 7, SettingKey: "report_timeout_seconds", SettingValue: "45" },
            { ID: 8, SettingKey: "report_retry_limit", SettingValue: "7" },
          ],
        },
        assertBody: (body) => {
          expect(body).toEqual({
            items: [
              { settingKey: "day_end_time", settingValue: "19:00" },
              { settingKey: "syslog_retention_days", settingValue: "60" },
              { settingKey: "feishu_app_id", settingValue: "cli_yyy" },
              { settingKey: "feishu_app_secret", settingValue: "secret_yyy" },
              { settingKey: "feishu_location_name", settingValue: "一号门考勤点" },
              { settingKey: "report_timeout_seconds", settingValue: "45" },
              { settingKey: "report_retry_limit", settingValue: "7" },
            ],
          });
        },
      },
    ]);

    render(<SettingsPage />);

    expect(screen.getByRole("heading", { name: "系统设置" })).toBeInTheDocument();
    expect(await screen.findByText("已装载当前运行参数")).toBeInTheDocument();
    expect(screen.getByLabelText("日切时间")).toHaveValue("18:30");

    fireEvent.input(screen.getByLabelText("日切时间"), {
      target: { value: "19:00" },
    });
    fireEvent.change(screen.getByLabelText("日志保留天数"), {
      target: { value: "60" },
    });
    fireEvent.change(screen.getByLabelText("Feishu App ID"), {
      target: { value: "cli_yyy" },
    });
    fireEvent.change(screen.getByLabelText("Feishu App Secret"), {
      target: { value: "secret_yyy" },
    });
    fireEvent.change(screen.getByLabelText("打卡地点名称"), {
      target: { value: "一号门考勤点" },
    });
    fireEvent.change(screen.getByLabelText("报告超时秒数"), {
      target: { value: "45" },
    });
    fireEvent.change(screen.getByLabelText("重试次数"), {
      target: { value: "7" },
    });
    fireEvent.click(screen.getByRole("button", { name: "保存设置" }));

    expect(await screen.findByText("设置已保存到后端")).toBeInTheDocument();
    expect(fetchMock.mock.calls).toHaveLength(3);
    expect(requests[2]?.body).toEqual({
      items: [
        { settingKey: "day_end_time", settingValue: "19:00" },
        { settingKey: "syslog_retention_days", settingValue: "60" },
        { settingKey: "feishu_app_id", settingValue: "cli_yyy" },
        { settingKey: "feishu_app_secret", settingValue: "secret_yyy" },
        { settingKey: "feishu_location_name", settingValue: "一号门考勤点" },
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
            { ID: 3, SettingKey: "feishu_app_id", SettingValue: "cli_xxx" },
            { ID: 4, SettingKey: "feishu_app_secret", SettingValue: "secret_xxx" },
            { ID: 6, SettingKey: "feishu_location_name", SettingValue: "总部办公区" },
            { ID: 7, SettingKey: "report_timeout_seconds", SettingValue: "30" },
            { ID: 8, SettingKey: "report_retry_limit", SettingValue: "5" },
          ],
        },
      },
      {
        method: "GET",
        path: "/api/syslog-rules",
        response: { items: [] },
      },
    ]);

    render(<SettingsPage />);

    expect(screen.getByRole("heading", { name: "系统设置" })).toBeInTheDocument();
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

  it("rejects an out-of-range time even when other values are valid", async () => {
    mockJsonFetch([
      {
        method: "GET",
        path: "/api/settings",
        response: {
          items: [
            { ID: 1, SettingKey: "day_end_time", SettingValue: "18:30" },
            { ID: 2, SettingKey: "syslog_retention_days", SettingValue: "45" },
            { ID: 3, SettingKey: "feishu_app_id", SettingValue: "cli_xxx" },
            { ID: 4, SettingKey: "feishu_app_secret", SettingValue: "secret_xxx" },
            { ID: 6, SettingKey: "feishu_location_name", SettingValue: "总部办公区" },
            { ID: 7, SettingKey: "report_timeout_seconds", SettingValue: "30" },
            { ID: 8, SettingKey: "report_retry_limit", SettingValue: "5" },
          ],
        },
      },
      {
        method: "GET",
        path: "/api/syslog-rules",
        response: { items: [] },
      },
    ]);

    render(<SettingsPage />);

    await screen.findByText("已装载当前运行参数");

    fireEvent.input(screen.getByLabelText("日切时间"), {
      target: { value: "25:99" },
    });
    fireEvent.change(screen.getByLabelText("日志保留天数"), {
      target: { value: "45" },
    });
    fireEvent.change(screen.getByLabelText("Feishu App ID"), {
      target: { value: "cli_xxx" },
    });
    fireEvent.change(screen.getByLabelText("Feishu App Secret"), {
      target: { value: "secret_xxx" },
    });
    fireEvent.change(screen.getByLabelText("打卡地点名称"), {
      target: { value: "总部办公区" },
    });
    fireEvent.change(screen.getByLabelText("报告超时秒数"), {
      target: { value: "30" },
    });
    fireEvent.change(screen.getByLabelText("重试次数"), {
      target: { value: "5" },
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
      {
        method: "GET",
        path: "/api/syslog-rules",
        response: { items: [] },
      },
    ]);

    render(<SettingsPage />);

    expect(screen.getByRole("heading", { name: "系统设置" })).toBeInTheDocument();
    expect(await screen.findByText("设置装载失败，请稍后重试")).toBeInTheDocument();
    expect(
      screen.getByRole("button", { name: "保存设置" }),
    ).toBeDisabled();
  });

  it("loads syslog rules and allows creating a new rule", async () => {
    const { requests } = mockJsonFetch([
      {
        method: "GET",
        path: "/api/settings",
        response: {
          items: [
            { ID: 1, SettingKey: "day_end_time", SettingValue: "18:30" },
            { ID: 2, SettingKey: "syslog_retention_days", SettingValue: "45" },
            { ID: 3, SettingKey: "feishu_app_id", SettingValue: "cli_xxx" },
            { ID: 4, SettingKey: "feishu_app_secret", SettingValue: "secret_xxx" },
            { ID: 6, SettingKey: "feishu_location_name", SettingValue: "总部办公区" },
            { ID: 7, SettingKey: "report_timeout_seconds", SettingValue: "30" },
            { ID: 8, SettingKey: "report_retry_limit", SettingValue: "5" },
          ],
        },
      },
      {
        method: "GET",
        path: "/api/syslog-rules",
        response: {
          items: [
            {
              id: 11,
              name: "默认 connect 规则",
              enabled: true,
              eventType: "connect",
              messagePattern: "connect Station\\[(?P<station_mac>[^\\]]+)\\]",
              stationMacGroup: "station_mac",
              apMacGroup: "",
              ssidGroup: "",
              ipv4Group: "",
              ipv6Group: "",
              hostnameGroup: "",
              osVendorGroup: "",
              eventTimeGroup: "",
              eventTimeLayout: "",
            },
          ],
        },
      },
      {
        method: "POST",
        path: "/api/syslog-rules",
        response: {
          id: 12,
          name: "默认 disconnect 规则",
          enabled: true,
          eventType: "disconnect",
          messagePattern: "disconnect Station\\[(?P<station_mac>[^\\]]+)\\]",
          stationMacGroup: "station_mac",
          apMacGroup: "",
          ssidGroup: "",
          ipv4Group: "",
          ipv6Group: "",
          hostnameGroup: "",
          osVendorGroup: "",
          eventTimeGroup: "",
          eventTimeLayout: "",
        },
        assertBody: (body) => {
          expect(body).toEqual({
            name: "默认 disconnect 规则",
            enabled: true,
            eventType: "disconnect",
            messagePattern: "disconnect Station\\[(?P<station_mac>[^\\]]+)\\]",
            stationMacGroup: "station_mac",
            apMacGroup: "",
            ssidGroup: "",
            ipv4Group: "",
            ipv6Group: "",
            hostnameGroup: "",
            osVendorGroup: "",
            eventTimeGroup: "",
            eventTimeLayout: "",
          });
        },
      },
    ]);

    render(<SettingsPage />);

    expect(await screen.findByRole("heading", { name: "Syslog 接收规则" })).toBeInTheDocument();
    expect(screen.getByText("默认 connect 规则")).toBeInTheDocument();
    expect(screen.getByText("启用规则 1 条")).toBeInTheDocument();

    fireEvent.click(screen.getByRole("button", { name: "新建规则" }));
    fireEvent.change(screen.getByLabelText("规则名称"), {
      target: { value: "默认 disconnect 规则" },
    });
    fireEvent.change(screen.getByLabelText("事件类型"), {
      target: { value: "disconnect" },
    });
    fireEvent.change(screen.getByLabelText("原始消息匹配正则"), {
      target: { value: "disconnect Station\\[(?P<station_mac>[^\\]]+)\\]" },
    });
    fireEvent.change(screen.getByLabelText("站点 MAC 分组"), {
      target: { value: "station_mac" },
    });
    fireEvent.click(screen.getByRole("button", { name: "保存规则" }));

    expect(await screen.findByText("规则已保存")).toBeInTheDocument();
    expect(requests[2]?.body).toEqual({
      name: "默认 disconnect 规则",
      enabled: true,
      eventType: "disconnect",
      messagePattern: "disconnect Station\\[(?P<station_mac>[^\\]]+)\\]",
      stationMacGroup: "station_mac",
      apMacGroup: "",
      ssidGroup: "",
      ipv4Group: "",
      ipv6Group: "",
      hostnameGroup: "",
      osVendorGroup: "",
      eventTimeGroup: "",
      eventTimeLayout: "",
    });
  });

  it("supports moving and previewing syslog rules", async () => {
    const { requests } = mockJsonFetch([
      {
        method: "GET",
        path: "/api/settings",
        response: {
          items: [
            { ID: 1, SettingKey: "day_end_time", SettingValue: "18:30" },
            { ID: 2, SettingKey: "syslog_retention_days", SettingValue: "45" },
            { ID: 3, SettingKey: "feishu_app_id", SettingValue: "cli_xxx" },
            { ID: 4, SettingKey: "feishu_app_secret", SettingValue: "secret_xxx" },
            { ID: 6, SettingKey: "feishu_location_name", SettingValue: "总部办公区" },
            { ID: 7, SettingKey: "report_timeout_seconds", SettingValue: "30" },
            { ID: 8, SettingKey: "report_retry_limit", SettingValue: "5" },
          ],
        },
      },
      {
        method: "GET",
        path: "/api/syslog-rules",
        response: {
          items: [
            {
              id: 11,
              name: "默认 connect 规则",
              enabled: true,
              eventType: "connect",
              messagePattern: "connect Station\\[(?P<station_mac>[^\\]]+)\\]",
              stationMacGroup: "station_mac",
              apMacGroup: "",
              ssidGroup: "",
              ipv4Group: "",
              ipv6Group: "",
              hostnameGroup: "",
              osVendorGroup: "",
              eventTimeGroup: "",
              eventTimeLayout: "",
            },
            {
              id: 12,
              name: "默认 disconnect 规则",
              enabled: true,
              eventType: "disconnect",
              messagePattern: "disconnect Station\\[(?P<station_mac>[^\\]]+)\\]",
              stationMacGroup: "station_mac",
              apMacGroup: "",
              ssidGroup: "",
              ipv4Group: "",
              ipv6Group: "",
              hostnameGroup: "",
              osVendorGroup: "",
              eventTimeGroup: "",
              eventTimeLayout: "",
            },
          ],
        },
      },
      {
        method: "POST",
        path: "/api/syslog-rules/11/move",
        response: {
          id: 11,
          name: "默认 connect 规则",
          enabled: true,
          eventType: "connect",
          messagePattern: "connect Station\\[(?P<station_mac>[^\\]]+)\\]",
          stationMacGroup: "station_mac",
          apMacGroup: "",
          ssidGroup: "",
          ipv4Group: "",
          ipv6Group: "",
          hostnameGroup: "",
          osVendorGroup: "",
          eventTimeGroup: "",
          eventTimeLayout: "",
        },
        assertBody: (body) => {
          expect(body).toEqual({ direction: "down" });
        },
      },
      {
        method: "POST",
        path: "/api/syslog-rules/preview",
        response: {
          matched: true,
          event: {
            eventType: "connect",
            stationMac: "aa:bb:cc:dd:ee:ff",
            hostname: "scanner-01",
          },
        },
      },
    ]);

    render(<SettingsPage />);

    expect(await screen.findByRole("heading", { name: "Syslog 接收规则" })).toBeInTheDocument();

    fireEvent.click(screen.getByRole("button", { name: "下移规则" }));
    expect(await screen.findByText("规则顺序已更新")).toBeInTheDocument();

    fireEvent.change(screen.getByLabelText("预览原始消息"), {
      target: {
        value: "Mar 22 09:15:00 stamgr: client_footprints connect Station[aa:bb:cc:dd:ee:ff]",
      },
    });
    fireEvent.click(screen.getByRole("button", { name: "预览命中" }));

    expect(await screen.findByText("命中规则，已提取结构化字段")).toBeInTheDocument();
    expect(screen.getByText("aa:bb:cc:dd:ee:ff")).toBeInTheDocument();
    expect(requests[2]?.body).toEqual({ direction: "down" });
    expect(requests[3]?.body).toEqual({
      receivedAt: expect.any(String),
      rawMessage: "Mar 22 09:15:00 stamgr: client_footprints connect Station[aa:bb:cc:dd:ee:ff]",
      rule: {
        name: "默认 connect 规则",
        enabled: true,
        eventType: "connect",
        messagePattern: "connect Station\\[(?P<station_mac>[^\\]]+)\\]",
        stationMacGroup: "station_mac",
        apMacGroup: "",
        ssidGroup: "",
        ipv4Group: "",
        ipv6Group: "",
        hostnameGroup: "",
        osVendorGroup: "",
        eventTimeGroup: "",
        eventTimeLayout: "",
      },
    });
  });
});
