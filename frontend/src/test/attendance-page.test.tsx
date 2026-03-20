import "@testing-library/jest-dom/vitest";
import { render, screen } from "@testing-library/react";
import { describe, expect, it } from "vitest";

import { AttendancePage } from "../features/attendance/AttendancePage";

describe("attendance page", () => {
  it("shows manual correction action for exception rows", async () => {
    render(<AttendancePage />);

    expect(await screen.findByText("人工修正")).toBeInTheDocument();
  });
});
