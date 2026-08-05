# Riffpad

The pocket remote for your AI coding agents. Watch, approve, and steer Claude Code, Codex, DeepSeek CLI, and other AI coding agents from your phone — without staying chained to the desk.

## What is Riffpad?

A lightweight bridge between your local AI coding agents and your phone. Your agents keep running on your own machine; Riffpad mirrors them to your phone for supervision, approval, and remote steering.

## Quickstart

Install the daemon:

**macOS / Linux**

```bash
curl -fsSL https://riffpad.ai/install.sh | sh
```

**Windows PowerShell**

```powershell
# Coming soon — build from source for now.
```

Then start the daemon and attach to your CLI:

```bash
riffpad daemon start
riffpad attach
claude
```

Open `http://127.0.0.1:8787`, pair your phone, and approve from anywhere.

## How it works

```
Adapter → Daemon → Encrypted Relay → Mobile app
```

- **Adapter** — parses each CLI's structured output and hooks into a unified event stream (Claude Code, Codex, DeepSeek, Kimi, …).
- **Daemon** — runs on your computer, owns the keys, manages sessions, and bridges adapters to the relay.
- **Relay** — lightweight encrypted WebSocket relay. Zero-knowledge: it routes ciphertext and stores nothing.
- **Mobile app** — shows sessions and approval cards, and sends approve / reject / prompt actions back.

## Is it safe?

Yes, by design:

- End-to-end encrypted — X25519 key exchange + AES-256-GCM.
- Keys live only on your daemon and phone; the relay never sees plaintext.
- Local-first — code, repositories, and API keys never leave your computer.
- Read-only by default — every approve, reject, or prompt is an explicit action.

## Documentation

[riffpad.ai/docs](https://riffpad.ai/docs)
