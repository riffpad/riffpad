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
    body: JSON.stringify({ code, name: "web", curve: "p256", publicKey: b64u(raw) }),
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
