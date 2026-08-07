import { expect, test } from "@playwright/test";

test("app loads and shows pairing onboarding for an unpaired device", async ({ page }) => {
  await page.goto("/");
  await expect(page).toHaveTitle(/Riffpad/);
  await expect(page.locator("#pair-view")).toBeVisible();
  await expect(page.locator("#pin-input input")).toHaveCount(6);
  await expect(page.locator("#scan-btn")).toBeVisible();
  await expect(page.locator("#copy-btn")).toBeVisible();
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

test("pin input auto-submits when the 6th digit is filled", async ({ page }) => {
  await page.goto("/");
  const inputs = page.locator("#pin-input input");
  await inputs.nth(0).fill("A");
  await inputs.nth(1).fill("1");
  await inputs.nth(2).fill("B");
  await inputs.nth(3).fill("2");
  await inputs.nth(4).fill("C");
  await inputs.nth(5).fill("3");
  await expect(page.locator("#pair-err")).toBeVisible();
});
