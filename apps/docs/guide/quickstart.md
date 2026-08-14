# Quickstart

> **Using an AI agent?** Paste this into its chat — it reads the skill and sets up Riffpad for you:
>
> ```bash
> curl -fsSL https://riffpad.ai/SKILL.md
> ```

## 1. Install the CLI

On your computer:

::: code-group

```bash [macOS / Linux]
curl -fsSL https://riffpad.ai/install.sh | sh
```

```powershell [Windows]
irm https://riffpad.ai/install.ps1 | iex
```

:::

The Windows script downloads the latest binary, adds it to your user PATH, and registers a logon autostart task for the daemon; remove it with `riffpad setup --remove`.

Verify:

```bash
riffpad version
```

## 2. Sign in

```bash
riffpad login
```

This opens a browser for GitHub authorization (password login stays available via `--username`). The daemon restarts automatically after login.

Check your account:

```bash
riffpad auth
```

## 3. Pair your phone

```bash
riffpad pair
```

The terminal prints a 6-character code and a QR code. Open <https://app.riffpad.ai> on your phone, sign in with the same GitHub account, then scan or enter the code.

## 4. Start and watch a session

```bash
riffpad run codex
```

Or with a specific agent and prompt:

```bash
riffpad run claude --prompt "Refactor the auth middleware"
```

Sessions appear in the app's Sessions page; open one to watch progress, approve actions, and send instructions.

## Common commands

| Command | Purpose |
|---|---|
| `riffpad login` / `logout` | GitHub device sign-in / sign-out |
| `riffpad auth` | Show the current account |
| `riffpad pair` | Print pairing code + QR |
| `riffpad run <claude\|codex\|kimi>` | Create and run a session |
| `riffpad sessions` | List sessions |
| `riffpad update` | Self-update the binary |
| `riffpad kill` | Kill switch: stop sessions + revoke devices |
