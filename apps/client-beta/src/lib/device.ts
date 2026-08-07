import { genPair, jwkToRaw, b64u, deviceSecret } from "./crypto";
import { api, deviceStore } from "./store";
import type { Device } from "./types";

export async function ensureIdentity(): Promise<Device> {
  let dev = deviceStore.get();
  if (!dev) {
    const kp = await genPair();
    const jwk = await crypto.subtle.exportKey("jwk", kp.privateKey);
    dev = { deviceId: null, serverPub: null, jwk };
    deviceStore.set(dev);
  }
  return dev;
}

export async function pairDevice(code: string): Promise<Device> {
  const dev = await ensureIdentity();
  const raw = jwkToRaw(dev.jwk);
  const res = await api("/api/pair", {
    method: "POST",
    body: JSON.stringify({ code, name: deviceDisplayName(), curve: "p256", publicKey: b64u(raw) }),
  });
  const data = await res.json();
  if (!res.ok) throw new Error(data.error || "配对失败");
  dev.deviceId = data.deviceId;
  dev.serverPub = data.serverPublicKey;
  deviceStore.set(dev);
  return dev;
}

export async function getDeviceSecret(dev: Device): Promise<Uint8Array<ArrayBuffer>> {
  return deviceSecret(dev);
}

// deviceDisplayName derives a readable device label from the user agent,
// e.g. "macOS · Chrome" or "iOS · Safari".
export function deviceDisplayName(): string {
  const ua = navigator.userAgent;
  const os = /iPhone|iPad|iPod/.test(ua) ? "iOS"
    : /Android/.test(ua) ? "Android"
    : /Macintosh|Mac OS X/.test(ua) ? "macOS"
    : /Windows/.test(ua) ? "Windows"
    : /Linux/.test(ua) ? "Linux"
    : "Web";
  const browser = /Edg\//.test(ua) ? "Edge"
    : /OPR\/|Opera/.test(ua) ? "Opera"
    : /Chrome\//.test(ua) ? "Chrome"
    : /Safari\//.test(ua) ? "Safari"
    : /Firefox\//.test(ua) ? "Firefox"
    : "Browser";
  return os + " · " + browser;
}
