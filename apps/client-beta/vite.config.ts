import { defineConfig } from "vite";
import react from "@vitejs/plugin-react";

// Build output goes straight into packages/webui/dist so the Go binaries
// (daemon + relay) embed the same client.
export default defineConfig({
  plugins: [react()],
  build: {
    outDir: "../../packages/webui/dist",
    emptyOutDir: true,
  },
});
