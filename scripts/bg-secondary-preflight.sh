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
else
  echo "(no legacy material.json)"
fi
echo "== preferred_secondary_ip =="
cat /etc/netductor/preferred_secondary_ip 2>/dev/null || echo "(unset)"
echo "== active peer =="
cat /etc/netductor/svc-paths/active_peer_id 2>/dev/null || echo "(unset)"
echo "OK preflight done"
