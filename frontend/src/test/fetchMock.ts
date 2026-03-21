import { vi } from "vitest";

type MockRoute = {
  method: string;
  path: string | RegExp;
  response: unknown;
  status?: number;
  assertBody?: (body: unknown) => void;
};

type RecordedRequest = {
  body: unknown;
  method: string;
  url: string;
};

function toResponseBody(initBody: BodyInit | null | undefined): unknown {
  if (typeof initBody !== "string") {
    return initBody ?? null;
  }

  try {
    return JSON.parse(initBody);
  } catch {
    return initBody;
  }
}

function matchesPath(path: string | RegExp, url: string): boolean {
  return typeof path === "string" ? url === path : path.test(url);
}

export function mockJsonFetch(routes: MockRoute[]) {
  const pendingRoutes = [...routes];
  const requests: RecordedRequest[] = [];

  const fetchMock = vi.fn(async (input: RequestInfo | URL, init?: RequestInit) => {
    const url = typeof input === "string" ? input : input.toString();
    const method =
      (init?.method ?? (typeof input !== "string" && !(input instanceof URL) ? input.method : "GET")).toUpperCase();
    const body = toResponseBody(init?.body as BodyInit | null | undefined);

    requests.push({ body, method, url });

    const routeIndex = pendingRoutes.findIndex(
      (route) => route.method === method && matchesPath(route.path, url),
    );

    if (routeIndex === -1) {
      throw new Error(`Unexpected request: ${method} ${url}`);
    }

    const [route] = pendingRoutes.splice(routeIndex, 1);
    route.assertBody?.(body);

    return new Response(JSON.stringify(route.response), {
      status: route.status ?? 200,
      headers: {
        "Content-Type": "application/json",
      },
    });
  });

  vi.stubGlobal("fetch", fetchMock);

  return {
    fetchMock,
    requests,
    assertAllMatched() {
      if (pendingRoutes.length > 0) {
        throw new Error(`Unmatched mock routes: ${pendingRoutes.map((route) => `${route.method} ${route.path.toString()}`).join(", ")}`);
      }
    },
  };
}
