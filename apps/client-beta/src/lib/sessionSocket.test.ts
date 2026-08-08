import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { webcrypto } from "node:crypto";
import { b64u, encryptEvent } from "./crypto";
import {
  dedupeBounded,
  dedupeEvent,
  MAX_SEEN_IDS,
  openSessionSocket,
  Outbox,
  reconnectBackoff,
  SeqTracker,
  SESSION_KEY_FAILURE_LIMIT,
  WS_STALE_TIMEOUT_MS,
  WS_WATCHDOG_INTERVAL_MS,
  type ConnTone,
  type SocketHandlers,
} from "./sessionSocket";
import type { Device, RiffpadEvent } from "./types";

function ev(id: string): RiffpadEvent {
  return { id, sessionId: "s1", timestamp: 1, type: "agent_message", payload: {} };
}

describe("dedupeEvent", () => {
  it("passes new events and skips replays", () => {
    const seen = new Set<string>();
    const a = ev("e1");
    const a2 = ev("e1");
    const b = ev("e2");
    expect(dedupeEvent(seen, a)).toBe(false);
    expect(dedupeEvent(seen, a2)).toBe(true); // same id replayed after reconnect
    expect(dedupeEvent(seen, b)).toBe(false);
    expect(seen.size).toBe(2);
  });

  it("keeps distinct events from the same replay burst", () => {
    const seen = new Set<string>();
    const burst = [ev("a"), ev("b"), ev("c"), ev("a")];
    const skipped = burst.filter((e) => dedupeEvent(seen, e));
    expect(skipped).toHaveLength(1);
  });
});

// dedupeBounded caps the dedupe set (#174): a session open for days must not
// grow it without limit, while recent replays must still be recognized.
describe("dedupeBounded", () => {
  it("bounds the set and still dedupes recent events", () => {
    const seen = new Set<string>();
    for (let i = 0; i < MAX_SEEN_IDS + 100; i++) {
      expect(dedupeBounded(seen, ev("e" + i))).toBe(false);
    }
    expect(seen.size).toBe(MAX_SEEN_IDS);
    // Recent ids are still remembered...
    expect(dedupeBounded(seen, ev("e" + (MAX_SEEN_IDS + 99)))).toBe(true);
    // ...and the oldest ones have been evicted (a stale replay of those would
    // render twice, which the bounded-set trade-off accepts).
    expect(seen.has("e0")).toBe(false);
  });
});

// SeqTracker powers the #173 gap detection: a jump in the per-session seq
// means a send buffer somewhere dropped events. Holes may still be filled by
// replay/live interleaving on the same connection, so only a hole that stays
// open is reported.
describe("SeqTracker", () => {
  it("stays quiet on a contiguous stream", () => {
    const tr = new SeqTracker();
    for (const seq of [1, 2, 3, 4]) tr.note(seq);
    expect(tr.pendingGap()).toBeNull();
  });

  it("ignores events without a seq (old daemon)", () => {
    const tr = new SeqTracker();
    tr.note(undefined);
    tr.note(0);
    tr.note(7);
    expect(tr.pendingGap()).toBeNull(); // first seq establishes the baseline
  });

  it("reports a hole that stays open", () => {
    const tr = new SeqTracker();
    tr.note(10);
    tr.note(13); // 11..12 lost
    expect(tr.pendingGap()).toEqual({ floor: 11, ceil: 12, missing: 2 });
    tr.clearGap();
    expect(tr.pendingGap()).toBeNull();
  });

  it("treats late in-stream arrivals as fills, not losses", () => {
    const tr = new SeqTracker();
    tr.note(10);
    tr.note(13); // apparent hole 11..12 (replay/live interleave)
    tr.note(11);
    tr.note(12);
    expect(tr.pendingGap()).toBeNull();
  });

  it("ignores replayed events at or below the high-water mark", () => {
    const tr = new SeqTracker();
    tr.note(10);
    tr.note(3); // replayed old event: no gap, lastSeq unchanged
    tr.note(11);
    expect(tr.pendingGap()).toBeNull();
  });
});

describe("Outbox", () => {
  it("keeps items until drained, preserving order", () => {
    const box = new Outbox<string>();
    box.push("a");
    box.push("b");
    expect(box.size).toBe(2);
    expect(box.drain()).toEqual(["a", "b"]);
    expect(box.size).toBe(0);
    expect(box.drain()).toEqual([]);
  });

  it("clear drops pending items", () => {
    const box = new Outbox<number>();
    box.push(1);
    box.clear();
    expect(box.size).toBe(0);
  });
});

