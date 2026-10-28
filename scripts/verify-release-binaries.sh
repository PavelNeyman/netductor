#!/usr/bin/env bash
# Verify dist/* binaries embed the expected version (VERSION file or $VER).
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
fail=0
shopt -s nullglob
bins=(dist/netductor-linux-* dist/netductor-tg-linux-* dist/netductor-agent-linux-* dist/netductor-op-*)
if [ ${#bins[@]} -eq 0 ]; then
  echo "FAIL: no release binaries in dist/" >&2
  exit 1
fi
for bin in "${bins[@]}"; do
  [ -f "$bin" ] || continue
  [ -x "$bin" ] || continue
  # Prefer runtime version when the binary can execute on this host.
  if out=$("$bin" version 2>/dev/null | head -1); then
    if echo "$out" | grep -qF "$VER"; then
      echo "OK  $bin => $out"
      continue
    fi
    echo "FAIL $bin version output: $out (want $VER)" >&2
    fail=1
    continue
  fi
  if command -v strings >/dev/null; then
    if strings "$bin" | grep -qF "$VER"; then
      echo "OK  $bin (strings has $VER)"
      continue
    fi
  fi
  echo "FAIL $bin: cannot verify version $VER" >&2
  fail=1
done
if [ -f dist/SHA256SUMS ]; then
  echo "checking dist/SHA256SUMS..."
  (cd dist && sha256sum -c SHA256SUMS) || fail=1
else
  echo "WARN: dist/SHA256SUMS missing"
fi
[ "$fail" -eq 0 ] && echo "verify-release-binaries: OK ($VER)" || exit 1
