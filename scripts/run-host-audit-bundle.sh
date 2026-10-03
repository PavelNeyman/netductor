#!/usr/bin/env bash
# One-shot host audit for netductor VPS (primary or secondary).
# Downloads latest scripts from GitHub main, runs read-only collection,
# writes text + optional tar.gz you can hand to an operator / AI.
#
# Usage (on the VPS as root):
#   curl -fsSL https://raw.githubusercontent.com/PavelNeyman/netductor/main/scripts/run-host-audit-bundle.sh | bash
# Or pin a ref:
#   NETDUCTOR_AUDIT_REF=v0.9.196 bash run-host-audit-bundle.sh
#
# Env:
#   NETDUCTOR_AUDIT_REF   git ref (default: main)
#   NETDUCTOR_AUDIT_OUT   output directory (default: /tmp/nd-host-audit-<host>-<utc>)
#   NETDUCTOR_AUDIT_NO_TAR=1  skip tarball
set -euo pipefail

REF="${NETDUCTOR_AUDIT_REF:-main}"
BASE_URL="https://raw.githubusercontent.com/PavelNeyman/netductor/${REF}/scripts"
HOST="$(hostname -s 2>/dev/null || hostname)"
UTC="$(date -u +%Y%m%dT%H%MZ)"
OUT="${NETDUCTOR_AUDIT_OUT:-/tmp/nd-host-audit-${HOST}-${UTC}}"
WORKDIR="${OUT}/.scripts"

mkdir -p "$WORKDIR" "$OUT"
chmod 700 "$OUT" 2>/dev/null || true

echo "==> netductor host-audit bundle"
echo "    ref=$REF host=$HOST out=$OUT"
echo "    downloading scripts…"

fetch() {
  local name="$1"
  local dest="$WORKDIR/$name"
  if command -v curl >/dev/null 2>&1; then
    curl -fsSL "${BASE_URL}/${name}" -o "$dest"
  elif command -v wget >/dev/null 2>&1; then
    wget -qO "$dest" "${BASE_URL}/${name}"
  else
    echo "ERROR: need curl or wget" >&2
    exit 1
  fi
  chmod +x "$dest" || true
}

fetch collect-host-audit.sh
fetch audit-hoster-agents.sh

# Prefer local repo scripts if this file is run from a checkout
HERE="$(cd "$(dirname "${BASH_SOURCE[0]:-$0}")" 2>/dev/null && pwd || true)"
if [[ -n "$HERE" && -f "$HERE/collect-host-audit.sh" ]]; then
  cp -f "$HERE/collect-host-audit.sh" "$WORKDIR/" 2>/dev/null || true
  cp -f "$HERE/audit-hoster-agents.sh" "$WORKDIR/" 2>/dev/null || true
fi

echo "    running collect-host-audit (read-only)…"
# shellcheck disable=SC2094
bash "$WORKDIR/collect-host-audit.sh" >"$OUT/host-audit.txt" 2>"$OUT/host-audit.stderr" || true

# Extra focused dumps (also read-only) for AI / offline review
{
  echo "=== extra: processes (top CPU) ==="
  ps aux --sort=-%cpu 2>/dev/null | head -40 || true
  echo
  echo "=== extra: processes matching denylist names ==="
  ps aux 2>/dev/null | grep -iE 'zabbix|telegraf|netdata|node_exporter|salt|puppet|chef|datadog|newrelic|elastic|nrpe|collectd|avahi|meshagent|tactical|anydesk|teamviewer|otel|cloudwatch|wazuh|ossec|snmpd' | grep -v grep || echo "(none)"
  echo
  echo "=== extra: established non-local TCP (sample) ==="
  ss -tpn state established 2>/dev/null | head -80 || true
  echo
  echo "=== extra: failed systemd units ==="
  systemctl --failed --no-pager 2>/dev/null || true
  echo
  echo "=== extra: last 30 auth log lines ==="
  if [[ -f /var/log/auth.log ]]; then
    tail -30 /var/log/auth.log 2>/dev/null || true
  elif [[ -f /var/log/secure ]]; then
    tail -30 /var/log/secure 2>/dev/null || true
  else
    journalctl -u ssh -u sshd -n 30 --no-pager 2>/dev/null || true
  fi
} >"$OUT/extra-signals.txt" 2>/dev/null || true

# Standalone denylist report
bash "$WORKDIR/audit-hoster-agents.sh" >"$OUT/hoster-agents.txt" 2>"$OUT/hoster-agents.stderr" || true

# Manifest for the reviewer
{
  echo "netductor_host_audit_bundle=1"
  echo "ref=$REF"
  echo "host=$HOST"
  echo "utc=$UTC"
  echo "uname=$(uname -a 2>/dev/null || true)"
  echo "files:"
  ls -la "$OUT" | sed 's/^/  /'
} >"$OUT/MANIFEST.txt"

TAR=""
if [[ "${NETDUCTOR_AUDIT_NO_TAR:-0}" != "1" ]]; then
  TAR="${OUT}.tar.gz"
  tar -C "$(dirname "$OUT")" -czf "$TAR" "$(basename "$OUT")" 2>/dev/null || TAR=""
fi

echo
echo "==> DONE"
echo "    text:  $OUT/host-audit.txt"
echo "    extra: $OUT/extra-signals.txt"
echo "    agents:$OUT/hoster-agents.txt"
if [[ -n "$TAR" && -f "$TAR" ]]; then
  echo "    archive: $TAR"
  echo
  echo "Copy the archive off the box, e.g.:"
  echo "  scp root@HOST:$TAR ."
else
  echo "    (no tar; directory is $OUT)"
fi
echo
echo "Hand $TAR (or the directory) to the operator / AI for review."
exit 0
