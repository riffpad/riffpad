import { test, expect } from "@playwright/test";
import { createWorkspace } from "./helpers";

test.use({ viewport: { width: 1280, height: 800 } });

test("chat input auto-grows and respects max height", async ({ page }) => {
  await createWorkspace(page);

  const textarea = page.getByPlaceholder(/describe your idea|描述你的想法/i);
  await expect(textarea).toBeVisible();

  const initialBox = await textarea.boundingBox();
  expect(initialBox).not.toBeNull();
  const initialHeight = initialBox!.height;

  // Type enough lines to trigger auto-grow.
  await textarea.fill("line 1\nline 2\nline 3\nline 4\nline 5\nline 6\nline 7\nline 8");
  await page.waitForTimeout(200);

  const grownBox = await textarea.boundingBox();
  expect(grownBox).not.toBeNull();
  expect(grownBox!.height).toBeGreaterThan(initialHeight);

  // Type many more lines to approach the max height.
  const manyLines = Array.from({ length: 30 }, (_, i) => `line ${i + 1}`).join("\n");
  await textarea.fill(manyLines);
  await page.waitForTimeout(200);

  const maxBox = await textarea.boundingBox();
  expect(maxBox).not.toBeNull();
  expect(maxBox!.height).toBeLessThanOrEqual(210);
});
