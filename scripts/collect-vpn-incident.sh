#!/usr/bin/env bash
# Collect ~N minutes of VPN / uplink / control-plane logs for offline review.
# Redacts tokens/keys; does not dump private keys.
#
# Usage (on the node, as root):
#   curl -fsSL https://raw.githubusercontent.com/PavelNeyman/netductor/main/scripts/collect-vpn-incident.sh | bash -s -- 90
#
# Output: /tmp/nd-vpn-incident-<host>-<ts>.tar.gz
set -uo pipefail

MINUTES="${1:-90}"
MINUTES="${MINUTES%m}"
if ! [[ "$MINUTES" =~ ^[0-9]+$ ]] || [[ "$MINUTES" -lt 15 ]] || [[ "$MINUTES" -gt 10080 ]]; then
  echo "usage: $0 [minutes]   (15..10080, default 90)" >&2
  exit 2
fi

TS=$(date -u +%Y%m%dT%H%M%SZ)
HOST=$(hostname -s 2>/dev/null || true)
HOST=${HOST:-$(hostname 2>/dev/null || true)}
HOST=${HOST:-node}
HOST=${HOST//[^a-zA-Z0-9._-]/_}

OUTDIR="/tmp/nd-vpn-incident-${HOST}-${TS}"
mkdir -p "$OUTDIR"
cd "$OUTDIR"

pack() {
  local rc=${1:-0}
  cd /tmp || true
  if [[ -d "$OUTDIR" ]]; then
    if tar -C /tmp -czf "${OUTDIR}.tar.gz" "$(basename "$OUTDIR")"; then
      echo
      echo "OK bundle: ${OUTDIR}.tar.gz"
      du -h "${OUTDIR}.tar.gz" 2>/dev/null || true
    else
      echo "WARN: tar failed; directory left at $OUTDIR" >&2
      rc=1
    fi
  fi
  exit "$rc"
}
trap 'pack $?' EXIT

SINCE="${MINUTES} min ago"
{
  echo "collect vpn incident host=$HOST ts=$TS since=$SINCE"
  date -u
  uptime 2>/dev/null || true
} | tee meta.txt

redact() {
  sed -E \
    -e 's/(token|password|secret|passwd|authorization|api_key)[=: ]+[^[:space:]]+/\1=***REDACTED***/Ig' \
    -e 's/Bearer [A-Za-z0-9._-]{8,}/Bearer ***REDACTED***/g' \
    -e 's/[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}/***UUID***/g' \
    -e 's/[A-Za-z0-9+\/=]{100,}/***LONG_B64***/g'
}

{
  echo "##### versions #####"
  cat /etc/netductor/VERSION 2>/dev/null || true
  command -v netductor >/dev/null && netductor version 2>/dev/null || true
  uname -a
  head -6 /etc/os-release 2>/dev/null || true

  echo
  echo "##### role / doctor #####"
  cat /etc/netductor/role 2>/dev/null || true
  netductor doctor 2>/dev/null | head -c 16000 || true

  echo
  echo "##### units #####"
  for u in sing-box netductor-api netductor-telegram-bot netductor-redirect \
           netductor-secondary-agent netductor-agent blocky \
           nd-wss-sp-server nd-wss-sp-client nd-wss-ps-server nd-wss-ps-client \
           netductor-stack-watchdog.timer; do
    printf '%s active=%s enabled=%s\n' "$u" \
      "$(systemctl is-active "$u" 2>/dev/null || echo n/a)" \
      "$(systemctl is-enabled "$u" 2>/dev/null || echo n/a)"
  done

  echo
  echo "##### load / mem #####"
  uptime 2>/dev/null || true
  free -h 2>/dev/null || true
  cat /proc/loadavg 2>/dev/null || true
  df -h / /var /var/lib 2>/dev/null || true

  echo
  echo "##### listen #####"
  ss -tulnp 2>/dev/null | grep -E ':(443|52222|8443|8445|8787|8789|53)\s' || ss -tulnp 2>/dev/null | head -40 || true

  echo
  echo "##### firewall #####"
  netductor firewall status 2>/dev/null || true

  echo
  echo "##### sing-box summary #####"
  CONF=""
  for p in /usr/local/etc/sing-box/config.json /etc/sing-box/config.json; do
    [[ -f $p ]] && CONF=$p && break
  done
  if [[ -n ${CONF} ]]; then
    echo "path=$CONF"
    ls -la "$CONF" || true
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
    echo "no sing-box config"
  fi

  echo
  echo "##### secondary / svc-paths #####"
  netductor secondary status 2>/dev/null || true
  netductor svc-paths status 2>/dev/null || true

  echo
  echo "##### sing-box restarts #####"
  systemctl show sing-box -p ActiveEnterTimestamp -p NRestarts -p ExecMainStatus 2>/dev/null || true
  journalctl -u sing-box --since "$SINCE" -o short-iso --no-pager 2>/dev/null \
    | grep -iE 'Started|Stopped|Failed|Main process' | tail -40 || true
} 2>/dev/null | redact | tee summary.txt >/dev/null || true

for u in sing-box netductor-api netductor-telegram-bot netductor-secondary-agent netductor-agent blocky; do
  journalctl -u "$u" --since "$SINCE" -o short-iso --no-pager 2>/dev/null | redact > "journal-${u}.log" || true
done
journalctl -k --since "$SINCE" -o short-iso --no-pager 2>/dev/null \
  | grep -iE 'oom|kill|net|udp|tcp|link|nf_|conntrack' | redact > journal-kernel-net.log || true

{
  echo "=== error/reject/mux counters (last ${MINUTES}m) ==="
  shopt -s nullglob
  for f in journal-*.log; do
    echo "-- $f"
    n=$(grep -ciE 'error|fail|reject|timeout|reset|refused|invalid' "$f" 2>/dev/null || echo 0)
    echo "  match_lines=$n"
    grep -iE 'reject|remote error|REALITY|invalid connection|mux|uplink|10\.88|deadline|i/o timeout|connection reset' "$f" 2>/dev/null | tail -80 || true
    echo
  done
} > error-index.txt 2>/dev/null || true

for f in journal-sing-box.log journal-netductor-api.log; do
  if [[ -f $f ]]; then
    {
      echo "=== head $f ==="
      head -50 "$f" || true
      echo "=== tail $f ==="
      tail -100 "$f" || true
    } > "${f%.log}-headtail.txt" || true
  fi
done

echo "dir ready: $OUTDIR"
ls -la "$OUTDIR" | head -30 || true
