#!/bin/sh
#
# Copies the canonical installer scripts (scripts/install.sh, install.ps1)
# into apps/landing/public so https://riffpad.ai/install.sh and
# /install.ps1 are served from a single source of truth. Runs automatically
# before the landing dev/build commands.
set -eu

root="$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)"
cp "$root/scripts/install.sh" "$root/apps/landing/public/install.sh"
cp "$root/scripts/install.ps1" "$root/apps/landing/public/install.ps1"
