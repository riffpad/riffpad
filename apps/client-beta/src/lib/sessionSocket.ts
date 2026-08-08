import {
  b64u,
  deriveSessionKey,
  decryptEvent,
  encryptEvent,
  genPair,
} from "./crypto";
import { getDeviceSecret } from "./device";
import { getT } from "./i18n";
import { api, isRelay, localTokenStore, relayStore } from "./store";
import type { Device, RiffpadEvent } from "./types";

// SendStatus tells the UI apart what used to be a single "true": the event
// was written to the socket ("sent"), parked in the in-memory outbox until
// the next reconnect ("queued"), or could not be handed to a socket at all
// ("failed"). Approvals must never show "已批准" for anything but "sent"
// (or a later successful flush).
export type SendStatus = "sent" | "queued" | "failed";

export interface SendResult {
  status: SendStatus;
  // id of the queued/sent event; correlates with onOutbox callbacks.
  id: string;
}

export interface SessionSocket {
  close(): void;
  send(type: string, payload: Record<string, unknown>): Promise<SendResult>;
  requestHistory(before: string, limit: number): boolean;
  retry(): void;
}

// ConnTone grades the connection status for the status light, so the UI
// never has to regex-match translated label text (#174: English labels used
// to fall through every Chinese pattern and always render green).
export type ConnTone = "good" | "pending" | "bad";

export interface SocketHandlers {
  onConn(label: string, tone: ConnTone): void;
  onEvent(ev: RiffpadEvent): void;
  onError(message: string): void;
  onFatal?(message: string): void;
  // onOffline fires with a message when the device is confirmed online but
  // unreachable (daemon down / weak network), and with null once the
  // connection recovers.
  onOffline?(message: string | null): void;
  onHistory?(events: RiffpadEvent[]): void;
  // onOutbox reports the fate of a queued event: "flushed" once it was
  // written after a reconnect, "dropped" when close() discards it.
  onOutbox?(id: string, status: "flushed" | "dropped"): void;
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

// MAX_SEEN_IDS bounds the dedupe set: a session open for days would otherwise
// grow it without limit (#174). JS Sets iterate in insertion order, so the
// oldest ids are evicted first; reconnect replays only cover recent history,
// far below this bound, so dedupe accuracy is unaffected in practice.
export const MAX_SEEN_IDS = 5000;

// dedupeBounded is dedupeEvent plus the size bound. Exported for tests.
export function dedupeBounded(seen: Set<string>, ev: RiffpadEvent): boolean {
  if (dedupeEvent(seen, ev)) return true;
  if (seen.size > MAX_SEEN_IDS) {
    const it = seen.values();
    for (let i = seen.size - MAX_SEEN_IDS; i > 0; i--) {
      seen.delete(it.next().value as string);
    }
  }
  return false;
}

// SeqTracker detects holes in the per-session event sequence (#173). Send
// buffers that overflow drop messages, so a client that only counts what
// arrives can never notice a missing approval card. Events carry an
// increasing per-session seq; a jump means something was lost in between.
//
// Replay and live events can interleave on one connection (the daemon queues
// the replay burst while the pump keeps broadcasting), so a hole may be
// filled by messages that simply arrive later in the same stream: note()
// only records the gap, and pendingGap() reports it once the drain finished
// with the hole still open. Exported for tests.
export class SeqTracker {
  private lastSeq = 0;
  private gapFloor = 0;
  private gapCeil = 0;
  private gapMissing = 0;

  note(seq: number | undefined): void {
    if (!seq || seq <= 0) return; // no seq (old daemon): skip detection
    if (seq > this.lastSeq) {
      if (this.lastSeq > 0 && seq > this.lastSeq + 1) {
        if (this.gapMissing === 0) this.gapFloor = this.lastSeq + 1;
        this.gapCeil = seq - 1;
        this.gapMissing += seq - this.lastSeq - 1;
      }
      this.lastSeq = seq;
    } else if (this.gapMissing > 0 && seq >= this.gapFloor && seq <= this.gapCeil) {
      this.gapMissing--;
    }
  }