// FakeWebSocket stands in for the browser WebSocket: it opens on a microtask
// and records close() calls so the watchdog tests can assert on them.
class FakeWebSocket {
  static instances: FakeWebSocket[] = [];
  static CONNECTING = 0;
  static OPEN = 1;
  static CLOSED = 3;

  readyState = FakeWebSocket.CONNECTING;
  onopen: (() => void) | null = null;
  onclose: (() => void) | null = null;
  onerror: (() => void) | null = null;
  onmessage: ((ev: { data: unknown }) => void) | null = null;
  closedByUs = false;
  sent: string[] = [];

  constructor(public url: string) {
    FakeWebSocket.instances.push(this);
    queueMicrotask(() => {
      if (this.readyState !== FakeWebSocket.CONNECTING) return;
      this.readyState = FakeWebSocket.OPEN;
      this.onopen?.();
    });
  }

  send(data: string) {
    this.sent.push(data);
  }

  close() {
    if (this.readyState === FakeWebSocket.CLOSED) return;
    this.readyState = FakeWebSocket.CLOSED;
    this.closedByUs = true;
    queueMicrotask(() => this.onclose?.());
  }

  emit(data: string) {
    this.onmessage?.({ data });
  }
}

describe("silence watchdog", () => {
  const dev: Device = { deviceId: "d1", serverPub: null, jwk: {} as JsonWebKey };
  const handlers = {
    onConn: () => {},
    onEvent: () => {},
    onError: () => {},
  };

  beforeEach(() => {
    FakeWebSocket.instances = [];
    vi.useFakeTimers();
    vi.stubGlobal("WebSocket", FakeWebSocket);
    if (!globalThis.crypto?.subtle) vi.stubGlobal("crypto", webcrypto);
  });

  afterEach(() => {
    vi.useRealTimers();
    vi.unstubAllGlobals();
  });

  it("closes a stale socket so the reconnect logic takes over", async () => {
    const sock = await openSessionSocket("s1", dev, handlers);
    const first = FakeWebSocket.instances[0];
    await vi.advanceTimersByTimeAsync(0); // let the open microtask run
    expect(first.readyState).toBe(FakeWebSocket.OPEN);

    // No message (data or keepalive) arrives for longer than the stale
    // timeout: the watchdog must close the half-open socket.
    await vi.advanceTimersByTimeAsync(WS_STALE_TIMEOUT_MS + WS_WATCHDOG_INTERVAL_MS);
    expect(first.closedByUs).toBe(true);

    // The existing reconnect logic then dials a fresh socket. connect()
    // awaits real async crypto ops that fake timers cannot settle, so flush
    // them on the real event loop before asserting.
    await vi.advanceTimersByTimeAsync(60_000);
    vi.useRealTimers();
    await new Promise((r) => setTimeout(r, 50));
    expect(FakeWebSocket.instances.length).toBeGreaterThan(1);
    sock.close();
  });

  it("stays connected while keepalives arrive", async () => {
    const sock = await openSessionSocket("s1", dev, handlers);
    const ws = FakeWebSocket.instances[0];
    await vi.advanceTimersByTimeAsync(0);

    // App-level {"kind":"ping"} keepalives every 20s for well over the stale
    // timeout: the socket must stay open.
    for (let i = 0; i < 6; i++) {
      ws.emit(`{"kind":"ping"}`);
      await vi.advanceTimersByTimeAsync(20_000);
    }
    expect(ws.closedByUs).toBe(false);
    sock.close();
  });
});

