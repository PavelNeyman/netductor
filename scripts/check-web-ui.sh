#!/usr/bin/env bash
set -euo pipefail
ROOT="$(cd "$(dirname "$0")/.." && pwd)"
JS="$ROOT/internal/operator/web/app.js"
HTML="$ROOT/internal/operator/web/index.html"
fail=0
cd "$ROOT"

if ! grep -q "classList.toggle('hidden'" "$JS"; then
  echo "FAIL: showMain must classList.toggle('hidden' (CSS .hidden is !important)"
  fail=1
fi
if grep -n "window\\.ND_TOKEN" "$JS"; then
  echo "FAIL: use ND_TOKEN + X-Netductor-Token, not window.ND_TOKEN"
  fail=1
fi
if ! grep -q 'id="main-control" class="hidden"' "$HTML"; then
  echo "FAIL: main-control should start with class=hidden"
  fail=1
fi
if ! grep -q 'id="main-settings" class="hidden"' "$HTML"; then
  echo "FAIL: main-settings should start with class=hidden"
  fail=1
fi

tmp=$(mktemp)
grep -nE "fetch\\('/v1/" "$JS" > "$tmp" || true
while IFS= read -r line; do
  num=${line%%:*}
  rest=${line#*:}
  if echo "$rest" | grep -q "X-Netductor-Token"; then continue; fi
  if sed -n "${num},$((num+4))p" "$JS" | grep -q "X-Netductor-Token"; then continue; fi
  echo "WARN: /v1 fetch without X-Netductor-Token near line $num"
done < "$tmp"
rm -f "$tmp"

python3 -c '
import re, pathlib
html=pathlib.Path("internal/operator/web/index.html").read_text()
js=pathlib.Path("internal/operator/web/app.js").read_text()
keys=sorted(set(re.findall(r"data-i18n=\"([^\"]+)\"", html)))
missing=[k for k in keys if f"{k}:" not in js]
if missing:
  print("WARN: data-i18n keys missing:", ", ".join(missing[:40]))
'

[ "$fail" -eq 0 ] && echo "check-web-ui: OK" || exit 1
