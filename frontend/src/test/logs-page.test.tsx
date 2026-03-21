import "@testing-library/jest-dom/vitest";
import { render, screen } from "@testing-library/react";
import { afterEach, describe, expect, it, vi } from "vitest";

import { LogsPage } from "../features/logs/LogsPage";
import { mockJsonFetch } from "./fetchMock";

afterEach(() => {
  vi.unstubAllGlobals();
  vi.restoreAllMocks();
});

describe("logs page", () => {
  it("loads logs from the API and exposes message and event fields", async () => {
    const { fetchMock } = mockJsonFetch([
      {
        method: "GET",
        path: "/api/logs",
        response: {
          items: [
            {
              message: {
                ID: 10,
                ReceivedAt: "2026-03-21T08:01:00Z",
                LogTime: "2026-03-21T08:01:00Z",
                ParseStatus: "parsed",
                RawMessage: "gate-1 access granted",
              },
              event: {
                ID: 20,
                EventType: "badge.scan",
                StationMac: "AA:BB:CC:DD:EE:01",
                Hostname: "gate-1",
              },
            },
          ],
        },
      },
    ]);

    render(<LogsPage />);

    expect(await screen.findByText("gate-1 access granted")).toBeInTheDocument();
    expect(screen.getByText("parsed")).toBeInTheDocument();
    expect(screen.getByText("badge.scan")).toBeInTheDocument();
    expect(screen.getByText("AA:BB:CC:DD:EE:01")).toBeInTheDocument();
    expect(screen.getByText("gate-1")).toBeInTheDocument();
    expect(fetchMock.mock.calls).toHaveLength(1);
  });
});
