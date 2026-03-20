import "@testing-library/jest-dom/vitest";
import { render, screen } from "@testing-library/react";
import { createMemoryRouter, RouterProvider } from "react-router-dom";
import { describe, expect, it } from "vitest";

import { appRoutes } from "../app/router";

describe("console router", () => {
  it("renders all five navigation items on the dashboard route", () => {
    const router = createMemoryRouter(appRoutes, {
      initialEntries: ["/"],
    });

    render(<RouterProvider router={router} />);

    expect(
      screen.getByRole("link", { name: /dashboard plant pulse and active alerts/i }),
    ).toBeInTheDocument();
    expect(
      screen.getByRole("link", { name: /logs ingestion stream and exception tail/i }),
    ).toBeInTheDocument();
    expect(
      screen.getByRole("link", { name: /employees roster status and certifications/i }),
    ).toBeInTheDocument();
    expect(
      screen.getByRole("link", { name: /attendance shift coverage and check-in drift/i }),
    ).toBeInTheDocument();
    expect(
      screen.getByRole("link", { name: /settings runtime controls and audit locks/i }),
    ).toBeInTheDocument();
    expect(screen.getByRole("heading", { name: "Dashboard" })).toBeInTheDocument();
  });

  it("renders the logs page on a non-default route", () => {
    const router = createMemoryRouter(appRoutes, {
      initialEntries: ["/logs"],
    });

    render(<RouterProvider router={router} />);

    expect(screen.getByRole("heading", { name: "Logs" })).toBeInTheDocument();
    expect(
      screen.getByRole("link", { name: /logs ingestion stream and exception tail/i }),
    ).toHaveAttribute("aria-current", "page");
  });
});
