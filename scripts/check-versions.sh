#!/usr/bin/env bash
set -euo pipefail
ROOT="$(cd "$(dirname "$0")/.." && pwd)"
cd "$ROOT"
fail=0
ver=$(tr -d ' \n' < VERSION)
if [[ ! "$ver" =~ ^[0-9]+\.[0-9]+\.[0-9]+$ ]]; then
  echo "FAIL: VERSION invalid: '$ver'"
  fail=1
fi
# Accept var or const Release = "x.y.z"
rel=$(grep -E '(var|const) Release[[:space:]]*=' internal/version/version.go | head -1 | sed -E 's/.*"([^"]+)".*/\1/')
if [ -z "$rel" ]; then
  echo "FAIL: could not extract internal/version.Release"
  fail=1
fi
if [ -n "$rel" ] && [ "$ver" != "$rel" ]; then
  echo "FAIL: VERSION ($ver) != internal/version.Release ($rel)"
  fail=1
fi
tmp=$(mktemp)
find cmd -name 'main.go' -print > "$tmp"
while IFS= read -r f; do
  if grep -nE 'var version = "[0-9]+\.[0-9]+\.[0-9]+"' "$f"; then
    echo "FAIL: $f hardcodes version — use ndver.Release"
    fail=1
  fi
done < "$tmp"
rm -f "$tmp"
echo "VERSION=$ver Release=$rel"
[ "$fail" -eq 0 ] && echo "check-versions: OK" || exit 1
