# Riffpad Relay deployment

> **Want to self-host in one command?** On any machine with Docker:
>
> ```bash
> curl -fsSL https://riffpad.ai/selfhost.sh | sh
> ```
>
> Add a domain for automatic HTTPS: `sh -s -- --domain relay.example.com`.
> See the [self-host docs](https://riffpad.ai/docs/guide/self-host).
> This file covers the advanced cases (building from source / manual VPS
> setup / migration) below.

The relay is the cloud WebSocket broker: the daemon (host) on a user's computer and the phone (viewer) both connect to it outbound, so no port forwarding is needed. User/host/device/session metadata is persisted to SQLite by default (`RELAY_DATA_DIR/relay.db`, WAL mode); setting `DATABASE_URL` (a Postgres DSN) switches to Postgres automatically — both drivers are implemented, and the relay survives restarts.

## Options at a glance

| Option | Cost | TLS | Good for | Notes |
|---|---|---|---|---|
| Fly.io | free tier / ~$3-5/mo | automatic | overseas testing, MVP | `fly deploy` and done |
| Railway / Render | limited free tier | automatic | quick spin-up | free tiers sleep; needs always-on config |
| VPS + Caddy | ~$5/mo | Caddy automatic | long-term / China-friendly | most flexible, recommended for production |
| Cloudflare Tunnel | free | automatic | no public port wanted | enable WebSocket |
| China cloud (Tencent/Alibaba) + filed domain | ~¥50-100/mo | Caddy/Nginx | official China launch | requires ICP filing, 1-3 week lead time |

## Fly.io

```bash
# 1. Install flyctl and sign in (needs a Fly account)
curl -L https://fly.io/install.sh | sh
fly auth login

# 2. Edit infra/relay/fly.toml: app name (database lives on a /data volume)
# 3. Deploy from the repo root
fly launch --no-deploy --name riffpad-relay --dockerfile infra/relay/Dockerfile
fly deploy

# 4. Get the public URL (automatic HTTPS)
fly open
```

After deploying, register and sign in on the relay web UI (https://…); configure the daemon:

```bash
export RIFFPAD_RELAY_URL=wss://riffpad-relay.fly.dev
export RIFFPAD_RELAY_USER=<your-username>
export RIFFPAD_RELAY_PASSWORD=<your-password>
```

Or run `riffpad relay login --url wss://riffpad-relay.fly.dev --username <your-username>`.
On first start the daemon logs in and registers the host automatically (hostId + hostSecret are saved to `~/.config/riffpad/config.json`). `riffpad pair` returns the relay page URL (https://…/?pair=CODE); open it on a signed-in phone and enter the code to pair.

## VPS + Caddy

```bash
# On the server
useradd -r -m riffpad
cp riffpad-relay.service /etc/systemd/system/
cat > /etc/riffpad-relay.env <<EOF
RELAY_PORT=9090
RELAY_LISTEN=127.0.0.1
RELAY_DATA_DIR=/var/lib/riffpad-relay
EOF
mkdir -p /var/lib/riffpad-relay && chown riffpad:riffpad /var/lib/riffpad-relay
systemctl daemon-reload && systemctl enable --now riffpad-relay

# nginx reverse proxy (after pointing api.riffpad.ai DNS at this host)
cp nginx-api-riffpad-ai.conf /etc/nginx/sites-available/api-riffpad-ai
ln -sf /etc/nginx/sites-available/api-riffpad-ai /etc/nginx/sites-enabled/api-riffpad-ai
nginx -t && systemctl reload nginx
# then certbot --nginx -d api.riffpad.ai to issue HTTPS

# Caddy (issues a cert automatically once DNS resolves to the server)
apt install caddy   # or run caddy in docker
cp Caddyfile /etc/caddy/Caddyfile   # replace relay.example.com with your domain
systemctl reload caddy
```

## Docker Compose (relay + Postgres bundle)

Recommended for production: keep nginx/certbot on the host, containerize relay and Postgres with compose. Prepare a secrets file (mode 600, never committed):

```bash
install -m 600 /dev/null /opt/riffpad/.env
# Fill in POSTGRES_USER / POSTGRES_PASSWORD / POSTGRES_DB / GITHUB_CLIENT_ID / GITHUB_CLIENT_SECRET
$EDITOR /opt/riffpad/.env

cd /path/to/riffpad   # repo root (compose build context)
docker compose --env-file /opt/riffpad/.env -f infra/docker-compose.yml up -d --build
```

The relay connects to Postgres via `DATABASE_URL` and creates its schema (AutoMigrate), listening only on `127.0.0.1:9090` behind the host nginx (api.riffpad.ai / app.riffpad.ai). To go back to SQLite, just drop the `DATABASE_URL` variable.

### Migrate from SQLite to Postgres

Start Postgres only first (the relay isn't switched yet, so the running service is unaffected):

```bash
docker compose --env-file /opt/riffpad/.env -f infra/docker-compose.yml up -d postgres
```

Then stop the old relay (to avoid writes during migration) and copy the data with the migration tool:

```bash
sudo systemctl stop riffpad-relay
go run ./apps/relay/cmd/migrate-sqlite \
  -sqlite /var/lib/riffpad-relay/relay.db \
  -postgres 'postgres://<user>:<password>@127.0.0.1:5432/riffpad?sslmode=disable'
```

The tool copies users / oauth_accounts / auth_tokens / host_records / devices / session_meta table by table and refuses if the target tables are non-empty (`--force` overrides — use with care). Once done, bring up compose:

```bash
docker compose --env-file /opt/riffpad/.env -f infra/docker-compose.yml up -d
```

After confirming `/api/status` is healthy and the login/pair/approve flow works, retire the old systemd service:

```bash
sudo systemctl disable --now riffpad-relay
```

## Same-WiFi real-device testing (zero deployment)

The relay listens on all interfaces by default (`:9090`). With the computer and phone on the same WiFi:

1. Run `riffpad pair` on the computer and note the 6-character code
2. Open `http://<computer-LAN-IP>:9090/` in the phone browser, register/sign in
3. Enter the code to pair; you can now see the computer's claude sessions and approve them

> Note: this has no TLS — trusted-LAN testing only.

## Security notes

- Passwords are bcrypt-hashed; login tokens expire after 30 days and are revoked on logout.
- After first registration the daemon connects with its own hostSecret and no longer shares the password.
- The relay data directory (SQLite) must be persisted; mount a dedicated volume in production. SQLite is fine for early single-instance use; switch to Postgres (`DATABASE_URL`) when scaling to multiple instances or wanting managed backups.
- The relay is zero-knowledge: it forwards encrypted envelopes and stores no content; metadata (devices/sessions) is visible, so for public deployments connect Postgres and add auditing early.
- Multiple production instances need shared session routing (Redis pub/sub or sticky sessions); not needed at the single-instance stage.
