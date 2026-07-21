import { test, expect } from "@playwright/test";

test.use({ viewport: { width: 1280, height: 800 } });
test.setTimeout(180_000);

test("dockable panels render with default layout", async ({ page }) => {
  await page.goto("/");
  await page.getByRole("button", { name: /new workspace|新工作区/i }).nth(1).click();

  // All three panels should be present.
  await expect(page.getByTestId("file-tree-panel")).toBeVisible();
  await expect(page.getByRole("heading", { name: /chat|聊天/i })).toBeVisible();
  await expect(page.getByText(/select a file|从左侧文件树/i)).toBeVisible();
});

test("left panel can be resized and respects minimum width", async ({ page }) => {
  await page.goto("/");
  await page.getByRole("button", { name: /new workspace|新工作区/i }).nth(1).click();

  const leftPanel = page.getByTestId("file-tree-panel");
  await expect(leftPanel).toBeVisible();

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
  await page.mouse.move(newBox!.x + newBox!.width + sepBox!.width / 2, sepBox!.y + sepBox!.height / 2);
  await page.mouse.down();
  await page.mouse.move(0, sepBox!.y + sepBox!.height / 2);
  await page.mouse.up();
  await page.waitForTimeout(200);

  const minBox = await leftPanel.boundingBox();
  expect(minBox).not.toBeNull();
  expect(minBox!.width).toBeGreaterThanOrEqual(170);
});


test("markdown file opens with preview and can toggle raw", async ({ page }) => {
  await page.goto("/");
  await page.getByRole("button", { name: /new workspace|新工作区/i }).nth(1).click();

  const promptBox = page.getByPlaceholder(/describe your idea|描述你的想法/i);
  await expect(promptBox).toBeVisible();

  // Ask the agent to create a small markdown file.
  await promptBox.fill(
    "Create a README.md file with exactly '# Hello' on the first line and '- world' on the second line. No other output."
  );
  await page.getByRole("button", { name: /send|发送/i }).click();

  const mdFile = page
    .getByTestId("file-tree-panel")
    .locator("button")
    .filter({ hasText: /README\.md$/i });
  await expect(mdFile).toBeVisible({ timeout: 90_000 });
  await mdFile.click();

  // Markdown defaults to preview: the heading should be rendered as an <h1>.
  const content = page.getByTestId("file-editor-content");
  await expect(content.locator("h1").filter({ hasText: "Hello" })).toBeVisible({ timeout: 10_000 });

  // Switch to raw view.
  await page.getByRole("button", { name: "Raw" }).click();
  await expect(content.locator("pre")).toContainText("# Hello");
  await expect(content.locator("pre")).toContainText("- world");

  // Switch back to preview.
  await page.getByRole("button", { name: "Preview" }).click();
  await expect(content.locator("h1").filter({ hasText: "Hello" })).toBeVisible();
});
