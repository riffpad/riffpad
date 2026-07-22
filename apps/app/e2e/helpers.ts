import { expect, type Page } from "@playwright/test";

export const API_URL = process.env.API_URL || "http://localhost:8080";

/**
 * Navigate home, create a fresh workspace via the UI, and land on its
 * workspace page (/w/:id) with the chat input ready.
 */
export async function createWorkspace(page: Page) {
  await page.goto("/");
  await page
    .getByRole("button", { name: /new workspace|新工作区/i })
    .first()
    .click();
  await page.waitForURL(/\/w\/[a-z0-9]+/i);
  await expect(
    page.getByPlaceholder(/describe your idea|描述你的想法/i)
  ).toBeVisible();
}

export interface WorkspaceSummary {
  id: string;
  slug: string;
}

/** Create a workspace directly through the API (skips the UI). */
export async function createWorkspaceViaApi(
  page: Page
): Promise<WorkspaceSummary> {
  const res = await page.request.post(`${API_URL}/api/v1/workspaces`);
  if (!res.ok()) {
    throw new Error(`create workspace failed: ${res.status()}`);
  }
  return res.json();
}
