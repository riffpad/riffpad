import {
  b64u,
  deriveSessionKey,
  decryptEvent,
  encryptEvent,
  genPair,
} from "./crypto";
import { getDeviceSecret } from "./device";
import { isRelay, relayStore } from "./store";
import type { Device, RiffpadEvent } from "./types";

export interface SessionSocket {
  close(): void;
  send(type: string, payload: Record<string, unknown>): Promise<boolean>;
}

export interface SocketHandlers {
  onConn(label: string): void;
  onEvent(ev: RiffpadEvent): void;
  onError(message: string): void;
}

// dedupeEvent returns true when ev has already been seen (replay after a
// reconnect). Exported for tests.
export function dedupeEvent(seen: Set<string>, ev: RiffpadEvent): boolean {
  if (seen.has(ev.id)) return true;
  seen.add(ev.id);
  return false;
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
  let everConnected = false;
  const seenIds = new Set<string>();

  // Messages must be handled in order: the hello handshake derives the
  // session key asynchronously, and replayed events arriving before the key is
  // ready would otherwise be dropped by concurrent async handlers.
  const queue: string[] = [];
  let draining = false;

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
          reconnectAttempt = 0;
          everConnected = true;
          handlers.onConn("已连接（加密）");
          continue;
        }
        if (sessionKey) {
          const pt = await decryptEvent(sessionKey, sid, data.nonce, data.ciphertext);
          const ev = pt as RiffpadEvent;
          if (dedupeEvent(seenIds, ev)) continue; // replay dedup across reconnects
          handlers.onEvent(ev);
        }
      } catch (e) {
        handlers.onError(e instanceof Error ? e.message : String(e));
      }
    }
    draining = false;
  }

  // The ephemeral key is captured per connection attempt; TS needs a stable
  // binding for the async drain above, so keep it at module-of-function scope.
  let eph: CryptoKeyPair;

  async function connect() {
    eph = await genPair();
    const ephPub = b64u(await crypto.subtle.exportKey("raw", eph.publicKey));
    const proto = location.protocol === "https:" ? "wss" : "ws";
    const tok = isRelay ? relayStore.get()?.token || "" : "";
    const url =
      `${proto}://${location.host}/ws?device=${dev.deviceId}&session=${sid}&eph=${ephPub}` +
      (tok ? "&token=" + encodeURIComponent(tok) : "");
    ws = new WebSocket(url);

    ws.onopen = () => handlers.onConn(reconnectAttempt > 0 ? "重连中…" : "连接中…");
    ws.onerror = () => {
      // onclose follows; keep state updates there.
    };
    ws.onclose = () => {
      sessionKey = null;
      if (closed) {
        handlers.onConn("已断开");
        return;
      }
      if (!everConnected && reconnectAttempt >= 3) {
        handlers.onConn("连接失败：设备可能已被撤销，请刷新页面并重新配对");
        return;
      }
      const delay = everConnected
        ? Math.min(1000 * 2 ** reconnectAttempt, 30000)
        : 1000 * 2 ** reconnectAttempt;
      reconnectAttempt++;
      handlers.onConn(`连接断开，${Math.round(delay / 1000)}s 后自动重连…`);
      window.setTimeout(() => {
        if (!closed) void connect();
      }, delay);
    };
    ws.onmessage = (msg) => {
      queue.push(String(msg.data));
      if (!draining) {
        draining = true;
        void drainQueue();
      }
    };
  }

  await connect();
  return {
    close() {
      closed = true;
      ws?.close();
    },
    async send(type: string, payload: Record<string, unknown>): Promise<boolean> {
      if (!ws || ws.readyState !== WebSocket.OPEN || !sessionKey) return false;
      const ev = {
        id: String(Date.now()) + Math.random().toString(16).slice(2),
        sessionId: sid,
        timestamp: Date.now(),
        type,
        payload,
      };
      const boxed = await encryptEvent(sessionKey, sid, ev);
      ws.send(JSON.stringify({ v: 1, kind: "event", sessionId: sid, ...boxed }));
      return true;
    },
  };
}
