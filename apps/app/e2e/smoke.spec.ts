import { test, expect } from "@playwright/test";

test("landing page renders", async ({ page }) => {
  await page.goto("/");
  await expect(page.getByRole("heading", { name: "Riffpad" }).first()).toBeVisible();
  await expect(page.getByRole("button", { name: /new workspace|新工作区/i })).toHaveCount(2);
});

test("toggles dark mode", async ({ page }) => {
  await page.goto("/");
  const toggle = page.getByRole("button", { name: "toggle theme" });
  await toggle.click();
  await expect(page.locator("html")).toHaveClass(/dark/);
  await toggle.click();
  await expect(page.locator("html")).not.toHaveClass(/dark/);
});

test("switches language to English", async ({ page }) => {
  await page.goto("/");
  const langButton = page.locator("button[aria-haspopup='listbox']");
  await langButton.click();
  await page.getByRole("option", { name: "English" }).click();
  await expect(page.getByRole("button", { name: "New workspace" }).nth(1)).toBeVisible();
});

test("creates a workspace", async ({ page }) => {
  await page.goto("/");
  await page.getByRole("button", { name: /new workspace|新工作区/i }).nth(1).click();
  await expect(page.getByPlaceholder(/describe your idea|描述你的想法/i)).toBeVisible();
  await expect(page.getByRole("heading", { name: /agent events|agent 事件/i })).toBeVisible();
});

test("opens mobile file drawer", async ({ page }) => {
  await page.setViewportSize({ width: 390, height: 844 });
  await page.goto("/");
  await page.getByRole("button", { name: /new workspace|新工作区/i }).nth(1).click();
  await page.getByRole("button", { name: /files|文件/i }).click();
  await expect(page.getByRole("heading", { name: /files|文件/i }).first()).toBeVisible();
  await expect(page.getByRole("button", { name: /close|关闭/i })).toBeVisible();
});
