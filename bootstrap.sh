#!/usr/bin/env bash
# Netductor bootstrap for Debian VPS (root)
set -euo pipefail
[[ ${EUID:-} -eq 0 ]] || { echo "run as root"; exit 1; }
arch=$(uname -m)
case "$arch" in x86_64) a=amd64;; aarch64) a=arm64;; *) echo "unsupported arch $arch"; exit 1;; esac
VER="${NETDUCTOR_VERSION:-0.8.12}"
curl -fsSL -o /usr/local/bin/netductor \
  "https://github.com/PavelNeyman/netductor/releases/download/v${VER}/netductor-linux-${a}"
chmod 755 /usr/local/bin/netductor
netductor version
echo "Next: netductor tui --mode vps   OR   netductor install"
