#!/usr/bin/env bash
# Full post-deploy / post-promote diagnostics for Secondary B (slot-aware).
#
# Download and run (public raw URL if repo is readable):
#   curl -fsSL https://raw.githubusercontent.com/PavelNeyman/netductor/main/scripts/diag-secondary-b.sh | bash
#
# Private repo (token with contents:read):
#   GH_TOKEN=ghp_...  # or export from /etc/netductor/secrets
#   curl -fsSL -H "Authorization: Bearer ${GH_TOKEN}" \
#     https://raw.githubusercontent.com/PavelNeyman/netductor/main/scripts/diag-secondary-b.sh | bash
#
# Or pin a commit/tag:
#   curl -fsSL .../netductor/<sha-or-tag>/scripts/diag-secondary-b.sh | bash
#
# Override targets without editing the script:
#   ROLE=primary PEER_IP=195.140.147.20 PEER_ID=peer-195-140-147-20 \
#     AGENT_ID=secondary-bb9952c66617ca91 bash
#   ROLE=secondary bash
#
# Exit 0 = no FAIL lines; non-zero = at least one FAIL.
set -euo pipefail

ROLE="${ROLE:-auto}"
PEER_IP="${PEER_IP:-195.140.147.20}"
PEER_ID="${PEER_ID:-peer-195-140-147-20}"
AGENT_ID="${AGENT_ID:-secondary-bb9952c66617ca91}"
SLOT1_P_SP="${SLOT1_P_SP:-10.87.20.1}"
SLOT1_S_SP="${SLOT1_S_SP:-10.87.20.2}"
SLOT1_P_PS="${SLOT1_P_PS:-10.87.21.1}"
SLOT1_S_PS="${SLOT1_S_PS:-10.87.21.2}"

PASS=0
FAIL=0
WARN=0

ok()   { echo "OK   $*"; PASS=$((PASS+1)); }
bad()  { echo "FAIL $*"; FAIL=$((FAIL+1)); }
warn() { echo "WARN $*"; WARN=$((WARN+1)); }
section() { echo; echo "=== $* ==="; }

detect_role() {
  if [[ -f /etc/netductor/svc-paths/peers.json ]] && grep -q active_id /etc/netductor/svc-paths/peers.json 2>/dev/null; then
    echo primary
  elif systemctl is-active netductor-secondary-agent &>/dev/null || [[ -f /etc/netductor/secrets/mtls/client.crt ]]; then
    echo secondary
  else
    echo unknown
  fi
}

if [[ "$ROLE" == auto ]]; then
  ROLE=$(detect_role)
fi
echo "ROLE=$ROLE host=$(hostname -f 2>/dev/null || hostname) $(date -Is)"

# ---------- PRIMARY ----------
if [[ "$ROLE" == primary ]]; then
  section "peer registry"
  if command -v netductor >/dev/null; then
    netductor svc-paths peer list || true
    netductor secondary status || true
  fi
  if [[ -f /etc/netductor/svc-paths/peers.json ]]; then
    python3 - << PY
import json
idx=json.load(open("/etc/netductor/svc-paths/peers.json"))
print("active_id=", idx.get("active_id"))
for p in idx.get("peers") or []:
    print(f"  {p.get('id')} state={p.get('state')} ip={p.get('public_ip')} slot={p.get('slot',{}).get('index')}")
want="$PEER_ID"
active=idx.get("active_id")
if active==want:
    print("CHECK active is B")
else:
    print("CHECK active is NOT B (active=%s)"%active)
PY
    ACT=$(python3 -c "import json;print(json.load(open('/etc/netductor/svc-paths/peers.json')).get('active_id',''))")
    if [[ "$ACT" == "$PEER_ID" ]]; then ok "active_id=$PEER_ID"; else bad "active_id=$ACT want $PEER_ID"; fi
  else
    bad "peers.json missing"
  fi

  PREF=$(cat /etc/netductor/preferred_secondary_ip 2>/dev/null || true)
  if [[ "$PREF" == "$PEER_IP" ]]; then ok "preferred_secondary_ip=$PREF"; else bad "preferred_secondary_ip='$PREF' want $PEER_IP"; fi

  section "slot1 plane on primary"
  for u in wg-quick@nd-svc-sp1 wg-quick@nd-svc-ps1 nd-wss-sp-server-nd-svc-sp1 nd-wss-ps-client-nd-svc-ps1; do
    if systemctl is-active "$u" &>/dev/null; then ok "unit $u active"
    else bad "unit $u not active"; fi
  done
  if ping -c1 -W3 "$SLOT1_S_SP" &>/dev/null; then ok "ping SP peer $SLOT1_S_SP"; else bad "ping SP peer $SLOT1_S_SP"; fi
  if ping -c1 -W3 "$SLOT1_S_PS" &>/dev/null; then ok "ping PS peer $SLOT1_S_PS"; else bad "ping PS peer $SLOT1_S_PS"; fi

  section "agent / devices"
  if [[ -f /var/lib/netductor/secondary/devices.json ]]; then
    python3 - << PY
import json, time
from datetime import datetime, timezone
f=json.load(open("/var/lib/netductor/secondary/devices.json"))
devs=f.get("devices") or f if isinstance(f, list) else f.get("Devices") or []
# support both shapes
if isinstance(f, dict) and "devices" in f:
    devs=f["devices"]
now=time.time()
found=None
for d in devs:
    ip=str(d.get("public_ip") or d.get("PublicIP") or "")
    if ip=="$PEER_IP" or str(d.get("id") or d.get("ID") or "")=="$AGENT_ID":
        found=d
        break
