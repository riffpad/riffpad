---
name: riffpad
description: Control AI coding agents (Claude Code, Codex, Kimi) from your phone via Riffpad. Use when the user wants to watch, approve, or steer a coding agent remotely / set up Riffpad / pair their phone / keep a long-running agent session going while away from the computer.
---

# riffpad

Watch, approve, and steer AI coding agents (Claude Code, Codex, Kimi) from your phone. A local daemon bridges the CLI agents running on the user's own machine to their phone, end-to-end encrypted through a zero-knowledge relay. **Not a cloud IDE** — agents keep running on the user's machine with their own API keys and repos; the phone is just a mirror + remote control.

## Install

If `riffpad` is not installed:

```bash
# macOS / Linux
curl -fsSL https://riffpad.ai/install.sh | sh

# Windows (PowerShell)
irm https://riffpad.ai/install.ps1 | iex
```

Verify with `riffpad version`. If still not found after install, open a new terminal.

---

## Onboarding flow (run these in order)

1. **Install** the daemon (above), then `riffpad version` to confirm.
2. **Sign in on the computer**: `riffpad login`
   - Opens a browser for GitHub authorization (like `gh auth login`). The **user** completes it in the browser; the daemon registers this machine and restarts automatically.
   - Non-GitHub / password login: set `RIFFPAD_RELAY_USER` + `RIFFPAD_RELAY_PASSWORD`, then `riffpad login --username`.
3. **Pair the phone**: `riffpad pair`
   - Prints a 6-char code + QR. **The user** must open https://app.riffpad.ai on their phone, sign in with the same account, and enter the code (or scan the QR). You can't complete this step for them — tell them what to do.
4. **Start a session**: `riffpad run codex` (or `claude` / `kimi`)
   - The terminal stays visible and interactive. The session mirrors to the phone immediately; the user can watch progress, approve permission prompts, and send new instructions from the phone.

> To capture a Claude session the user **already started themselves** (instead of launching a new one), use `riffpad attach` (injects Claude hooks) — not `run`.

---

## CLI reference

| Command | Description |
|---|---|
| `riffpad login [--url wss://…]` | GitHub device sign-in (default); `--username` for password login |
| `riffpad logout` | Sign out and revoke the relay token |
| `riffpad auth` | Show the current account (live validation) |
| `riffpad pair` | Print a 6-char pairing code and QR |
| `riffpad run [claude\|codex\|kimi] [--name N] [--prompt P] [--cwd D]` | Create a hosted session; terminal stays visible/interactive |
| `riffpad attach` / `detach` | Inject/remove Claude hooks to capture the user's own running session |
| `riffpad sessions` | List sessions |
| `riffpad status` | Daemon status |
| `riffpad setup [--remove]` | Install/remove the Linux systemd autostart |
| `riffpad kill` | Kill switch — stop all sessions + revoke all paired devices |
| `riffpad update` | Replace with the latest version (SHA256 + atomic swap) |
| `riffpad logs` | Show daemon logs |
| `riffpad version` | Print the version |

### `run` options
- `--name N` — label the session
- `--prompt P` — start with an initial prompt
- `--cwd D` — set the working directory

### Languages
Output follows the system language (zh/en); override with `--lang zh|en`.

### Environment variables
| Variable | Default | Purpose |
|---|---|---|
| `RIFFPAD_RELAY_URL` | `wss://api.riffpad.ai` | Relay address (change to self-host) |
| `RIFFPAD_RELAY_USER` / `RIFFPAD_RELAY_PASSWORD` | — | Password-login credentials |
| `RIFFPAD_URL` | `http://127.0.0.1:8787` | Local daemon address |
| `RIFFPAD_DIR` | `~/.config/riffpad` | Data directory |

---

## Self-host the relay (optional)
The relay is the only server component, and it's zero-knowledge — it forwards ciphertext and stores nothing. To point Riffpad at your own relay, set `RIFFPAD_RELAY_URL` before `login`. Relay source + self-host notes: https://github.com/riffpad/riffpad (`apps/relay`).

---

## Common pitfalls

| Pitfall | Correct approach |
|---|---|
| Treating it like a cloud IDE | Agents run on the user's machine with their own keys; the phone only mirrors + sends approvals/instructions. Don't try to move the session to the cloud. |
| Skipping pairing | `riffpad pair` must run, and the user must complete pairing on their phone at app.riffpad.ai. Without it, nothing reaches the phone. |
| Using `run` for a session the user already started | Use `riffpad attach` (injects Claude hooks into an existing session), not `run` (which starts a new one). |
| Approvals not showing on the phone | Confirm `riffpad status` is healthy and the phone is paired; check `riffpad logs`. |
| `login` can't open a browser (headless) | Use `--username` with `RIFFPAD_RELAY_USER` / `RIFFPAD_RELAY_PASSWORD`. |
| Need to stop everything immediately | `riffpad kill` — stops all sessions + revokes all paired devices (kill switch). |

---

## Notes
- **Local-first**: code, repos, and API keys never leave the user's machine.
- **End-to-end encrypted** (X25519 + AES-256-GCM); the relay is zero-knowledge.
- **Read-only by default** on the phone — every approve/reject/prompt is an explicit tap.
- Docs: https://www.riffpad.ai/docs/guide/quickstart · Repo: https://github.com/riffpad/riffpad · App: https://app.riffpad.ai
