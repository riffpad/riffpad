import {
  b64u,
  deriveSessionKey,
  decryptEvent,
  encryptEvent,
  genPair,
} from "./crypto";
import { getDeviceSecret } from "./device";
import { getT } from "./i18n";
import { isRelay, localTokenStore, relayStore } from "./store";
import type { Device, RiffpadEvent } from "./types";

export interface SessionSocket {
  close(): void;
  send(type: string, payload: Record<string, unknown>): Promise<boolean>;
  requestHistory(before: string, limit: number): boolean;
}

export interface SocketHandlers {
  onConn(label: string): void;
  onEvent(ev: RiffpadEvent): void;
  onError(message: string): void;
  onFatal?(message: string): void;
  onHistory?(events: RiffpadEvent[]): void;
}

// Outbox keeps outgoing events that could not be written while the socket was
// down. Items are flushed in order after the next successful hello, so nothing
// the user typed is silently lost. Events that were already written to the
// socket are never queued, which avoids duplicate execution on replay.
export class Outbox<T> {
  private items: T[] = [];

  push(item: T) {
    this.items.push(item);
  }

  drain(): T[] {
    const out = this.items;
    this.items = [];
    return out;
  }

  clear() {
    this.items = [];
  }

  get size() {
    return this.items.length;
  }
}

// dedupeEvent returns true when ev has already been seen (replay after a
// reconnect). Exported for tests.
export function dedupeEvent(seen: Set<string>, ev: RiffpadEvent): boolean {
  if (seen.has(ev.id)) return true;
  seen.add(ev.id);
  return false;
}

// Watchdog parameters: if nothing (data or keepalive) arrives for
// WS_STALE_TIMEOUT_MS, the socket is closed so the reconnect logic kicks in.
// Exported for tests.
export const WS_STALE_TIMEOUT_MS = 75_000;
export const WS_WATCHDOG_INTERVAL_MS = 15_000;

