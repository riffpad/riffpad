#!/usr/bin/env node
//
// Riffpad M1 acceptance: full-stack pass — real browser-equivalent viewer
// through the production relay to the local daemon and a real agent CLI.
//
// Flow:
//   1. daemon reachable, relay auth rejects anonymous
//   2. viewer logs in with the daemon's saved relay token (same owner)
//   3. host creates a real agent session (kimi/codex/claude)
//   4. host generates a pairing code (forwarded to relay)
//   5. viewer pairs, connects WS (E2EE), sees session_start + agent reply
//   6. viewer sends an encrypted prompt, receives the agent's reply
//
// Usage:
//   node scripts/e2e-acceptance.mjs [--cli kimi] [--prompt "..."] [--cwd ...]
//
// Env overrides:
//   RIFFPAD_URL   daemon base (default http://127.0.0.1:8787)
//   RIFFPAD_DIR   daemon data dir (default ~/.config/riffpad)
//
// Exit 0 = all checks passed; nonzero = first failing check index.

import { readFileSync } from "node:fs";
import { homedir } from "node:os";
import { join } from "node:path";
import { setTimeout as sleep } from "node:timers/promises";

const args = process.argv.slice(2);
const opt = (name, def) => {
  const i = args.indexOf(`--${name}`);
  return i >= 0 && args[i + 1] ? args[i + 1] : def;
};

const DAEMON = process.env.RIFFPAD_URL || "http://127.0.0.1:8787";
const DATA_DIR = process.env.RIFFPAD_DIR || join(homedir(), ".config", "riffpad");
const CLI = opt("cli", "kimi");
const PROMPT = opt("prompt", "Reply with exactly: OK");
const CWD = opt("cwd", "/tmp/riffpad-acp-test");
const EXISTING_SESSION = opt("session", "");
const TIMEOUT_MS = 120_000;

const enc = new TextEncoder();
const dec = new TextDecoder();

