// Shared bootstrap for the core-path tests (#314).
//
// Boots the compiled relay + riffpad binaries and walks the real CLI path
// (login → pair → run) so a test can then drive the chain from whichever side
// it cares about: a WebCrypto viewer (scripts/e2e-core-smoke.mjs) or a real
// browser against the real client (apps/client-beta/e2e/core-flow.spec.ts).
//
// Everything here is the real thing — real binaries, real HTTP, real
// WebSocket, real E2EE. Only the agent CLI is scripted (`run demo`).

import { spawn, spawnSync } from "node:child_process";
import { createServer } from "node:http";
import { existsSync, mkdirSync, mkdtempSync, readFileSync, readdirSync, rmSync, statSync, writeFileSync } from "node:fs";
import { createServer as createNetServer } from "node:net";
import { tmpdir } from "node:os";
import { dirname, join, resolve } from "node:path";
import { fileURLToPath } from "node:url";
import { setTimeout as sleep } from "node:timers/promises";

export const ROOT = resolve(dirname(fileURLToPath(import.meta.url)), "..", "..");
export const ACCOUNT = { username: "smoke", password: "smoke-password-123" };

export function freePort() {
  return new Promise((res, rej) => {
    const srv = createNetServer();
    srv.on("error", rej);
    srv.listen(0, "127.0.0.1", () => {
      const { port } = srv.address();
      srv.close(() => res(port));
    });
  });
}

export async function waitFor(fn, { timeout = 30_000, interval = 250, what = "condition" } = {}) {
  const deadline = Date.now() + timeout;
  for (;;) {
    try {
      const v = await fn();
      if (v) return v;
    } catch {
      // keep polling
    }
    if (Date.now() > deadline) throw new Error(`timed out waiting for ${what}`);
    await sleep(interval);
  }
}

export async function jfetch(url, { method = "GET", body, token } = {}) {
  const headers = {};
  if (token) headers.Authorization = `Bearer ${token}`;
  if (body) headers["Content-Type"] = "application/json";
  const res = await fetch(url, { method, headers, body: body ? JSON.stringify(body) : undefined });
  const data = await res.json().catch(() => ({}));
  return { res, data };
}

// walk lists every file under dir (used by the zero-trust check).
export function walk(dir, out = []) {
  if (!existsSync(dir)) return out;
  for (const e of readdirSync(dir, { withFileTypes: true })) {
    const p = join(dir, e.name);
    if (e.isDirectory()) walk(p, out);
    else if (e.isFile()) out.push(p);
  }
  return out;
}

export { readFileSync, statSync, sleep };

// startGitHubStub stands in for github.com so the whole OAuth flow — popup,
// redirect, code exchange, user lookup — stays on the test machine. It serves
// all three endpoints the relay talks to; the redirect target comes from the
// relay's own redirect_uri, so the browser lands back on the local relay.
export async function startGitHubStub({ verbose = false } = {}) {
  const port = await freePort();
  const requests = [];
  const srv = createServer((req, res) => {
    const url = new URL(req.url, `http://127.0.0.1:${port}`);
    requests.push(`${req.method} ${url.pathname}`);
    if (verbose) console.log("[github-stub]", req.method, url.pathname);

    if (url.pathname === "/authorize") {
      const redirect = url.searchParams.get("redirect_uri") || "";
      const state = url.searchParams.get("state") || "";
      if (!redirect) {
        res.writeHead(400).end("missing redirect_uri");
        return;
      }
      const back = new URL(redirect);
      back.searchParams.set("code", "stub-code");
      back.searchParams.set("state", state);
      res.writeHead(302, { Location: back.toString() }).end();
      return;
    }

    if (url.pathname === "/token") {
      res.writeHead(200, { "Content-Type": "application/json" });
      res.end(JSON.stringify({ access_token: "stub-gh-token", token_type: "bearer", scope: "read:user" }));
      return;
    }

    if (url.pathname === "/user") {
      res.writeHead(200, { "Content-Type": "application/json" });
      res.end(JSON.stringify({ id: 424242, login: "smoke-user", email: "smoke@example.com" }));
      return;
    }

    res.writeHead(404).end("not found");
  });
  await new Promise((r) => srv.listen(port, "127.0.0.1", r));
  return {
    base: `http://127.0.0.1:${port}`,
    requests,
    close: () => new Promise((r) => srv.close(r)),
  };
}

