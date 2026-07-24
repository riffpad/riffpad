import { test, expect } from "@playwright/test";
import {
  API_URL,
  createNamedWorkspace,
  createWorkspaceViaApi,
  deleteWorkspaceViaApi,
  patchWorkspaceViaApi,
  workspaceCard,
} from "./helpers";

test.use({ viewport: { width: 1280, height: 800 } });
test.setTimeout(120_000);

const searchBox = (page: import("@playwright/test").Page) =>
  page.getByPlaceholder(/search by name|搜索名称/i);

async function openMenu(page: import("@playwright/test").Page, cardText: string) {
  const card = workspaceCard(page, cardText);
  await expect(card).toBeVisible();
  await card.hover();
  await card
    .getByRole("button", { name: /workspace actions|工作区操作/i })
    .click();
}

test("search filters workspaces by name", async ({ page }) => {
  const ws = await createNamedWorkspace(page, "e2e-search-target");
  try {
    await page.goto("/");
    await searchBox(page).fill("e2e-search-target");
    await expect(workspaceCard(page, "e2e-search-target")).toBeVisible();

    await searchBox(page).fill("no-such-workspace-zzz");
    await expect(
      page.getByText(/no workspaces match|没有匹配的工作区/i)
    ).toBeVisible();
  } finally {
    await deleteWorkspaceViaApi(page, ws.id);
  }
});

test("renames a workspace via the card menu and persists", async ({ page }) => {
  const ws = await createWorkspaceViaApi(page);
  const newName = "e2e-renamed-ws";
  try {
    await page.goto("/");
    await openMenu(page, ws.slug);
    await page.getByRole("menuitem", { name: /^rename$|^重命名$/i }).click();
    const input = page.getByRole("textbox", { name: /rename|重命名/i });
    await input.fill(newName);
    await input.press("Enter");

    await expect(workspaceCard(page, newName)).toBeVisible();

    // Persistence: reload and the new name is still there.
    await page.reload();
    await expect(workspaceCard(page, newName)).toBeVisible();
  } finally {
    await deleteWorkspaceViaApi(page, ws.id);
  }
});

test("edits the note of a workspace", async ({ page }) => {
  const ws = await createWorkspaceViaApi(page);
  try {
    await page.goto("/");
    await openMenu(page, ws.slug);
    await page
      .getByRole("menuitem", { name: /edit note|编辑备注/i })
      .click();
    const input = page.getByRole("textbox", { name: /edit note|编辑备注/i });
    await input.fill("a quick note");
    await input.press("Enter");

    await expect(
      workspaceCard(page, ws.slug).getByText("a quick note")
    ).toBeVisible();
  } finally {
    await deleteWorkspaceViaApi(page, ws.id);
  }
});

test("pinning a workspace moves it above newer ones", async ({ page }) => {
  const a = await createNamedWorkspace(page, "e2e-pin-test-a");
  const b = await createNamedWorkspace(page, "e2e-pin-test-b");
  try {
    await page.goto("/");
    await searchBox(page).fill("e2e-pin-test");

    const titles = page.locator("div[role='button'] h3");
    await expect(titles).toHaveCount(2);
    // Default order: newest (b) first.
    await expect(titles.nth(0)).toHaveText("e2e-pin-test-b");

    const cardA = workspaceCard(page, "e2e-pin-test-a");
    await cardA.hover();
    await cardA.getByRole("button", { name: /^pin$|^置顶$/i }).click();

    await expect(titles.nth(0)).toHaveText("e2e-pin-test-a");
  } finally {
    await deleteWorkspaceViaApi(page, a.id);
    await deleteWorkspaceViaApi(page, b.id);
  }
});

test("archives a workspace and finds it via the status filter", async ({
  page,
}) => {
  const ws = await createNamedWorkspace(page, "e2e-archive-me");
  try {
    await page.goto("/");
    await searchBox(page).fill("e2e-archive-me");
    await openMenu(page, "e2e-archive-me");
    await page.getByRole("menuitem", { name: /^archive$|^归档$/i }).click();

    // Archived card shows the badge in the default "all" filter.
    await expect(
      workspaceCard(page, "e2e-archive-me").getByText(/archived|已归档/i)
    ).toBeVisible();

    // It disappears under the "active" filter...
    await page
      .getByRole("button", { name: /^active$|^活跃$/i })
      .click();
    await expect(workspaceCard(page, "e2e-archive-me")).toHaveCount(0);

    // ...and shows up under the "archived" filter.
    await page
      .getByRole("button", { name: /^archived$|^已归档$/i })
      .click();
    await expect(workspaceCard(page, "e2e-archive-me")).toBeVisible();
  } finally {
    await deleteWorkspaceViaApi(page, ws.id);
  }
});