if not found:
    print("FAIL device not in devices.json")
else:
    ls=found.get("last_seen") or found.get("LastSeen")
    print("device id=", found.get("id") or found.get("ID"))
    print("public_ip=", found.get("public_ip") or found.get("PublicIP"))
    print("pbk_tail=", (found.get("pbk") or found.get("PBK") or "")[-8:])
    print("sid=", found.get("sid") or found.get("SID"))
    print("sni=", found.get("sni") or found.get("SNI"))
    print("last_seen=", ls)
    print("singbox_ok=", found.get("singbox_ok") or found.get("SingboxOK"))
PY
  else
    warn "devices.json missing"
  fi

  section "public face B :443 from primary"
  if timeout 5 bash -c "echo >/dev/tcp/$PEER_IP/443" 2>/dev/null; then
    ok "tcp $PEER_IP:443 connect"
  else
    # second try with nc
    if command -v nc >/dev/null && nc -z -w5 "$PEER_IP" 443; then ok "tcp $PEER_IP:443 nc"; else bad "tcp $PEER_IP:443 unreachable"; fi
  fi

  section "doctor snippet"
  if command -v netductor >/dev/null; then
    netductor doctor 2>/dev/null | grep -E 'fail|WARN|secondary|snapshot|channel' | head -40 || true
  fi
fi

# ---------- SECONDARY ----------
if [[ "$ROLE" == secondary ]]; then
  section "units"
  for u in sing-box netductor-secondary-agent wg-quick@nd-svc-sp1 wg-quick@nd-svc-ps1; do
    if systemctl is-active "$u" &>/dev/null; then ok "unit $u active"
    else
      # slot0 names on single-secondary installs
      alt=${u/sp1/sp}; alt=${alt/ps1/ps}
      if systemctl is-active "$alt" &>/dev/null; then warn "unit $u missing but $alt active (slot0?)"; else bad "unit $u not active"; fi
    fi
  done

  section "listen"
  if ss -tlnp | grep -qE ':443\b'; then ok "listen :443"; else bad "no listen :443"; fi
  if ss -tlnp | grep -qE ':52222\b'; then ok "listen :52222"; else warn "no :52222 (factory still on 22?)"; fi

  section "WG slot1"
  if wg show nd-svc-sp1 &>/dev/null; then
    wg show nd-svc-sp1 | head -20
    ok "wg nd-svc-sp1 present"
  else
    bad "wg nd-svc-sp1 missing"
  fi
  if wg show nd-svc-ps1 &>/dev/null; then
    wg show nd-svc-ps1 | head -20
    ok "wg nd-svc-ps1 present"
  else
    bad "wg nd-svc-ps1 missing"
  fi
  if ping -c1 -W3 "$SLOT1_P_SP" &>/dev/null; then ok "ping primary SP $SLOT1_P_SP"; else bad "ping primary SP $SLOT1_P_SP"; fi
  if ping -c1 -W3 "$SLOT1_P_PS" &>/dev/null; then ok "ping primary PS $SLOT1_P_PS"; else bad "ping primary PS $SLOT1_P_PS"; fi

  section "agent URL (must be slot1 VIP, not public and not slot0)"
  journalctl -u netductor-secondary-agent --since "2 hours ago" --no-pager 2>/dev/null \
    | grep -E 'secondary agent →|agent URL|svc-paths apply' | tail -20 || true
  if journalctl -u netductor-secondary-agent --since "30 min ago" --no-pager 2>/dev/null \
      | grep -qE '10\.87\.10\.|10\.87\.11\.'; then
    warn "recent logs mention slot0 10.87.10/11 — check CF-3"
  else
    ok "no slot0 VIP in last 30m agent logs"
  fi
  if journalctl -u netductor-secondary-agent --since "2 hours ago" --no-pager 2>/dev/null \
      | grep -q 'uplink watchdog: restarting sing-box'; then
    warn "watchdog restarted sing-box in last 2h — CF-2"
  else
    ok "no watchdog sing-box restart in last 2h"
  fi

  section "sing-box reality inbound"
  if [[ -f /usr/local/etc/sing-box/config.json ]]; then
    python3 - << 'PY'
import json, hashlib
c=json.load(open("/usr/local/etc/sing-box/config.json"))
relay=None
for ib in c.get("inbounds") or []:
    if ib.get("tag")=="relay-in" or ib.get("listen_port")==443:
        relay=ib; break
if not relay:
    print("FAIL no relay-in / :443 inbound")
else:
    tls=relay.get("tls") or {}
    r=tls.get("reality") or {}
    priv=(r.get("private_key") or "").strip()
    print("tag", relay.get("tag"), "port", relay.get("listen_port"))
    print("server_name", tls.get("server_name"))
    print("short_id", r.get("short_id"))
    print("priv_sha12", hashlib.sha256(priv.encode()).hexdigest()[:12] if priv else "EMPTY")
    print("users", len(relay.get("users") or []))
PY
    ok "parsed sing-box config"
  else
    bad "no sing-box config"
  fi

  section "mtls material"
  for f in /etc/netductor/secrets/mtls/ca.crt /etc/netductor/secrets/mtls/client.crt /etc/netductor/secrets/mtls/client.key; do
    if [[ -f "$f" ]]; then ok "exists $f"; else bad "missing $f"; fi
  done
fi

section "summary"
echo "PASS=$PASS WARN=$WARN FAIL=$FAIL"
if [[ "$FAIL" -gt 0 ]]; then exit 1; fi
exit 0
