const enc = new TextEncoder();
const dec = new TextDecoder();

export function b64u(buf: ArrayBuffer | Uint8Array): string {
  const b = buf instanceof Uint8Array ? buf : new Uint8Array(buf);
  let s = "";
  for (let i = 0; i < b.length; i += 0x8000) {
    s += String.fromCharCode.apply(null, Array.from(b.subarray(i, i + 0x8000)));
  }
  return btoa(s).replace(/\+/g, "-").replace(/\//g, "_").replace(/=+$/, "");
}

export function b64uToBytes(s: string): Uint8Array<ArrayBuffer> {
  const b = atob(s.replace(/-/g, "+").replace(/_/g, "/"));
  const u = new Uint8Array(b.length);
  for (let i = 0; i < b.length; i++) u[i] = b.charCodeAt(i);
  return u;
}

export function jwkToRaw(jwk: JsonWebKey): Uint8Array<ArrayBuffer> {
  const x = b64uToBytes(jwk.x!);
  const y = b64uToBytes(jwk.y!);
  const raw = new Uint8Array(65);
  raw[0] = 4;
  raw.set(x, 1);
  raw.set(y, 33);
  return raw;
}

export async function genPair(): Promise<CryptoKeyPair> {
  return crypto.subtle.generateKey({ name: "ECDH", namedCurve: "P-256" }, true, ["deriveBits"]);
}

export async function importIdentity(jwk: JsonWebKey): Promise<CryptoKey> {
  return crypto.subtle.importKey("jwk", jwk, { name: "ECDH", namedCurve: "P-256" }, false, ["deriveBits"]);
}

export async function deviceSecret(dev: { jwk: JsonWebKey; serverPub: string | null }): Promise<Uint8Array<ArrayBuffer>> {
  if (!dev.serverPub) throw new Error("device not paired");
  const priv = await importIdentity(dev.jwk);
  const spub = await crypto.subtle.importKey(
    "raw",
    b64uToBytes(dev.serverPub),
    { name: "ECDH", namedCurve: "P-256" },
    false,
    [],
  );
  const bits = await crypto.subtle.deriveBits({ name: "ECDH", public: spub }, priv, 256);
  return new Uint8Array(bits);
}

export async function deriveSessionKey(
  ephPriv: CryptoKey,
  serverEphPubB64: string,
  deviceSecret: Uint8Array<ArrayBuffer>,
  sid: string,
): Promise<CryptoKey> {
  const serverEph = await crypto.subtle.importKey(
    "raw",
    b64uToBytes(serverEphPubB64),
    { name: "ECDH", namedCurve: "P-256" },
    false,
    [],
  );
  const ephBits = await crypto.subtle.deriveBits({ name: "ECDH", public: serverEph }, ephPriv, 256);
  const hkdf = await crypto.subtle.importKey("raw", ephBits, "HKDF", false, ["deriveKey"]);
  const info = new Uint8Array(enc.encode("riffpad/session-v1/" + sid));
  return crypto.subtle.deriveKey(
    { name: "HKDF", hash: "SHA-256", salt: deviceSecret, info },
    hkdf,
    { name: "AES-GCM", length: 256 },
    false,
    ["encrypt", "decrypt"],
  );
}

export async function encryptEvent(
  key: CryptoKey,
  sessionId: string,
  event: unknown,
): Promise<{ nonce: string; ciphertext: string }> {
  const iv = crypto.getRandomValues(new Uint8Array(12));
  const pt = enc.encode(JSON.stringify(event));
  const aad = new Uint8Array(enc.encode(sessionId));
  const ct = await crypto.subtle.encrypt(
    { name: "AES-GCM", iv, additionalData: aad },
    key,
    pt,
  );
  return { nonce: b64u(iv), ciphertext: b64u(ct) };
}

export async function decryptEvent(
  key: CryptoKey,
  sessionId: string,
  nonceB64: string,
  ciphertextB64: string,
): Promise<unknown> {
  const pt = await crypto.subtle.decrypt(
    {
      name: "AES-GCM",
      iv: b64uToBytes(nonceB64),
      additionalData: new Uint8Array(enc.encode(sessionId)),
    },
    key,
    b64uToBytes(ciphertextB64),
  );
  return JSON.parse(dec.decode(pt));
}
