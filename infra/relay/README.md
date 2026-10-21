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

## Credentials in WebSocket URLs (needs a manual proxy update)

Both WebSocket connections carry a **long-lived credential in the URL query string**:

| connection | URL today | credential | lifetime |
|---|---|---|---|
| daemon → relay | `/ws/host?hostId=…&token=<hostSecret>` | per-host secret | never expires |
| browser → relay | `/ws?device=…&session=…&eph=…&token=<relayToken>` | account token | 30 days |

Boot ordering, tab reloads and WebSocket upgrade failures are all written to the proxy's access log, so with a default config those tokens end up in `access.log`, in logrotate archives, in whatever ships the logs, and in backups. A leaked account token can list sessions, revoke paired devices and fire the kill switch (content stays end-to-end encrypted, so it is availability + metadata, not plaintext); a leaked `hostSecret` can impersonate the host and kick the real daemon.

So the shipped proxy configs **disable request logging for `/ws` and nothing else** (issue #349):

- `nginx-api-riffpad-ai.conf`: `location /ws { access_log off; … }`, plus `limit_req_zone` rules keyed on `$binary_remote_addr` (stricter for `/api/auth/`, looser for the rest of `/api/`).
- `Caddyfile`: `log_skip /ws*` — Caddy keeps logging every other path; it has no built-in rate limiting, so per-IP limits come from nginx config or a plugin (see the comment in the file).

**Apply it yourself — the deploy pipeline does not.** `deploy-relay` on the server only runs `git pull` + `docker compose up -d`; it never touches the host nginx/Caddy config.

`nginx-api-riffpad-ai.conf` is a **template**: an existing `api.riffpad.ai` vhost has usually been rewritten by certbot (`listen 443 ssl`, `ssl_certificate`, HTTP→HTTPS redirect), so copying the template over it deletes HTTPS and takes the site down. **Merge the pieces instead.**

1. Add the two zones **at `http` level** (`/etc/nginx/conf.d/riffpad-limits.conf` is the usual place; they are not allowed inside `server`):

   ```nginx
   limit_req_zone $binary_remote_addr zone=riffpad_auth:10m rate=1r/s;
   limit_req_zone $binary_remote_addr zone=riffpad_api:10m rate=20r/s;
   ```

2. Inside the **existing** `server` block of the api vhost (the one with your certbot `ssl_certificate` lines), add:

   ```nginx
   limit_req_status 429;   # 429, not nginx's default 503

   # credentials ride in the /ws query string (#349): never log them
   location /ws {
       access_log off;
       proxy_pass http://127.0.0.1:9090;
       proxy_http_version 1.1;
       proxy_set_header Upgrade $http_upgrade;
       proxy_set_header Connection "upgrade";
       proxy_set_header Host $host;
       proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
       proxy_set_header X-Forwarded-Proto $scheme;
       proxy_read_timeout 3600s;
       proxy_send_timeout 3600s;
   }

   location /api/auth/ {
       limit_req zone=riffpad_auth burst=5 nodelay;
       proxy_pass http://127.0.0.1:9090;
       proxy_http_version 1.1;
       proxy_set_header Host $host;
       proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
       proxy_set_header X-Forwarded-Proto $scheme;
   }

   location /api/ {
       limit_req zone=riffpad_api burst=40 nodelay;
       proxy_pass http://127.0.0.1:9090;
       proxy_http_version 1.1;
       proxy_set_header Host $host;
       proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
       proxy_set_header X-Forwarded-Proto $scheme;
   }
   ```

   Keep your existing `location /` (or the catch-all proxy) as it is — only these three paths are new. On a fresh machine you can start from the template file wholesale.

3. Validate and reload, then confirm the site still serves HTTPS:

   ```bash
   sudo nginx -t && sudo systemctl reload nginx
   curl -sSI https://api.riffpad.ai/api/status | head -1
   ```

Caddy needs no merge if you own `/etc/caddy/Caddyfile`; just add the two lines to your site block:

```caddyfile
log
log_skip /ws*
```

```bash
sudo caddy validate --config /etc/caddy/Caddyfile && sudo systemctl reload caddy
```

Then check that the request line with the credential is no longer written:

```bash
grep -c 'token=' /var/log/nginx/access.log   # expect 0
tail -n 5 /var/log/nginx/access.log          # /ws lines are absent entirely
```

Also note: the relay no longer trusts `X-Forwarded-For` at all (it was trusted before #349, which made every rate limit bypassable by sending a header). Rate limits are now keyed on usernames, emails, device codes, user IDs and a shared global bucket; per-source floods are the proxy's job. Removing the credential from the URL entirely (header auth for the daemon, short-lived tickets for browsers) is tracked separately in #349.

## Security notes

- Passwords are bcrypt-hashed; login tokens expire after 30 days and are revoked on logout.
- After first registration the daemon connects with its own hostSecret and no longer shares the password.
- The relay data directory (SQLite) must be persisted; mount a dedicated volume in production. SQLite is fine for early single-instance use; switch to Postgres (`DATABASE_URL`) when scaling to multiple instances or wanting managed backups.
- The relay is zero-knowledge: it forwards encrypted envelopes and stores no content; metadata (devices/sessions) is visible, so for public deployments connect Postgres and add auditing early.
- Multiple production instances need shared session routing (Redis pub/sub or sticky sessions); not needed at the single-instance stage.
- WebSocket credentials live in the URL, so the bundled nginx/Caddy configs disable access logging for `/ws` — apply them by hand after upgrading (see the section above). Rate limits are keyed per identity, never per client IP header.
