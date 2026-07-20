import { test, expect } from "@playwright/test";

test.use({ viewport: { width: 1280, height: 800 } });
test.setTimeout(180_000);

async function waitForCitedReply(page: any, timeoutMs = 120_000) {
  // The agent may emit a short pre-tool message first, then call web_search,
  // then emit the final answer. Poll all assistant messages until one contains
  // a citation marker [N] or a rendered citation link.
  const start = Date.now();
  while (Date.now() - start < timeoutMs) {
    const contents = page.locator('[data-testid="assistant-content"]');
    const count = await contents.count();
    for (let i = 0; i < count; i++) {
      const text = await contents.nth(i).textContent();
      const hasLink = await contents.nth(i).locator('a[href^="http"]').count();
      if (text && (/\[\d+\]/.test(text) || hasLink > 0)) {
        return contents.nth(i);
      }
    }
    await page.waitForTimeout(500);
  }
  throw new Error("Timed out waiting for an assistant reply with citations");
}

test("agent uses web search and cites sources", async ({ page }) => {
  await page.goto("/");
  await page.getByRole("button", { name: /new workspace|新工作区/i }).nth(1).click();

  const promptBox = page.getByPlaceholder(/describe your idea|描述你的想法/i);
  await expect(promptBox).toBeVisible();

  const prompt = "Search the web for the current weather in Beijing and tell me the source.";
  await promptBox.fill(prompt);
  await page.getByRole("button", { name: /send|发送/i }).click();

  // Wait for a web search tool card to appear and complete.
  const searchTool = page.getByText(/Searching/i).first();
  await expect(searchTool).toBeVisible({ timeout: 60_000 });
  await page.screenshot({ path: "test-results/websearch-tool-appears.png" });

  // Wait until the search is done so the result is available to render.
  const doneTool = page.getByText(/Done/i).first();
  await expect(doneTool).toBeVisible({ timeout: 90_000 });

  // Expand the tool card to verify card rendering.
  await doneTool.click();
  await page.waitForTimeout(300);
  await page.screenshot({ path: "test-results/websearch-tool-expanded.png" });

  // Wait for an assistant reply that contains citation markers.
  const assistantContent = await waitForCitedReply(page);
  await page.waitForTimeout(1000);
  await page.screenshot({ path: "test-results/websearch-reply.png" });

  const text = await assistantContent.textContent();
  console.log("Assistant reply text:", text);

  // The reply should contain a citation marker or a rendered citation link.
  const hasCitationMarker = /\[\d+\]/.test(text);
  const linkCount = await assistantContent.locator('a[href^="http"]').count();
  expect(hasCitationMarker || linkCount > 0).toBe(true);
});
