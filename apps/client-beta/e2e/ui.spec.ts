import { expect, test } from "@playwright/test";

test("app loads and shows pairing view for an unpaired device", async ({ page }) => {
  await page.goto("/");
  await expect(page).toHaveTitle(/Riffpad/);
  await expect(page.locator("text=配对设备")).toBeVisible();
  await expect(page.locator("#pair-view input")).toBeVisible();
});

test("homepage shows disconnected status for an unpaired device", async ({ page }) => {
  await page.goto("/");
  await expect(page.locator("#conn")).toContainText("未配对");
});

test("topbar has logo and theme toggle", async ({ page }) => {
  await page.goto("/");
  await expect(page.locator(".brand .logo")).toBeVisible();
  await expect(page.locator("#theme-toggle")).toBeVisible();
  await expect(page.locator("html")).toHaveAttribute("data-theme", "light");
  await page.click("#theme-toggle");
  await expect(page.locator("html")).toHaveAttribute("data-theme", "dark");
});

test("mobile topbar becomes a collapsible sidebar", async ({ page }) => {
  await page.setViewportSize({ width: 390, height: 844 });
  await page.goto("/");
  await expect(page.locator("#menu-toggle")).toBeVisible();
  await page.click("#menu-toggle");
  await expect(page.locator("#topbar")).toHaveClass(/open/);
  await page.click("#sidebar-close");
  await expect(page.locator("#topbar")).not.toHaveClass(/open/);
});
