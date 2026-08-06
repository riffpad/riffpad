import { defineConfig } from "@playwright/test";

// UI smoke tests against a real local daemon (separate data dir + port).
export default defineConfig({
  testDir: "./e2e",
  timeout: 60_000,
  use: {
    baseURL: "http://127.0.0.1:8792",
    locale: "zh-CN",
  },
  webServer: {
    command:
      "bash -c 'mkdir -p /tmp/riffpad-pw && printf \"{\\\"port\\\":8792}\" > /tmp/riffpad-pw/config.json && ${RIFFPAD_DAEMON:-$HOME/.local/bin/riffpad} _daemon --data-dir /tmp/riffpad-pw'",
    url: "http://127.0.0.1:8792/api/status",
    reuseExistingServer: false,
    timeout: 30_000,
  },
});
