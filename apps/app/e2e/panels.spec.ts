import { test, expect } from "@playwright/test";

test.use({ viewport: { width: 1280, height: 800 } });

test("dockable panels render with default layout", async ({ page }) => {
  await page.goto("/");
  await page.getByRole("button", { name: /new workspace|新工作区/i }).nth(1).click();

  // All three panels should be present.
  await expect(page.getByTestId("file-tree-panel")).toBeVisible();
  await expect(page.getByRole("heading", { name: /chat|聊天/i })).toBeVisible();
  await expect(page.getByRole("button", { name: /preview|预览/i })).toBeVisible();
});

test("left panel can be resized and respects minimum width", async ({ page }) => {
  await page.goto("/");
  await page.getByRole("button", { name: /new workspace|新工作区/i }).nth(1).click();

  const fileTree = page.getByTestId("file-tree-panel");
  await expect(fileTree).toBeVisible();

  const leftPanel = page.locator('[data-testid="file-tree"]').first();
  const firstSeparator = page.locator("[data-separator]").first();
  await expect(firstSeparator).toBeVisible();

  const initialBox = await leftPanel.boundingBox();
  expect(initialBox).not.toBeNull();
  const initialWidth = initialBox!.width;

  // Drag the first separator to the left.
  const sepBox = await firstSeparator.boundingBox();
  expect(sepBox).not.toBeNull();
  await page.mouse.move(sepBox!.x + sepBox!.width / 2, sepBox!.y + sepBox!.height / 2);
  await page.mouse.down();
  await page.mouse.move(sepBox!.x - 100, sepBox!.y + sepBox!.height / 2);
  await page.mouse.up();

  // Give the layout a moment to settle.
  await page.waitForTimeout(200);

  const newBox = await leftPanel.boundingBox();
  expect(newBox).not.toBeNull();
  expect(newBox!.width).toBeLessThan(initialWidth);

  // Try to drag the separator far off-screen to the left.
  await page.mouse.move(sepBox!.x - 100 + sepBox!.width / 2, sepBox!.y + sepBox!.height / 2);
  await page.mouse.down();
  await page.mouse.move(0, sepBox!.y + sepBox!.height / 2);
  await page.mouse.up();
  await page.waitForTimeout(200);

  const minBox = await leftPanel.boundingBox();
  expect(minBox).not.toBeNull();
  expect(minBox!.width).toBeGreaterThanOrEqual(170);
});
