import { defineConfig } from "@playwright/test";

// Core-path browser test (#314): the real client against the real relay +
// daemon binaries. Unlike playwright.config.ts this boots nothing itself —
// e2e/core-flow.spec.ts owns the stack through scripts/lib/core-harness.mjs,
// because the relay has to come up with the GitHub stub URLs already set.
export default defineConfig({
  testDir: "./e2e",
  testMatch: "core-flow.spec.ts",
  timeout: 120_000,
  workers: 1,
  use: {
    locale: "zh-CN",
    // The relay serves the embedded client, so there is no baseURL to pin.
  },
});
