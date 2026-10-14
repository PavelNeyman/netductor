#!/usr/bin/env bash
# Static guards for operator web UI traps that pure Go review misses.
set -euo pipefail
ROOT="$(cd "$(dirname "$0")/.." && pwd)"
JS="$ROOT/internal/operator/web/app.js"
CSS="$ROOT/internal/operator/web/app.css"
HTML="$ROOT/internal/operator/web/index.html"
fail=0
cd "$ROOT"

echo "== web UI static guards =="
if command -v node >/dev/null 2>&1; then
  if ! node --check "$JS" 2>/tmp/nd-web-js.err; then
    echo "FAIL: app.js syntax error"
    cat /tmp/nd-web-js.err
    fail=1
  fi
else
  echo "WARN: node not available — skip JS syntax check"
fi


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
# style.display alone on main-control without classList in same function is banned pattern
if grep -n "getElementById('main-control').style.display" "$JS" | grep -v classList; then
  if ! grep -q "setVis('main-control'" "$JS" && ! grep -q "classList.toggle('hidden'" "$JS"; then
    echo "FAIL: main-control style.display without .hidden toggle"
    fail=1
  fi
fi

# CSS must keep .hidden !important documented — if removed, showMain can use display only
if ! grep -qE '\.hidden\s*\{[^}]*display:\s*none\s*!important' "$CSS" && ! grep -q 'display:none!important' "$CSS"; then
  echo "WARN: .hidden !important not found — verify showMain still correct"
fi

tmp=$(mktemp)
grep -nE "fetch\\('/v1/" "$JS" > "$tmp" || true
while IFS= read -r line; do
  [ -z "$line" ] && continue
  num=${line%%:*}
  rest=${line#*:}
  if echo "$rest" | grep -q "X-Netductor-Token"; then continue; fi
  if sed -n "${num},$((num+5))p" "$JS" | grep -q "X-Netductor-Token"; then continue; fi
  echo "WARN: /v1 fetch without X-Netductor-Token near line $num"
done < "$tmp"
rm -f "$tmp"

python3 - <<'PY'
import re, pathlib, sys
html = pathlib.Path("internal/operator/web/index.html").read_text()
js = pathlib.Path("internal/operator/web/app.js").read_text()
keys = sorted(set(re.findall(r'data-i18n="([^"]+)"', html)))
missing = [k for k in keys if f"{k}:" not in js]
# buttons that should have i18n on critical paths
critical = ["btn_edge_probe", "set_phost", "btn_tunnel_start"]
for c in critical:
    if c not in js:
        print(f"FAIL: critical i18n key missing in app.js: {c}")
        sys.exit(2)
if missing:
    print("WARN: data-i18n keys missing from app.js dict:", ", ".join(missing[:40]))
# both lang objects should include btn_edge_probe roughly twice
if js.count("btn_edge_probe:") < 2:
    print("WARN: btn_edge_probe expected in EN and RU dicts")
print(f"i18n: {len(keys)} html keys, {len(missing)} missing from js")
PY
ec=$?
[ "$ec" -eq 2 ] && fail=1

[ "$fail" -eq 0 ] && echo "check-web-ui: OK" || exit 1
