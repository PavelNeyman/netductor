#!/usr/bin/env bash
# Verify dist/ has the full required release set, correct versions, and SHA256SUMS.
set -euo pipefail
ROOT="$(cd "$(dirname "$0")/.." && pwd)"
cd "$ROOT"
VER="${VER:-$(tr -d ' \n' < VERSION)}"
VER="${VER#v}"
if [[ ! "$VER" =~ ^[0-9]+\.[0-9]+\.[0-9]+$ ]]; then
  echo "FAIL: invalid VER='$VER'" >&2
  exit 1
fi
if [ ! -d dist ]; then
  echo "FAIL: dist/ missing — build first" >&2
  exit 1
fi

# Required assets for a full node release (matches scripts/release.sh matrix).
REQUIRED=(
  netductor-linux-amd64
  netductor-linux-arm64
  netductor-tg-linux-amd64
  netductor-tg-linux-arm64
  netductor-agent-linux-amd64
  netductor-agent-linux-arm64
  netductor-agent-linux-arm
  netductor-agent-linux-mipsle
  netductor-agent-linux-riscv64
  netductor-op-linux-amd64
  netductor-op-darwin-arm64
  netductor-op-darwin-amd64
  SHA256SUMS
)

# Minimal set when ND_RELEASE_MINIMAL=1 (amd64-only smoke).
if [ "${ND_RELEASE_MINIMAL:-}" = "1" ]; then
  REQUIRED=(
    netductor-linux-amd64
    netductor-tg-linux-amd64
    netductor-agent-linux-amd64
    netductor-op-linux-amd64
    SHA256SUMS
  )
fi

fail=0
for name in "${REQUIRED[@]}"; do
  path="dist/$name"
  if [ ! -e "$path" ]; then
    echo "FAIL: missing required asset: $name" >&2
    fail=1
    continue
  fi
  if [ "$name" = "SHA256SUMS" ]; then
    continue
  fi
  if [ ! -f "$path" ]; then
    echo "FAIL: $name is not a regular file" >&2
    fail=1
    continue
  fi
  if [ ! -x "$path" ]; then
    echo "FAIL: $name is not executable" >&2
    fail=1
    continue
  fi
  if out=$("$path" version 2>/dev/null | head -1); then
    if echo "$out" | grep -qF "$VER"; then
      echo "OK  $name => $out"
      continue
    fi
    echo "FAIL $name version output: $out (want $VER)" >&2
    fail=1
    continue
  fi
  if command -v strings >/dev/null && strings "$path" | grep -qF "$VER"; then
    echo "OK  $name (strings has $VER)"
    continue
  fi
  echo "FAIL $name: cannot verify version $VER" >&2
  fail=1
done

if [ -f dist/SHA256SUMS ]; then
  echo "checking dist/SHA256SUMS..."
  # Every REQUIRED binary (except SHA256SUMS) must appear in the manifest.
  for name in "${REQUIRED[@]}"; do
    [ "$name" = "SHA256SUMS" ] && continue
    if ! grep -qE "[[:space:]]${name}\$" dist/SHA256SUMS; then
      echo "FAIL: $name missing from SHA256SUMS" >&2
      fail=1
    fi
  done
  (cd dist && sha256sum -c SHA256SUMS) || fail=1
else
  echo "FAIL: dist/SHA256SUMS missing" >&2
  fail=1
fi

[ "$fail" -eq 0 ] && echo "verify-release-binaries: OK ($VER)" || exit 1
