<p align="center">
  <picture>
    <source media="(prefers-color-scheme: dark)" srcset="docs/assets/riffpad-banner-dark.png" />
    <source media="(prefers-color-scheme: light)" srcset="docs/assets/riffpad-banner-light.png" />
    <img src="docs/assets/riffpad-banner-light.png" alt="Riffpad" width="100%" />
  </picture>
</p>

<p align="center">
  <strong>Riffpad: The AI-native workspace in seconds.</strong><br />
</p>

<p align="center">
  <a href="https://riffpad.ai">Website</a> ·
  <a href="docs/prd.md">PRD</a> ·
  <a href="docs/tsd.md">TSD</a> ·
  <a href="LICENSE">License</a>
</p>

---

## What is Riffpad?

Riffpad is an **AI-native sketchbook for code**.

Describe an idea in plain language — on your phone, tablet, or desktop — and watch it become a running workspace: files, a live preview, and an autonomous agent inside a sandbox. When the prototype is validated, bridge it to Cursor, Claude Code, or your own pipeline in one click.

No setup. No local environment. No lost context.

## Why Riffpad?

Most AI coding tools sit at one of two extremes: a chat window that thinks but doesn't build, or a heavy IDE that builds but demands a desk. Riffpad lives upstream of both.

### vs. ChatGPT / Gemini

|                  | ChatGPT / Gemini                 | Riffpad                               |
| ---------------- | -------------------------------- | ------------------------------------- |
| Where ideas live | Buried in an endless chat scroll | A file tree with version history      |
| Output           | Text replies                     | A running sandbox with live preview   |
| Environment      | None                             | Stateful, sandboxed workspace         |
| Persistence      | Copy-paste snippets              | Branch, persist, refactor, and export |

### vs. Claude Code / Codex

|                   | Claude Code / Codex               | Riffpad                         |
| ----------------- | --------------------------------- | ------------------------------- |
| Runtime           | Your local machine                | Cloud                           |
| Setup             | Repo, dependencies, terminal, Git | None — describe it and start    |
| Devices           | Desktop, phone perhaps?           | Phone, tablet, desktop          |
| Workflow position | Downstream production IDE         | Upstream idea incubator         |
| Handoff           | —                                 | Zip, MCP server, or Agent Skill |

## Local Development

This repo is a pnpm monorepo. You need:

- Node.js >= 18.17
- pnpm >= 9.0
- Go >= 1.22
- Docker & Docker Compose (for local Postgres / Redis / MinIO)

### Quick start

```bash
# 1. Install dependencies
cp .env.example .env
make install

# 2. Start the full local stack
make dev-local
```

> The compose Postgres is mapped to host port **5433** (container 5432) so it
> does not clash with a system-wide Postgres. `DATABASE_URL` in `.env.example`
> already points there. The API auto-creates the `workspaces` / `messages`
> tables on startup (GORM AutoMigrate).

Services will be available at:

| Service | URL |
| ------- | --- |
| App (app.riffpad.ai) | http://localhost:3000 |
| Landing (riffpad.ai) | http://localhost:3001 |
| API (api.riffpad.ai) | http://localhost:8080 |
| API health check | http://localhost:8080/healthz |
| Postgres | localhost:5433 |
| MinIO console | http://localhost:9001 |

### Useful commands

```bash
make build     # Build all apps
make test      # Run all tests
make lint      # Run all linters
make down      # Stop Docker Compose services
make clean     # Remove build artifacts and node_modules
```

### Project structure

```
riffpad/
├── apps/
│   ├── app/       # app.riffpad.ai (Next.js 14 + shadcn/ui)
│   ├── landing/   # riffpad.ai marketing site
│   └── api/       # Go backend (Echo + GORM)
├── packages/
│   └── shared-types/  # OpenAPI definitions
├── infra/
│   └── docker-compose.yml  # Local dev infrastructure
└── docs/
    ├── prd.md     # Product requirements
    └── tsd.md     # Technical specification
```

## License

Licensed under the [Apache License 2.0](LICENSE).
