#!/usr/bin/env bash
# Static guards for operator web UI traps that pure Go review misses.
set -euo pipefail
ROOT="$(cd "$(dirname "$0")/.." && pwd)"
JS="$ROOT/internal/operator/web/app.js"
HTML="$ROOT/internal/operator/web/index.html"
fail=0

if ! grep -q "classList.toggle('hidden'" "$JS"; then
  echo "FAIL: showMain must classList.toggle('hidden' (CSS .hidden is !important)"
  fail=1
fi

if grep -n "window\\.ND_TOKEN" "$JS"; then
  echo "FAIL: use ND_TOKEN const + X-Netductor-Token, not window.ND_TOKEN"
  fail=1
fi

# fetch('/v1/...') should usually include X-Netductor-Token on same statement or nearby
while IFS= read -r line; do
  num=${line%%:*}
  rest=${line#*:}
  if echo "$rest" | grep -q "X-Netductor-Token"; then
    continue
  fi
  # multi-line headers: allow next 3 lines
  if sed -n "${num},$((num+4))p" "$JS" | grep -q "X-Netductor-Token"; then
    continue
  fi
  echo "WARN: /v1 fetch without X-Netductor-Token near line $num: $rest"
done < <(grep -nE "fetch\\('/v1/" "$JS" || true)

if ! grep -q 'id="main-control" class="hidden"' "$HTML"; then
  echo "FAIL: main-control should start with class=hidden"
  fail=1
fi
if ! grep -q 'id="main-settings" class="hidden"' "$HTML"; then
  echo "FAIL: main-settings should start with class=hidden"
  fail=1
fi

python3 - <<'PY'
import re, pathlib
html = pathlib.Path("internal/operator/web/index.html").read_text()
js = pathlib.Path("internal/operator/web/app.js").read_text()
keys = sorted(set(re.findall(r'data-i18n="([^"]+)"', html)))
missing = [k for k in keys if f"{k}:" not in js]
if missing:
    print("WARN: data-i18n keys missing from app.js dict:", ", ".join(missing[:40]))
PY

if [ "$fail" -ne 0 ]; then
  exit 1
fi
echo "check-web-ui: OK"
