# Architecture

```
┌─────────────────┐      ┌────────────────────┐      ┌──────────────┐
│  Your computer  │      │  Cloud relay       │      │  Phone/Web   │
│                 │ WSS  │                    │ WSS  │              │
│  coding CLI     │◀────▶│  encrypted envelopes│◀────▶│  session list │
│  (claude/codex) │      │  no decrypt/storage│      │  approvals    │
│     ↕ adapters  │      └────────────────────┘      │  instructions │
│  daemon         │                                   └──────────────┘
└─────────────────┘
```

- **coding CLI**: Claude Code, Codex, Kimi CLI run on your computer; API keys never leave your machine.
- **daemon**: captures events, injects instructions, and relays approvals through adapters (Claude host protocol / Codex app-server / Kimi ACP).
- **relay**: WebSocket rooms that forward encrypted envelopes; it cannot read content and only stores non-sensitive metadata.
- **client**: the web app (PWA direction), read-only by default; approvals and instructions are explicit actions.

## Session history

Session messages are encrypted and persisted locally by the daemon. The relay never stores message content. On every open, the daemon replays recent history; older messages load on demand as you scroll up.
