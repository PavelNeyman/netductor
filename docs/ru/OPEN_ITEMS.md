**Стоп-линия v0.9.319.** Закрыто: path e2e, presets, canary, incident, backup verify, clip token, session на release GET, doctor perms nvr.

Открыто (lab): OpenWrt/Tapo/MikroTik hardware e2e — см. OPENWRT-LAB.md.

---

**RU** · [EN](../OPEN_ITEMS.md)

# Открытые пункты

План: [ARCHITECTURE-PLAN.md](ARCHITECTURE-PLAN.md) · freeze: [ARCHITECTURE-FREEZE.md](ARCHITECTURE-FREEZE.md)

## Планы (документы)

- [x] [PLAN-SERVICE-ACCESS-POLICY](PLAN-SERVICE-ACCESS-POLICY.md) **P0–P4** поверхность политики; dual-node smoke — на операторе
- [x] [HOST-AUDIT](HOST-AUDIT.md) — one-shot + residual (0.9.197–0.9.198); live CLEAN 2026-10-03
- [x] Denylist host-audit (RMM/otel/avahi + timeweb-zabbix) — script ↔ Go (0.9.198)
- [x] Firewall status во всех UI + алерты (0.9.259)
- [x] Baseline snapshot install + doctor auto-diff (0.9.259–0.9.260)

## Закрыто (ядро)

- [x] Architecture freeze
- [x] Recover automation + live drill
- [x] Доки EN/RU
- [x] Updates (node/TG/Web/TUI) + secondary pre-backup wait
- [x] TG menu categories (0.9.288–289) — [PLAN-TG-MENU.md](PLAN-TG-MENU.md)
- [x] Site rooms (0.9.305) — [PLAN-SITE-ROOMS.md](PLAN-SITE-ROOMS.md)
- [x] NVR UI + doctor + retention alerts (код; e2e железо отдельно)
- [x] Channel health + export логов 1h + ротация (0.9.275–0.9.276)
- [x] Internal releases P0–P4 + multi-repo projects (0.9.282–0.9.283)
- [x] Channel face vs uplink alerts (0.9.284)
- [x] Private GH runbook — [GH-PRIVATE-SETUP.md](GH-PRIVATE-SETUP.md)
- [x] Workflow resolve только `.github/workflows/<Name>.yml` (0.9.307–0.9.308); `ci/` убран
- [x] Mac queue + TG/CLI (0.9.307); op `project build` (0.9.309)

## У владельца / лаб

- [ ] Edge stock-OpenWrt **lab e2e** (код hybrid готов) — [OPENWRT-LAB.md](OPENWRT-LAB.md)
- [ ] NVR hardware e2e (C200 record/live/clip) — [PLAN-NVR-IMPLEMENTATION.md](PLAN-NVR-IMPLEMENTATION.md)
- [ ] Hardware e2e (OpenWrt / Tapo / MikroTik)
- [x] Homebrew Formula SHA (`release.sh` + `update-formula-sha.sh`) (0.9.310)
- [x] Internal releases «later»: auto-build on mirror-fetch, metrics local_release_* (0.9.310)

## Channel / CI (справка)

- [x] Alert batch adaptive (0.9.274)
- [x] 10 реп: default `main`, Actions off, entry yaml по имени репы

Ревью: [REVIEW-PROGRESS.md](REVIEW-PROGRESS.md).
