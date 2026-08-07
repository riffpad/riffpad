import { defineConfig } from "vite";
import react from "@vitejs/plugin-react";
import type { Plugin } from "vite";

const devTarget = process.env.RIFFPAD_DEV_TARGET || "https://api.riffpad.ai";

// In dev, when talking to the cloud relay, inject RIFFPAD_RELAY so the client
// boots into relay mode automatically (same as the production web app).
function relayInject(): Plugin {
  return {
    name: "dev-relay-inject",
    apply: "serve",
    transformIndexHtml(html) {
      if (devTarget.startsWith("https://")) {
        return html.replace("</head>", "<script>window.RIFFPAD_RELAY=1;</script></head>");
      }
      return html;
    },
  };
}

// Build output goes straight into packages/webui/dist so the Go binaries
// (daemon + relay) embed the same client.
export default defineConfig({
  plugins: [react(), relayInject()],
  server: {
    // Dev-only proxy so the client can talk to a local daemon (default) or a
    // remote relay. Default is the cloud relay; use dev:local for a daemon.
    proxy: {
      "/api": {
        target: devTarget,
        changeOrigin: true,
      },
      "/ws": {
        target: devTarget,
        ws: true,
        changeOrigin: true,
      },
    },
  },
  build: {
    outDir: "../../packages/webui/dist",
    emptyOutDir: true,
  },
});
