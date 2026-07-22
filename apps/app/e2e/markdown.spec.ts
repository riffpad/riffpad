import { test, expect } from "@playwright/test";
import { createWorkspace } from "./helpers";

test.use({ viewport: { width: 1280, height: 800 } });
test.setTimeout(120_000);

test("agent markdown output is rendered as HTML", async ({ page }) => {
  await createWorkspace(page);
  const promptBox = page.getByPlaceholder(/describe your idea|描述你的想法/i);

  // Ask for a markdown-formatted reply with no tool calls
  const prompt =
    "Reply using Markdown only. Include a level-1 heading starting with '# ', a bullet list with '- ', and inline code wrapped in backticks. Do not use any tools.";
  await promptBox.fill(prompt);
  await page.getByRole("button", { name: /send|发送/i }).click();

  const assistantContent = page.getByTestId("assistant-content");
  await expect(assistantContent).toBeVisible({ timeout: 90_000 });

  // Confirm markdown is rendered into HTML during/after streaming.
  await expect(assistantContent.locator("h1")).toHaveCount(1, {
    timeout: 30_000,
  });
  await expect.poll(async () => await assistantContent.locator("ul > li").count()).toBeGreaterThanOrEqual(1);
  await expect.poll(async () => await assistantContent.locator("code").count()).toBeGreaterThanOrEqual(1);
});
