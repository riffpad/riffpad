"use strict";

const $ = (id) => document.getElementById(id);
const enc = new TextEncoder();
const dec = new TextDecoder();

const store = {
  get() {
    try { return JSON.parse(localStorage.getItem("riffpad.device")); }
    catch { return null; }
  },
  set(v) { localStorage.setItem("riffpad.device", JSON.stringify(v)); }
};

function b64u(buf) {
  const b = new Uint8Array(buf);
  let s = "";
  for (let i = 0; i < b.length; i += 0x8000) {
    s += String.fromCharCode.apply(null, b.subarray(i, i + 0x8000));
  }
  return btoa(s).replace(/\+/g, "-").replace(/\//g, "_").replace(/=+$/, "");
}
function b64uToBytes(s) {
  const b = atob(s.replace(/-/g, "+").replace(/_/g, "/"));
  const u = new Uint8Array(b.length);
  for (let i = 0; i < b.length; i++) u[i] = b.charCodeAt(i);
  return u;
}
function jwkToRaw(jwk) {
  const x = b64uToBytes(jwk.x), y = b64uToBytes(jwk.y);
  const raw = new Uint8Array(65);
  raw[0] = 4;
  raw.set(x, 1);
  raw.set(y, 33);
  return raw;
}

async function genPair() {
  return crypto.subtle.generateKey({ name: "ECDH", namedCurve: "P-256" }, true, ["deriveBits"]);
}
async function importIdentity(jwk) {
  return crypto.subtle.importKey("jwk", jwk, { name: "ECDH", namedCurve: "P-256" }, false, ["deriveBits"]);
}
async function ensureIdentity() {
  let dev = store.get();
  if (!dev) {
    const kp = await genPair();
    const jwk = await crypto.subtle.exportKey("jwk", kp.privateKey);
    dev = { deviceId: null, serverPub: null, jwk };
    store.set(dev);
  }
  return dev;
}
async function deviceSecret(dev) {
  const priv = await importIdentity(dev.jwk);
  const spub = await crypto.subtle.importKey("raw", b64uToBytes(dev.serverPub), { name: "ECDH", namedCurve: "P-256" }, false, []);
  const bits = await crypto.subtle.deriveBits({ name: "ECDH", public: spub }, priv, 256);
  return new Uint8Array(bits);
}

let ws = null;
let sessionKey = null;
let currentSession = null;

async function pair(code) {
  const dev = await ensureIdentity();
  const raw = jwkToRaw(dev.jwk);
  const res = await fetch("/api/pair", {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ code, name: "web", curve: "p256", publicKey: b64u(raw) })
  });
  const data = await res.json();
  if (!res.ok) throw new Error(data.error || "配对失败");
  dev.deviceId = data.deviceId;
  dev.serverPub = data.serverPublicKey;
  store.set(dev);
}

async function refreshSessions() {
  const res = await fetch("/api/sessions");
  const data = await res.json();
  const list = $("session-list");
  list.innerHTML = "";
  for (const s of (data.sessions || [])) {
    const li = document.createElement("li");
    const name = document.createElement("span");
    name.textContent = s.name || s.id;
    const st = document.createElement("span");
    st.className = "status " + s.status;
    st.textContent = s.status;
    li.append(name, st);
    li.onclick = () => openDetail(s.id, s.name);
    list.appendChild(li);
  }
}

async function createSession(ev) {
  ev.preventDefault();
  const res = await fetch("/api/sessions", {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({
      name: $("s-name").value,
      prompt: $("s-prompt").value,
      cwd: $("s-cwd").value
    })
  });
  const data = await res.json();
  if (!res.ok) throw new Error(data.error || "启动失败");
  $("s-name").value = ""; $("s-prompt").value = ""; $("s-cwd").value = "";
  await refreshSessions();
  openDetail(data.id, data.name);
}

async function openDetail(sid, name) {
  currentSession = sid;
  $("d-title").textContent = name || sid;
  $("events").innerHTML = "";
  $("detail").classList.remove("hidden");
  const dev = await ensureIdentity();
  if (!dev.deviceId) return;
  const eph = await genPair();
  const ephPub = b64u(await crypto.subtle.exportKey("raw", eph.publicKey));
  const proto = location.protocol === "https:" ? "wss" : "ws";
  ws = new WebSocket(`${proto}://${location.host}/ws?device=${dev.deviceId}&session=${sid}&eph=${ephPub}`);
  ws.onopen = () => setConn("连接中…");
  ws.onclose = () => { setConn("已断开"); sessionKey = null; };
  ws.onerror = () => setConn("连接错误");
  ws.onmessage = async (msg) => {
    const data = JSON.parse(msg.data);
    if (data.kind === "hello") {
      const dsec = await deviceSecret(dev);
      const serverEph = await crypto.subtle.importKey("raw", b64uToBytes(data.serverEphPub), { name: "ECDH", namedCurve: "P-256" }, false, []);
      const ephBits = await crypto.subtle.deriveBits({ name: "ECDH", public: serverEph }, eph.privateKey, 256);
      const hkdf = await crypto.subtle.importKey("raw", ephBits, "HKDF", false, ["deriveKey"]);
      sessionKey = await crypto.subtle.deriveKey(
        { name: "HKDF", hash: "SHA-256", salt: dsec, info: enc("riffpad/session-v1/" + sid) },
        hkdf, { name: "AES-GCM", length: 256 }, false, ["encrypt", "decrypt"]
      );
      setConn("已连接（加密）");
      return;
    }
    if (sessionKey) {
      const iv = b64uToBytes(data.nonce);
      const ct = b64uToBytes(data.ciphertext);
      const pt = await crypto.subtle.decrypt({ name: "AES-GCM", iv, additionalData: enc(sid) }, sessionKey, ct);
      renderEvent(JSON.parse(dec.decode(pt)));
    }
  };
}

