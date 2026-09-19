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

import { bootCore, jfetch, readFileSync, sleep, statSync, waitFor, walk } from "./lib/core-harness.mjs";
import { dirname, join, resolve } from "node:path";
import { fileURLToPath } from "node:url";

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

let core = null;
let relayBase = "";
let daemonBase = "";
let localToken = "";
let workDir = "";

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

// ---------- main ----------

async function main() {
  workDir = "starting…";
  console.log(`Riffpad core-path smoke — cli=${CLI}`);

  // 1-6: relay boots, account, `riffpad login`, `riffpad pair`,
  // host registration, `riffpad run <cli>` — bootCore throws on any failure,
  // so reaching here already proves the CLI half of the chain.
  core = await bootCore({
    cli: CLI,
    keep: KEEP,
    verbose: VERBOSE,
    databaseURL: DATABASE_URL,
    relayBin: opt("relay-bin", join(ROOT, "apps/relay/bin/relay")),
    riffpadBin: opt("riffpad-bin", join(ROOT, "apps/daemon/bin/riffpad")),
  });
  relayBase = core.relayBase;
  daemonBase = core.daemonBase;
  localToken = core.localToken;
  workDir = core.workDir;
  console.log(`workdir=${workDir}\n`);

  const { data: st } = await jfetch(`${relayBase}/api/status`);
  check("1 relay boots", !!st.version, `${st.name} ${st.version}`);
  check("2 account registered", !!core.ownerToken, `user=${ACCOUNT.username}`);
  check("3 riffpad login saved relay token", !!core.relayToken, `relayUrl=${core.relayWs}`);
  const code = await core.pair();
  check("4 riffpad pair printed a code", !!code, `code=${code}`);
  const { data: afterPair } = await jfetch(`${relayBase}/api/status`);
  check("5 daemon connected to relay as host", afterPair.hosts > 0, `hosts=${afterPair.hosts}`);
  const sid = core.runSession();
  check("6 riffpad run created a session", !!sid, `session=${sid}`);

  // ---- 7. viewer pairs with the code, exactly like the browser does ----
  const viewerKp = await genKeyPair();
  const viewerPub = b64u(await crypto.subtle.exportKey("raw", viewerKp.publicKey));
  const paired = await jfetch(`${relayBase}/api/pair`, {
    method: "POST",
    token: core.ownerToken,
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
  const wsUrl = `${core.relayWs.replace(/^http/, "ws")}/ws?device=${deviceId}&session=${sid}` +
    `&eph=${ephPub}&token=${encodeURIComponent(core.relayToken)}`;

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
  for (const f of walk(core.relayDir)) {
    if (statSync(f).size > 32 * 1024 * 1024) continue;
    const blob = readFileSync(f, "latin1");
    for (const n of needles) if (blob.includes(n)) hits.push(`${f}:${n}`);
  }
  check("13 relay storage holds no plaintext", hits.length === 0, hits.join(", ") || `${walk(core.relayDir).length} file(s) scanned`);

  ws.close();
}

let exitCode = 0;
try {
  await main();
} catch (e) {
  console.log(`\nERROR  ${e.message}`);
  exitCode = 99;
} finally {
  if (core) await core.shutdown();
  const passed = checks.filter((c) => c.ok).length;
  if (!exitCode && failed) exitCode = failed;
  console.log(`\n${passed}/${checks.length} checks passed in ${((Date.now() - started) / 1000).toFixed(1)}s`);
  process.exit(exitCode);
}
