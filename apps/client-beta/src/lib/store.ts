import type { Device, RelaySession } from "./types";

export const isRelay: boolean =
  typeof (window as unknown as { RIFFPAD_RELAY?: number }).RIFFPAD_RELAY !== "undefined";

const relayKey = "riffpad.relay";
const deviceKey = "riffpad.device";
const localTokenKey = "riffpad.localtoken";

// localTokenStore holds the daemon's local API token (non-relay mode). The
// pairing/session URLs printed by the CLI carry it as ?token=...; capture it
// once and reuse it for every API call and WS handshake. The daemon rejects
// calls without it.
export const localTokenStore = {
  get(): string {
    try {
      const q = new URLSearchParams(location.search).get("token");
      if (q) {
        localStorage.setItem(localTokenKey, q);
        return q;
      }
      return localStorage.getItem(localTokenKey) || "";
    } catch {
      return "";
    }
  },
};

export const relayStore = {
  get(): RelaySession | null {
    try {
      const v = localStorage.getItem(relayKey);
      return v ? (JSON.parse(v) as RelaySession) : null;
    } catch {
      return null;
    }
  },
  set(v: RelaySession) {
    localStorage.setItem(relayKey, JSON.stringify(v));
  },
  clear() {
    localStorage.removeItem(relayKey);
  },
};

export const deviceStore = {
  get(): Device | null {
    try {
      const v = localStorage.getItem(deviceKey);
      return v ? (JSON.parse(v) as Device) : null;
    } catch {
      return null;
    }
  },
  set(v: Device) {
    localStorage.setItem(deviceKey, JSON.stringify(v));
  },
};

export async function api(path: string, opts: RequestInit = {}): Promise<Response> {
  const headers: Record<string, string> = { ...(opts.headers as Record<string, string>) };
  const tok = relayStore.get()?.token;
  if (tok) headers["Authorization"] = "Bearer " + tok;
  if (!isRelay) {
    const ltok = localTokenStore.get();
    if (ltok) headers["X-Riffpad-Token"] = ltok;
  }
  if (opts.body) headers["Content-Type"] = "application/json";
  return fetch(path, { ...opts, headers });
}
