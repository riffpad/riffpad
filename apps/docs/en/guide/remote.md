# Remote control

After pairing, open <https://app.riffpad.ai> (or the local daemon page on port 8787):

- **Sessions**: see every running session with its status light (WAITING FOR INPUT / RUNNING / DONE) and recent activity; tap a card to open it.
- **Session detail**: live chat stream plus tool-call logs; an `Interrupt` button appears while the agent is running; scrolling up loads older history in pages.
- **Approvals**: approve or reject permission requests with one tap.
- **Instructions**: type a message and send it with the `→` button; it travels end-to-end encrypted to your daemon.
- **Devices**: list paired devices, identify this device, revoke access.

## Do I need tmux?

No. `riffpad run` hosts the session for you; `riffpad attach` can also capture a Claude session you started yourself.

If you want a session to survive closing the terminal, pair it with tmux:

```bash
tmux new -s work
riffpad run --cli codex
# Ctrl-b d to detach; reattach later with: tmux attach -t work
```