// Outbox send-status tests: an approval tapped while the socket is down must
// come back "queued" (never a fake success), resolve via onOutbox once the
// reconnect flushes it, and surface as "dropped" when close() discards it.
// These use real timers because the hello handshake awaits real crypto ops.
describe("outbox send status", () => {
  beforeEach(() => {
    FakeWebSocket.instances = [];
    vi.stubGlobal("WebSocket", FakeWebSocket);
    if (!globalThis.crypto?.subtle) vi.stubGlobal("crypto", webcrypto);
  });

  afterEach(() => {
    vi.unstubAllGlobals();
  });

  // makeDevice builds a paired device plus the server ephemeral key a fake
  // daemon would use, so tests can drive the hello handshake themselves.
  async function makeDevice() {
    const serverIdentity = await crypto.subtle.generateKey(
      { name: "ECDH", namedCurve: "P-256" },
      true,
      ["deriveBits"],
    );
    const serverPub = b64u(await crypto.subtle.exportKey("raw", serverIdentity.publicKey));
    const devIdentity = await crypto.subtle.generateKey(
      { name: "ECDH", namedCurve: "P-256" },
      true,
      ["deriveBits"],
    );
    const jwk = await crypto.subtle.exportKey("jwk", devIdentity.privateKey);
    const serverEph = await crypto.subtle.generateKey(
      { name: "ECDH", namedCurve: "P-256" },
      true,
      ["deriveBits"],
    );
    return { dev: { deviceId: "d1", serverPub, jwk } as Device, serverEph };
  }

  function collectOutbox() {
    const events: { id: string; status: string }[] = [];
    const handlers = {
      onConn: () => {},
      onEvent: () => {},
      onError: () => {},
      onOutbox: (id: string, status: "flushed" | "dropped") => events.push({ id, status }),
    };
    return { events, handlers };
  }

  async function settle() {
    await new Promise((r) => setTimeout(r, 20));
  }

  it("returns queued instead of a fake success while the socket is not ready", async () => {
    const { dev } = await makeDevice();
    const { events, handlers } = collectOutbox();
    const sock = await openSessionSocket("s1", dev, handlers);
    await settle(); // ws open microtask; no hello yet, so no session key

    const res = await sock.send("approval_response", { requestId: "r1", decision: "approve" });
    expect(res.status).toBe("queued");
    expect(res.id).toBeTruthy();
    expect(FakeWebSocket.instances[0].sent).toHaveLength(0);
    expect(events).toHaveLength(0);
    sock.close();
  });

  it("flushes queued events after reconnect hello and reports them", async () => {
    const { dev, serverEph } = await makeDevice();
    const { events, handlers } = collectOutbox();
    const sock = await openSessionSocket("s1", dev, handlers);
    await settle();
    const ws = FakeWebSocket.instances[0];

    const res = await sock.send("approval_response", { requestId: "r1", decision: "approve" });
    expect(res.status).toBe("queued");

    // The daemon answers with hello; the client derives the session key and
    // flushes the outbox through the now-open socket.
    const serverEphPub = b64u(await crypto.subtle.exportKey("raw", serverEph.publicKey));
    ws.emit(JSON.stringify({ kind: "hello", serverEphPub }));
    await vi.waitFor(() => expect(ws.sent).toHaveLength(1));
    expect(events).toEqual([{ id: res.id, status: "flushed" }]);
    const written = JSON.parse(ws.sent[0]);
    expect(written.kind).toBe("event");
    sock.close();
  });

  it("reports queued events as dropped when close() discards the outbox", async () => {
    const { dev } = await makeDevice();
    const { events, handlers } = collectOutbox();
    const sock = await openSessionSocket("s1", dev, handlers);
    await settle();

    const res = await sock.send("approval_response", { requestId: "r1", decision: "approve" });
    expect(res.status).toBe("queued");

    sock.close(); // user kills the tab while offline
    expect(events).toEqual([{ id: res.id, status: "dropped" }]);
  });
});

// FailingWebSocket simulates a WS handshake that always fails: it never opens
// and closes on a microtask, like a daemon that is down or a rejected upgrade.
class FailingWebSocket {
  static instances: FailingWebSocket[] = [];
  static CONNECTING = 0;
  static OPEN = 1;
  static CLOSED = 3;

  readyState = FailingWebSocket.CONNECTING;
  onopen: (() => void) | null = null;
  onclose: (() => void) | null = null;
  onerror: (() => void) | null = null;
  onmessage: ((ev: { data: unknown }) => void) | null = null;

  constructor(public url: string) {
    FailingWebSocket.instances.push(this);
    queueMicrotask(() => {
      if (this.readyState !== FailingWebSocket.CONNECTING) return;
      this.readyState = FailingWebSocket.CLOSED;
      this.onclose?.();
    });
  }

  send() {}
  close() {
    if (this.readyState === FailingWebSocket.CLOSED) return;
    this.readyState = FailingWebSocket.CLOSED;
    queueMicrotask(() => this.onclose?.());
  }
}

