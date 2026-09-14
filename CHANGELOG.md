# Changelog

## 0.7.0-dev — 2026-09-09 (Netductor)

- **Rename** product/repo: Netductor → **Netductor** (`PavelNeyman/netductor`)
- **Go orchestrator** `netductor`: version, doctor, vpn, edge, serve, install, probe
- **Go API**: edge, session, metrics, live probes, VPN users, admin SPA; proxy to Python
- **Units**: `netductor-api` (parallel :8790 / cutover :8787)
- **Agent**: `netductor-agent` (OpenWrt outbound)
- **Telegram**: `netductor-tg` + unit `netductor-telegram-bot`
- **G7 paths**: dual layout `/opt|/etc|/var/lib/netductor` → legacy symlinks
- Cleanup: prefer Go bots over `cmd/netductor-tg` and bash `bot.sh`

## 0.5.4 — 2026-09-08

- Hardening: blacklist virtio_gpu (headless KVM soft lockup mitigation)
- Hardening: 2G /swapfile by default (NETDUCTOR_SWAP_MB, NETDUCTOR_SKIP_SWAP=1)
- Kuma/Beszel optional off by default


## 0.5.2 — 2026-09-08

- Admin UI: auto-refresh, probe strip, session expiry, toasts, note edit, search, copy VLESS/HY2
- Metrics: net counters, probe 24h uptime %, Settings for alerts/probes JSON
- Alerts: recovery messages; UDP hy2; flock + cooldown before notify
- API: `/api/session`, `/vpn/users/:name/note`, `/api/probes/uptime`, POST probes config
- CLI: `netductor vpn note`
- Doctor: admin UI + metrics timer checks

## 0.5.1 — probes + charts + UDP hy2 fix
## 0.5.0 — Admin UI + self metrics (no Prometheus); Kuma/Beszel off by default
## 0.4.x — install/prepare, telegram bot, uninstall tiers
