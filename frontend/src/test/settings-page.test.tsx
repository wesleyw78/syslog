import "@testing-library/jest-dom/vitest";
import { fireEvent, render, screen } from "@testing-library/react";
import { beforeEach, describe, expect, it } from "vitest";

import { SettingsPage } from "../features/settings/SettingsPage";
import { resetMockData } from "../lib/api";

describe("settings page", () => {
  beforeEach(() => {
    resetMockData();
  });

  it("saves settings through the mock flow", async () => {
    render(<SettingsPage />);

    await screen.findByDisplayValue("3");

    fireEvent.change(screen.getByLabelText("扫描重试阈值"), {
      target: { value: "5" },
    });
    fireEvent.change(screen.getByLabelText("迟到容忍分钟"), {
      target: { value: "12" },
    });
    fireEvent.change(screen.getByLabelText("归档保留天数"), {
      target: { value: "60" },
    });
    fireEvent.click(screen.getByRole("button", { name: "保存设置" }));

    expect(await screen.findByText("设置已保存到本地 mock 控制面")).toBeInTheDocument();
    expect(screen.getByDisplayValue("5")).toBeInTheDocument();
    expect(screen.getByDisplayValue("12")).toBeInTheDocument();
    expect(screen.getByDisplayValue("60")).toBeInTheDocument();
  });

  it("rejects invalid numeric settings", async () => {
    render(<SettingsPage />);

    await screen.findByDisplayValue("3");

    fireEvent.change(screen.getByLabelText("扫描重试阈值"), {
      target: { value: "0" },
    });
    fireEvent.change(screen.getByLabelText("归档保留天数"), {
      target: { value: "-1" },
    });
    fireEvent.click(screen.getByRole("button", { name: "保存设置" }));

    expect(await screen.findByText("设置数值不合法")).toBeInTheDocument();
    expect(screen.queryByText("设置已保存到本地 mock 控制面")).not.toBeInTheDocument();
  });
});
