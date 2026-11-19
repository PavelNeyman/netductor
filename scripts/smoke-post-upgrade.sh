#!/usr/bin/env bash
# Post-upgrade smoke (control/primary). Read-only — does not change config.
# Exit 0 = critical checks passed; non-zero = fix before relying on the fleet.
#
# Usage (on primary as root):
#   WANT_VERSION=0.9.758 ./scripts/smoke-post-upgrade.sh
#   # or after curl from raw main:
#   curl -fsSL https://raw.githubusercontent.com/PavelNeyman/netductor/main/scripts/smoke-post-upgrade.sh | sudo bash
#
# Optional: WANT_VERSION=0.9.758  (default: /etc/netductor/VERSION or netductor version)
set -euo pipefail

FAIL=0
pass() { printf 'OK   %s\n' "$*"; }
warn() { printf 'WARN %s\n' "$*"; }
fail() { printf 'FAIL %s\n' "$*"; FAIL=1; }

if [[ "${EUID:-$(id -u)}" -ne 0 ]]; then
  warn "not root — some checks may be incomplete"
fi

# Resolve expected version
WANT="${WANT_VERSION:-}"
if [[ -z "$WANT" && -r /etc/netductor/VERSION ]]; then
  WANT="$(tr -d ' \nvV' </etc/netductor/VERSION)"
fi
if [[ -z "$WANT" ]] && command -v netductor >/dev/null; then
  WANT="$(netductor version 2>/dev/null | head -1 | grep -oE '[0-9]+\.[0-9]+\.[0-9]+' | head -1 || true)"
fi
WANT="${WANT#v}"
if [[ -z "$WANT" ]]; then
  echo "FAIL: set WANT_VERSION=x.y.z" >&2
  exit 2
fi

echo "=== netductor post-upgrade smoke (want $WANT) ==="
echo "host=$(hostname -f 2>/dev/null || hostname)  $(date -u +%Y-%m-%dT%H:%M:%SZ)"

# --- versions ---
VER_OUT="$(netductor version 2>/dev/null || true)"
echo "$VER_OUT"
echo "$VER_OUT" | grep -qE "$WANT" && pass "primary version contains $WANT" || fail "primary version missing $WANT"

FILE_VER="$(tr -d ' \n' </etc/netductor/VERSION 2>/dev/null || true)"
FILE_VER="${FILE_VER#v}"
[[ "$FILE_VER" == "$WANT" ]] && pass "VERSION file=$FILE_VER" || fail "VERSION file='$FILE_VER' want $WANT"

# --- units ---
for u in netductor-api netductor-telegram-bot; do
  if systemctl is-active --quiet "$u" 2>/dev/null; then
    pass "unit $u active"
  else
    fail "unit $u not active"
    systemctl --no-pager -l status "$u" 2>&1 | tail -8 || true
  fi
done

# --- API health (common ports) ---
health_ok=0
for url in \
  "http://127.0.0.1:8787/health" \
  "http://127.0.0.1:8788/health" \
  "https://127.0.0.1:8787/health"; do
  if curl -fsSk -m 5 "$url" >/dev/null 2>&1; then
    pass "API health $url"
    health_ok=1
    break
  fi
done
[[ "$health_ok" -eq 1 ]] || warn "loopback /health not probed OK — doctor below is authoritative"

# --- migrate ---
MIG="$(netductor migrate status 2>&1 || true)"
echo "$MIG" | head -25
if echo "$MIG" | grep -qiE 'status=failed|attention=true'; then
  fail "migrate journal attention/failed"
else
  pass "migrate status (no failed/attention)"
fi

# --- control role (mesh phase 3: sole node must stay active) ---
if CTRL="$(netductor control status 2>/dev/null)"; then
  echo "$CTRL" | head -10
  if echo "$CTRL" | grep -qi 'write_allowed=true' && echo "$CTRL" | grep -qi 'active'; then
    pass "control_role active write_allowed"
  elif echo "$CTRL" | grep -qi 'standby'; then
    fail "control_role=standby on this host — sole control must not be standby"
  else
    warn "control status ambiguous — review output"
  fi
else
  warn "control status N/A"
fi

# --- secondary ---
echo "--- secondary status ---"
SEC="$(netductor secondary status 2>&1 || true)"
echo "$SEC"
if echo "$SEC" | grep -qE "$WANT"; then
  pass "secondary agent version shows $WANT"
else
  fail "secondary version not $WANT (upgrade still queued? re-run later)"
fi
if echo "$SEC" | grep -qiE 'sb=true'; then
  pass "secondary singbox_ok (sb=true)"
else
  fail "secondary sb not true — peer may drop from /sub"
fi
if echo "$SEC" | grep -qiE 'online'; then
  pass "secondary online"
else
  fail "secondary not online"
fi
if echo "$SEC" | grep -qiE 'region=$|region=[[:space:]]'; then
  warn "secondary region unset — optional: netductor secondary region <id> <ru-msk|fi-hel|…>"
fi

# --- cluster (phase 4) ---
if CL="$(netductor cluster status 2>/dev/null)"; then
  echo "$CL"
  echo "$CL" | grep -qi 'write_allowed=true' && pass "cluster write_allowed" || warn "cluster write_allowed?"
  echo "$CL" | grep -qi "ver=$WANT" && pass "cluster members on $WANT" || warn "cluster version mismatch"
else
  warn "cluster status unavailable"
fi

# --- doctor ---
echo "--- doctor summary ---"
DOC="$(netductor doctor 2>&1 || true)"
echo "$DOC" | tail -50
FAIL_N=$(echo "$DOC" | grep -cE '^FAIL' || true)
WARN_N=$(echo "$DOC" | grep -cE '^WARN' || true)
if [[ "${FAIL_N:-0}" -gt 0 ]]; then
  fail "doctor FAIL lines=$FAIL_N"
else
  pass "doctor no FAIL lines (warn=$WARN_N)"
fi
echo "$DOC" | grep -qi 'control_role=standby' && fail "doctor: control_role=standby" || true
echo "$DOC" | grep -qi 'VERSION matches latest GitHub tag' && pass "VERSION matches latest GH tag" || warn "VERSION vs GH tag (doctor INFO/OK line)"

# --- sing-box unit if present ---
if systemctl list-unit-files 2>/dev/null | grep -q '^sing-box'; then
  systemctl is-active --quiet sing-box 2>/dev/null && pass "sing-box active" || warn "sing-box not active"
fi

echo "=== result ==="
if [[ "$FAIL" -eq 0 ]]; then
  echo "PASS smoke for $WANT"
  exit 0
fi
echo "FAIL smoke — fix items above; then: netductor secondary status"
exit 1
