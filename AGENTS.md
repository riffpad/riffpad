# Riffpad - Agent Guide

## Project

AI-Native lightweight code sketchbook. Capture inspiration on mobile, run prototypes in an isolated sandbox, then bridge validated ideas to downstream IDEs (Cursor / Claude Code).

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
2. **Minimal changes** — do only what was asked; don't refactor unrelated code.
3. **Mobile-first** — every UI change must work on phone/tablet.
4. **Sandbox safety** — treat sandbox isolation as non-negotiable.
5. **Test & verify** — run/build/test what you change; never commit secrets.
6. **Serverless by default** — use managed services for non-core infra.
7. **Update docs** — keep `AGENTS.md`, `TSD`, and `README` in sync with code.

## Dev Workflow

- Monorepo: `apps/{app,landing,api}`, `packages/{shared-types,ui,sandbox-images}`, `infra/`
- Backend layout: `cmd/{api,sandbox-worker,lifecycle-worker}`, `internal/{config,domain,repository,service,handler,client,middleware,pkg}`
- Local dev: `make dev` (Supabase) or `make dev-local` (full Docker Compose)
- Git: small commits, clear messages, reference issues when applicable
