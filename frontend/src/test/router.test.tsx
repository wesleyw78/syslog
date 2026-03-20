import "@testing-library/jest-dom/vitest";
import { render, screen } from "@testing-library/react";
import { describe, expect, it } from "vitest";

import { AppShell } from "../app/layout/AppShell";

describe("AppShell", () => {
  it("renders dashboard nav item", () => {
    render(<AppShell />);

    expect(screen.getByText("Dashboard")).toBeInTheDocument();
  });
});
