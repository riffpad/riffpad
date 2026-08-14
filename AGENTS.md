# Riffpad Agent Guide

## Project

Riffpad is a remote control for AI coding agents: a local daemon bridges the CLI sessions on your computer, an encrypted relay streams events, and the phone supervises, approves, and steers.

## Rules

1. **Read `docs/prd.md` and `docs/tsd.md` first** — they are the source of truth for product positioning and technical design; `docs/design.md` is a quick overview.
2. **Minimal changes** — do only what is asked.
3. **Security first** — the relay is zero-trust, transport is end-to-end encrypted, and the mobile client is read-only by default; treat any change involving credentials or keys with extra care.
4. **Test and verify** — run the relevant tests and build after a change.
5. **Keep the protocol in sync** — any event-protocol change must update both `packages/protocol` and `docs/design.md`.
6. **Use the `gh` CLI** — manage issues, branches, and PRs through the GitHub workflow.
7. **Keep the dev plan current** — `docs/dev-plan.md` is the progress tracker: record new tasks, check off completed ones, and note blockers there (follow its "Update rules" section).
8. **Log manual testing** — `test-notes/` (gitignored) holds manual test notes named `test-note-YYYY-MM-DD.txt`; record issues found during testing there and reference the issue number when fixing.

## Conventions

- Branches: `feature/<n>-<slug>` / `bugfix/<n>-<slug>`
- Commits: `feat(daemon): ... (#n)` / `fix(relay): ... (#n)`
- PRs: title + verification steps, body says `Closes #n`

## Done means

- Code builds and tests pass
- Protocol and docs agree
- PR reviewed and merged