test("copies a workspace link from the menu", async ({ page }) => {
  await page.context().grantPermissions(["clipboard-read", "clipboard-write"]);
  const ws = await createWorkspaceViaApi(page);
  try {
    await page.goto("/");
    await openMenu(page, ws.slug);
    await page
      .getByRole("menuitem", { name: /copy link|复制链接/i })
      .click();
    await expect(
      page.getByText(/link copied|链接已复制/i)
    ).toBeVisible();
  } finally {
    await deleteWorkspaceViaApi(page, ws.id);
  }
});

test("batch deletes selected workspaces", async ({ page }) => {
  const a = await createNamedWorkspace(page, "e2e-batch-del-1");
  const b = await createNamedWorkspace(page, "e2e-batch-del-2");
  await page.goto("/");
  await searchBox(page).fill("e2e-batch-del");

  for (const name of ["e2e-batch-del-1", "e2e-batch-del-2"]) {
    const card = workspaceCard(page, name);
    await card.hover();
    await card
      .getByRole("checkbox", { name: /select workspace|选择工作区/i })
      .click();
  }

  // Batch bar appears with the count.
  await expect(
    page.getByText(/2 selected|已选 2 项/i)
  ).toBeVisible();

  await page
    .getByRole("button", { name: /delete selected|删除所选/i })
    .click();

  // Custom confirm dialog (not a native browser dialog).
  const dialog = page.getByRole("alertdialog");
  await expect(dialog).toBeVisible();
  await dialog.getByRole("button", { name: /^delete$|^删除$/i }).click();

  await expect(workspaceCard(page, "e2e-batch-del-1")).toHaveCount(0);
  await expect(workspaceCard(page, "e2e-batch-del-2")).toHaveCount(0);
});

test("switches between grid and list view and persists the choice", async ({
  page,
}) => {
  await page.goto("/");
  await expect(page.locator("ul.grid")).toHaveCount(1);

  await page
    .getByRole("button", { name: /list view|列表视图/i })
    .click();
  await expect(page.locator("ul.grid")).toHaveCount(0);

  const stored = await page.evaluate(() =>
    window.localStorage.getItem("riffpad.viewMode")
  );
  expect(stored).toBe("list");

  // Persisted across reloads.
  await page.reload();
  await expect(page.locator("ul.grid")).toHaveCount(0);

  // Restore grid for other tests.
  await page
    .getByRole("button", { name: /grid view|网格视图/i })
    .click();
  await expect(page.locator("ul.grid")).toHaveCount(1);
});

test("empty state hides the toolbar and focuses on creation", async ({
  page,
}) => {
  // Intercept the list endpoint and return an empty array so the real data
  // stays untouched.
  await page.route(`${API_URL}/api/v1/workspaces`, (route) => {
    if (route.request().method() === "GET") {
      return route.fulfill({ json: [] });
    }
    return route.continue();
  });

  await page.goto("/");

  // The search / sort / filter / view toolbar must not be rendered.
  await expect(
    page.getByPlaceholder(/search by name|搜索名称/i)
  ).toHaveCount(0);
  await expect(
    page.getByRole("button", { name: /grid view|网格视图/i })
  ).toHaveCount(0);

  // The hero guides the user to create the first workspace.
  await expect(
    page.getByRole("heading", {
      name: /ready to turn your idea|准备好把你的想法/i,
    })
  ).toBeVisible();
  await expect(
    page.getByRole("button", { name: /new workspace|新工作区/i }).first()
  ).toBeVisible();
});

test("loads more workspaces when scrolling to the bottom", async ({ page }) => {
  // 26 workspaces with a shared prefix -> first page shows 24, scroll loads the rest.
  const ids: string[] = [];
  for (let i = 0; i < 26; i++) {
    const ws = await createWorkspaceViaApi(page);
    ids.push(ws.id);
    await patchWorkspaceViaApi(page, ws.id, {
      name: `e2e-scroll-ws-${String(i).padStart(2, "0")}`,
    });
  }

  try {
    await page.goto("/");
    await searchBox(page).fill("e2e-scroll-ws");
    const cards = page.locator("div[role='button']", {
      hasText: "e2e-scroll-ws",
    });
    await expect(cards).toHaveCount(24);

    await page.evaluate(() =>
      window.scrollTo(0, document.body.scrollHeight)
    );
    await expect(cards).toHaveCount(26);
  } finally {
    await Promise.all(ids.map((id) => deleteWorkspaceViaApi(page, id)));
  }
});
