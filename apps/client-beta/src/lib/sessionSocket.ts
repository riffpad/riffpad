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

export async function openSessionSocket(
  sid: string,
  dev: Device,
  handlers: SocketHandlers,
): Promise<SessionSocket> {
  const eph = await genPair();
  const ephPub = b64u(await crypto.subtle.exportKey("raw", eph.publicKey));
  const proto = location.protocol === "https:" ? "wss" : "ws";
  const tok = isRelay ? relayStore.get()?.token || "" : "";
  const ws = new WebSocket(
    `${proto}://${location.host}/ws?device=${dev.deviceId}&session=${sid}&eph=${ephPub}${tok ? "&token=" + encodeURIComponent(tok) : ""}`,
  );
  let sessionKey: CryptoKey | null = null;
  let everConnected = false;

  const api = {
    close() {
      ws.close();
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

  ws.onopen = () => handlers.onConn("连接中…");
  ws.onclose = () => {
    sessionKey = null;
    handlers.onConn("连接失败：请刷新页面重试");
  };
  ws.onerror = () => handlers.onConn("连接错误");

  // Messages must be handled in order: the hello handshake derives the
  // session key asynchronously, and replayed events arriving before the key is
  // ready would otherwise be dropped by concurrent async handlers.
  const queue: string[] = [];
  let draining = false;
  ws.onmessage = (msg) => {
    queue.push(String(msg.data));
    if (!draining) {
      draining = true;
      void drainQueue();
    }
  };

  async function drainQueue() {
    while (queue.length > 0) {
      const raw = queue.shift()!;
      try {
        const data = JSON.parse(raw);
        if (data.kind === "hello") {
          const dsec = await getDeviceSecret(dev);
          sessionKey = await deriveSessionKey(eph.privateKey, data.serverEphPub, dsec, sid);
          everConnected = true;
          handlers.onConn("已连接（加密）");
          continue;
        }
        if (sessionKey) {
          const pt = await decryptEvent(sessionKey, sid, data.nonce, data.ciphertext);
          handlers.onEvent(pt as RiffpadEvent);
        }
      } catch (e) {
        handlers.onError(e instanceof Error ? e.message : String(e));
      }
    }
    draining = false;
  }
  void everConnected;
  return api;
}
