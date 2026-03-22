import react from "@vitejs/plugin-react";
import { defineConfig, loadEnv } from "vite";
import { resolveApiProxyTarget } from "./src/lib/devProxy";

export default defineConfig(({ mode }) => {
  const env = loadEnv(mode, ".", "");

  return {
    plugins: [react()],
    server: {
      proxy: {
        "/api": {
          target: resolveApiProxyTarget(env.VITE_API_PROXY_TARGET),
          changeOrigin: true,
        },
      },
    },
    test: {
      css: true,
      environment: "jsdom",
      globals: true,
    },
  };
});
