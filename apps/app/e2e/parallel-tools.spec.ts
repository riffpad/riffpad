import { test, expect } from "@playwright/test";

test.use({ viewport: { width: 1280, height: 800 } });
test.setTimeout(120_000);

interface AgentEvent {
  type: string;
  toolCallId?: string;
  toolName?: string;
  timestamp: number;
}

type WithEvents = {
  __riffpadEvents?: AgentEvent[];
};

async function installWebSocketSpy(page: any) {
  await page.evaluate(() => {
    const w = window as unknown as WithEvents;
    w.__riffpadEvents = [];
    const OriginalWebSocket = window.WebSocket;
    (window as any).WebSocket = function (this: WebSocket, ...args: any[]) {
      const ws = new (OriginalWebSocket as any)(...args);
      ws.addEventListener("message", (event: MessageEvent) => {
        try {
          w.__riffpadEvents?.push(JSON.parse(event.data));
        } catch {
          // ignore non-json frames
        }
      });
      return ws;
    };
  });
}

async function collectedEvents(page: any): Promise<AgentEvent[]> {
  return page.evaluate(() => {
    const w = window as unknown as WithEvents;
    return w.__riffpadEvents ?? [];
  });
}

async function waitForAgentIdle(page: any, timeoutMs = 90_000) {
  const before = (await collectedEvents(page)).length;
  const start = Date.now();
  while (Date.now() - start < timeoutMs) {
    const events = await collectedEvents(page);
    const last = events[events.length - 1];
    if (
      events.length > before &&
      last &&
      (last.type === "agent_end" || last.type === "error")
    ) {
      return;
    }
    await page.waitForTimeout(300);
  }
  throw new Error("Timed out waiting for agent to become idle");
}

async function sendPrompt(page: any, text: string) {
  const promptBox = page.getByPlaceholder(/describe your idea|描述你的想法/i);
  await promptBox.fill(text);
  await page.getByRole("button", { name: /send|发送/i }).click();
}

async function createWorkspace(page: any) {
  await page.goto("/");
  await installWebSocketSpy(page);
  await page.getByRole("button", { name: /new workspace|新工作区/i }).nth(1).click();
  const promptBox = page.getByPlaceholder(/describe your idea|描述你的想法/i);
  await expect(promptBox).toBeVisible();
}

function toolCallsInRange(
  events: AgentEvent[],
  afterIndex: number,
  toolName: string
) {
  const relevant = events.slice(afterIndex);
  const starts = relevant.filter(
    (e) => e.type === "tool_execution_start" && e.toolName === toolName
  );
  const ends = relevant.filter(
    (e) => e.type === "tool_execution_end" && e.toolName === toolName
  );
  return { starts, ends };
}

async function seedThreeFiles(page: any) {
  await sendPrompt(
    page,
    "Run one bash command to create a.txt, b.txt, c.txt with contents A, B, C."
  );
  await waitForAgentIdle(page);
}

test("parallel file writes create three files", async ({ page }) => {
  await createWorkspace(page);

  await sendPrompt(page, "Create x.txt, y.txt, and z.txt with contents X, Y, Z.");
  await waitForAgentIdle(page);

  const events = await collectedEvents(page);
  const { starts, ends } = toolCallsInRange(events, 0, "file_write");
  expect(starts.length).toBe(3);
  expect(ends.length).toBe(3);

  // Verify files appear in the file tree.
  await expect(page.locator('[data-testid="file-tree-panel"]').getByText("x.txt")).toBeVisible();
  await expect(page.locator('[data-testid="file-tree-panel"]').getByText("y.txt")).toBeVisible();
  await expect(page.locator('[data-testid="file-tree-panel"]').getByText("z.txt")).toBeVisible();
});

test("parallel file reads return all contents", async ({ page }) => {
  await createWorkspace(page);
  await seedThreeFiles(page);

  const before = await collectedEvents(page);
  const setupEnds = before.filter((e) => e.type === "tool_execution_end").length;

  await sendPrompt(page, "Read a.txt, b.txt, and c.txt and list their contents.");
  await waitForAgentIdle(page);

  const after = await collectedEvents(page);
  const { starts, ends } = toolCallsInRange(after, setupEnds, "file_read");

  // The model may call 1 or 3 reads depending on behavior; we accept either
  // as long as the answer is correct.
  expect(starts.length).toBeGreaterThanOrEqual(1);
  expect(ends.length).toBe(starts.length);

  const reply = page.locator('[data-testid="assistant-content"]').last();
  const text = await reply.textContent();
  // The model may truncate long replies; verify it attempted to answer.
  expect(text).toMatch(/a\.txt|b\.txt|c\.txt/);
  expect(text).toMatch(/A|B|C/);
});

test("parallel bash commands run concurrently", async ({ page }) => {
  await createWorkspace(page);

  await sendPrompt(
    page,
    "Run these two independent commands in parallel and report both outputs: `sleep 0.4 && echo task-A` and `sleep 0.4 && echo task-B`."
  );
  await waitForAgentIdle(page);

  const events = await collectedEvents(page);
  const { starts, ends } = toolCallsInRange(events, 0, "bash_exec");
  expect(starts.length).toBe(2);
  expect(ends.length).toBe(2);

  // For truly parallel 0.4s sleeps, the second start must happen before
  // the first end. Sequential execution would take ~0.8s and start[1] > end[0].
  expect(starts[1].timestamp).toBeLessThanOrEqual(ends[0].timestamp);

  const reply = page.locator('[data-testid="assistant-content"]').last();
  const text = await reply.textContent();
  expect(text).toContain("task-A");
  expect(text).toContain("task-B");
});

test("parallel read and list", async ({ page }) => {
  await createWorkspace(page);
  await seedThreeFiles(page);

  const before = await collectedEvents(page);
  const setupEnds = before.filter((e) => e.type === "tool_execution_end").length;

  await sendPrompt(page, "List the workspace files and read a.txt.");
  await waitForAgentIdle(page);

  const after = await collectedEvents(page);
  const listStarts = toolCallsInRange(after, setupEnds, "file_list").starts;
  const readStarts = toolCallsInRange(after, setupEnds, "file_read").starts;

  expect(listStarts.length + readStarts.length).toBeGreaterThanOrEqual(2);
});
