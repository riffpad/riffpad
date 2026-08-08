#!/bin/sh
#
# Riffpad installer — downloads the single riffpad binary from GitHub
# Releases and installs it to ~/.local/bin (override with RIFFPAD_PREFIX).
#
# The copy served at https://riffpad.ai/install.sh lives in
# apps/landing/public/install.sh — keep it in sync when editing this file.
#
# Usage:
#   curl -fsSL https://raw.githubusercontent.com/riffpad/riffpad/main/scripts/install.sh | sh
#
# Overrides:
#   RIFFPAD_VERSION   release tag to install (default: latest)
#   RIFFPAD_PREFIX    install directory (default: ~/.local/bin)
#   RIFFPAD_DOWNLOAD_URL  direct binary URL (for testing/mirrors)
#   RIFFPAD_SUMS_URL      checksums URL when RIFFPAD_DOWNLOAD_URL is set
#   RIFFPAD_NO_AUTOSTART  set to 1 to skip registering daemon autostart
set -eu

REPO="riffpad/riffpad"
VERSION="${RIFFPAD_VERSION:-latest}"
PREFIX="${RIFFPAD_PREFIX:-$HOME/.local/bin}"

case "$(uname -s)" in
  Linux) OS="linux" ;;
  Darwin) OS="darwin" ;;
  *)
    echo "riffpad: unsupported OS: $(uname -s)" >&2
    exit 1
    ;;
esac

case "$(uname -m)" in
  x86_64|amd64) ARCH="amd64" ;;
  aarch64|arm64) ARCH="arm64" ;;
  *)
    echo "riffpad: unsupported architecture: $(uname -m)" >&2
    exit 1
    ;;
esac

ASSET="riffpad-${OS}-${ARCH}"
if [ -n "${RIFFPAD_DOWNLOAD_URL:-}" ]; then
  URL="$RIFFPAD_DOWNLOAD_URL"
  SUMS_URL="${RIFFPAD_SUMS_URL:-}"
elif [ "$VERSION" = "latest" ]; then
  URL="https://github.com/$REPO/releases/latest/download/$ASSET"
  SUMS_URL="https://github.com/$REPO/releases/latest/download/sha256sums.txt"
else
  URL="https://github.com/$REPO/releases/download/$VERSION/$ASSET"
  SUMS_URL="https://github.com/$REPO/releases/download/$VERSION/sha256sums.txt"
fi

tmp="$(mktemp -d)"

# --- 3x3 dot-matrix marquee spinner --------------------------------
# One dot lights at a time, sweeping left-to-right / top-to-bottom across
# the grid (9 frames, ~0.12s each), mirroring the DotMatrix indicator in
# the web client.
FRAME0=$(printf '●··\n···\n···')
FRAME1=$(printf '·●·\n···\n···')
FRAME2=$(printf '··●\n···\n···')
FRAME3=$(printf '···\n●··\n···')
FRAME4=$(printf '···\n·●·\n···')
FRAME5=$(printf '···\n··●\n···')
FRAME6=$(printf '···\n···\n●··')
FRAME7=$(printf '···\n···\n·●·')
FRAME8=$(printf '···\n···\n··●')

_spinner_pid=

tick() {
  if command -v perl >/dev/null 2>&1; then
    perl -e 'select undef,undef,undef,0.12'
  elif sleep 0.12 2>/dev/null; then
    :
  else
    sleep 1
  fi
}

spinner_start() {
  i=0
  first=1
  while :; do
    if [ "$first" -eq 1 ]; then
      first=0
    else
      printf '\033[3A' >&2
    fi
    eval "frame=\$FRAME$i"
    printf '\r\033[K%s\n\033[K%s\n\033[K%s\n' "$frame" >&2
    i=$(((i + 1) % 9))
    tick
  done
}

spinner_stop() {
  if [ -n "$_spinner_pid" ]; then
    kill "$_spinner_pid" 2>/dev/null || true
    wait "$_spinner_pid" 2>/dev/null || true
    _spinner_pid=
    if [ -t 2 ]; then
      printf '\033[3A\033[J' >&2
    fi
  fi
}