// bootCore starts the stack and returns handles. Call shutdown() in a finally.
export async function bootCore({
  cli = "demo",
  // "password": register an account and log the CLI in with it (what the
  // WebCrypto smoke test uses). "device": walk the real GitHub device flow —
  // the test opens the verification URL and authorises it — which is how a
  // user signs in, and leaves CLI and browser on the SAME identity.
  login = "password",
  keep = false,
  verbose = false,
  withGitHubStub = false,
  databaseURL = "",
  relayBin = join(ROOT, "apps/relay/bin/relay"),
  riffpadBin = join(ROOT, "apps/daemon/bin/riffpad"),
} = {}) {
  for (const b of [relayBin, riffpadBin]) {
    if (!existsSync(b)) throw new Error(`binary not found: ${b} (build it first)`);
  }

  const workDir = mkdtempSync(join(tmpdir(), "riffpad-core-"));
  const relayDir = join(workDir, "relay");
  const daemonDir = join(workDir, "daemon");
  mkdirSync(relayDir, { recursive: true });
  mkdirSync(daemonDir, { recursive: true });

  const relayPort = await freePort();
  const daemonPort = await freePort();
  const relayBase = `http://127.0.0.1:${relayPort}`;
  const daemonBase = `http://127.0.0.1:${daemonPort}`;
  const localToken = "smoke-local-token";
  writeFileSync(join(daemonDir, "config.json"), JSON.stringify({ port: daemonPort, localToken }));

  const procs = [];
  let githubStub = null;
  let githubEnv = {};
  if (withGitHubStub) {
    githubStub = await startGitHubStub({ verbose });
    githubEnv = {
      GITHUB_CLIENT_ID: "stub-client-id",
      GITHUB_CLIENT_SECRET: "stub-client-secret",
      GITHUB_AUTHORIZE_URL: `${githubStub.base}/authorize`,
      GITHUB_TOKEN_URL: `${githubStub.base}/token`,
      GITHUB_USER_URL: `${githubStub.base}/user`,
      // GitHub would redirect the browser back to the relay's public callback;
      // keep it local so the popup never leaves the test machine.
      GITHUB_REDIRECT_URL: `${relayBase}/api/auth/github/callback`,
    };
  }

  const relay = spawn(relayBin, [], {
    env: {
      ...process.env,
      RELAY_PORT: String(relayPort),
      RELAY_LISTEN: "127.0.0.1",
      RELAY_DATA_DIR: relayDir,
      // Everything the relay tells a client to visit must stay local: the
      // device-login verification URL is built from this.
      RIFFPAD_APP_URL: relayBase,
      ...(databaseURL ? { DATABASE_URL: databaseURL } : {}),
      ...githubEnv,
    },
    stdio: verbose ? "inherit" : "pipe",
  });
  procs.push(relay);

  // RIFFPAD_NO_BROWSER: the device flow would otherwise xdg-open the test
  // machine's real browser. This harness drives its own.
  const cliEnv = {
    RIFFPAD_DIR: daemonDir,
    RIFFPAD_URL: daemonBase,
    RIFFPAD_RELAY_PASSWORD: ACCOUNT.password,
    RIFFPAD_NO_BROWSER: "1",
  };

  const run = (args, { env = cliEnv, timeout = 90_000 } = {}) =>
    spawnSync(riffpadBin, ["--lang", "en", ...args], { encoding: "utf8", timeout, env: { ...process.env, ...env } });

  async function shutdown() {
    // Ask the daemon to stop first: it is an HTTP call, and after the SIGKILLs
    // below it would be pointless. A clean exit also closes its SQLite handles.
    await fetch(`${daemonBase}/api/shutdown`, {
      method: "POST",
      headers: { "X-Riffpad-Token": localToken },
    }).catch(() => {});
    for (const p of procs) {
      try {
        if (!p.killed && p.exitCode === null) p.kill("SIGKILL");
      } catch {
        // already gone
      }
    }
    // Wait for the children to exit before deleting their data: a killed
    // process still finishes in-flight writes (SQLite WAL/journal), and
    // removing the directory underneath it produced ENOTEMPTY (#322).
    await Promise.all(
      procs.map((p) =>
        p.exitCode !== null
          ? null
          : new Promise((res) => {
              p.once("exit", res);
              setTimeout(res, 3000).unref();
            }),
      ),
    );
    if (githubStub) await githubStub.close();
    if (keep) {
      console.log(`\nkept: ${workDir}`);
      return;
    }
    try {
      rmSync(workDir, { recursive: true, force: true, maxRetries: 5, retryDelay: 200 });
    } catch (e) {
      // Cleanup is housekeeping: it must never turn a passing run into a failure.
      console.log(`\nwarning: could not remove ${workDir} (${e.code || e.message})`);
    }
  }

  try {
    await waitFor(async () => {
      const { res } = await jfetch(`${relayBase}/api/status`);
      return res.ok;
    }, { what: "relay /api/status", timeout: 20_000 });

    const readCfg = () => {
      try {
        return JSON.parse(readFileSync(join(daemonDir, "config.json"), "utf8"));
      } catch {
        return {};
      }
    };

    let ownerToken = "";
    let verificationURL = "";
    let loginDone = Promise.resolve(0);

    if (login === "password") {
      const reg = await jfetch(`${relayBase}/api/auth/register`, { method: "POST", body: ACCOUNT });
      ownerToken = reg.data.token || "";
      if (!ownerToken) throw new Error("account registration failed");

      // `riffpad login --username` — the real CLI command against the relay.
      const res = run(["login", "--url", `ws://127.0.0.1:${relayPort}`, "--username", ACCOUNT.username]);
      if (res.status !== 0 || !readCfg().relayToken) {
        throw new Error(`riffpad login failed: ${(res.stdout || "") + (res.stderr || "")}`);
      }
    } else {
      // `riffpad login` with no --username starts the GitHub device flow: it
      // prints a verification URL and polls until the browser authorises it.
      const child = spawn(riffpadBin,
        ["--lang", "en", "login", "--url", `ws://127.0.0.1:${relayPort}`],
        { env: { ...process.env, ...cliEnv } });
      procs.push(child);
      let out = "";
      child.stdout.on("data", (d) => { out += d; });
      child.stderr.on("data", (d) => { out += d; });
      verificationURL = await waitFor(() => (out.match(/https?:\/\/\S+/) || [])[0] || null,
        { what: "device verification URL", timeout: 30_000 });
      loginDone = new Promise((res) => child.on("exit", res));
      // The device flow only completes once a browser authorises it, so the
      // caller drives that and then awaits waitForLogin().
    }

    // Wait for the device flow to finish before anything needs the token.
    async function waitForLogin() {
      if (login === "password") return;
      const code = await loginDone;
      if (code !== 0) throw new Error(`riffpad login exited ${code}`);
      await waitFor(() => !!readCfg().relayToken, { what: "relay token in config", timeout: 15_000 });
    }

    // `riffpad pair` — lazy-starts the daemon and mints a relay pairing code.
    // The CLI already retries while the daemon's relay socket comes up; the
    // extra wait just confirms the host registered before callers continue.
    async function pair() {
      if (!readCfg().relayToken) throw new Error("not logged in yet: await waitForLogin() first");
      const out = run(["pair"]);
      const code = (out.stdout.match(/Pairing code:\s*([A-Z0-9]{6})/) || [])[1] || "";
      if (out.status !== 0 || !code) {
        throw new Error(`riffpad pair failed: ${(out.stdout || "") + (out.stderr || "")}`);
      }
      await waitFor(async () => {
        const { data } = await jfetch(`${relayBase}/api/status`);
        return data.hosts > 0;
      }, { what: "host presence on relay", timeout: 20_000 });
      return code;
    }

    // `riffpad run <cli>` — a real session.
    function runSession() {
      if (!readCfg().relayToken) throw new Error("not logged in yet: await waitForLogin() first");
      const out = run(["run", cli]);
      const sid = (out.stdout.match(/session=([0-9a-zA-Z]+)/) || [])[1] || "";
      if (out.status !== 0 || !sid) {
        throw new Error(`riffpad run ${cli} failed: ${(out.stdout || "") + (out.stderr || "")}`);
      }
      return sid;
    }

    return {
      relayBase,
      daemonBase,
      get relayWs() { return readCfg().relayUrl; },
      get relayToken() { return readCfg().relayToken; },
      verificationURL,
      loginMode: login,
      waitForLogin,
      ownerToken,
      localToken,
      workDir,
      relayDir,
      daemonDir,
      githubStub,
      cliEnv,
      pair,
      runSession,
      shutdown,
    };
  } catch (e) {
    await shutdown();
    throw e;
  }
}
