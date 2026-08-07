import { defineConfig } from "vite";
import react from "@vitejs/plugin-react";

// Build output goes straight into packages/webui/dist so the Go binaries
// (daemon + relay) embed the same client.
export default defineConfig({
  plugins: [react()],
  server: {
    // Dev-only proxy so the client can talk to a local daemon (default) or a
    // remote relay (RIFFPAD_DEV_TARGET=https://api.riffpad.ai + ?relay=1).
    proxy: {
      "/api": {
        target: process.env.RIFFPAD_DEV_TARGET || "http://127.0.0.1:8787",
        changeOrigin: true,
      },
      "/ws": {
        target: process.env.RIFFPAD_DEV_TARGET || "http://127.0.0.1:8787",
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
