# AGENT Handoff (RU)
- **0.9.175:** test release (version bump only) for TG/stack update smoke.

**Точка останова: v0.9.189** (2026-10-02). Edge: сеть до harden; пароль до harden; полный набор agent.

## Текущий baseline

| Пункт | Состояние |
|--|--|
| **Релиз** | **v0.9.189** — 11 assets |
| **Плоскости** | `netductor-op` (Mac) · `netductor` + `netductor-tg` (node) · `netductor-agent` (OpenWrt/secondary) |
| **Web UI** | **Только** `internal/operator/web` (embed в op). Legacy `runtime/api/admin` **удалён** (0.9.171) |
| **Edge VPN шаблон** | UI TG/Web + API `GET|POST /api/edge/templates/vpn`; CLI `template-get` / `template-set-vpn` |
| **Policy** | TemplateWithVPN **не** затирает fallback/dns/mode; allowlist SetTemplateVPN; merge POST templates |
| **Агент** | `enabled=false` / `mode=off` / нет vless → stop `netductor-vpn`; fallback=block ⇒ soft off |
| **/sub/** | Публичный token sub + **60 req/min/IP** (0.9.170) |
| **Тесты** | `go test ./...` зелёный (0.9.172); уникальные ID opcatalog |

## Недавние версии (кратко)

- **0.9.189** — сеть/guest до harden; SSH с паролем до harden

- **0.9.188** — OpenWrt без base64
- **0.9.187** — persist operator token
- **0.9.186** — Dropbear ssh-pipe; mipsle

- **0.9.186** — Dropbear: agent через ssh stdin; mipsle+riscv64

- **0.9.185** — offline pubkey/~; cert по device_id
- **0.9.184** — offline pack Web/CLI

- **0.9.181** — WAN DNS (пробел/запятая) + подсказки
- **0.9.180** — подсказки guest; handoff; bind template только с device_id

- **0.9.172** — расширение unit-тестов; дубликаты ID opcatalog
- **0.9.171** — удаление legacy VPS admin UI
- **0.9.170** — rate-limit /sub/; mutex SaveTemplate; chmod github_token
- **0.9.166–169** — merge policy VPN; stop при disable; ValidName
- **0.9.164–165** — карточка Template VPN в Web/TG
- **0.9.162–163** — LAN через VLESS + soft WAN; CF-first DNS; CLI шаблона

## Apply на живых нодах

```bash
netductor stack apply v0.9.185
# OpenWrt: agent_update + apply_template после Save шаблона
# Mac
brew reinstall netductor
netductor-op version   # 0.9.179
```

## Напоминания

- Save шаблона на primary ≠ Apply на роутере.
- Одна operator-сессия = полный контроль (split read/destructive — опционально позже).
- EN/RU: полная **смысловая** паритетность (AGENTS §7).

## Не сделано / владелец

См. [OPEN_ITEMS.md](OPEN_ITEMS.md) / [ru/OPEN_ITEMS.md](OPEN_ITEMS.md): force-update VPS если отстаёт; hardware e2e; опционально CI SHA Formula.

---

## 0.9.163 Порядок DNS edge + CLI шаблона

- Удалённый DNS: сначала **1.1.1.1**, затем **9.9.9.9**.
- CLI: `edge template-get` / `template-set-vpn`. Web/TG: Template VPN/DNS → `GET|POST /api/edge/templates/vpn`.
## Документация EN/RU

**Жёсткое правило:** полный **смысловой** паритет `docs/` ↔ `docs/ru/` (не краткий пересказ). См. AGENTS.md §7, docs/I18N.md.

## 0.9.162 LAN через VLESS (мягкий fallback WAN)

- Шаблон: VPN вкл, tun, fallback=wan, dns=vpn
- Peer `edge-<id>` не в списке Users
- Soft fallback + DNS через secondary; агент к primary — direct

## Модель edge (решено, код частично)

**Сделано (0.9.160+):** LuCI вкл/выкл/продление/статус — агент + SSH в LAN + TG Роутеры + Web + TUI + CLI.

**Решено, ещё не в коде:**
- Private Wi‑Fi → VLESS на secondary; мягкий fallback на WAN ISP
- Агент mTLS вне user VLESS
- VPN-peer на каждый роутер (не в Users)
- Blocky/DNS для LAN через VPN

## 0.9.160–0.9.161 LuCI

- Действия агента: `luci_enable|disable|extend|status` (hours=, дефолт **1ч**)
- Оператор: `POST /v1/edge/luci` via ssh|agent
- Пакет не удаляется — только uhttpd

## 0.9.151 Control+Media hub

See docs/TG-MEDIA-TOPICS.md


## 0.9.150 — Versions hub + sub profiles

- TG **Versions** merges Fleet digest + Updates; multi-select primary/secondary Apply
- VPN **sub_profile**: `secondary` (default) | `primary` | `both` — subscription URL body replaced (client refresh)
- Operator unset profile → both; others → secondary
- Design: [TG-MEDIA-TOPICS.md](TG-MEDIA-TOPICS.md)


## Version policy (0.9.148+)

- **No auto upgrade/rollback** of node/tg/agent binaries.
- Operator chooses version (TG Updates / CLI `stack apply` / `stack rollback` / `stack heal`).
- Secondary: only explicit queue `upgrade` / `upgrade:vX` — not `desired_release` heartbeat.
- Watchdog: unit restart only, never swaps binaries.

**RU** · [EN](../AGENT_HANDOFF.md)

# Handoff для агента

**Репозиторий:** https://github.com/PavelNeyman/netductor  
**Версия:** **v0.9.122**

Читать: [ARCHITECTURE-FREEZE](ARCHITECTURE-FREEZE.md) · [RUNBOOK-FORCE-UPDATE](RUNBOOK-FORCE-UPDATE.md) · [RECOVER-DRILL](RECOVER-DRILL.md) · [BREW](BREW.md)

## Канон

primary + secondary · VLESS Reality · SP/PS · mTLS **:8789** · SSH **52222** · op только на Mac · redirect **:8443**

## Оркестратор (0.9.116–121)

stack status/apply/rollback/watchdog · pre-backup · secondary `upgrade:vX` · DR secondary без API primary · Web/TG Stack

Сломанный узел (0.9.111 и т.п.): [RUNBOOK-FORCE-UPDATE](RUNBOOK-FORCE-UPDATE.md) — ручная подмена бинарников, не in-process update.

## Дальше (владелец)

1. Обновить живые VPS до **0.9.122**  
2. Smoke  
3. Hardware e2e  
