const DEFAULT_API_PROXY_TARGET = "http://127.0.0.1:8080";

export function resolveApiProxyTarget(target = ""): string {
  const normalizedTarget = target.trim();

  return normalizedTarget || DEFAULT_API_PROXY_TARGET;
}

export { DEFAULT_API_PROXY_TARGET };
