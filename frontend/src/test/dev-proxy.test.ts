import { describe, expect, it } from "vitest";

import { resolveApiProxyTarget } from "../lib/devProxy";

describe("vite proxy target", () => {
  it("defaults to localhost for host-side development", () => {
    expect(resolveApiProxyTarget()).toBe("http://127.0.0.1:8080");
  });

  it("uses the configured backend target when provided", () => {
    expect(resolveApiProxyTarget("http://backend:8080")).toBe("http://backend:8080");
  });
});
