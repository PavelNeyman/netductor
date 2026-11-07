#!/usr/bin/env bash
# CF-15: ensure upgrade.json generates from in-code floors and has valid schema.
set -euo pipefail
ROOT="$(cd "$(dirname "$0")/.." && pwd)"
cd "$ROOT"

ver=$(tr -d ' \n' < VERSION)
tmp=$(mktemp)
trap 'rm -f "$tmp"' EXIT

go run scripts/gen-upgrade-json.go "$ver" > "$tmp"

python3 - "$tmp" "$ver" <<'PY'
import json, sys
path, ver = sys.argv[1], sys.argv[2]
meta = json.load(open(path))
if meta.get("version", "").lstrip("v") != ver.lstrip("v"):
    print(f"FAIL: upgrade.json version {meta.get('version')!r} != VERSION {ver!r}", file=sys.stderr)
    sys.exit(1)
for key in ("version",):
    if key not in meta:
        print(f"FAIL: missing {key}", file=sys.stderr)
        sys.exit(1)
# min_from optional; floors list optional
mf = meta.get("min_from") or ""
if mf and not all(c.isdigit() or c == "." for c in mf.lstrip("v")):
    print(f"FAIL: min_from looks invalid: {mf!r}", file=sys.stderr)
    sys.exit(1)
floors = meta.get("floors") or []
if not isinstance(floors, list):
    print("FAIL: floors must be list", file=sys.stderr)
    sys.exit(1)
for f in floors:
    if not f.get("id") or not f.get("min_binary"):
        print(f"FAIL: floor missing id/min_binary: {f}", file=sys.stderr)
        sys.exit(1)
print(f"check-upgrade-floors: OK version={meta['version']} min_from={mf or '(none)'} floors={len(floors)}")
PY
