#!/usr/bin/env bash
# Netductor read-only diagnostic collector.
# Does not read secret files, invoke doctor, migrate peer registry, or change services.
set -uo pipefail
umask 077

ROLE="${1:-}"
if [[ "$ROLE" != "P" && "$ROLE" != "S" ]]; then
  echo "Usage: sudo bash netductor-diag.sh P|S" >&2
  exit 2
fi
if [[ "$EUID" -ne 0 ]]; then
  echo "Run with sudo: sudo bash netductor-diag.sh P|S" >&2
  exit 2
fi

OUT_DIR="${TMPDIR:-/tmp}"
REPORT="$OUT_DIR/netductor-diagnostic-${ROLE}-$(date +%Y%m%d-%H%M%S).txt"
: > "$REPORT"
chmod 600 "$REPORT"

sanitize() {
  python3 -c '
import sys, re, ipaddress
s = sys.stdin.read()

# PEM private keys and common credential fields.
s = re.sub(r"-----BEGIN [A-Z0-9 ]*PRIVATE KEY-----.*?-----END [A-Z0-9 ]*PRIVATE KEY-----",
           "[REDACTED_PRIVATE_KEY]", s, flags=re.S)
s = re.sub(r"(?i)\bBearer\s+[A-Za-z0-9._~+/=-]+", "Bearer [REDACTED]", s)
s = re.sub(r"(?i)(\b(?:access[_-]?token|refresh[_-]?token|id[_-]?token|api[_-]?key|password|passwd|secret|authorization|client[_-]?secret|private[_-]?key)\b\s*[:=]\s*)(?:\"[^\"]*\"|\x27[^\x27]*\x27|[^\s,;]+)",
           r"\1[REDACTED]", s)
s = re.sub(r"(?i)(https?://)[^/\s:@]+:[^/\s@]+@", r"\1[REDACTED]@", s)
s = re.sub(r"(?i)([?&](?:access_token|refresh_token|token|api_key|key|secret|password|auth|signature)=)[^&#\s]+",
           r"\1[REDACTED]", s)
# JWT-shaped strings.
s = re.sub(r"\beyJ[A-Za-z0-9_-]{10,}\.[A-Za-z0-9_-]{10,}\.[A-Za-z0-9_-]{8,}\b",
           "[JWT_REDACTED]", s)
# MAC addresses.
s = re.sub(r"(?i)\b(?:[0-9a-f]{2}:){5}[0-9a-f]{2}\b", "[MAC_REDACTED]", s)
# Email addresses.
s = re.sub(r"\b[A-Za-z0-9._%+-]+@[A-Za-z0-9.-]+\.[A-Za-z]{2,}\b", "[EMAIL_REDACTED]", s)

# Replace globally routable IPv4s consistently; keep private/loopback/link-local VPN addresses.
ip_aliases = {}
def mask_ipv4(m):
    value = m.group(0)
    try: addr = ipaddress.ip_address(value)
    except ValueError: return value
    if addr.version == 4 and addr.is_global:
        if value not in ip_aliases:
            ip_aliases[value] = "[PUBLIC_IP_%d]" % (len(ip_aliases) + 1)
        return ip_aliases[value]
    return value
s = re.sub(r"(?<![\d.])(?:\d{1,3}\.){3}\d{1,3}(?![\d.])", mask_ipv4, s)

# Mask base64-like WireGuard/public-key identifiers and long opaque credentials.
s = re.sub(r"\b[A-Za-z0-9+/]{42,44}={0,2}\b", "[KEY_OR_TOKEN_REDACTED]", s)
s = re.sub(r"(?i)\b(?:token|secret|password|api[_-]?key)\s+\S{12,}", "[CREDENTIAL_REDACTED]", s)

sys.stdout.write(s)
'
}

section() { printf '\n\n===== %s =====\n' "$1" >> "$REPORT"; }

run() {
  local title="$1"; shift
  section "$title"
  "$@" 2>&1 | sanitize >> "$REPORT"
  local rc=${PIPESTATUS[0]}
  printf '[exit=%s]\n' "$rc" >> "$REPORT"
}

run_shell() {
  local title="$1"; shift
  section "$title"
  bash -o pipefail -c "$*" 2>&1 | sanitize >> "$REPORT"
  local rc=${PIPESTATUS[0]}
  printf '[exit=%s]\n' "$rc" >> "$REPORT"
}

{
  echo "Netductor diagnostic report (read-only)"
  echo "Role: $ROLE"
  echo "Collected: $(date -Is 2>/dev/null || date)"
  echo "Kernel: $(uname -sr 2>/dev/null || echo unavailable)"
  echo "This script does not intentionally read secret/config files."
  echo "Public IPv4s, MACs, email addresses, and key-shaped strings are masked."
} >> "$REPORT"

run "Netductor executable" bash -c 'command -v netductor; p="$(command -v netductor)" && readlink -f "$p"'
run "Netductor version" netductor version
run "Netductor status" netductor status
run "Service paths status" netductor svc-paths status

if [[ "$ROLE" == "P" ]]; then
  run "Secondary status" netductor secondary status
  run_shell "Netductor-related service units" 'systemctl list-units --all --type=service --no-legend --plain | grep -Ei "nd-svc|nd-wss|netductor" || true'
else
  run "Expected Secondary service units" systemctl is-active \
    nd-wss-sp-client nd-wss-ps-server \
    wg-quick@nd-svc-sp wg-quick@nd-svc-ps \
    nd-svc-paths-health.timer
fi

run "Failed systemd units" systemctl --failed --no-legend --plain
run "IPv4 interface addresses" ip -4 -brief address
run "IPv4 routes" ip -4 route
run "WireGuard interface names only" wg show interfaces
run "WireGuard latest handshake times (keys redacted)" wg show all latest-handshakes
run "Recent error-level journal entries (last 2 hours, max 100)" \
  journalctl -p err..alert --since "2 hours ago" -n 100 --no-pager

chmod 600 "$REPORT"
if [[ -n "${SUDO_UID:-}" && -n "${SUDO_GID:-}" ]]; then
  chown "$SUDO_UID:$SUDO_GID" "$REPORT" 2>/dev/null || true
fi
echo "Report saved to: $REPORT"
echo "Review it before sharing; automated redaction cannot guarantee removal of every sensitive value."
