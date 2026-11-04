#!/usr/bin/env bash
set -Eeuo pipefail
umask 077

ROLE="${1:-P}"
if [[ "$ROLE" != "P" ]]; then
  echo "Usage: sudo $0 P"
  exit 2
fi

STAMP="$(date -u +%Y%m%d-%H%M%S)"
OUT="/tmp/netductor-diag-P-${STAMP}.txt"
exec > >(tee "$OUT") 2>&1

section() { printf '\n===== %s =====\n' "$1"; }

redact() {
  sed -E \
    -e 's#(https?://[^? ]+)\?[^ ]+#\1?[REDACTED]#g' \
    -e 's/(Bearer )[A-Za-z0-9._~+\/-]+/\1[REDACTED]/Ig' \
    -e 's/(token|secret|password|passwd|api[_-]?key)([=: ]+)[^ ,;]+/\1\2[REDACTED]/Ig' \
    -e 's/[A-Fa-f0-9]{48,}/[LONG_HEX_REDACTED]/g'
}

section "IDENTITY / TIME"
date -Is
hostname
id

section "OS / ARCH"
uname -a
if [[ -r /etc/os-release ]]; then . /etc/os-release; printf '%s %s\n' "${PRETTY_NAME:-unknown}" "${VERSION_ID:-}"; fi

section "BINARIES / VERSIONS"
for bin in netductor netductor-agent; do
  if command -v "$bin" >/dev/null 2>&1; then
    path="$(command -v "$bin")"
    printf '%s: %s\n' "$bin" "$path"
    "$bin" version 2>&1 | head -n 8 || true
    stat -c 'file=%n size=%s mtime=%y mode=%a owner=%U:%G' "$path" 2>/dev/null || true
    sha256sum "$path" 2>/dev/null || true
  else
    printf '%s: NOT FOUND\n' "$bin"
  fi
done

section "RELEVANT SYSTEMD UNITS"
systemctl list-units --type=service --all --no-legend 2>/dev/null |
  grep -Ei 'netductor|telegram|notify|backup' | head -n 80 || true
for unit in netductor.service netductor-agent.service; do
  if systemctl cat "$unit" >/dev/null 2>&1; then
    printf '\n--- %s ---\n' "$unit"
    systemctl is-active "$unit" 2>/dev/null || true
    systemctl is-enabled "$unit" 2>/dev/null || true
    systemctl show "$unit" \
      -p ActiveState -p SubState -p Result -p ExecMainCode -p ExecMainStatus \
      -p ActiveEnterTimestamp -p InactiveEnterTimestamp 2>/dev/null || true
  fi
done

section "STACK APPLY SYSTEMD UNIT / LOG"
systemctl show netductor-stack-apply.service \
  -p LoadState -p ActiveState -p SubState -p Result -p ExecMainCode -p ExecMainStatus \
  -p ExecMainStartTimestamp -p ExecMainExitTimestamp 2>/dev/null || true
if [[ -r /var/log/netductor-stack-apply.log ]]; then
  tail -n 240 /var/log/netductor-stack-apply.log | redact || true
else
  echo "/var/log/netductor-stack-apply.log: NOT FOUND"
fi

section "API / TELEGRAM JOURNALS (LAST 48H)"
for unit in netductor-api.service netductor-telegram-bot.service netductor-stack-apply.service; do
  echo "--- $unit ---"
  journalctl -u "$unit" --since "48 hours ago" --no-pager -o short-iso 2>/dev/null |
    grep -Ei 'stack|apply|upgrade|update|release|version|sha256|download|tls|handshake|telegram|notify|notification|alert|send|webhook|error|failed|timeout|scheduled|DONE' |
    tail -n 160 | redact || true
done

section "UPDATE / STACK / NOTIFICATION JOURNAL (LAST 48H)"
for unit in netductor.service netductor-agent.service; do
  journalctl -u "$unit" --since "48 hours ago" --no-pager -o short-iso 2>/dev/null || true
done |
  grep -Ei 'stack|apply|upgrade|update|release|version|sha256|download|tls|handshake|telegram|notify|notification|alert|send|webhook|error|failed|timeout|DONE' |
  tail -n 220 | redact || true

section "GITHUB RELEASE CONNECTIVITY (NO URL / TOKEN OUTPUT)"
URL="https://github.com/PavelNeyman/netductor/releases/download/v0.9.656/SHA256SUMS"
if command -v curl >/dev/null 2>&1; then
  curl -sS -L -o /dev/null --max-time 25 \
    -w 'HTTP=%{http_code} DNS=%{time_namelookup}s TCP=%{time_connect}s TLS=%{time_appconnect}s TOTAL=%{time_total}s\n' \
    "$URL" 2>&1 || true
else
  echo "curl: NOT FOUND"
fi

section "DISK / CLOCK"
df -h / /var 2>/dev/null || true
timedatectl status 2>/dev/null | head -n 12 || true

section "END"
echo "Report: $OUT"
echo "Review the report before sharing; redact sensitive hostnames, addresses, or identifiers."
