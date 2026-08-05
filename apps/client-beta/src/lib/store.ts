import type { Device, RelaySession } from "./types";

export const isRelay: boolean =
  typeof (window as unknown as { RIFFPAD_RELAY?: number }).RIFFPAD_RELAY !== "undefined";

const relayKey = "riffpad.relay";
const deviceKey = "riffpad.device";

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
  if (opts.body) headers["Content-Type"] = "application/json";
  return fetch(path, { ...opts, headers });
}
