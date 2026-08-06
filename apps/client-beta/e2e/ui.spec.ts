import { expect, test } from "@playwright/test";

test("app loads and shows pairing view for an unpaired device", async ({ page }) => {
  await page.goto("/");
  await expect(page).toHaveTitle(/Riffpad/);
  await expect(page.locator("text=配对设备")).toBeVisible();
  await expect(page.locator("#pair-view input")).toBeVisible();
});

test("homepage shows online status", async ({ page }) => {
  await page.goto("/");
  await expect(page.locator("#conn")).toContainText("服务在线");
});
