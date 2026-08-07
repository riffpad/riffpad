import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { webcrypto } from "node:crypto";
import {
  dedupeEvent,
  openSessionSocket,
  Outbox,
  WS_STALE_TIMEOUT_MS,
  WS_WATCHDOG_INTERVAL_MS,
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
