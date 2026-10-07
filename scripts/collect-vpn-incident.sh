#!/usr/bin/env bash
# Collect ~N hours of VPN / uplink / control-plane logs for offline review.
# Safe-ish: redacts tokens/keys; does not dump full Reality private keys or VPN user UUIDs lists in clear beyond tags.
#
# Usage (on the node, as root):
#   curl -fsSL https://raw.githubusercontent.com/PavelNeyman/netductor/main/scripts/collect-vpn-incident.sh | bash -s -- 90
#   # 90 = minutes of journal history (default 90 ≈ “last hour with margin”)
#
# Or local:
#   sudo bash scripts/collect-vpn-incident.sh 90
#
# Output: /tmp/nd-vpn-incident-<host>-<ts>.tar.gz  (and extracted dir next to it)
set -euo pipefail

MINUTES="${1:-90}"
# allow trailing 'm' or pure number
MINUTES="${MINUTES%m}"
if ! [[ "$MINUTES" =~ ^[0-9]+$ ]] || [[ "$MINUTES" -lt 15 ]] || [[ "$MINUTES" -gt 10080 ]]; then
  echo "usage: $0 [minutes]   (15..10080, default 90)" >&2
  exit 2
fi

TS=$(date -u +%Y%m%dT%H%M%SZ)
HOST=$(hostname -s 2>/dev/null || hostname)
OUTDIR="/tmp/nd-vpn-incident-${HOST}-${TS}"
mkdir -p "$OUTDIR"
cd "$OUTDIR"

SINCE="${MINUTES} min ago"
echo "collect vpn incident host=$HOST ts=$TS since=$SINCE" | tee meta.txt
date -u >> meta.txt
uptime >> meta.txt 2>/dev/null || true

redact() {
  sed -E \
    -e 's/(token|password|secret|passwd|authorization|api_key)[=: ]+[^[:space:]]+/\1=***REDACTED***/Ig' \
    -e 's/Bearer [A-Za-z0-9._-]{8,}/Bearer ***REDACTED***/g' \
    -e 's/-----BEGIN [A-Z0-9 ]+PRIVATE KEY-----([^-]|-)*-----END [A-Z0-9 ]+PRIVATE KEY-----/***PRIVATE KEY REDACTED***/g' \
    -e 's/[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}/***UUID***/g' \
    -e 's/[A-Za-z0-9+\/=]{100,}/***LONG_B64***/g'
}

section() { echo; echo "##### $1 #####"; }

{
  section "versions"
  cat /etc/netductor/VERSION 2>/dev/null || true
  command -v netductor >/dev/null && netductor version 2>/dev/null || true
  uname -a
  cat /etc/os-release 2>/dev/null | head -6

  section "role / baseline"
  cat /etc/netductor/role 2>/dev/null || true
  netductor doctor 2>/dev/null | head -c 16000 || true

  section "units"
  for u in sing-box netductor-api netductor-telegram-bot netductor-redirect \
           netductor-secondary-agent netductor-agent blocky \
           nd-wss-sp-server nd-wss-sp-client nd-wss-ps-server nd-wss-ps-client \
           netductor-stack-watchdog.timer; do
    printf '%s active=%s enabled=%s\n' "$u" \
      "$(systemctl is-active "$u" 2>/dev/null || echo n/a)" \
      "$(systemctl is-enabled "$u" 2>/dev/null || echo n/a)"
  done

  section "load / mem"
  uptime
  free -h 2>/dev/null || true
  cat /proc/loadavg 2>/dev/null || true
  df -h / /var /var/lib 2>/dev/null || true

  section "listen ports (vpn-related)"
  ss -tulnp 2>/dev/null | grep -E ':(443|52222|8443|8445|8787|8789|53)\s' || ss -tulnp 2>/dev/null | head -40

  section "firewall"
  netductor firewall status 2>/dev/null || true
  ufw status verbose 2>/dev/null | head -40 || true

  section "sing-box config summary (no secrets)"
  CONF=""
  for p in /usr/local/etc/sing-box/config.json /etc/sing-box/config.json; do
    [[ -f $p ]] && CONF=$p && break
  done
  if [[ -n $CONF ]]; then
    echo "path=$CONF"
    ls -la "$CONF"
    python3 - "$CONF" <<'PY' 2>/dev/null || echo "parse failed"
import json,sys
d=json.load(open(sys.argv[1]))
print("inbounds:", [(x.get("tag"), x.get("type"), x.get("listen"), x.get("listen_port")) for x in d.get("inbounds",[])])
print("outbounds:", [(x.get("tag"), x.get("type")) for x in d.get("outbounds",[])][:30])
rules=d.get("route",{}).get("rules",[])
print("route_rules_n:", len(rules))
for r in rules[:25]:
    keys=sorted(k for k in r if k not in ("outbound","action"))
    print(" ", {k:r.get(k) for k in keys}, "->", r.get("outbound") or r.get("action"))
print("final:", d.get("route",{}).get("final"))
PY
  else
    echo "no sing-box config found"
  fi

  section "secondary / svc-paths (primary)"
  netductor secondary status 2>/dev/null || true
  netductor svc-paths status 2>/dev/null || true

  section "recent restarts (systemd)"
  systemctl show sing-box -p ActiveEnterTimestamp -p NRestarts -p ExecMainStatus 2>/dev/null || true
  journalctl -u sing-box --since "$SINCE" -o short-iso --no-pager 2>/dev/null \
    | grep -iE 'Started|Stopped|Failed|Main process' | tail -40 || true
} | redact | tee summary.txt >/dev/null

# Full journals (redacted) — separate files for grepping
junits=(sing-box netductor-api netductor-telegram-bot netductor-secondary-agent netductor-agent blocky)
for u in "${junits[@]}"; do
  if systemctl cat "$u" &>/dev/null || systemctl status "$u" &>/dev/null; then
    journalctl -u "$u" --since "$SINCE" -o short-iso --no-pager 2>/dev/null | redact > "journal-${u}.log" || true
  fi
done
# kernel / network noise
journalctl -k --since "$SINCE" -o short-iso --no-pager 2>/dev/null | grep -iE 'oom|kill|net|udp|tcp|link|nf_|conntrack' | redact > journal-kernel-net.log || true

# Compact error index
{
  echo "=== error/reject/mux counters (last ${MINUTES}m) ==="
  for f in journal-*.log; do
    [[ -f $f ]] || continue
    echo "-- $f"
    grep -ciE 'error|fail|reject|timeout|reset|refused|invalid' "$f" 2>/dev/null | xargs -I{} echo "  match_lines={}"
    grep -iE 'reject|remote error|REALITY|invalid connection|mux|uplink|10\.88|deadline|i/o timeout|connection reset' "$f" 2>/dev/null | tail -80 || true
    echo
  done
} | tee error-index.txt >/dev/null

# Optional: last 200 lines of each journal head+tail for quick view
for f in journal-sing-box.log journal-netductor-api.log; do
  if [[ -f $f ]]; then
    { echo "=== head $f ==="; head -50 "$f"; echo "=== tail $f ==="; tail -100 "$f"; } > "${f%.log}-headtail.txt"
  fi
done

tar -C /tmp -czf "${OUTDIR}.tar.gz" "$(basename "$OUTDIR")"
echo
echo "OK bundle: ${OUTDIR}.tar.gz"
echo "size: $(du -h "${OUTDIR}.tar.gz" | awk '{print $1}')"
echo "Upload that file (primary and secondary separately if both stormed)."
ls -la "$OUTDIR" | head -30
