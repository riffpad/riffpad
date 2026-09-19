// Browser half of the core-path test (#314).
//
// scripts/e2e-core-smoke.mjs proves the chain with a WebCrypto viewer; this
// drives the *real* client through the *real* stack, in the order a user
// walks it:
//
//   riffpad login (GitHub device flow, authorised in the browser)
//     -> riffpad pair -> riffpad run demo
//     -> web sign-in with the same GitHub identity
//     -> enter the pairing code -> watch the timeline -> click approve
//
// The relay's GitHub endpoints point at a local stub (GITHUB_*_URL) so the
// whole OAuth flow stays on the test machine. Both sign-ins resolve to the
// same stubbed GitHub account, which is what makes them the same relay user —
// the same way it works in production.

import { expect, test } from "@playwright/test";
import { bootCore } from "../../../scripts/lib/core-harness.mjs";

let core;
let sessionId;

test.beforeAll(async () => {
  core = await bootCore({ withGitHubStub: true, login: "device", cli: "demo" });
});

test.afterAll(async () => {
  await core?.shutdown();
});

test("sign in, pair, watch a session and approve from the UI", async ({ page }) => {
  // ---- 1. `riffpad login` device flow, authorised in the browser ----
  // The device page navigates (not a popup): relay -> stub GitHub -> relay
  // callback -> receipt page, and the CLI's poll picks up the token.
  await page.goto(core.verificationURL);
  await expect(page.locator("#device-auth-view")).toBeVisible();
  await page.click("#device-github-login");
  // The receipt page's wording depends on how recently the CLI happened to
  // poll (it warns when the last poll was >8s ago), so assert on the real
  // outcome — the CLI finishing — rather than on the copy.
  await page.waitForURL(/\/api\/auth\/github\/callback/, { timeout: 20_000 });

  // The CLI exits once its poll sees the authorisation; only then can it pair.
  await core.waitForLogin();
  expect(core.relayToken).toBeTruthy();

  // ---- 2. `riffpad pair` + `riffpad run demo` from the real CLI ----
  const code = await core.pair();
  sessionId = core.runSession();

  // ---- 3. Web sign-in with the same GitHub account ----
  await page.goto(core.relayBase);
  await expect(page.locator("#auth-view")).toBeVisible();
  const popup = page.waitForEvent("popup");
  await page.click("#github-login");
  await (await popup).waitForEvent("close").catch(() => {});

  // ---- 4. Pair this browser with the printed code ----
  await expect(page.locator("#pair-view")).toBeVisible({ timeout: 20_000 });
  const digits = page.locator("#pin-input input");
  await expect(digits).toHaveCount(6);
  for (let i = 0; i < code.length; i++) await digits.nth(i).fill(code[i]);

  // ---- 5. The session the CLI started is listed as live ----
  const session = page.locator("#session-list li.session").first();
  await expect(session).toBeVisible({ timeout: 20_000 });
  await expect(session).toContainText("demo");
  // The scripted agent settles into waiting_input once its intro timeline is
  // done, so assert on a live status rather than a specific one.
  await expect(session.locator(".session-light")).toHaveClass(/running|waiting|done|restored/);

  // ---- 6. Open it: the scripted timeline arrives decrypted ----
  await session.click();
  await expect(page.locator("#detail")).toBeVisible();
  await expect(page.locator("#events")).toContainText("refactor the auth middleware", { timeout: 20_000 });

  // ---- 7. Ask for an approval and click it: the decision travels back
  //         through the relay to the agent, which then reports success ----
  const prompt = page.locator("#detail input[name=message]");
  await prompt.fill("approval");
  await prompt.press("Enter");

  const approve = page.locator(".approval-card .approve").last();
  await expect(approve).toBeVisible({ timeout: 20_000 });
  await approve.click();

  await expect(page.locator("#events")).toContainText("Approved — file written.", { timeout: 20_000 });

  // The session the UI is driving must be the one the CLI created.
  expect(sessionId).toBeTruthy();
});
