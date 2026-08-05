<p align="center">
  <img src="docs/assets/riffpad-logo.svg" alt="Riffpad" width="96" />
</p>

<p align="center">
  <strong>The pocket remote for your AI coding agents.</strong><br />
  Watch, approve, and steer Claude Code, Codex, DeepSeek CLI, and other AI coding agents from your phone — without staying chained to the desk.
</p>

<p align="center">
  <a href="https://riffpad.ai">Website</a> ·
  <a href="docs/prd.md">PRD</a> ·
  <a href="docs/tsd.md">TSD</a> ·
  <a href="docs/design.md">Design</a> ·
  <a href="docs/dev-plan.md">Roadmap</a>
</p>

---

## What is Riffpad?

Riffpad is a **mobile remote control for AI coding agents**.

Your agents keep running on your own computer — with your API keys, your repositories, and your toolchain. Riffpad bridges them to your phone over an end-to-end encrypted relay:

- **Supervise** — watch structured progress, tool calls, file changes, and commands in real time, with a terminal fallback when a CLI exposes no structure.
- **Approve** — approval requests arrive as push notifications; allow, deny, or edit the condition with one tap.
- **Steer** — send a new instruction to a running session from anywhere.
- **Stay private** — the relay is zero-knowledge: it routes ciphertext and stores nothing. Code and keys never leave your machine.

One sentence: *when your AI is waiting for you, it calls you in your pocket.*

## Why Riffpad?

Most AI coding tools keep you at the desk — either watching a terminal, or missing the moment your agent needs a decision.

### vs. staying at the desk

|                         | Terminal-only workflow        | Riffpad                              |
| ----------------------- | ----------------------------- | ------------------------------------ |
| Where you need to be    | At the computer               | Anywhere — phone, subway, bed        |
| Approval requests       | Missed when you step away     | Push notification, one-tap response  |
| Steering a long task    | Return to the terminal        | Send a prompt from your phone        |
| Multiple sessions       | Hard to track                 | Session list with live status        |

### vs. official mobile apps

|               | Official mobile apps        | Riffpad                             |
| ------------- | --------------------------- | ----------------------------------- |
| Coverage      | One vendor's CLI            | Claude Code, Codex, DeepSeek, Kimi… |
| Runtime       | Vendor cloud                | Your own computer                   |
| Openness      | Locked to one ecosystem     | CLI-agnostic adapters               |
| Privacy       | Depends on vendor           | Local-first, end-to-end encrypted   |

### vs. SSH + tmux

|              | SSH / tmux            | Riffpad                             |
| ------------ | --------------------- | ----------------------------------- |
| Setup        | Network, keys, config | One daemon + QR pairing             |
| View         | Raw terminal          | Structured events, approval cards   |
| Control      | Manual keystrokes     | Approve / reject / prompt actions   |

## How it works

1. **Daemon** — a small Go binary runs on your computer, wraps or attaches to CLI sessions (Claude Code, Codex, …), and owns the keys.
2. **Encrypted relay** — events travel through a WebSocket relay as end-to-end encrypted envelopes (X25519 + AES-256-GCM). The relay cannot read or store session content.
3. **Your phone** — the Riffpad app shows sessions, approval cards, and a terminal fallback, and sends explicit approve / reject / prompt actions back.

## Status

- **M0 closed loop verified** — attach a Claude session, receive an approval card on the phone, approve, and the agent continues.
- **M1 in progress** — relay production deployment (api.riffpad.ai), mobile PWA, Web Push, Codex adapters, and seed users.

## Local development

Requirements:

- Go 1.25+
- Node.js >= 18.17 and pnpm >= 9
- Claude Code (for the M0 flow) or any supported CLI

Quick start:

```bash
make build-daemon                 # build riffpad + riffpadd
./apps/daemon/bin/riffpad daemon start
./apps/daemon/bin/riffpad pair    # show pairing code + QR
# open http://127.0.0.1:8787, enter the pairing code, start a Claude session
```

Attach mode (recommended for approvals):

```bash
make install-daemon               # install to ~/.local/bin
riffpad attach                    # inject hooks into ~/.claude/settings.json
claude                            # run your normal interactive session
```

Web apps:

```bash
pnpm install
pnpm dev:mobile                   # mobile PWA
pnpm --filter landing dev         # marketing site
```

## Project structure

```
riffpad/
├── apps/
│   ├── daemon/       # Go — local bridge on the user's computer
│   ├── relay/        # Go — encrypted WebSocket relay (zero-knowledge)
│   ├── mobile/       # Mobile PWA (Next.js 14)
│   └── landing/      # Marketing site (Next.js 14 + Tailwind)
├── packages/
│   └── protocol/     # Shared event protocol (daemon ↔ relay ↔ mobile)
├── docs/             # PRD, TSD, design overview, development plan
├── infra/            # Relay deployment and Docker Compose
├── Makefile
└── AGENTS.md
```

## Documentation

- [Product requirements (PRD)](docs/prd.md)
- [Technical specification (TSD)](docs/tsd.md)
- [Design overview](docs/design.md)
- [Development plan](docs/dev-plan.md)
