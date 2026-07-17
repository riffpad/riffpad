import { defineConfig, devices } from "@playwright/test";

/**
 * Formal E2E smoke tests for the Riffpad app.
 *
 * Before running:
 *   pnpm exec playwright install chromium
 *   pnpm dev:app      # http://localhost:3000
 *   cd apps/api && go run ./cmd/api   # http://localhost:8080
 *
 * Then:
 *   pnpm e2e
 */
export default defineConfig({
  testDir: "./e2e",
  fullyParallel: true,
  forbidOnly: !!process.env.CI,
  retries: process.env.CI ? 2 : 0,
  workers: process.env.CI ? 1 : undefined,
  reporter: "list",
  use: {
    baseURL: process.env.BASE_URL || "http://localhost:3000",
    trace: "on-first-retry",
    screenshot: "only-on-failure",
  },

  projects: [
    {
      name: "chromium",
      use: { ...devices["Desktop Chrome"] },
    },
    {
      name: "mobile-chrome",
      use: { ...devices["Pixel 5"] },
    },
  ],

  webServer: [
    {
      command: "pnpm dev:app",
      url: "http://localhost:3000",
      timeout: 120_000,
      reuseExistingServer: !process.env.CI,
    },
    {
      command: "cd apps/api && go run ./cmd/api",
      url: "http://localhost:8080/healthz",
      timeout: 120_000,
      reuseExistingServer: !process.env.CI,
    },
  ],
});
