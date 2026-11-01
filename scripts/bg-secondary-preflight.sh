#!/usr/bin/env bash
# Preflight on Primary before Blue-Green add. Does not modify state.
set -euo pipefail
echo "== netductor version =="
netductor version 2>/dev/null || true
echo "== peers =="
netductor svc-paths peer list 2>/dev/null || echo "(peer list failed)"
echo "== inventory =="
netductor inventory list 2>/dev/null || echo "(inventory empty/unavailable)"
echo "== legacy material =="
if [[ -f /etc/netductor/svc-paths/material.json ]]; then
  python3 -c "import json;m=json.load(open('/etc/netductor/svc-paths/material.json'));print('secondary',m.get('secondary_ip'),'slot',m.get('slot',0))"
elif [[ -f /etc/netductor/svc-paths/peers.json ]] && python3 -c "import json,sys;d=json.load(open('/etc/netductor/svc-paths/peers.json'));sys.exit(0 if d.get('peers') else 1)"; then
  echo "(legacy material absent; managed peer registry already exists)"
else
  echo "(no legacy material.json and no managed peer registry)"
  # An empty registry plus missing material is not proof of a fresh host.
  # Refuse slot-0 reuse if legacy WG/WSS or preferred-secondary state remains.
  legacy_found=0
  for p in \
    /etc/wireguard/nd-svc-sp.conf \
    /etc/wireguard/nd-svc-ps.conf \
    /etc/systemd/system/nd-wss-sp-server.service \
    /etc/systemd/system/nd-wss-ps-client.service \
    /etc/netductor/preferred_secondary_ip \
    /etc/netductor/secrets/preferred_secondary_ip; do
    if [[ -e "$p" || -L "$p" ]]; then
      echo "BLOCK: legacy slot-0 artifact exists: $p"
      legacy_found=1
    fi
  done
  for iface in nd-svc-sp nd-svc-ps; do
    if [[ -e "/sys/class/net/$iface" ]]; then
      echo "BLOCK: legacy slot-0 kernel interface exists: $iface"
      legacy_found=1
    fi
  done
  for u in wg-quick@nd-svc-sp wg-quick@nd-svc-ps nd-wss-sp-server nd-wss-ps-client; do
    if systemctl is-active --quiet "$u" 2>/dev/null; then
      echo "BLOCK: legacy slot-0 unit is active: $u"
      legacy_found=1
    elif systemctl is-enabled "$u" 2>/dev/null | grep -qxE 'enabled|enabled-runtime'; then
      echo "BLOCK: legacy slot-0 unit is enabled: $u"
      legacy_found=1
    fi
  done
  if (( legacy_found )); then
    echo "ERROR: legacy slot-0 state is ambiguous; do not allocate another peer until the old material is restored or the old peer is explicitly retired."
    exit 1
  fi
fi
echo "== preferred_secondary_ip =="
cat /etc/netductor/preferred_secondary_ip 2>/dev/null || echo "(unset)"
echo "== active peer =="
cat /etc/netductor/svc-paths/active_peer_id 2>/dev/null || echo "(unset)"
echo "OK preflight done"
