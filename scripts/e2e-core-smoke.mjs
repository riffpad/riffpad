#!/usr/bin/env node
//
// Core-path smoke test (#314): the whole chain, the way a user walks it.
//
//   riffpad login -> riffpad pair -> riffpad run <cli>
//        -> daemon announces the session to the relay
//        -> a viewer pairs with the printed code, does the E2EE handshake,
//           receives the event timeline, approves a request and gets a reply.
//
// Everything is real here: compiled `relay` and `riffpad` binaries, real HTTP
// and WebSocket, real E2EE (the viewer side is a browser-equivalent WebCrypto
// client, so the Go daemon and the TS client are cross-checked against each
// other). Only the agent CLI is scripted — `--cli demo` replays a fixed
// timeline, which is what makes this deterministic and free of API quota.
// Pass `--cli codex|claude|kimi` to run the same chain against a real agent.
//
// Usage:
//   node scripts/e2e-core-smoke.mjs [--cli demo] [--relay-bin PATH] [--riffpad-bin PATH]
//                                   [--database-url DSN] [--keep] [--verbose]
//
// Exit 0 = every check passed; nonzero = the index of the first failing check.

import { spawn, spawnSync } from "node:child_process";
import { existsSync, mkdirSync, mkdtempSync, readFileSync, readdirSync, rmSync, statSync, writeFileSync } from "node:fs";
import { createServer } from "node:net";
import { tmpdir } from "node:os";
import { dirname, join, resolve } from "node:path";
import { fileURLToPath } from "node:url";
import { setTimeout as sleep } from "node:timers/promises";

const ROOT = resolve(dirname(fileURLToPath(import.meta.url)), "..");
const args = process.argv.slice(2);
const opt = (name, def) => {
  const i = args.indexOf(`--${name}`);
  return i >= 0 && args[i + 1] && !args[i + 1].startsWith("--") ? args[i + 1] : def;
};
const flag = (name) => args.includes(`--${name}`);

const CLI = opt("cli", "demo");
const KEEP = flag("keep");
const VERBOSE = flag("verbose");
const DATABASE_URL = opt("database-url", "");
const ACCOUNT = { username: "smoke", password: "smoke-password-123" };
const MARKER = "SMOKE-" + Math.random().toString(36).slice(2, 10).toUpperCase();

let started = Date.now();
let failed = 0;
const checks = [];

function check(name, ok, detail = "") {
  checks.push({ name, ok });
  console.log(`${ok ? "PASS" : "FAIL"}  ${name}${detail ? " — " + detail : ""}`);
  if (!ok && !failed) failed = checks.length;
  return ok;
}

function log(...a) {
  if (VERBOSE) console.log("[smoke]", ...a);
}

// ---------- process helpers ----------

function freePort() {
  return new Promise((res, rej) => {
    const srv = createServer();
    srv.on("error", rej);
    srv.listen(0, "127.0.0.1", () => {
      const { port } = srv.address();
      srv.close(() => res(port));
    });
  });
}

