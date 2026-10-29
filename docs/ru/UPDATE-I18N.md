# Update system localization (Phase 7)

## Canonical sources

| Surface | Source | Languages | Fallback |
|---------|--------|-----------|----------|
| Telegram | `cmd/netductor-tg/i18n.go` `dict` + `T(key)` | `en`, `ru` (`telegram_lang`) | `en` if key missing |
| Web operator | `internal/operator/web/app.js` `I18N.en` / `I18N.ru` + `t(key)` | `en`, `ru` (`uiLang`) | English literal in `t()` call site |
| CLI | mostly English machine-oriented messages | — | — |

## Stable (do not translate)

- Component IDs: `node`, `tg`, `agent`, `op`
- Binary / asset names: `netductor-linux-amd64`, …
- Job states: `queued`, `downloading`, `succeeded`, …
- `FailureClass` tokens: `checksum`, `health`, `conflict`, …
- Version numbers, SHAs, tags (`v0.9.x`)

Human text only at the presentation boundary (TG/Web).

## Backend errors

API returns `error` strings and structured fields (`job.state`, `display_kind`). UI may show them; codes remain English.

## Checks

```bash
bash scripts/check-update-i18n.sh
bash scripts/check-web-ui.sh   # broader web i18n
```

## Docs pairs

UPDATE-CONTRACT, UPDATE-RECOVERY, VERSION-SEMANTICS, EDGE-AGENT-IDENTITY, UPDATE-SYSTEM-IMPLEMENTATION-PLAN — maintain EN + `docs/ru/` with the same operational meaning.
