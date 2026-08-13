#!/bin/sh
#
# Riffpad self-hosted relay installer — run your own relay on a VPS (or any
# Docker host) instead of relying on app.riffpad.ai. One command, no clone,
# no build: it pulls the public image ghcr.io/riffpad/relay and brings up a
# single relay container (embedded SQLite by default).
#
# The copy served at https://riffpad.ai/selfhost.sh is generated from this
# file by scripts/sync-installers.sh before the landing dev/build runs.
#
# Usage:
#   curl -fsSL https://riffpad.ai/selfhost.sh | sh
#
# With automatic TLS on a public domain (Caddy gets the certificate):
#   curl -fsSL https://riffpad.ai/selfhost.sh | sh -s -- --domain relay.example.com
#
# Options (env equivalents in parens):
#   --domain HOST    public domain; enables a Caddy sidecar with auto-HTTPS
#                    (RIFFPAD_DOMAIN)
#   --port N         host port for HTTP mode, default 9090 (RIFFPAD_RELAY_PORT)
#   --tag TAG        image tag, default latest (RIFFPAD_RELAY_TAG)
#   --email ADDR     ACME account email for Caddy (RIFFPAD_RELAY_EMAIL)
#   -h, --help       show this help
#
# Files are written to ~/.riffpad-relay (RIFFPAD_RELAY_DIR): docker-compose.yml,
# .env, and a ./data volume with the encrypted session/event store. Manage the
# relay with `docker compose` from that directory (logs / ps / restart / down).
set -eu

REPO_IMAGE="ghcr.io/riffpad/relay"

usage() {
  sed -n '2,/^set -eu$/p' "$0" | sed '/^set -eu$/d' | sed 's/^# \{0,1\}//'
}

# Best-effort LAN IPv4 of this host, for printing a reachable address in HTTP
# mode. Empty if nothing can be determined (the caller falls back to localhost).
detect_lan_ip() {
  case "$(uname -s)" in
    Darwin)
      ipconfig getifaddr en0 2>/dev/null || ipconfig getifaddr en1 2>/dev/null
      ;;
    *)
      hostname -I 2>/dev/null | awk '{print $1}' || true
      ;;
  esac
}

DOMAIN="${RIFFPAD_DOMAIN:-}"
PORT="${RIFFPAD_RELAY_PORT:-9090}"
TAG="${RIFFPAD_RELAY_TAG:-latest}"
EMAIL="${RIFFPAD_RELAY_EMAIL:-}"
DIR="${RIFFPAD_RELAY_DIR:-$HOME/.riffpad-relay}"

while [ $# -gt 0 ]; do
  case "$1" in
    --domain)   DOMAIN="$2"; shift 2 ;;
    --domain=*) DOMAIN="${1#*=}"; shift ;;
    --port)     PORT="$2"; shift 2 ;;
    --port=*)   PORT="${1#*=}"; shift ;;
    --tag)      TAG="$2"; shift 2 ;;
    --tag=*)    TAG="${1#*=}"; shift ;;
    --email)    EMAIL="$2"; shift 2 ;;
    --email=*)  EMAIL="${1#*=}"; shift ;;
    -h|--help)  usage; exit 0 ;;
    *) echo "riffpad: unknown option: $1 (try --help)" >&2; exit 1 ;;
  esac
done

# --- prerequisites --------------------------------------------------------
if ! command -v docker >/dev/null 2>&1; then
  echo "riffpad: docker not found. Install Docker first:" >&2
  echo "  https://docs.docker.com/engine/install/" >&2
  exit 1
fi
if docker compose version >/dev/null 2>&1; then
  DC="docker compose"
elif command -v docker-compose >/dev/null 2>&1; then
  DC="docker-compose"
else
  echo "riffpad: 'docker compose' (Compose v2) not found." >&2
  echo "  Update Docker Desktop / engine, or: https://docs.docker.com/compose/install/" >&2
  exit 1
fi

IMAGE="${RIFFPAD_RELAY_IMAGE:-$REPO_IMAGE}"
echo "riffpad: target image $IMAGE:$TAG"
[ -n "$DOMAIN" ] && echo "riffpad: TLS domain $DOMAIN (Caddy auto-HTTPS)"

# --- install directory ----------------------------------------------------
mkdir -p "$DIR/data"
cd "$DIR"
echo "riffpad: install dir $DIR"

# A starter .env the user can edit later (GitHub OAuth, app URL, ...). The
# relay reads it via `env_file` in the compose file. SQLite is the default —
# leave DATABASE_URL unset and the relay stores everything in /data/relay.db.
if [ ! -f .env ]; then
  cat > .env <<'EOF'
# Riffpad self-hosted relay — runtime config. Edit, then restart with:
#   docker compose up -d
#
# Database: embedded SQLite at /data/relay.db by default. To use Postgres
# instead, set DATABASE_URL and add a postgres service (see the docs page).
# DATABASE_URL=postgres://user:pass@postgres:5432/riffpad?sslmode=disable
#
# GitHub sign-in (optional):
# GITHUB_CLIENT_ID=
# GITHUB_CLIENT_SECRET=
#
# Canonical address users reach the relay at (optional):
# RIFFPAD_APP_URL=https://relay.example.com
EOF
fi

