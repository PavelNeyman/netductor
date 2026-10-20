> **Стоп: v0.9.427.** Thin-UI jobs core + аудит Phase 3 + Operate Result dock. Остаётся: доработки по ревью F1–F10 (ниже), lab e2e (OpenWrt/NVR/MikroTik), future (OIDC/guest session).

**RU** · [EN](../OPEN_ITEMS.md)

# Открытые пункты

План: [ARCHITECTURE-PLAN.md](ARCHITECTURE-PLAN.md) · freeze: [ARCHITECTURE-FREEZE.md](ARCHITECTURE-FREEZE.md)

## Стоп-линия

**v0.9.427** (tip кода 2026-10-19; доки синхронизированы 2026-10-20) — Operate Result dock (0.9.422–0.9.426), fallback токена в offline-prep (0.9.427). Последний опубликованный релиз: v0.9.426. Ранее: 0.9.414 thin-UI jobs core; 0.9.404 чистка.

### Доработки по ревью 2026-10-20 (статический полный комплекс; тесты не запускались)

Отчёт: [REVIEW-REPORT-2026-10-19.md](REVIEW-REPORT-2026-10-19.md) · план: [REVIEW-PLAN-2026-10-19.md](REVIEW-PLAN-2026-10-19.md). Пункты с пометкой **owner** ждут согласия, потому что меняют код.

- [x] **Выполнить `go test ./internal/... ./cmd/... -count=1`** — прошло 2026-10-20 на tip 0.9.429 (`GOMAXPROCS=1 NETDUCTOR_DISABLE_VPS_WORKER=1`, timeout 180s). `internal/incident` 0.027s, без форк-бомбы. Все пакеты ok.
- [x] **F1 (S-1, Medium, owner)** отзывать VPN sub-токен при удалении пользователя (`RevokeNative`) + отклонять отключённых в `SubscriptionBody` + тест
- [x] **F2 (Q-1, Medium, owner)** добавить Web i18n-ключи `edge_st_g_bak`, `rd_mode`, `rd_pos`, `result_hint` (EN+RU)
- [x] **F3 (Q-2, owner)** самоприсваивание в `internal/git/projects.go:108`
- [x] **F4 (Q-3, owner)** коммит gofmt (129 файлов) + `gofmt -l` в комплексе
- [x] **F5 (owner)** удалить проверенный мёртвый код DC-A (~45 функций, ~25 констант/переменных); по DC-B решать по каждому пункту
- [x] **F6 (S-2, Medium, owner)** recovery TLS: fail closed при невалидном пине, явный флаг первого контакта
- [x] **F7 (S-5/R-4, owner)** `httpx.NewServer` с таймаутами для всех слушателей
- [x] **F8 (S-6/S-7/R-3, owner)** без shell-сборки для `ufw delete` / `wg pubkey`; заменить установку Docker через `curl | sh`
- [x] **F9 (S-8, owner)** пробрасывать ошибки `saveDevices` в `internal/edge`
- [ ] **F10 (R-1…R-8, owner)** рефакторинги: разбить функции-монстры (`registerEdgeAPI` 937 строк, TG `handleCallback` 779, …), общие helper-ы
- [x] **D-1 (Medium, owner)** — RU pairs for surviving EN-only ops docs (2026-10-20). See PLAN-DOCS-D1.md.
- [ ] **D-1 (Medium, owner)** RU-пары для 19 документов, которые есть только на EN: CI-WORKFLOWS, CLEANUP, DAY2-GROUPS, DISASTER-RECOVERY, GH-PRIVATE-SETUP, OPENWRT-LAB, PLAN-CI-NEXT, PLAN-EDGE-DEPLOY-HYBRID, PLAN-GIT-CI-OPS, PLAN-INTERNAL-RELEASES, PLAN-NVR-IMPLEMENTATION, PLAN-NVR-UI, PLAN-REMOVE-TG-TOPICS, PLAN-TG-MENU, RECOVER-BOT, RELEASE, REVIEW-PROGRESS, TG-ALERTS-CHANNEL, YOUTUBE-ADS
- [ ] **§8 live smoke флота** (`netductor doctor`, `host-audit`, `channel status`, версия агента на secondary) — нужен доступ оператора
- [x] Статический прогон ревью, отчёт и план опубликованы (EN+RU); стоп-линии handoff / OPEN_ITEMS / AGENTS приведены к 0.9.427

- [x] План улучшений продукта (без железа e2e) — [PLAN-PRODUCT-IMPROVEMENTS.md](PLAN-PRODUCT-IMPROVEMENTS.md) **закрыт** (2026-10-12); включает проход по структуре TG-меню


