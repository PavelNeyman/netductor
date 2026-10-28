#!/usr/bin/env bash
# Netductor post-update health report (sanitized).
# Run on Primary and on Secondary; paste both outputs.
# Does NOT dump keys, tokens, material secrets, or full VPN user lists.
set -euo pipefail

mask_ip() {
  # 1.2.3.4 -> 1.2.x.x
  sed -E 's/\b([0-9]{1,3}\.)([0-9]{1,3}\.)([0-9]{1,3}\.)([0-9]{1,3})\b/\1\2x.x/g'
}

mask_secrets() {
  sed -E \
    -e 's/(token|secret|password|passwd|api[_-]?key|private[_-]?key|recovery)(=|:[[:space:]]*)[^[:space:]]+/\1\2***REDACTED***/Ig' \
    -e 's/\b(ghp_|gho_|github_pat_|sk-|eyJ)[A-Za-z0-9._-]{8,}/***REDACTED***/g' \
    -e 's/-----BEGIN[^-]+-----.*-----END[^-]+-----/***REDACTED-PEM***/g'
}

section() { printf '\n======== %s ========\n' "$1"; }

HOST="$(hostname 2>/dev/null || echo unknown)"
ROLE_GUESS="unknown"
if [[ -f /etc/netductor/role ]]; then
  ROLE_GUESS="$(tr -d ' \n' </etc/netductor/role 2>/dev/null || true)"
fi
# crude role hint
if systemctl is-active netductor-agent >/dev/null 2>&1 && ! systemctl is-active netductor >/dev/null 2>&1; then
  ROLE_GUESS="${ROLE_GUESS:-secondary}"
fi

echo "ND_POST_UPDATE_REPORT"
echo "generated_utc: $(date -u +%Y-%m-%dT%H:%M:%SZ)"
echo "hostname: $HOST"
echo "role_file_or_guess: ${ROLE_GUESS:-unknown}"
echo "kernel: $(uname -srm 2>/dev/null || true)"

section "VERSION"
{
  command -v netductor >/dev/null && netductor version 2>&1 || echo "(no netductor in PATH)"
  command -v netductor-agent >/dev/null && netductor-agent version 2>&1 || true
  echo -n "VERSION_file: "
  cat /etc/netductor/VERSION 2>/dev/null || echo "(missing)"
} 2>&1 | mask_ip | mask_secrets || true

section "SYSTEMD"
{
  for u in netductor netductor-agent netductor-tg netductor-stack \
           'wg-quick@nd-svc-sp' 'wg-quick@nd-svc-ps' \
           'wg-quick@nd-svc-sp1' 'wg-quick@nd-svc-ps1'; do
    st=$(systemctl is-active "$u" 2>/dev/null || echo n/a)
    en=$(systemctl is-enabled "$u" 2>/dev/null || echo n/a)
    printf '%-28s active=%-10s enabled=%s\n' "$u" "$st" "$en"
  done
  echo "--- failed units ---"
  systemctl --failed --no-pager 2>/dev/null | head -30 || true
} 2>&1 | mask_ip | mask_secrets || true

section "DOCTOR"
{
  if command -v netductor >/dev/null; then
    # Prefer non-verbose; still sanitize
    netductor doctor 2>&1 | head -120 || true
  else
    echo "(netductor not installed)"
  fi
} 2>&1 | mask_ip | mask_secrets || true

section "FLEET_SECONDARY_STATUS"
{
  if command -v netductor >/dev/null; then
    netductor fleet status 2>&1 | head -40 || true
    netductor secondary status 2>&1 | head -40 || true
  else
    echo "(skip: no netductor)"
  fi
} 2>&1 | mask_ip | mask_secrets || true

section "SVC_PATHS"
{
  if command -v netductor >/dev/null; then
    netductor svc-paths status 2>&1 | head -40 || true
    echo "--- peer list ---"
    netductor svc-paths peer list 2>&1 | head -60 || true
  else
    echo "(skip: no netductor)"
  fi
  echo "--- preferred / active (masked) ---"
  echo -n "preferred_secondary_ip: "
  if [[ -f /etc/netductor/preferred_secondary_ip ]]; then
    mask_ip </etc/netductor/preferred_secondary_ip | tr -d '\n'; echo
  else
    echo "(unset)"
  fi
  echo -n "active_peer_id: "
  cat /etc/netductor/svc-paths/active_peer_id 2>/dev/null || echo "(unset)"
} 2>&1 | mask_ip | mask_secrets || true

section "LEGACY_MATERIAL_META"
{
  # Only non-secret metadata — never dump keys
  f=/etc/netductor/svc-paths/material.json
  if [[ ! -f $f ]]; then
    echo "(no legacy material.json)"
  else
    python3 - <<'PY' 2>/dev/null || echo "(python parse failed)"
import json
m=json.load(open("/etc/netductor/svc-paths/material.json"))
keys=sorted(m.keys())
# show structure only + safe fields
safe={}
for k in ("slot","peer_id","PeerID","secondary","secondary_ip","primary","primary_ip"):
    if k in m and m[k] not in (None,""):
        safe[k]=m[k]
print("keys:", ", ".join(keys))
print("safe_fields:", safe)
# presence flags for secrets without values
for k in keys:
    kl=k.lower()
    if any(x in kl for x in ("key","psk","priv","token","cert","secret")):
        v=m.get(k)
        print(f"has_{k}:", "yes" if v else "empty")
PY
  fi
} 2>&1 | mask_ip | mask_secrets || true

section "LOCAL_HEALTH"
{
  for url in \
    http://127.0.0.1:8787/health \
    http://127.0.0.1:8787/v1/health \
    http://127.0.0.1:8789/health
  do
    code=$(curl -sS -m 2 -o /dev/null -w "%{http_code}" "$url" 2>/dev/null || echo "err")
    echo "$url -> $code"
  done
} 2>&1 | mask_secrets || true

section "NETWORK_HINTS"
{
  # Interface names + whether 10.87 exists — addresses masked
  ip -br addr 2>/dev/null | mask_ip | head -30 || true
  echo "--- wg interfaces (names only) ---"
  wg show interfaces 2>/dev/null || true
} 2>&1 | mask_ip | mask_secrets || true

section "JOURNAL_TAIL_SANITIZED"
{
  for u in netductor netductor-agent netductor-tg; do
    if systemctl cat "$u" >/dev/null 2>&1; then
      echo "--- $u (last 25) ---"
      journalctl -u "$u" -n 25 --no-pager -o cat 2>/dev/null | mask_ip | mask_secrets || true
    fi
  done
} 2>&1 || true

section "END"
echo "OK report complete (sanitized). Send this whole output."
