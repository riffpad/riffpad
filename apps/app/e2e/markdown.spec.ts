import { test, expect } from "@playwright/test";

test.use({ viewport: { width: 1280, height: 800 } });
test.setTimeout(120_000);

test("agent markdown output is rendered as HTML", async ({ page }) => {
  await page.goto("/");

  // Create a workspace
  await page.getByRole("button", { name: /new workspace|新工作区/i }).nth(1).click();
  const promptBox = page.getByPlaceholder(/describe your idea|描述你的想法/i);
  await expect(promptBox).toBeVisible();

  // Ask for a markdown-formatted reply with no tool calls
  const prompt =
    "Reply using Markdown only. Include a level-1 heading starting with '# ', a bullet list with '- ', and inline code wrapped in backticks. Do not use any tools.";
  await promptBox.fill(prompt);
  await page.getByRole("button", { name: /send|发送/i }).click();

  // Wait for a complete assistant message (no blinking cursor)
  const assistantContent = page.getByTestId("assistant-content");
  await expect(assistantContent).toBeVisible({ timeout: 90_000 });

  // Wait until streaming cursor disappears
  await expect(assistantContent.locator("span.animate-pulse")).toHaveCount(0, {
    timeout: 30_000,
  });

  // Confirm markdown was rendered, not shown raw
  await expect(assistantContent.locator("h1")).toHaveCount(1, {
    timeout: 5_000,
  });
  await expect.poll(async () => await assistantContent.locator("ul > li").count()).toBeGreaterThanOrEqual(1);
  await expect.poll(async () => await assistantContent.locator("code").count()).toBeGreaterThanOrEqual(1);
});