export async function openSessionSocket(
  sid: string,
  dev: Device,
  handlers: SocketHandlers,
): Promise<SessionSocket> {
  let closed = false;
  let ws: WebSocket | null = null;
  let sessionKey: CryptoKey | null = null;
  let reconnectAttempt = 0;
  let everConnected = false;
  // Watchdog state: browsers cannot observe protocol-level ping/pong, so any
  // incoming message (data or app-level {"kind":"ping"}) counts as activity.
  // A half-open connection (lock screen, network switch) stays silent forever;
  // closing the socket lets the existing reconnect logic take over.
  let lastActivity = Date.now();
  const seenIds = new Set<string>();
  let historyState: { remaining: number; events: RiffpadEvent[] } | null = null;
  const outbox = new Outbox<{
    id: string;
    sessionId: string;
    timestamp: number;
    type: string;
    payload: Record<string, unknown>;
  }>();

  // Messages must be handled in order: the hello handshake derives the
  // session key asynchronously, and replayed events arriving before the key is
  // ready would otherwise be dropped by concurrent async handlers.
  const queue: string[] = [];
  let draining = false;
  const t = getT();

  async function drainQueue() {
    while (queue.length > 0) {
      const raw = queue.shift()!;
      try {
        const data = JSON.parse(raw);
        if (data.kind === "hello") {
          const dsec = await getDeviceSecret(dev);
          sessionKey = await deriveSessionKey(
            eph.privateKey,
            data.serverEphPub,
            dsec,
            sid,
          );
          await flushOutbox();
          reconnectAttempt = 0;
          everConnected = true;
          handlers.onConn(t("connected_encrypted"));
          continue;
        }
        if (data.kind === "ping") {
          continue; // app-level keepalive: activity already recorded in onmessage
        }
        if (data.kind === "history") {
          historyState = { remaining: Number(data.count) || 0, events: [] };
          continue;
        }
        if (data.kind === "history_done") {
          finalizeHistory();
          continue;
        }
        if (sessionKey) {
          const pt = await decryptEvent(sessionKey, sid, data.nonce, data.ciphertext);
          const ev = pt as RiffpadEvent;
          if (historyState) {
            if (historyState.remaining > 0) historyState.remaining--;
            if (!dedupeEvent(seenIds, ev)) historyState.events.push(ev);
            if (historyState.remaining <= 0) finalizeHistory();
            continue;
          }
          if (dedupeEvent(seenIds, ev)) continue; // replay dedup across reconnects
          handlers.onEvent(ev);
        }
      } catch (e) {
        handlers.onError(e instanceof Error ? e.message : String(e));
      }
    }
    draining = false;
  }

  function finalizeHistory() {
    if (!historyState) return;
    const batch = historyState.events;
    historyState = null;
    handlers.onHistory?.(batch);
  }

  async function flushOutbox() {
    const pending = outbox.drain();
    for (const ev of pending) {
      if (!ws || ws.readyState !== WebSocket.OPEN || !sessionKey) {
        outbox.push(ev);
        break;
      }
      try {
        await writeEvent(ev);
      } catch {
        outbox.push(ev);
        break;
      }
    }
  }

  // The ephemeral key is captured per connection attempt; TS needs a stable
  // binding for the async drain above, so keep it at module-of-function scope.
  let eph: CryptoKeyPair;

  async function writeEvent(ev: {
    id: string;
    sessionId: string;
    timestamp: number;
    type: string;
    payload: Record<string, unknown>;
  }) {
    if (!ws || ws.readyState !== WebSocket.OPEN || !sessionKey) {
      throw new Error("socket not open");
    }
    const boxed = await encryptEvent(sessionKey, sid, ev);
    ws.send(JSON.stringify({ v: 1, kind: "event", sessionId: sid, ...boxed }));
  }

  async function connect() {
    eph = await genPair();
    const ephPub = b64u(await crypto.subtle.exportKey("raw", eph.publicKey));
    const proto = location.protocol === "https:" ? "wss" : "ws";
    const tok = isRelay ? relayStore.get()?.token || "" : localTokenStore.get();
    const url =
      `${proto}://${location.host}/ws?device=${dev.deviceId}&session=${sid}&eph=${ephPub}` +
      (tok ? "&token=" + encodeURIComponent(tok) : "");
    ws = new WebSocket(url);
    lastActivity = Date.now();

    ws.onopen = () => {
      lastActivity = Date.now();
      handlers.onConn(reconnectAttempt > 0 ? t("reconnecting") : t("connecting"));
    };
    ws.onerror = () => {
      // onclose follows; keep state updates there.
    };
    ws.onclose = () => {
      sessionKey = null;
      historyState = null;
      if (closed) {
        handlers.onConn(t("disconnected"));
        return;
      }
      if (!everConnected && reconnectAttempt >= 3) {
        handlers.onFatal?.(t("device_revoked"));
        handlers.onConn(t("device_revoked"));
        return;
      }
      const delay = everConnected
        ? Math.min(1000 * 2 ** reconnectAttempt, 30000)
        : 1000 * 2 ** reconnectAttempt;
      reconnectAttempt++;
      handlers.onConn(t("reconnect_in", { s: Math.round(delay / 1000) }));
      window.setTimeout(() => {
        if (!closed) void connect();
      }, delay);
    };
    ws.onmessage = (msg) => {
      lastActivity = Date.now();
      queue.push(String(msg.data));
      if (!draining) {
        draining = true;
        void drainQueue();
      }
    };
  }

  await connect();
  const watchdog = window.setInterval(() => {
    if (closed || !ws || ws.readyState !== WebSocket.OPEN) return;
    if (Date.now() - lastActivity > WS_STALE_TIMEOUT_MS) {
      ws.close(); // stale connection: let onclose drive the reconnect
    }
  }, WS_WATCHDOG_INTERVAL_MS);
  return {
    close() {
      closed = true;
      window.clearInterval(watchdog);
      outbox.clear();
      ws?.close();
    },
    async send(type: string, payload: Record<string, unknown>): Promise<boolean> {
      const ev = {
        id: String(Date.now()) + Math.random().toString(16).slice(2),
        sessionId: sid,
        timestamp: Date.now(),
        type,
        payload,
      };
      if (!ws || ws.readyState !== WebSocket.OPEN || !sessionKey) {
        outbox.push(ev);
        return true;
      }
      try {
        await writeEvent(ev);
      } catch {
        outbox.push(ev);
      }
      return true;
    },
    requestHistory(before: string, limit: number): boolean {
      if (!ws || ws.readyState !== WebSocket.OPEN) return false;
      ws.send(JSON.stringify({ kind: "history_query", before, limit }));
      return true;
    },
  };
}
