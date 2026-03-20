import "@testing-library/jest-dom/vitest";
import { fireEvent, render, screen, within } from "@testing-library/react";
import { beforeEach, describe, expect, it } from "vitest";

import { AttendancePage } from "../features/attendance/AttendancePage";
import { resetMockData } from "../lib/api";

describe("attendance page", () => {
  beforeEach(() => {
    resetMockData();
  });

  it("shows manual correction action for exception rows", async () => {
    render(<AttendancePage />);

    const exceptionRow = await screen.findByRole("group", {
      name: /Arjun Patel attendance row/i,
    });
    const normalRow = await screen.findByRole("group", {
      name: /Lena Wu attendance row/i,
    });

    expect(
      within(exceptionRow).getByRole("button", { name: "人工修正" }),
    ).toBeInTheDocument();
    expect(
      within(normalRow).queryByRole("button", { name: "人工修正" }),
    ).not.toBeInTheDocument();

    fireEvent.click(
      within(exceptionRow).getByRole("button", { name: "人工修正" }),
    );

    expect(await within(exceptionRow).findByText("已修正")).toBeInTheDocument();
    expect(await screen.findByText("已提交 Arjun Patel 的人工修正")).toBeInTheDocument();
    expect(
      within(exceptionRow).queryByRole("button", { name: "人工修正" }),
    ).not.toBeInTheDocument();
  });
});
