#!/usr/bin/env bash
# Documents CLI ↔ UI surface. Exit 1 if critical CLI groups lack any UI mention.
set -euo pipefail
ROOT=$(cd "$(dirname "$0")/.." && pwd)
fail=0
check() {
  local cmd="$1" pattern="$2"
  if ! grep -RInq "$pattern" "$ROOT/cmd/netductor-tg" "$ROOT/cmd/netductor/tui"*.go "$ROOT/runtime/api/admin" 2>/dev/null; then
    echo "MISSING UI surface for CLI group: $cmd (pattern $pattern)"
    fail=1
  else
    echo "OK $cmd"
  fi
}
check doctor "doctor"
check vpn "vpn"
check edge "edge"
check secondary "secondary"
check mtls "mtls"
check nodes "nodes"
check sites "sites"
check backup "backup"
check nvr "nvr"
check deploy "DeployPrimary\|deploy primary\|wizardPrimary"
check fleet "fleet\|provision-secondary"
check ssh-hosts "ssh-hosts\|sshhosts\|ssh_hosts"
exit $fail
