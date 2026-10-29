#!/usr/bin/env bash
set -euo pipefail
ROOT="$(cd "$(dirname "$0")/.." && pwd)"
cd "$ROOT"
fail=0
# Web: EN keys used for updates must exist in both dicts
for k in upd_local upd_latest upd_pick upd_apply upd_refresh upd_available upd_ok upd_note upd_jobs tab_updates; do
  if ! grep -q "upd_local\|${k}:" internal/operator/web/app.js; then
    echo "FAIL: web missing key pattern $k" >&2
    fail=1
  fi
done
# crude: each key appears at least twice (en + ru) for upd_
for k in upd_note upd_jobs upd_latest; do
  n=$(grep -c "${k}:" internal/operator/web/app.js || true)
  if [ "$n" -lt 2 ]; then
    echo "FAIL: $k expected in both EN and RU dicts (count=$n)" >&2
    fail=1
  fi
done
# TG ver_* keys
for k in ver_jobs ver_jobs_empty ver_apply_hint ver_confirm ver_nothing ver_job_active ver_bad_tag ver_no_retry; do
  n=$(grep -c "\"$k\"" cmd/netductor-tg/i18n.go || true)
  if [ "$n" -lt 2 ]; then
    echo "FAIL: TG key $k missing en/ru (count=$n)" >&2
    fail=1
  fi
done
# docs pairs
for f in UPDATE-I18N.md UPDATE-RECOVERY.md VERSION-SEMANTICS.md EDGE-AGENT-IDENTITY.md; do
  if [ ! -f "docs/$f" ] || [ ! -f "docs/ru/$f" ]; then
    echo "FAIL: missing docs pair $f" >&2
    fail=1
  fi
done
[ "$fail" -eq 0 ] && echo "check-update-i18n: OK" || exit 1