describe("offline vs revoked", () => {
  const dev: Device = { deviceId: "d1", serverPub: null, jwk: {} as JsonWebKey };

  beforeEach(() => {
    FailingWebSocket.instances = [];
    vi.stubGlobal("WebSocket", FailingWebSocket);
    if (!globalThis.crypto?.subtle) vi.stubGlobal("crypto", webcrypto);
    // Shrink the reconnect backoff so the real-timer tests stay fast.
    reconnectBackoff.baseMs = 2;
    reconnectBackoff.maxMs = 50;
  });

  afterEach(() => {
    reconnectBackoff.baseMs = 1000;
    reconnectBackoff.maxMs = 30000;
    vi.unstubAllGlobals();
  });

  function wait(ms: number) {
    return new Promise((r) => setTimeout(r, ms));
  }

  function makeHandlers(extra: Partial<SocketHandlers> = {}) {
    return {
      onConn: () => {},
      onEvent: () => {},
      onError: () => {},
      ...extra,
    };
  }

  it("treats a 401 probe as revoked and stops reconnecting", async () => {
    vi.stubGlobal(
      "fetch",
      vi.fn(async () => new Response("unauthorized", { status: 401 })),
    );
    const onFatal = vi.fn();
    const onOffline = vi.fn();
    const sock = await openSessionSocket("s1", dev, makeHandlers({ onFatal, onOffline }));

    await wait(300); // 4 failed attempts -> probe fires
    expect(onFatal).toHaveBeenCalledTimes(1);
    expect(onOffline).not.toHaveBeenCalled();

    // No reconnect is scheduled after a confirmed revocation.
    const n = FailingWebSocket.instances.length;
    await wait(300);
    expect(FailingWebSocket.instances.length).toBe(n);
    sock.close();
  });

  it("treats a missing device in a 200 probe as revoked", async () => {
    vi.stubGlobal(
      "fetch",
      vi.fn(async () =>
        new Response(JSON.stringify({ devices: [{ id: "other" }] }), { status: 200 }),
      ),
    );
    const onFatal = vi.fn();
    const sock = await openSessionSocket("s1", dev, makeHandlers({ onFatal }));

    await wait(300);
    expect(onFatal).toHaveBeenCalledTimes(1);
    sock.close();
  });

  it("keeps reconnecting with an offline notice when the device still exists", async () => {
    vi.stubGlobal(
      "fetch",
      vi.fn(async () =>
        new Response(JSON.stringify({ devices: [{ id: "d1" }] }), { status: 200 }),
      ),
    );
    const onFatal = vi.fn();
    const onOffline = vi.fn();
    const sock = await openSessionSocket("s1", dev, makeHandlers({ onFatal, onOffline }));

    await wait(300);
    expect(onFatal).not.toHaveBeenCalled();
    expect(onOffline).toHaveBeenCalled();
    expect(typeof onOffline.mock.calls[0][0]).toBe("string");

    // Reconnecting continues instead of declaring the device revoked.
    const n = FailingWebSocket.instances.length;
    await wait(300);
    expect(FailingWebSocket.instances.length).toBeGreaterThan(n);
    sock.close();
  });

  it("keeps reconnecting when the probe itself fails (daemon down)", async () => {
    vi.stubGlobal("fetch", vi.fn(async () => Promise.reject(new Error("network down"))));
    const onFatal = vi.fn();
    const onOffline = vi.fn();
    const sock = await openSessionSocket("s1", dev, makeHandlers({ onFatal, onOffline }));

    await wait(300);
    expect(onFatal).not.toHaveBeenCalled();
    expect(onOffline).toHaveBeenCalled();

    const n = FailingWebSocket.instances.length;
    await wait(300);
    expect(FailingWebSocket.instances.length).toBeGreaterThan(n);
    sock.close();
  });

  it("retry() reconnects immediately without waiting for the backoff", async () => {
    reconnectBackoff.baseMs = 10_000; // make the scheduled backoff far away
    vi.stubGlobal("fetch", vi.fn(async () => Promise.reject(new Error("network down"))));
    const sock = await openSessionSocket("s1", dev, makeHandlers({ onOffline: () => {} }));

    await wait(300); // probe fired, reconnect scheduled far in the future
    const n = FailingWebSocket.instances.length;
    sock.retry();
    await wait(100);
    expect(FailingWebSocket.instances.length).toBeGreaterThan(n);
    sock.close();
  });
});