# --- compose file ---------------------------------------------------------
# Regenerated each run, so editing options (tag, port, domain) just means
# re-running the installer. The data volume is preserved.
if [ -n "$DOMAIN" ]; then
  # HTTPS via Caddy: relay is internal-only, Caddy terminates TLS on 80/443.
  if [ -n "$EMAIL" ]; then
    cat > Caddyfile <<EOF
{
  email "$EMAIL"
}
$DOMAIN {
  reverse_proxy relay:9090
}
EOF
  else
    cat > Caddyfile <<EOF
$DOMAIN {
  reverse_proxy relay:9090
}
EOF
  fi
  cat > docker-compose.yml <<EOF
services:
  relay:
    image: $IMAGE:$TAG
    restart: unless-stopped
    expose:
      - "9090"
    env_file: [.env]
    volumes:
      - ./data:/data
    environment:
      RELAY_LISTEN: "0.0.0.0"
      RELAY_PORT: "9090"
      RELAY_DATA_DIR: /data
    healthcheck:
      test: ["CMD", "wget", "-q", "-O", "/dev/null", "http://127.0.0.1:9090/api/status"]
      interval: 10s
      timeout: 3s
      retries: 5
  caddy:
    image: caddy:2-alpine
    restart: unless-stopped
    ports:
      - "80:80"
      - "443:443"
    volumes:
      - ./Caddyfile:/etc/caddy/Caddyfile:ro
      - caddy_data:/data
      - caddy_config:/config
    depends_on:
      - relay
volumes:
  caddy_data:
  caddy_config:
EOF
else
  # Plain HTTP: relay published on the host port for LAN / direct access.
  # Put it behind your own nginx/Caddy for public TLS (see the docs page).
  cat > docker-compose.yml <<EOF
services:
  relay:
    image: $IMAGE:$TAG
    restart: unless-stopped
    ports:
      - "$PORT:9090"
    env_file: [.env]
    volumes:
      - ./data:/data
    environment:
      RELAY_LISTEN: "0.0.0.0"
      RELAY_PORT: "9090"
      RELAY_DATA_DIR: /data
    healthcheck:
      test: ["CMD", "wget", "-q", "-O", "/dev/null", "http://127.0.0.1:9090/api/status"]
      interval: 10s
      timeout: 3s
      retries: 5
EOF
fi

# --- pull & start ---------------------------------------------------------
echo "riffpad: pulling image (this can take a minute on first run)"
if ! $DC pull 2>/dev/null; then
  echo "riffpad: could not pull $IMAGE:$TAG." >&2
  echo "  If the image was just published, it may still be propagating;" >&2
  echo "  otherwise the org may not have made the package public." >&2
  echo "  Check: https://github.com/riffpad/riffpad/pkgs/container/relay" >&2
  exit 1
fi

echo "riffpad: starting relay"
$DC up -d

# --- wait for health ------------------------------------------------------
CID="$($DC ps -q relay)"
STATUS=""
i=0
while [ "$i" -lt 40 ]; do
  STATUS="$(docker inspect -f '{{.State.Health.Status}}' "$CID" 2>/dev/null || echo "")"
  [ "$STATUS" = "healthy" ] && break
  i=$((i + 1))
  sleep 1
done

if [ "$STATUS" != "healthy" ]; then
  echo "riffpad: relay did not become healthy in time. Recent logs:" >&2
  $DC logs --tail=20 relay >&2 || true
  echo "riffpad: inspect with: cd \"$DIR\" && $DC logs -f relay" >&2
  exit 1
fi

# --- resolve the address users will connect to ---------------------------
if [ -n "$DOMAIN" ]; then
  SCHEME="https"
  URL="https://$DOMAIN"
  WS="wss://$DOMAIN"
else
  SCHEME="http"
  LAN_IP="$(detect_lan_ip 2>/dev/null || true)"
  HOST_PART="${LAN_IP:-localhost}"
  URL="http://$HOST_PART:$PORT"
  WS="ws://$HOST_PART:$PORT"
fi

echo ""
echo "riffpad: relay is up at $URL"
echo ""
echo "Next:"
echo "  1. Open $URL in a browser and register an account."
echo "  2. On each computer running a daemon, point it at this relay and log in:"
echo "       export RIFFPAD_RELAY_URL=$WS"
echo "       riffpad relay login --url $WS --username <your-username>"
echo "  3. Run 'riffpad pair' there; open the printed URL on your phone to pair."
echo ""
echo "Manage the relay from: $DIR"
echo "  $DC logs -f relay    # follow logs"
echo "  $DC restart relay    # restart"
echo "  $DC down             # stop"
echo ""
if [ -z "$DOMAIN" ]; then
  echo "Note: HTTP mode is fine for trusted LAN / testing. For public access"
  echo "with TLS, re-run with --domain your.domain, or put the relay behind"
  echo "your own reverse proxy. See: https://riffpad.ai/docs/guide/self-host"
else
  echo "TLS is provisioned automatically by Caddy; the first request may take"
  echo "a few seconds while the certificate is issued. See:"
  echo "  https://riffpad.ai/docs/guide/self-host"
fi