async function sendEvent(type, payload) {
  if (!ws || !sessionKey) {
    setConn("未连接，无法发送");
    return;
  }
  const ev = { id: String(Date.now()) + Math.random().toString(16).slice(2), sessionId: currentSession, timestamp: Date.now(), type, payload };
  const iv = crypto.getRandomValues(new Uint8Array(12));
  const pt = enc(JSON.stringify(ev));
  const ct = await crypto.subtle.encrypt({ name: "AES-GCM", iv, additionalData: enc(currentSession) }, sessionKey, pt);
  ws.send(JSON.stringify({ v: 1, kind: "event", sessionId: currentSession, nonce: b64u(iv), ciphertext: b64u(ct) }));
}

const LABELS = {
  session_start: "会话开始", session_end: "会话结束", agent_status: "状态",
  agent_message: "Agent", tool_call: "工具调用", file_change: "文件变更",
  command: "命令", approval_request: "审批", approval_response: "审批回复",
  prompt: "指令", control: "控制", notify: "通知"
};

function esc(s) {
  return String(s ?? "").replace(/[&<>"']/g, (c) => ({
    "&": "&amp;", "<": "&lt;", ">": "&gt;", '"': "&quot;", "'": "&#39;"
  }[c]));
}

function eventText(ev) {
  const p = ev.payload || {};
  switch (ev.type) {
    case "agent_message": return p.text || "";
    case "tool_call": return (p.tool || "") + " " + (p.summary || "");
    case "file_change": return p.path || "";
    case "command": return p.command || "";
    case "agent_status": return p.status || "";
    case "notify": return p.message || "";
    case "session_end": return "原因: " + (p.reason || "结束");
    default: return JSON.stringify(p);
  }
}

function renderEvent(ev) {
  const el = document.createElement("div");
  el.className = "ev " + ev.type;
  const head = document.createElement("div");
  head.className = "ev-head";
  head.textContent = LABELS[ev.type] || ev.type;
  el.appendChild(head);
  if (ev.type === "approval_request") {
    const p = ev.payload || {};
    const body = document.createElement("div");
    body.className = "ev-body";
    body.textContent = (p.action ? p.action + "：" : "") + (p.summary || "");
    el.appendChild(body);
    const row = document.createElement("div");
    row.className = "row";
    const ok = document.createElement("button");
    ok.textContent = "同意";
    ok.onclick = () => sendEvent("approval_response", { requestId: p.requestId, decision: "approve" });
    const no = document.createElement("button");
    no.className = "danger";
    no.textContent = "拒绝";
    no.onclick = () => sendEvent("approval_response", { requestId: p.requestId, decision: "reject" });
    row.append(ok, no);
    el.appendChild(row);
  } else {
    const body = document.createElement("div");
    body.className = "ev-body";
    body.textContent = eventText(ev);
    el.appendChild(body);
  }
  const events = $("events");
  events.appendChild(el);
  events.scrollTop = events.scrollHeight;
}

function setConn(s) {
  $("conn").textContent = s;
}

async function init() {
  const dev = await ensureIdentity();
  const params = new URLSearchParams(location.search);
  if (params.get("pair")) $("pair-code").value = params.get("pair");
  if (params.get("session")) {
    $("sessions-view").classList.remove("hidden");
    await refreshSessions();
    openDetail(params.get("session"), params.get("session"));
    return;
  }
  if (dev.deviceId) {
    $("sessions-view").classList.remove("hidden");
    await refreshSessions();
  } else {
    $("pair-view").classList.remove("hidden");
  }
}

$("pair-btn").onclick = async () => {
  $("pair-err").textContent = "";
  try {
    await pair($("pair-code").value.trim());
    $("pair-view").classList.add("hidden");
    $("sessions-view").classList.remove("hidden");
    await refreshSessions();
  } catch (e) {
    $("pair-err").textContent = e.message;
  }
};
$("create-form").onsubmit = (e) => { e.preventDefault(); createSession(e).catch((err) => alert(err.message)); };
$("refresh-btn").onclick = refreshSessions;
$("leave-btn").onclick = () => {
  if (ws) ws.close();
  currentSession = null; sessionKey = null;
  $("detail").classList.add("hidden");
};
$("stop-btn").onclick = async () => {
  if (!currentSession) return;
  await fetch("/api/sessions/" + currentSession + "/stop", { method: "POST" });
  if (ws) ws.close();
  await refreshSessions();
};
$("prompt-form").onsubmit = (e) => {
  e.preventDefault();
  const text = $("prompt-text").value.trim();
  if (!text) return;
  $("prompt-text").value = "";
  sendEvent("prompt", { text });
};

init().catch((e) => alert(e.message));
