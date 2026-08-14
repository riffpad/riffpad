# Security model

## End-to-end encryption

- Device keys are exchanged during pairing (X25519 + P-256); the session key is derived on both sides.
- Messages are encrypted with AES-GCM; the relay only sees ciphertext.
- Client keys live in browser storage (native apps will use Keychain/Keystore).

## Zero-knowledge relay

The relay does not persist session content — only necessary metadata (accounts, devices, online state). No plaintext messages are stored.

## Devices & revocation

- One account can pair many devices; each device can be revoked independently.
- `riffpad kill` is a kill switch: stop all sessions and revoke every device.
- Revoking the current device sends the client back to pairing immediately.

## Accounts

- GitHub OAuth is supported today (username/password login remains).
- Login tokens last 30 days and are revoked on sign-out.