Полный комплекс ревью: [REVIEW-SUITE.md](REVIEW-SUITE.md) · [../REVIEW-SUITE.md](../REVIEW-SUITE.md).

Журнал чисток / allowlist (tip после 0.9.403): [CLEANUP-PASSES-2026-10.md](../CLEANUP-PASSES-2026-10.md) · [ru](CLEANUP-PASSES-2026-10.md). Tip может быть **впереди** GitHub Release до 0.9.404+.

**v0.9.446** (2026-10-13) — build-host recover + fleet optional; ранее 0.9.349 SR General.

### Закрыто в 0.9.320–0.9.331 (security / DNS / SSH)

- [x] Metrics UI + HTML-таблицы channel + recovery-алерты EN/RU
- [x] unattended-upgrades (только security), integrity manifest, backup age, SERVFAIL-алерты
- [x] TG reboot: выбор ноды + one-shot коды; Blocky DoT (primary)
- [x] SSH allowlist: TG Tools, API, agent, Web Advanced, opcatalog (`m:sshkeys:`)
- [x] Reboot challenge: паритет API + Web Advanced + opcatalog (те же коды, что TG)

### Закрыто в 0.9.310–0.9.318 (ops / security)

- [x] Path e2e, custom policy presets, devices, canary mismatch, incident pack
- [x] Backup verify CLI/timer + TG/Web/API
- [x] Post-review: verify caps, incident без shell, update session, decrypt RAM, stream tar
- [x] Clip share token без op-сессии (`GET /api/nvr/clip` exempt)
- [x] Release GET — requireSession; perms nvr secrets/go2rtc в doctor
- [x] TG Day2 Tools (без клонов Status/Users/Fleet); подписи хабов через `T()`

### Открыто (lab / продукт, не code blockers)

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
- [x] TG menu categories (0.9.288–289) — [PLAN-TG-MENU.md](../PLAN-TG-MENU.md)
- [x] Site rooms (0.9.305) — [PLAN-SITE-ROOMS.md](PLAN-SITE-ROOMS.md)
- [x] NVR UI + doctor + retention alerts (код; e2e железо отдельно)
- [x] Channel health + export логов 1h + ротация (0.9.275–0.9.276)
- [x] Internal releases P0–P4 + multi-repo projects (0.9.282–0.9.283)
- [x] Channel face vs uplink alerts (0.9.284)
- [x] Private GH runbook — [GH-PRIVATE-SETUP.md](../GH-PRIVATE-SETUP.md)
- [x] Workflow resolve только `.github/workflows/<Name>.yml` (0.9.307–0.9.308); `ci/` убран
- [x] Mac queue + TG/CLI (0.9.307); op `project build` (0.9.309)
- [x] REVIEW-SUITE EN/RU + unit-test debt P0/P1 (0.9.329–0.9.331)

## У владельца / лаб

- [ ] Edge stock-OpenWrt **lab e2e** (код hybrid готов) — [OPENWRT-LAB.md](../OPENWRT-LAB.md)
- [ ] NVR hardware e2e (C200 record/live/clip) — [PLAN-NVR-IMPLEMENTATION.md](../PLAN-NVR-IMPLEMENTATION.md)
- [ ] Hardware e2e (OpenWrt / Tapo / MikroTik)
- [x] Homebrew Formula SHA (`release.sh` + `update-formula-sha.sh`) (0.9.310)
- [x] Internal releases «later»: auto-build on mirror-fetch, metrics local_release_* (0.9.310)

## Channel / CI (справка)

- [x] Alert batch adaptive (0.9.274)
- [x] 10 реп: default `main`, Actions off, entry yaml по имени репы

Ревью: [REVIEW-PROGRESS.md](../REVIEW-PROGRESS.md) · комплекс: [REVIEW-SUITE.md](REVIEW-SUITE.md).


- [ ] Read-only guest operator session — будущее
- [ ] OIDC operator login — [OIDC-DESIGN](../OIDC-DESIGN.md); когда появится IdP

## USB на edge

- План: [PLAN-EDGE-USB-STORAGE.md](../PLAN-EDGE-USB-STORAGE.md) / [ru](PLAN-EDGE-USB-STORAGE.md) — extroot + pin/sync + алерты (после 0.9.402).

| Тонкий UI (jobs) | draft | [PLAN-THIN-UI-JOBS](PLAN-THIN-UI-JOBS.md) |

### Отложено (продукт)

- **Web-терминал** (xterm.js + agent stream/PTY или exec allowlist) — не в дереве; предпочтителен путь через агент, не сырой SSH.
- **Web-файловый менеджер** (agent `fs_list`/`fs_read`/`fs_write` + allowlist путей) — отложено.
- **UI-скины A/B/C** — реализованы в operator Web (Settings → skin).
