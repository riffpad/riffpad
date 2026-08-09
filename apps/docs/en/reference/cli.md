# CLI reference

| Command | Description |
|---|---|
| `riffpad login [--url wss://…]` | GitHub device sign-in (default); `--username` for password login |
| `riffpad logout` | Sign out and revoke the relay token |
| `riffpad auth` | Show the current account (live validation) |
| `riffpad pair` | Print a 6-char pairing code and QR |
| `riffpad run [claude\|codex\|kimi] [--name N] [--prompt P] [--cwd D]` | Create a hosted session; the terminal stays visible and interactive |
| `riffpad attach` / `detach` | Inject/remove Claude hooks to capture your own session |
| `riffpad sessions` | List sessions |
| `riffpad status` | Daemon status |
| `riffpad setup [--remove]` | Install/remove the Linux systemd autostart |
| `riffpad kill` | Kill switch: stop all sessions + revoke all devices |
| `riffpad update` | Check and replace with the latest version (SHA256 + atomic swap) |
| `riffpad logs` | Show daemon logs |
| `riffpad version` | Print the version |

## Languages

The CLI output supports zh/en and follows the system language; override with `--lang zh|en`.

## Environment variables

| Variable | Description |
|---|---|
| `RIFFPAD_RELAY_URL` | Relay address (default `wss://api.riffpad.ai`) |
| `RIFFPAD_RELAY_USER` / `RIFFPAD_RELAY_PASSWORD` | Password login credentials |
| `RIFFPAD_URL` | Local daemon address (default `http://127.0.0.1:8787`) |
| `RIFFPAD_DIR` | Data directory (default `~/.config/riffpad`) |