async function waitFor(fn, { timeout = 30_000, interval = 250, what = "condition" } = {}) {
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

// run executes a binary to completion and returns its combined output.
function run(bin, cmdArgs, { env = {}, timeout = 60_000 } = {}) {
  const r = spawnSync(bin, cmdArgs, {
    encoding: "utf8",
    timeout,
    env: { ...process.env, ...env },
  });
  const out = `${r.stdout || ""}${r.stderr || ""}`;
  log("$", bin, cmdArgs.join(" "), "->", r.status, out.trim().split("\n")[0]);
  return { status: r.status, out };
}

async function jfetch(url, { method = "GET", body, token } = {}) {
  const headers = {};
  if (token) headers.Authorization = `Bearer ${token}`;
  if (body) headers["Content-Type"] = "application/json";
  const res = await fetch(url, { method, headers, body: body ? JSON.stringify(body) : undefined });
  const data = await res.json().catch(() => ({}));
  return { res, data };
}

// ---------- viewer crypto (browser-equivalent WebCrypto) ----------

const enc = new TextEncoder();
const dec = new TextDecoder();

const b64u = (buf) => {
  const b = buf instanceof Uint8Array ? buf : new Uint8Array(buf);
  let s = "";
  for (let i = 0; i < b.length; i += 0x8000) s += String.fromCharCode(...b.subarray(i, i + 0x8000));
  return btoa(s).replace(/\+/g, "-").replace(/\//g, "_").replace(/=+$/, "");
};
const b64uToBytes = (s) => {
  const b = atob(s.replace(/-/g, "+").replace(/_/g, "/"));
  const u = new Uint8Array(b.length);
  for (let i = 0; i < b.length; i++) u[i] = b.charCodeAt(i);
  return u;
};
const genKeyPair = () =>
  crypto.subtle.generateKey({ name: "ECDH", namedCurve: "P-256" }, true, ["deriveBits"]);

async function deriveSessionKey(ephPriv, serverEphPubB64, deviceSecret, sid) {
  const serverEph = await crypto.subtle.importKey(
    "raw", b64uToBytes(serverEphPubB64), { name: "ECDH", namedCurve: "P-256" }, false, [],
  );
  const bits = await crypto.subtle.deriveBits({ name: "ECDH", public: serverEph }, ephPriv, 256);
  const hkdf = await crypto.subtle.importKey("raw", bits, "HKDF", false, ["deriveKey"]);
  return crypto.subtle.deriveKey(
    {
      name: "HKDF", hash: "SHA-256", salt: deviceSecret,
      info: enc.encode(`riffpad/session-v1/${sid}`),
    },
    hkdf, { name: "AES-GCM", length: 256 }, false, ["encrypt", "decrypt"],
  );
}

async function encryptEvent(key, sid, ev) {
  const iv = crypto.getRandomValues(new Uint8Array(12));
  const ct = await crypto.subtle.encrypt(
    { name: "AES-GCM", iv, additionalData: enc.encode(sid) }, key, enc.encode(JSON.stringify(ev)),
  );
  return { nonce: b64u(iv), ciphertext: b64u(ct) };
}

async function decryptEvent(key, sid, nonceB64, ctB64) {
  const pt = await crypto.subtle.decrypt(
    { name: "AES-GCM", iv: b64uToBytes(nonceB64), additionalData: enc.encode(sid) },
    key, b64uToBytes(ctB64),
  );
  return JSON.parse(dec.decode(pt));
}

// ---------- teardown ----------

const procs = [];
let relayBase = "";
let daemonBase = "";
let localToken = "";
let workDir = "";

async function shutdown() {
  for (const p of procs.reverse()) {
    try {
      if (!p.killed) p.kill("SIGKILL");
    } catch {
      // already gone
    }
  }
  if (daemonBase) {
    await fetch(`${daemonBase}/api/shutdown`, {
      method: "POST",
      headers: { "X-Riffpad-Token": localToken },
    }).catch(() => {});
  }
  if (!KEEP && workDir) rmSync(workDir, { recursive: true, force: true });
  else if (workDir) console.log(`\nkept: ${workDir}`);
}

// Recursively collect every file under dir (used for the zero-trust check).
function walk(dir, out = []) {
  if (!existsSync(dir)) return out;
  for (const e of readdirSync(dir, { withFileTypes: true })) {
    const p = join(dir, e.name);
    if (e.isDirectory()) walk(p, out);
    else if (e.isFile()) out.push(p);
  }
  return out;
}

// ---------- main ----------

async function main() {
  const relayBin = opt("relay-bin", join(ROOT, "apps/relay/bin/relay"));
  const riffpadBin = opt("riffpad-bin", join(ROOT, "apps/daemon/bin/riffpad"));
  for (const b of [relayBin, riffpadBin]) {
    if (!existsSync(b)) throw new Error(`binary not found: ${b} (build it first)`);
  }

  workDir = mkdtempSync(join(tmpdir(), "riffpad-smoke-"));
  const relayDir = join(workDir, "relay");
  const daemonDir = join(workDir, "daemon");
  mkdirSync(relayDir, { recursive: true });
  mkdirSync(daemonDir, { recursive: true });

  const relayPort = await freePort();
  const daemonPort = await freePort();
  relayBase = `http://127.0.0.1:${relayPort}`;
  daemonBase = `http://127.0.0.1:${daemonPort}`;
  localToken = "smoke-local-token";
  writeFileSync(join(daemonDir, "config.json"), JSON.stringify({ port: daemonPort, localToken }));

  console.log(`Riffpad core-path smoke — cli=${CLI} workdir=${workDir}\n`);

  // ---- 1. relay boots ----
  const relay = spawn(relayBin, [], {
    env: {
      ...process.env,
      RELAY_PORT: String(relayPort),
      RELAY_LISTEN: "127.0.0.1",
      RELAY_DATA_DIR: relayDir,
      ...(DATABASE_URL ? { DATABASE_URL } : {}),
    },
    stdio: VERBOSE ? "inherit" : "pipe",
  });
  procs.push(relay);
  let relayStatus = null;
  try {
    relayStatus = await waitFor(async () => {
      const { res, data } = await jfetch(`${relayBase}/api/status`);
      return res.ok ? data : null;
    }, { what: "relay /api/status", timeout: 20_000 });
  } catch (e) {
    check("1 relay boots", false, e.message);
    return;
  }
  check("1 relay boots", !!relayStatus?.version, `${relayStatus?.name} ${relayStatus?.version}`);

  // ---- 2. account exists (the web signup step; GitHub OAuth creates the same
  //          user in production, the password path is what the CLI can drive) ----
  const reg = await jfetch(`${relayBase}/api/auth/register`, {
    method: "POST",
    body: ACCOUNT,
  });
  const ownerToken = reg.data.token || "";
  check("2 account registered", !!ownerToken, `user=${reg.data.user?.username}`);

  // ---- 3. `riffpad login` against that relay ----
  const login = run(riffpadBin, ["--lang", "en", "login", "--url", `ws://127.0.0.1:${relayPort}`, "--username", ACCOUNT.username], {
    env: { RIFFPAD_DIR: daemonDir, RIFFPAD_RELAY_PASSWORD: ACCOUNT.password },
  });
  let cfg = {};
  try {
    cfg = JSON.parse(readFileSync(join(daemonDir, "config.json"), "utf8"));
  } catch {
    cfg = {};
  }
  check(
    "3 riffpad login saved relay token",
    login.status === 0 && !!cfg.relayToken,
    `relayUrl=${cfg.relayUrl || "-"}`,
  );
  if (!cfg.relayToken) return;

  const cliEnv = {
    RIFFPAD_DIR: daemonDir,
    RIFFPAD_URL: daemonBase,
    RIFFPAD_RELAY_PASSWORD: ACCOUNT.password,
  };

  // ---- 4. `riffpad pair` (lazy-starts the daemon, mints a relay pairing code) ----
  const pair = run(riffpadBin, ["--lang", "en", "pair"], { env: cliEnv, timeout: 90_000 });
  const code = (pair.out.match(/Pairing code:\s*([A-Z0-9]{6})/) || [])[1] || "";
  check("4 riffpad pair printed a code", pair.status === 0 && !!code, `code=${code}`);
  if (!code) return;

  // ---- 5. daemon registered itself as a host on the relay ----
  const status = await waitFor(async () => {
    const { data } = await jfetch(`${relayBase}/api/status`);
    return data.hosts > 0 ? data : null;
  }, { what: "host presence on relay", timeout: 20_000 }).catch(() => null);
  check("5 daemon connected to relay as host", !!status, `hosts=${status?.hosts ?? 0}`);

  // ---- 6. `riffpad run <cli>` creates a real session ----
  const runRes = run(riffpadBin, ["--lang", "en", "run", CLI], { env: cliEnv, timeout: 90_000 });
  const sid = (runRes.out.match(/session=([0-9a-zA-Z]+)/) || [])[1] || "";
  check("6 riffpad run created a session", runRes.status === 0 && !!sid, `session=${sid}`);
  if (!sid) return;

  // ---- 7. viewer pairs with the code, exactly like the browser does ----
  const viewerKp = await genKeyPair();
  const viewerPub = b64u(await crypto.subtle.exportKey("raw", viewerKp.publicKey));
  const paired = await jfetch(`${relayBase}/api/pair`, {
    method: "POST",
    token: ownerToken,
    body: { code, name: "core-smoke", curve: "p256", publicKey: viewerPub },
  });
  const deviceId = paired.data.deviceId || "";
  const hostPub = paired.data.serverPublicKey || "";
  check("7 viewer paired via relay", paired.res.ok && !!deviceId && !!hostPub, `device=${deviceId}`);
  if (!deviceId) return;

  const hostPubKey = await crypto.subtle.importKey(
    "raw", b64uToBytes(hostPub), { name: "ECDH", namedCurve: "P-256" }, false, [],
  );
  const deviceSecret = new Uint8Array(
    await crypto.subtle.deriveBits({ name: "ECDH", public: hostPubKey }, viewerKp.privateKey, 256),
  );

  // ---- 8. viewer connects the E2EE WebSocket ----
  const eph = await genKeyPair();
  const ephPub = b64u(await crypto.subtle.exportKey("raw", eph.publicKey));
  const wsUrl = `${cfg.relayUrl.replace(/^http/, "ws")}/ws?device=${deviceId}&session=${sid}` +
    `&eph=${ephPub}&token=${encodeURIComponent(cfg.relayToken)}`;

  const events = [];
  let sessionKey = null;
  let wsError = null;
  let wsOpen = false;
  const pending = [];

  const ws = new WebSocket(wsUrl);
  ws.onopen = () => { wsOpen = true; };
  ws.onerror = (e) => { wsError = e?.message || "ws error"; };
  ws.onclose = (e) => { if (!sessionKey && e?.code) wsError = `closed ${e.code}`; };
  ws.onmessage = async (msg) => {
    pending.push((async () => {
      try {
        const data = JSON.parse(String(msg.data));
        if (data.kind === "hello") {
          sessionKey = await deriveSessionKey(eph.privateKey, data.serverEphPub, deviceSecret, sid);
          return;
        }
        if (sessionKey && data.nonce && data.ciphertext) {
          events.push(await decryptEvent(sessionKey, sid, data.nonce, data.ciphertext));
        }
      } catch (e) {
        wsError = e?.message || "decrypt error";
      }
    })());
  };

  await waitFor(() => sessionKey || wsError, { what: "E2EE handshake", timeout: 20_000 }).catch(() => {});
  check("8 viewer connected with E2EE", wsOpen && !!sessionKey, wsError || "");
  if (!sessionKey) return;

  const settle = () => Promise.all(pending.splice(0));
  const waitEvent = (pred, timeout = 30_000, from = 0) =>
    waitFor(async () => {
      await settle();
      return events.slice(from).find(pred) || null;
    }, { what: "event", timeout }).catch(() => null);

  // ---- 9. the scripted agent's timeline arrives through the relay ----
  const firstMsg = await waitEvent((e) => e.type === "agent_message");
  const fileChange = await waitEvent((e) => e.type === "file_change");
  check(
    "9 agent events reach the viewer",
    !!firstMsg && !!fileChange,
    `types=${[...new Set(events.map((e) => e.type))].join(",")}`,
  );

  // ---- 10. viewer prompt -> relay -> daemon -> agent -> back, encrypted ----
  const send = async (ev) => {
    const boxed = await encryptEvent(sessionKey, sid, ev);
    ws.send(JSON.stringify({ v: 1, kind: "event", sessionId: sid, ...boxed }));
  };
  await send({
    id: "smoke-prompt-" + Date.now(),
    sessionId: sid,
    timestamp: Date.now(),
    type: "prompt",
    payload: { text: MARKER },
  });
  const echoed = await waitEvent((e) => e.type === "agent_message" && String(e.payload?.text || "").includes(MARKER));
  check("10 encrypted prompt round-trips to the agent", !!echoed, MARKER);

  // ---- 11. approval round trip: request -> viewer approves -> agent continues ----
  await send({
    id: "smoke-approval-" + Date.now(),
    sessionId: sid,
    timestamp: Date.now(),
    type: "prompt",
    payload: { text: "approval" },
  });
  const approvalReq = await waitEvent((e) => e.type === "approval_request");
  if (!approvalReq) {
    check("11 approval request reaches the viewer", false, `types=${[...new Set(events.map((e) => e.type))].join(",")}`);
  } else {
    check("11 approval request reaches the viewer", true, `req=${approvalReq.payload?.requestId}`);
    // Only accept events produced *after* the decision: the scripted timeline
    // already contains tool_call rows, so a stale match would fake a pass.
    await settle();
    const from = events.length;
    await send({
      id: "smoke-decision-" + Date.now(),
      sessionId: sid,
      timestamp: Date.now(),
      type: "approval_response",
      payload: { requestId: approvalReq.payload?.requestId, decision: "approve" },
    });
    // "Approved — file written." only appears when the demo's pending
    // approval channel actually received the decision.
    const resolved = await waitEvent(
      (e) => e.type === "agent_message" && /approved/i.test(String(e.payload?.text || "")),
      30_000,
      from,
    );
    check(
      "12 approval decision takes effect",
      !!resolved,
      resolved ? String(resolved.payload?.text).slice(0, 40) :
        `no follow-up; new types=${[...new Set(events.slice(from).map((e) => e.type))].join(",") || "none"}`,
    );
  }

  // ---- 13. zero-trust: the relay never holds plaintext ----
  await settle();
  const needles = [MARKER, "scripted demo reply", "refactor the auth middleware"];
  const hits = [];
  for (const f of walk(relayDir)) {
    if (statSync(f).size > 32 * 1024 * 1024) continue;
    const blob = readFileSync(f, "latin1");
    for (const n of needles) if (blob.includes(n)) hits.push(`${f}:${n}`);
  }
  check("13 relay storage holds no plaintext", hits.length === 0, hits.join(", ") || `${walk(relayDir).length} file(s) scanned`);

  ws.close();
  void relayStatus;
}

let exitCode = 0;
try {
  await main();
} catch (e) {
  console.log(`\nERROR  ${e.message}`);
  exitCode = 99;
} finally {
  await shutdown();
  const passed = checks.filter((c) => c.ok).length;
  if (!exitCode && failed) exitCode = failed;
  console.log(`\n${passed}/${checks.length} checks passed in ${((Date.now() - started) / 1000).toFixed(1)}s`);
  process.exit(exitCode);
}