// Session key mismatch (#172): if the daemon's identity key is regenerated
// (e.g. the user deleted keys.json to "reset"), the device record still
// exists, so nothing server-side rejects the connection — but the stored
// serverPub no longer matches and every event fails to decrypt. After enough
// consecutive failures the socket must escalate to the fatal re-pair prompt
// (the same UI path as a revoked device) instead of spinning silently.
describe("session key mismatch", () => {
  beforeEach(() => {
    FakeWebSocket.instances = [];
    vi.stubGlobal("WebSocket", FakeWebSocket);
    if (!globalThis.crypto?.subtle) vi.stubGlobal("crypto", webcrypto);
  });

  afterEach(() => {
    vi.unstubAllGlobals();
  });

  async function genEcdh() {
    return crypto.subtle.generateKey({ name: "ECDH", namedCurve: "P-256" }, true, ["deriveBits"]);
  }

  async function settle(ms = 30) {
    await new Promise((r) => setTimeout(r, ms));
  }

  it("escalates to a fatal re-pair prompt after repeated decrypt failures", async () => {
    // The device is paired against the OLD server identity.
    const oldServerIdentity = await genEcdh();
    const serverPub = b64u(await crypto.subtle.exportKey("raw", oldServerIdentity.publicKey));
    const devIdentity = await genEcdh();
    const jwk = await crypto.subtle.exportKey("jwk", devIdentity.privateKey);
    const dev = { deviceId: "d1", serverPub, jwk } as Device;

    const onFatal = vi.fn();
    const onError = vi.fn();
    const sock = await openSessionSocket("s1", dev, {
      onConn: () => {},
      onEvent: () => {},
      onError,
      onFatal,
    });
    await settle();
    const ws = FakeWebSocket.instances[0];

    // The daemon (now running a NEW identity) answers hello; the client
    // derives a session key that does not match the daemon's.
    const serverEph = await genEcdh();
    ws.emit(JSON.stringify({ kind: "hello", serverEphPub: b64u(await crypto.subtle.exportKey("raw", serverEph.publicKey)) }));
    await settle(50);

    // Events arrive encrypted under the daemon's (correct) key; to the client
    // they are undecryptable. A random AES key simulates that mismatch.
    const daemonKey = await crypto.subtle.generateKey({ name: "AES-GCM", length: 256 }, true, ["encrypt", "decrypt"]);
    for (let i = 0; i < SESSION_KEY_FAILURE_LIMIT + 2; i++) {
      const boxed = await encryptEvent(daemonKey, "s1", ev("bad" + i));
      ws.emit(JSON.stringify({ v: 1, kind: "event", sessionId: "s1", ...boxed }));
      await settle(10);
    }

    expect(onFatal).toHaveBeenCalledTimes(1); // once, not per failure
    expect(typeof onFatal.mock.calls[0][0]).toBe("string");
    expect(onFatal.mock.calls[0][0]).not.toBe("");
    // Failures below the limit are ordinary errors, not fatal.
    expect(onError).toHaveBeenCalledTimes(SESSION_KEY_FAILURE_LIMIT - 1);
    sock.close();
  });
});

// Connection tone (#174): the status light consumes a structured tone from
// the socket instead of regex-matching the (translated) label, so English
// UI no longer renders every state green.
describe("connection tone", () => {
  beforeEach(() => {
    FakeWebSocket.instances = [];
    vi.stubGlobal("WebSocket", FakeWebSocket);
    if (!globalThis.crypto?.subtle) vi.stubGlobal("crypto", webcrypto);
  });

  afterEach(() => {
    vi.unstubAllGlobals();
  });

  async function genEcdh() {
    return crypto.subtle.generateKey({ name: "ECDH", namedCurve: "P-256" }, true, ["deriveBits"]);
  }

  it("reports pending while connecting, good after the hello handshake, bad on close", async () => {
    const serverIdentity = await genEcdh();
    const serverPub = b64u(await crypto.subtle.exportKey("raw", serverIdentity.publicKey));
    const devIdentity = await genEcdh();
    const jwk = await crypto.subtle.exportKey("jwk", devIdentity.privateKey);
    const dev = { deviceId: "d1", serverPub, jwk } as Device;

    const tones: ConnTone[] = [];
    const sock = await openSessionSocket("s1", dev, {
      onConn: (_label, tone) => tones.push(tone),
      onEvent: () => {},
      onError: () => {},
    });
    await new Promise((r) => setTimeout(r, 20)); // ws open microtask
    expect(tones.at(-1)).toBe("pending");

    const serverEph = await genEcdh();
    const ws = FakeWebSocket.instances[0];
    ws.emit(JSON.stringify({ kind: "hello", serverEphPub: b64u(await crypto.subtle.exportKey("raw", serverEph.publicKey)) }));
    await vi.waitFor(() => expect(tones.at(-1)).toBe("good"));

    sock.close();
    await new Promise((r) => setTimeout(r, 20));
    expect(tones.at(-1)).toBe("bad");
  });
});
