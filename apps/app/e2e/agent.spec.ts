import { test, expect } from "@playwright/test";
import { createWorkspace } from "./helpers";

test.use({ viewport: { width: 1280, height: 800 } });
test.setTimeout(180_000);

test("agent writes a file from a prompt", async ({ page }) => {
  await createWorkspace(page);
  const promptBox = page.getByPlaceholder(/describe your idea|描述你的想法/i);

  // Send a prompt
  const prompt = "Create a simple HTML landing page with a title and a button. Save it as index.html.";
  await promptBox.fill(prompt);
  await page.getByRole("button", { name: /send|发送/i }).click();

  // Wait for an HTML file to appear in the file tree
  const htmlFile = page
    .getByTestId("file-tree-panel")
    .locator("button")
    .filter({ hasText: /\.html$/ });
  await expect(htmlFile).toBeVisible({ timeout: 90_000 });

  // Open the file and verify it contains expected HTML tags
  await htmlFile.click();
  const viewer = page.locator("pre");
  await expect(viewer).toContainText("<html", { timeout: 10_000 });
  await expect(viewer).toContainText(/<button|<h1|<title/i);


});
