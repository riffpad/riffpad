import { test, expect } from "@playwright/test";
import { createWorkspace, createWorkspaceViaApi } from "./helpers";

test("landing page renders", async ({ page }) => {
  await page.goto("/");
  await expect(page.getByRole("heading", { name: "Riffpad" }).first()).toBeVisible();
  await expect(
    page.getByRole("button", { name: /new workspace|新工作区/i }).first()
  ).toBeVisible();
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
  await expect(page.getByRole("button", { name: "New workspace" }).first()).toBeVisible();
});

test("creates a workspace", async ({ page }) => {
  await createWorkspace(page);
  await expect(page.getByRole("heading", { name: /chat|聊天/i })).toBeVisible();
});

test("lists existing workspaces and navigates into one", async ({ page }) => {
  const ws = await createWorkspaceViaApi(page);
  await page.goto("/");
  const card = page.locator("div[role='button']", { hasText: ws.slug });
  await expect(card).toBeVisible();

  await card.click();
  await page.waitForURL(new RegExp(`/w/${ws.id}`));
  await expect(page.getByRole("heading", { name: /chat|聊天/i })).toBeVisible();

  // Back to the list via the header button.
  await page
    .getByRole("button", { name: /all workspaces|工作区列表/i })
    .click();
  await expect(card).toBeVisible();
});

test("reopens a persisted workspace directly by URL", async ({ page }) => {
  const ws = await createWorkspaceViaApi(page);
  await page.goto(`/w/${ws.id}`);
  await expect(page.getByRole("heading", { name: /chat|聊天/i })).toBeVisible();
  await expect(page.getByPlaceholder(/describe your idea|描述你的想法/i)).toBeVisible();
});

test("deletes a workspace from the list", async ({ page }) => {
  const ws = await createWorkspaceViaApi(page);
  await page.goto("/");
  const card = page.locator("div[role='button']", { hasText: ws.slug });
  await expect(card).toBeVisible();

  await card.hover();
  await card
    .getByRole("button", { name: /workspace actions|工作区操作/i })
    .click();
  page.once("dialog", (dialog) => void dialog.accept());
  await page
    .getByRole("menuitem", { name: /delete workspace|删除工作区/i })
    .click();
  await expect(card).toHaveCount(0);
});

test("opens mobile file drawer", async ({ page }) => {
  await page.setViewportSize({ width: 390, height: 844 });
  await createWorkspace(page);
  await page.getByRole("button", { name: /files|文件/i }).click();
  await expect(page.getByRole("heading", { name: /files|文件/i }).first()).toBeVisible();
  await expect(page.getByRole("button", { name: /close|关闭/i })).toBeVisible();
});
