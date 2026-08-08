#!/bin/sh
#
# Riffpad installer — downloads the single riffpad binary from GitHub
# Releases and installs it to ~/.local/bin (override with RIFFPAD_PREFIX).
#
# The copy served at https://riffpad.ai/install.sh is generated from this
# file by scripts/sync-installers.sh before the landing dev/build runs.
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
trap 'rm -rf "$tmp"' EXIT INT TERM

echo "riffpad: downloading $URL"
if ! curl -fsSL "$URL" -o "$tmp/$ASSET"; then
  echo "riffpad: download failed" >&2
  exit 1
fi

if [ -n "$SUMS_URL" ]; then
  echo "riffpad: verifying checksum"
  if ! curl -fsSL "$SUMS_URL" -o "$tmp/sha256sums.txt"; then
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
chmod 0755 "$tmp/$ASSET"
mkdir -p "$PREFIX"
install -m 0755 "$tmp/$ASSET" "$PREFIX/riffpad"

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
