#!/usr/bin/env bash
# Fail if operator-facing paths still advertise ancient v0.7 downloads.
set -euo pipefail
ROOT="$(cd "$(dirname "$0")/.." && pwd)"
cd "$ROOT"
VER="$(tr -d '[:space:]' < VERSION)"
echo "VERSION=$VER"

bad=0
check_file() {
  local f="$1"
  [[ -f "$f" ]] || return 0
  if grep -nE 'v0\.7\.|0\.7\.0-dev' "$f" 2>/dev/null | grep -vE 'CHANGELOG|REVIEW-|OPEN_ITEMS|AGENT_HANDOFF|PLAN-|EDGE-|older|historical|was |legacy|v0\.7\.x|pinned to|base64' >/tmp/nd-pin-hits 2>/dev/null; then
    if [[ -s /tmp/nd-pin-hits ]]; then
      echo "FAIL stale pin: $f"
      cat /tmp/nd-pin-hits
      bad=1
    fi
  fi
}

check_file bootstrap.sh
check_file Formula/netductor.rb
check_file README.md
check_file runtime/api/admin/app.js
check_file cmd/netductor-tg/format_sites.go
check_file internal/secondary/agent.go
check_file internal/deploy/version.go
check_file internal/version/version.go
check_file internal/install/services.go
check_file internal/install/components.go
check_file edge/openwrt/INSTALL.md

while IFS= read -r f; do
  check_file "$f"
done <<FIND
$(find docs -type f \( -name '*.md' -o -name '*.sh' \) 2>/dev/null)
FIND

rel="$(grep -E 'const Release' internal/version/version.go | sed -E 's/.*"([0-9.]+)".*/\1/')"
if [[ "$rel" != "$VER" ]]; then
  echo "FAIL version.Release=$rel != VERSION=$VER"
  bad=1
fi

if [[ $bad -ne 0 ]]; then
  exit 1
fi
echo "OK version pins"
