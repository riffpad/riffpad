# Riffpad - Agent Guide

## Project

AI-Native lightweight code sketchbook. Capture inspiration on mobile, run prototypes in an isolated sandbox, then bridge validated ideas to downstream IDEs.

## Key Docs

- `docs/prd.md` — Product requirements
- `docs/tsd.md` — Technical spec (source of truth; read before implementing)

## Tech Stack

| Layer | Choice |
|-------|--------|
| Frontend | Next.js 14 + TypeScript + Tailwind + shadcn/ui |
| Landing | Next.js 14 + Framer Motion |
| Backend | Go 1.22+ + Echo/Fiber + GORM |
| Auth / DB / Storage | Supabase |
| Cache | Upstash Redis |
| Sandbox | Docker + gVisor (dev); Firecracker (prod) |
| AI | OpenAI-compatible providers (Kimi / GLM / Qwen / DeepSeek) |

## Agent Rules

1. **Read TSD first** — architecture decisions live there.
2. **Minimal changes** — do only what was asked.
3. **Mobile-first** — every UI change must work on phone/tablet.
4. **Sandbox safety** — treat isolation as non-negotiable.
5. **Test & verify** — run/build/test what you change; never commit secrets.
6. **Prefer serverless** — use managed services for non-core infra.
7. **Keep docs in sync** — update TSD/AGENTS/README when architecture changes.

## Dev Workflow

### Track every requirement with a GitHub Issue

- **Epic** → large feature module (e.g., "Agent Chat System")
- **Story** → user-visible feature (e.g., "switch model in chat")
- **Task** → concrete dev work (e.g., "add `Message.ModelID` field")

Every Issue must include:
- Why / What
- Acceptance Criteria (checkable)
- TSD reference (if any)
- Estimated size: S / M / L

### Code-link discipline

- Branch: `feature/42-model-toggle`, `bugfix/51-sandbox-timeout`
- Commit: `feat(api): add per-message model override (#42)`
- PR: title + summary + how to verify; body should say `Closes #42`

### TSD sync rule

If a change makes `docs/tsd.md` inaccurate, update TSD in the same PR.

### Local dev

- `make dev` — connect to Supabase
- `make dev-local` — full Docker Compose stack

## Done Means

- Code works and is tested
- TSD/docs are consistent
- PR is reviewed and merged
- Issue is closed
