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
curl -fsSL "$URL" -o "$tmp/$ASSET"

if [ -n "$SUMS_URL" ]; then
  echo "riffpad: verifying checksum"
  curl -fsSL "$SUMS_URL" -o "$tmp/sha256sums.txt"
  if command -v sha256sum >/dev/null 2>&1; then
    (cd "$tmp" && grep " $ASSET$" sha256sums.txt | sha256sum -c - >/dev/null)
  elif command -v shasum >/dev/null 2>&1; then
    (cd "$tmp" && grep " $ASSET$" sha256sums.txt | shasum -a 256 -c - >/dev/null)
  else
    echo "riffpad: warning: no sha256sum/shasum found, skipping verification" >&2
  fi
fi

chmod 0755 "$tmp/$ASSET"
mkdir -p "$PREFIX"
install -m 0755 "$tmp/$ASSET" "$PREFIX/riffpad"

echo "riffpad: installed $PREFIX/riffpad ($OS/$ARCH, $VERSION)"
case ":${PATH:-}:" in
  *":$PREFIX:"*) ;;
  *) echo "riffpad: note: $PREFIX is not on PATH — add it or run: export PATH=\"$PREFIX:\$PATH\"" ;;
esac
