import { test, expect } from "@playwright/test";
import { createWorkspace } from "./helpers";

test.use({ viewport: { width: 1280, height: 800 } });
test.setTimeout(120_000);

test("agent code block shows language label and copy button", async ({ page }) => {
  await createWorkspace(page);
  const promptBox = page.getByPlaceholder(/describe your idea|描述你的想法/i);

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

  // Language label should be visible (exact match so the model's reasoning
  // text, which echoes the prompt, doesn't also match).
  const langLabel = assistantContent.getByText("python", { exact: true });
  await expect(langLabel).toBeVisible({ timeout: 5_000 });

  // Copy button should be visible and clickable
  const copyButton = assistantContent.locator('button[aria-label="Copy"]');
  await expect(copyButton).toBeVisible({ timeout: 5_000 });
});
