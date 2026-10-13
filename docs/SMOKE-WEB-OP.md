# Operator Web UI — browser smoke (mandatory for `internal/operator/web/` diffs)

Run against **local** `netductor-op operator serve` (or `serve`) after rebuilding op. ~2 minutes.

## Preconditions

- `netductor-op version` matches tip `VERSION`
- Browser hard-refresh (Cmd+Shift+R) after binary replace
- DevTools → Network available

## Checklist

| # | Step | Expect |
|--|--|--|
| 1 | Open Installer | Fleet/Primary forms visible, not blank page |
| 2 | Tab **Control** | Tunnel card + sub-nav visible (not empty white) |
| 3 | Tab **Settings** | Primary host / key fields visible |
| 4 | Settings: save host+key | “OK” / persisted; reload still filled |
| 5 | OpenWrt → **Probe device** | Not bare `Unauthorized`; either card or SSH error text |
| 6 | Network: any `/v1/*` | Request has header `X-Netductor-Token` |
| 7 | Language RU/EN | Probe button and Settings labels switch |

## Automated static (always, CI/local)

```bash
./scripts/check-web-ui.sh
./scripts/check-versions.sh
```

## Rule

Diff under `internal/operator/web/` **or** `cmd/netductor-op/` UI paths: suite incomplete without static scripts **and** this smoke (or Playwright later). Mark in REVIEW-SUITE output: `Web smoke: done | skipped (reason)`.