function b64u(buf) {
  const b = buf instanceof Uint8Array ? buf : new Uint8Array(buf);
  let s = "";
  for (let i = 0; i < b.length; i += 0x8000) {
    s += String.fromCharCode(...b.subarray(i, i + 0x8000));
  }
  return btoa(s).replace(/\+/g, "-").replace(/\//g, "_").replace(/=+$/, "");
}

function b64uToBytes(s) {
  const b = atob(s.replace(/-/g, "+").replace(/_/g, "/"));
  const u = new Uint8Array(b.length);
  for (let i = 0; i < b.length; i++) u[i] = b.charCodeAt(i);
  return u;
}

async function jfetch(url, opts = {}, token) {
  const headers = { ...(opts.headers || {}) };
  if (token) headers.Authorization = `Bearer ${token}`;
  if (opts.body) headers["Content-Type"] = "application/json";
  const res = await fetch(url, { ...opts, headers });
  const data = await res.json().catch(() => ({}));
  return { res, data };
}

const checks = [];
function check(name, ok, detail = "") {
  checks.push({ name, ok, detail });
  console.log(`${ok ? "PASS" : "FAIL"}  ${name}${detail ? " — " + detail : ""}`);
  return ok;
}

// ---------- crypto helpers (viewer side) ----------
async function genKeyPair() {
  return crypto.subtle.generateKey({ name: "ECDH", namedCurve: "P-256" }, true, [
    "deriveBits",
  ]);
}

async function deriveSessionKey(ephPriv, serverEphPubB64, deviceSecret, sid) {
  const serverEph = await crypto.subtle.importKey(
    "raw",
    b64uToBytes(serverEphPubB64),
    { name: "ECDH", namedCurve: "P-256" },
    false,
    [],
  );
  const bits = await crypto.subtle.deriveBits(
    { name: "ECDH", public: serverEph },
    ephPriv,
    256,
  );
  const hkdf = await crypto.subtle.importKey("raw", bits, "HKDF", false, ["deriveKey"]);
  return crypto.subtle.deriveKey(
    {
      name: "HKDF",
      hash: "SHA-256",
      salt: deviceSecret,
      info: enc.encode(`riffpad/session-v1/${sid}`),
    },
    hkdf,
    { name: "AES-GCM", length: 256 },
    false,
    ["encrypt", "decrypt"],
  );
}

async function encryptEvent(key, sid, ev) {
  const iv = crypto.getRandomValues(new Uint8Array(12));
  const pt = enc.encode(JSON.stringify(ev));
  const ct = await crypto.subtle.encrypt(
    { name: "AES-GCM", iv, additionalData: enc.encode(sid) },
    key,
    pt,
  );
  return { nonce: b64u(iv), ciphertext: b64u(ct) };
}

async function decryptEvent(key, sid, nonceB64, ctB64) {
  const pt = await crypto.subtle.decrypt(
    { name: "AES-GCM", iv: b64uToBytes(nonceB64), additionalData: enc.encode(sid) },
    key,
    b64uToBytes(ctB64),
  );
  return JSON.parse(dec.decode(pt));
}

// ---------- main ----------
const started = Date.now();
console.log(`Riffpad M1 acceptance — cli=${CLI} daemon=${DAEMON}`);

// 1. daemon reachable
let daemonStatus = null;
try {
  const r = await fetch(`${DAEMON}/api/status`);
  daemonStatus = await r.json();
} catch {
  daemonStatus = null;
}
if (!check("1 daemon reachable", !!daemonStatus, daemonStatus ? `port=${daemonStatus.port}` : "")) {
  process.exit(1);
}

// 2. relay config from daemon (same owner token)
let cfg = {};
try {
  cfg = JSON.parse(readFileSync(join(DATA_DIR, "config.json"), "utf8"));
} catch {
  cfg = {};
}
const relayWS = cfg.relayUrl || cfg.RelayURL || "";
const relayBase = relayWS.replace(/^wss:\/\//, "https://").replace(/^ws:\/\//, "http://");
const token = cfg.relayToken || cfg.RelayToken || "";
if (!check("2 relay token from daemon config", !!relayWS && !!token, relayWS)) {
  process.exit(2);
}

// anonymous rejected
const anon = await jfetch(`${relayBase}/api/auth/me`);
check("3 relay rejects anonymous", anon.res.status === 401, `status=${anon.res.status}`);

// 4. host creates a real agent session (or reuses an existing one)
let sid = EXISTING_SESSION;
if (sid) {
  check("4 reuse existing agent session", true, `session=${sid}`);
} else {
  const created = await jfetch(`${DAEMON}/api/sessions`, {
    method: "POST",
    body: JSON.stringify({ name: "acceptance", cli: CLI, prompt: PROMPT, cwd: CWD }),
  });
  sid = created.data.id;
  if (!check("4 host created agent session", created.res.ok && !!sid, `session=${sid}`)) {
    process.exit(4);
  }
}

// 5. host pairing code (daemon forwards to relay)
const pairRes = await jfetch(`${DAEMON}/api/pairings`, { method: "POST" });
const code = pairRes.data.code;
if (!check("5 host pairing code", pairRes.res.ok && !!code, `code=${code}`)) {
  process.exit(5);
}

// 6. viewer: keypair + pair via relay
const viewerKp = await genKeyPair();
const viewerPub = b64u(await crypto.subtle.exportKey("raw", viewerKp.publicKey));
const paired = await jfetch(
  `${relayBase}/api/pair`,
  {
    method: "POST",
    body: JSON.stringify({ code, name: "e2e-acceptance", curve: "p256", publicKey: viewerPub }),
  },
  token,
);
const deviceId = paired.data.deviceId;
const hostPub = paired.data.serverPublicKey;
if (!check("6 viewer paired via relay", paired.res.ok && !!deviceId && !!hostPub, `device=${deviceId}`)) {
  process.exit(6);
}

// device secret (viewer private key + host public key)
const hostPubKey = await crypto.subtle.importKey(
  "raw",
  b64uToBytes(hostPub),
  { name: "ECDH", namedCurve: "P-256" },
  false,
  [],
);
const secretBits = await crypto.subtle.deriveBits(
  { name: "ECDH", public: hostPubKey },
  viewerKp.privateKey,
  256,
);
const deviceSecret = new Uint8Array(secretBits);

// 7. connect WS with a fresh ephemeral key
const eph = await genKeyPair();
const ephPub = b64u(await crypto.subtle.exportKey("raw", eph.publicKey));
const wsUrl =
  `${relayWS.replace(/^wss:\/\//, "wss://")}/ws` +
  `?device=${deviceId}&session=${sid}&eph=${ephPub}&token=${encodeURIComponent(token)}`;

const events = [];
let sessionKey = null;
let wsError = null;
let wsConnected = false;
const DEBUG = process.env.RIFFPAD_E2E_DEBUG === "1";

const ws = new WebSocket(wsUrl);
const wsReady = new Promise((resolve) => {
  ws.onopen = () => {
    wsConnected = true;
    if (DEBUG) console.log("[ws] open");
    resolve();
  };
  ws.onerror = (e) => {
    wsError = e?.message || "ws error";
    if (DEBUG) console.log("[ws] error", wsError);
    resolve();
  };
  ws.onclose = (e) => {
    if (DEBUG) console.log("[ws] close", e?.code, e?.reason);
    resolve();
  };
});
ws.onmessage = async (msg) => {
  try {
    if (DEBUG) console.log("[ws] recv", String(msg.data).slice(0, 200));
    const data = JSON.parse(String(msg.data));
    if (data.kind === "hello") {
      sessionKey = await deriveSessionKey(eph.privateKey, data.serverEphPub, deviceSecret, sid);
      return;
    }
    if (sessionKey && data.nonce && data.ciphertext) {
      const ev = await decryptEvent(sessionKey, sid, data.nonce, data.ciphertext);
      events.push(ev);
    }
  } catch (e) {
    wsError = e?.message || "decrypt error";
  }
};

await Promise.race([wsReady, sleep(15_000)]);
const helloDeadline = Date.now() + 15_000;
while (!sessionKey && !wsError && Date.now() < helloDeadline) {
  await sleep(200);
}
check("7 viewer WS connected (E2EE)", wsConnected && !!sessionKey, wsError || "");
if (!wsConnected || !sessionKey) process.exit(7);

// 8. wait for initial agent reply (or existing history when reusing a session)
const deadline = Date.now() + TIMEOUT_MS;
let gotReply = false;
while (Date.now() < deadline && !gotReply) {
  gotReply = events.some((e) => e.type === "agent_message");
  if (!gotReply) await sleep(1000);
}
const replyText = events.find((e) => e.type === "agent_message")?.payload?.text || "";
check("8 agent reply visible to viewer", gotReply, replyText.slice(0, 60));
if (!gotReply) {
  console.log("  events seen:", events.map((e) => `${e.type}:${e.payload?.status || e.payload?.text?.slice(0, 20) || ""}`).join(" | "));
}

// 9. viewer sends an encrypted prompt and receives the reply
const sent = await (async () => {
  if (!sessionKey) return false;
  const ev = {
    id: "acc-" + Date.now(),
    sessionId: sid,
    timestamp: Date.now(),
    type: "prompt",
    payload: { text: "Reply with exactly: PONG" },
  };
  const boxed = await encryptEvent(sessionKey, sid, ev);
  ws.send(JSON.stringify({ v: 1, kind: "event", sessionId: sid, ...boxed }));
  return true;
})();
check("9 viewer sent encrypted prompt", sent);

const before = events.length;
let gotPong = false;
const pongDeadline = Date.now() + TIMEOUT_MS;
while (Date.now() < pongDeadline && !gotPong) {
  const tail = events.slice(before).map((e) => e.payload?.text || "").join(" ");
  gotPong = /pong/i.test(tail);
  if (!gotPong) await sleep(1000);
}
check("10 agent replied to viewer prompt", gotPong, "reply contains PONG");

ws.close();
await jfetch(`${DAEMON}/api/sessions/${sid}/stop`, { method: "POST" });

const passed = checks.filter((c) => c.ok).length;
console.log(`\n${passed}/${checks.length} checks passed in ${((Date.now() - started) / 1000).toFixed(1)}s`);
process.exit(passed === checks.length ? 0 : 10);
