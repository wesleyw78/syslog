import "@testing-library/jest-dom/vitest";
import { render, screen, within } from "@testing-library/react";
import { describe, expect, it } from "vitest";

import { AttendancePage } from "../features/attendance/AttendancePage";

describe("attendance page", () => {
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
  });
});
