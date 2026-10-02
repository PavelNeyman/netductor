#!/usr/bin/env bash
# Collect primary/secondary netductor state for offline review (no secrets in clear if possible).
# Usage:
#   PRIMARY:  sudo bash collect-vps-state.sh primary  > /tmp/nd-state-primary.txt
#   SECONDARY: sudo bash collect-vps-state.sh secondary > /tmp/nd-state-secondary.txt
# Redacts: tokens, private keys, long base64 blobs.
set -euo pipefail
ROLE="${1:-primary}"
TS=$(date -u +%Y%m%dT%H%M%SZ)
HOST=$(hostname -f 2>/dev/null || hostname)
redact() {
  sed -E \
    -e 's/(token|password|secret|passwd|authorization)[=: ]+[^[:space:]]+/\1=***REDACTED***/Ig' \
    -e 's/Bearer [A-Za-z0-9._-]{8,}/Bearer ***REDACTED***/g' \
    -e 's/-----BEGIN [A-Z0-9 ]+PRIVATE KEY-----/***PRIVATE KEY REDACTED***/g' \
    -e 's/[A-Za-z0-9+\/=]{80,}/***LONG_B64***/g'
}

echo "=== netductor state collect ==="
echo "role=$ROLE host=$HOST ts=$TS"
echo

echo "=== versions ==="
command -v netductor >/dev/null && netductor version || echo "netductor: missing"
command -v netductor-tg >/dev/null && netductor-tg version 2>/dev/null || echo "netductor-tg: missing"
# secondary runs: netductor secondary agent (same node binary)
if command -v netductor-agent >/dev/null; then netductor-agent version 2>/dev/null
elif systemctl is-active --quiet netductor-secondary-agent 2>/dev/null; then
  echo -n "netductor-secondary-agent: "; systemctl show -p ExecStart --value netductor-secondary-agent 2>/dev/null
  /usr/local/bin/netductor version 2>/dev/null || true
else echo "netductor-agent: n/a"; fi
cat /etc/netductor/VERSION 2>/dev/null || echo "VERSION file: missing"
echo

echo "=== units ==="
for u in netductor-api netductor-telegram-bot netductor-redirect sing-box blocky netductor-backup.timer \
         netductor-secondary-agent netductor-agent netductor-vpn nd-wss-sp-server nd-wss-sp-client nd-wss-ps-server nd-wss-ps-client \
         wg-quick@nd-svc-sp wg-quick@nd-svc-ps docker; do
  st=$(systemctl is-active "$u" 2>/dev/null || echo n/a)
  en=$(systemctl is-enabled "$u" 2>/dev/null || echo n/a)
  echo "$u active=$st enabled=$en"
done
echo

echo "=== stack status (json if any) ==="
netductor stack status 2>/dev/null | head -c 8000 || true
echo
ls -la /var/lib/netductor/stack/ 2>/dev/null || true
cat /var/lib/netductor/stack/current.json 2>/dev/null || true
echo
cat /var/lib/netductor/stack/prev/VERSION 2>/dev/null || true
echo

echo "=== doctor (truncated) ==="
netductor doctor 2>/dev/null | head -c 12000 || true
echo

echo "=== conf (redacted) ==="
if [[ -f /etc/netductor/netductor.conf ]]; then
  redact < /etc/netductor/netductor.conf
else
  echo "no netductor.conf"
fi
echo

echo "=== ports listen ==="
ss -lntup 2>/dev/null | grep -E ':(22|443|80|52222|8787|8788|8789|8443|8444|8445|9118)\s' || ss -lntup 2>/dev/null | head -40
echo

echo "=== ufw / nft sample ==="
ufw status verbose 2>/dev/null | head -60 || true
echo
nft list ruleset 2>/dev/null | head -80 || true
echo

echo "=== sing-box ==="
systemctl is-active sing-box 2>/dev/null || true
# config existence only, not dump secrets
ls -la /etc/sing-box/ 2>/dev/null || ls -la /etc/netductor/sing-box* 2>/dev/null || true
test -f /etc/sing-box/config.json && python3 -c "
import json,sys
p='/etc/sing-box/config.json'
try:
  d=json.load(open(p))
  print('inbounds', [x.get('tag') or x.get('type') for x in d.get('inbounds',[])])
  print('outbounds', [x.get('tag') or x.get('type') for x in d.get('outbounds',[])][:20])
except Exception as e:
  print('parse',e)
" 2>/dev/null || echo "sing-box config not parsed"
echo

echo "=== agents / devices (ids only) ==="
# avoid dumping tokens
for f in /var/lib/netductor/secondary/devices.json /var/lib/netductor/edge/devices.json; do
  if [[ -f $f ]]; then
    echo "-- $f"
    python3 -c "
import json,sys
d=json.load(open('$f'))
if isinstance(d, dict):
  items=d.get('devices') or d
  if isinstance(items, dict):
    for k,v in list(items.items())[:50]:
      if isinstance(v, dict):
        print(k, 'status=', v.get('status'), 'host=', v.get('hostname') or v.get('host'), 'ip=', v.get('ip') or v.get('wan_ip'), 'last=', v.get('last_seen'))
      else:
        print(k)
  elif isinstance(items, list):
    for v in items[:50]:
      print(v.get('id') or v.get('device_id'), v.get('status'), v.get('ip'))
" 2>/dev/null || echo "(parse fail)"
  fi
done
echo

echo "=== backups ==="
find /var/lib/netductor/backups -type f 2>/dev/null | head -30
ls -la /var/lib/netductor/backups/peers 2>/dev/null || true
echo

echo "=== LE certs ==="
ls -la /etc/letsencrypt/live/ 2>/dev/null || echo "no LE live"
echo

echo "=== svc-paths health ==="
cat /var/lib/netductor/svc-paths-health.json 2>/dev/null || true
ip -br a 2>/dev/null | head -30
echo

echo "=== resources ==="
free -h 2>/dev/null || true
df -h / /var /var/lib 2>/dev/null || df -h
uptime
echo

echo "=== recent journal (no secrets filter) ==="
journalctl -u netductor-api -u netductor-telegram-bot -u sing-box -u netductor-secondary-agent -u netductor-agent --since "24 hours ago" -n 120 --no-pager 2>/dev/null | redact | tail -150
echo
echo "=== host monitoring (zabbix) ===
systemctl is-active zabbix-agent zabbix-agent2 zabbix-agentd 2>/dev/null || true
ss -tlnp 2>/dev/null | grep -E ':10050|:10051' || echo "ports 10050/10051 not listening"
dpkg -l 'zabbix*' 2>/dev/null | awk '/^ii/{print}' || true

=== END ==="
