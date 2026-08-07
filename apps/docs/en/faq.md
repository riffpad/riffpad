# FAQ

## Does my computer need to stay on?

Yes. Agents run on your own computer — Riffpad is the remote control. Sessions stop when the computer sleeps.

## Is my data uploaded to the cloud?

Session content is not. Only encrypted envelopes pass through the relay, which never decrypts or stores them; message history stays on your computer.

## Which coding CLIs are supported?

Claude Code, Codex, and Kimi CLI today; DeepSeek / GLM are on the roadmap.

## How many devices can one daemon pair?

Unlimited. Each device is managed and revocable independently.

## What happens on a network drop or daemon restart?

The client reconnects automatically and de-duplicates replay. After a daemon restart, unfinished sessions recover (Codex reattaches its TUI; Kimi/Claude recover read-only).

## How do I report issues?

Open an issue at <https://github.com/riffpad/riffpad/issues> or email hi@riffpad.ai.
