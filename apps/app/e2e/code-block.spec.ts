import { test, expect } from "@playwright/test";

test.use({ viewport: { width: 1280, height: 800 } });
test.setTimeout(120_000);

test("agent code block shows language label and copy button", async ({ page }) => {
  await page.goto("/");

  // Create a workspace
  await page.getByRole("button", { name: /new workspace|新工作区/i }).nth(1).click();
  const promptBox = page.getByPlaceholder(/describe your idea|描述你的想法/i);
  await expect(promptBox).toBeVisible();

  // Ask for a code block in a specific language
  const prompt =
    "Reply with a single markdown code block labeled `python` that contains `print('hello')`. Do not use any tools.";
  await promptBox.fill(prompt);
  await page.getByRole("button", { name: /send|发送/i }).click();

  const assistantContent = page.getByTestId("assistant-content");
  await expect(assistantContent).toBeVisible({ timeout: 90_000 });

  // Wait for a rendered code block
  const codeBlock = assistantContent.locator("pre");
  await expect(codeBlock).toBeVisible({ timeout: 30_000 });

  // Language label should be visible
  const langLabel = assistantContent.locator("text=PYTHON");
  await expect(langLabel).toBeVisible({ timeout: 5_000 });

  // Copy button should be visible and clickable
  const copyButton = assistantContent.locator('button[aria-label="Copy"]');
  await expect(copyButton).toBeVisible({ timeout: 5_000 });
});