cleanup() {
  spinner_stop
  rm -rf "$tmp"
}
trap cleanup EXIT INT TERM

# Runs a command with the spinner animating on stderr; returns the command's
# status and always stops the spinner first.
with_spinner() {
  if [ -t 2 ]; then
    spinner_start &
    _spinner_pid=$!
  fi
  "$@" || {
    rc=$?
    spinner_stop
    return $rc
  }
  spinner_stop
  return 0
}

echo "riffpad: downloading $URL"
if ! with_spinner curl -fsSL "$URL" -o "$tmp/$ASSET"; then
  echo "riffpad: download failed" >&2
  exit 1
fi

if [ -n "$SUMS_URL" ]; then
  echo "riffpad: verifying checksum"
  if ! with_spinner curl -fsSL "$SUMS_URL" -o "$tmp/sha256sums.txt"; then
    echo "riffpad: failed to fetch checksums" >&2
    exit 1
  fi
  if command -v sha256sum >/dev/null 2>&1; then
    if ! (cd "$tmp" && grep " $ASSET$" sha256sums.txt | sha256sum -c - >/dev/null); then
      echo "riffpad: checksum mismatch" >&2
      exit 1
    fi
  elif command -v shasum >/dev/null 2>&1; then
    if ! (cd "$tmp" && grep " $ASSET$" sha256sums.txt | shasum -a 256 -c - >/dev/null); then
      echo "riffpad: checksum mismatch" >&2
      exit 1
    fi
  else
    echo "riffpad: warning: no sha256sum/shasum found, skipping verification" >&2
  fi
fi

echo "riffpad: installing"
if ! with_spinner sh -c 'chmod 0755 "$1" && mkdir -p "$2" && install -m 0755 "$1" "$2/riffpad"' sh "$tmp/$ASSET" "$PREFIX"; then
  echo "riffpad: install failed" >&2
  exit 1
fi

if [ -t 1 ]; then
  printf '\033[32m'
fi
cat <<'LOGO'
 ██ ██ ██
       ██
 ██ ██ ██
       ██
 ██ ██ ██
LOGO
if [ -t 1 ]; then
  printf '\033[0m'
fi

echo "riffpad: installed $PREFIX/riffpad ($OS/$ARCH, $VERSION)"
case ":${PATH:-}:" in
  *":$PREFIX:"*) ;;
  *) echo "riffpad: note: $PREFIX is not on PATH — add it or run: export PATH=\"$PREFIX:\$PATH\"" ;;
esac

# Best-effort autostart: register a systemd user service so the daemon
# starts at login and restarts after crashes. This never fails the install —
# if systemd is unavailable (WSL without systemd, containers, macOS), warn
# and let the user run `riffpad setup` later. Opt out with
# RIFFPAD_NO_AUTOSTART=1.
if [ "$(uname -s)" = "Linux" ] &&
  [ -z "${RIFFPAD_NO_AUTOSTART:-}" ] &&
  command -v systemctl >/dev/null 2>&1; then
  if systemctl --user is-enabled riffpad.service >/dev/null 2>&1; then
    echo "riffpad: daemon autostart already enabled"
  else
    # A manually started daemon would conflict with the systemd unit, so
    # stop it first and let the service take over.
    if "$PREFIX/riffpad" status >/dev/null 2>&1; then
      "$PREFIX/riffpad" daemon stop >/dev/null 2>&1 || true
    fi
    echo "riffpad: enabling daemon autostart (systemd user service)"
    if "$PREFIX/riffpad" setup >/dev/null 2>&1; then
      echo "riffpad: daemon autostart enabled"
    else
      echo "riffpad: warning: could not enable daemon autostart (systemd unavailable?)" >&2
      echo "riffpad: run 'riffpad setup' manually when systemd is available" >&2
    fi
  fi
fi