  // pendingGap returns the still-unfilled hole, if any.
  pendingGap(): { floor: number; ceil: number; missing: number } | null {
    if (this.gapMissing <= 0) return null;
    return { floor: this.gapFloor, ceil: this.gapCeil, missing: this.gapMissing };
  }

  clearGap(): void {
    this.gapMissing = 0;
    this.gapFloor = 0;
    this.gapCeil = 0;
  }
}

// Watchdog parameters: if nothing (data or keepalive) arrives for
// WS_STALE_TIMEOUT_MS, the socket is closed so the reconnect logic kicks in.
// Exported for tests.
export const WS_STALE_TIMEOUT_MS = 75_000;
export const WS_WATCHDOG_INTERVAL_MS = 15_000;

// Reconnect backoff parameters, mutable so tests can shrink the delays.
export const reconnectBackoff = { baseMs: 1000, maxMs: 30000 };

// SESSION_KEY_FAILURE_LIMIT is how many consecutive events may fail to
// decrypt before the pairing is declared broken. This catches the case where
// the daemon's identity key was regenerated (e.g. the user deleted keys.json
// to "reset"): the device record still exists, so nothing server-side rejects
// the connection, but the stored serverPub no longer matches and every event
// is undecryptable. Exported for tests.
export const SESSION_KEY_FAILURE_LIMIT = 3;

// probeDeviceRevoked decides whether a device has truly been revoked.
// Browsers cannot read the HTTP status of a failed WS handshake, so the
// devices endpoint is probed over HTTP instead. It works in both modes:
// relay (/api/devices answers even while the daemon is offline) and local
// (a stopped daemon makes the fetch itself fail). Exported for tests.
export async function probeDeviceRevoked(dev: Device): Promise<boolean> {
  try {
    const res = await api("/api/devices");
    if (res.status === 401 || res.status === 403) return true;
    if (!res.ok) return false; // 5xx etc: server trouble, keep reconnecting
    const data = await res.json();
    const list = (data.devices || []) as { id?: string }[];
    return !list.some((d) => d.id === dev.deviceId);
  } catch {
    // Network error: daemon stopped or weak network — not a revocation.
    return false;
  }
}

export async function openSessionSocket(
  sid: string,
  dev: Device,
  handlers: SocketHandlers,
): Promise<SessionSocket> {
  let closed = false;
  let ws: WebSocket | null = null;
  let sessionKey: CryptoKey | null = null;
  let reconnectAttempt = 0;
  let reconnectTimer: number | null = null;
  let everConnected = false;
  // Consecutive decrypt failures: past SESSION_KEY_FAILURE_LIMIT the stored
  // pairing no longer matches the daemon's identity key, so escalate to a
  // fatal re-pair prompt (same UI path as a revoked device).
  let keyFailures = 0;
  let keyMismatchFatal = false;
  // Watchdog state: browsers cannot observe protocol-level ping/pong, so any
  // incoming message (data or app-level {"kind":"ping"}) counts as activity.
  // A half-open connection (lock screen, network switch) stays silent forever;
  // closing the socket lets the existing reconnect logic take over.
  let lastActivity = Date.now();
  const seenIds = new Set<string>();
  const seqTracker = new SeqTracker();
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
          keyFailures = 0;
          await flushOutbox();
          reconnectAttempt = 0;
          everConnected = true;
          handlers.onOffline?.(null);
          handlers.onConn(t("connected_encrypted"), "good");
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
          let ev: RiffpadEvent;
          try {
            ev = (await decryptEvent(sessionKey, sid, data.nonce, data.ciphertext)) as RiffpadEvent;
          } catch (e) {
            // Undecryptable events mean the derived session key is wrong —
            // after enough of them in a row the pairing itself is broken
            // (daemon identity regenerated, e.g. deleted keys.json), so
            // surface the fatal re-pair prompt instead of silent errors.
            keyFailures++;
            if (keyFailures >= SESSION_KEY_FAILURE_LIMIT) {
              if (!keyMismatchFatal) {
                keyMismatchFatal = true;
                handlers.onFatal?.(t("session_key_mismatch"));
                handlers.onConn(t("session_key_mismatch"), "bad");
              }
            } else {
              handlers.onError(e instanceof Error ? e.message : String(e));
            }
            continue;
          }
          keyFailures = 0;
          if (historyState) {
            if (historyState.remaining > 0) historyState.remaining--;
            if (!dedupeBounded(seenIds, ev)) historyState.events.push(ev);
            if (historyState.remaining <= 0) finalizeHistory();
            continue;
          }
          if (dedupeBounded(seenIds, ev)) continue; // replay dedup across reconnects
          seqTracker.note(ev.seq);
          handlers.onEvent(ev);
        }
      } catch (e) {
        handlers.onError(e instanceof Error ? e.message : String(e));
      }
    }
    draining = false;
    // A still-open seq hole after the whole burst was drained means events
    // were really lost (not just reordered in transit): reconnect so the
    // daemon replays recent history and refills it (#173).
    const gap = seqTracker.pendingGap();
    if (gap) {
      seqTracker.clearGap();
      console.warn(
        `[riffpad] session ${sid}: ${gap.missing} event(s) lost ` +
          `(seq ${gap.floor}..${gap.ceil}); reconnecting to replay history`,
      );
      if (ws && ws.readyState === WebSocket.OPEN) ws.close();
    }
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
        handlers.onOutbox?.(ev.id, "flushed");
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

  function scheduleReconnect() {
    const delay = Math.min(
      reconnectBackoff.baseMs * 2 ** reconnectAttempt,
      reconnectBackoff.maxMs,
    );
    reconnectAttempt++;
    handlers.onConn(t("reconnect_in", { s: Math.round(delay / 1000) }), "pending");
    reconnectTimer = window.setTimeout(() => {
      reconnectTimer = null;
      if (!closed) void connect();
    }, delay);
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
      handlers.onConn(reconnectAttempt > 0 ? t("reconnecting") : t("connecting"), "pending");
    };
    ws.onerror = () => {
      // onclose follows; keep state updates there.
    };
    ws.onclose = () => {
      sessionKey = null;
      historyState = null;
      if (closed) {
        handlers.onConn(t("disconnected"), "bad");
        return;
      }
      if (!everConnected && reconnectAttempt >= 3) {
        // The WS handshake failure carries no HTTP status in the browser, so
        // probe the devices endpoint: only a confirmed revocation (401/403 or
        // device missing) is fatal; anything else means the daemon is simply
        // offline and we keep reconnecting.
        void (async () => {
          const revoked = await probeDeviceRevoked(dev);
          if (closed) return;
          if (revoked) {
            handlers.onFatal?.(t("device_revoked"));
            handlers.onConn(t("device_revoked"), "bad");
            return;
          }
          handlers.onOffline?.(t("daemon_offline"));
          scheduleReconnect();
        })();
        return;
      }
      scheduleReconnect();
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
      if (reconnectTimer !== null) {
        window.clearTimeout(reconnectTimer);
        reconnectTimer = null;
      }
      // Queued events die with the socket (e.g. the tab is closed): tell the
      // UI each one was dropped so it can show 未送达 instead of 已批准.
      for (const ev of outbox.drain()) handlers.onOutbox?.(ev.id, "dropped");
      ws?.close();
    },
    retry() {
      if (closed) return;
      if (reconnectTimer !== null) {
        window.clearTimeout(reconnectTimer);
        reconnectTimer = null;
      }
      reconnectAttempt = 0;
      void connect();
    },
    async send(type: string, payload: Record<string, unknown>): Promise<SendResult> {
      const ev = {
        id: String(Date.now()) + Math.random().toString(16).slice(2),
        sessionId: sid,
        timestamp: Date.now(),
        type,
        payload,
      };
      if (!ws || ws.readyState !== WebSocket.OPEN || !sessionKey) {
        outbox.push(ev);
        return { status: "queued", id: ev.id };
      }
      try {
        await writeEvent(ev);
        return { status: "sent", id: ev.id };
      } catch {
        outbox.push(ev);
        return { status: "queued", id: ev.id };
      }
    },
    requestHistory(before: string, limit: number): boolean {
      if (!ws || ws.readyState !== WebSocket.OPEN) return false;
      ws.send(JSON.stringify({ kind: "history_query", before, limit }));
      return true;
    },
  };
}
