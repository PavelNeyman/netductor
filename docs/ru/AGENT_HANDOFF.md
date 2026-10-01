
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
