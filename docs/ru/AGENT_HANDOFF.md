
- **0.9.166:** TemplateWithVPN no longer overwrites `vpn.fallback`/policy; `SetTemplateVPN` allowlist; `POST /api/edge/templates` merges by default (`replace=true` for full replace).

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
