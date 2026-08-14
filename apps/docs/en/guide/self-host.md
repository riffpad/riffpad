# Self-host the Relay

By default the daemon and your phone connect to the hosted relay at `app.riffpad.ai`. If you'd rather own the relay — on your own VPS, intranet, or a machine at home — a single command brings up an independent relay container, and **the data stays entirely on your server**.

The relay is zero-knowledge: it forwards end-to-end-encrypted envelopes and never sees session content. It only stores **metadata** — accounts, devices, sessions.

## One command

::: tip Prerequisite
A machine with Docker installed (a VPS, a home server, even a NAS).
:::

```bash
curl -fsSL https://riffpad.ai/selfhost.sh | sh
```

The script:

1. pulls the public image `ghcr.io/riffpad/relay`;
2. writes `docker-compose.yml`, `.env`, and a `data/` volume into `~/.riffpad-relay/`;
3. starts the relay (embedded SQLite by default — no separate database);
4. prints the address to open, and how to point your computer's daemon at it.

> After the image is first published, set `riffpad/relay` to **public** under [GitHub Packages](https://github.com/riffpad/riffpad/pkgs/container/relay), otherwise `docker pull` returns 401.

## Public + automatic HTTPS

If you have a domain and want the relay reachable from the public internet, pass `--domain` and the script adds a Caddy sidecar that provisions certificates automatically:

```bash
curl -fsSL https://riffpad.ai/selfhost.sh | sh -s -- --domain relay.example.com
```

Point the domain's DNS A record at this machine first. Caddy obtains a Let's Encrypt certificate on the first request (add `--email you@x.com` for the ACME account).

No domain? The default HTTP mode is fine for a **trusted LAN / testing**. You can also put your own nginx/Caddy in front — see the [relay deployment README](https://github.com/riffpad/riffpad/tree/main/infra/relay#readme).

## Point your computer at the self-hosted relay

Once the relay is up, register and sign in to it on each machine running a daemon:

```bash
# HTTPS mode (your domain) or HTTP mode (ws://<IP>:9090)
export RIFFPAD_RELAY_URL=wss://relay.example.com

riffpad relay login --url "$RIFFPAD_RELAY_URL" --username <your-username>
```

You'll be prompted for the password (or set `RIFFPAD_RELAY_PASSWORD`). After login the daemon reconnects automatically; then `riffpad pair` to pair your phone as usual.

::: tip
The username/password is the account **you register on this self-hosted relay** — open the relay address in a browser and sign up to create it.
:::

## Upgrade to Postgres (optional)

Default SQLite is plenty for most self-hosted setups. For higher concurrency or managed backups, switch to Postgres: edit `~/.riffpad-relay/docker-compose.yml`, replacing the relay service and adding a postgres service:

```yaml
services:
  relay:
    image: ghcr.io/riffpad/relay:latest
    restart: unless-stopped
    expose: ["9090"]
    env_file: [.env]
    volumes:
      - ./data:/data
    environment:
      RELAY_LISTEN: "0.0.0.0"
      RELAY_PORT: "9090"
      DATABASE_URL: postgres://riffpad:${POSTGRES_PASSWORD}@postgres:5432/riffpad?sslmode=disable
    depends_on:
      postgres:
        condition: service_healthy

  postgres:
    image: postgres:17-alpine
    restart: unless-stopped
    environment:
      POSTGRES_USER: riffpad
      POSTGRES_PASSWORD: ${POSTGRES_PASSWORD}
      POSTGRES_DB: riffpad
    volumes:
      - pg-data:/var/lib/postgresql/data
    healthcheck:
      test: ["CMD-SHELL", "pg_isready -U riffpad"]
      interval: 3s
      timeout: 3s
      retries: 10

volumes:
  pg-data:
```

Set `POSTGRES_PASSWORD=...` in `.env`, then `docker compose up -d`. The relay detects `DATABASE_URL` and auto-migrates to Postgres.

## Manage, upgrade, back up

Everything lives in the install dir, `~/.riffpad-relay/`:

```bash
cd ~/.riffpad-relay

docker compose logs -f relay     # follow logs
docker compose restart relay     # restart
docker compose pull && docker compose up -d   # upgrade to the latest image
docker compose down               # stop
```

**Backups**: stop the relay and copy the `data/` directory (SQLite mode) or dump Postgres. Metadata is encrypted at rest; session content never touches the relay's disk.

**Change port / domain / image tag**: re-run the installer with new flags — it regenerates the compose file and leaves the data volume intact:

```bash
curl -fsSL https://riffpad.ai/selfhost.sh | sh -s -- --port 8080 --tag v0.2.5
```

## Security notes

- Default HTTP mode is **unencrypted** — trusted LAN only. For public access use `--domain` (Caddy auto-TLS) or your own reverse proxy.
- The relay stores only metadata (accounts/devices/sessions). Session content is end-to-end encrypted and unreadable to the relay.
- `~/.riffpad-relay/.env` holds credentials — `chmod 600` it and keep it out of git.
- For GitHub sign-in, set `GITHUB_CLIENT_ID` / `GITHUB_CLIENT_SECRET` in `.env` and point the OAuth callback at your relay domain.
