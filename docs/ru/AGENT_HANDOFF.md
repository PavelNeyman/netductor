# Handoff агента

**RU** · [EN](../AGENT_HANDOFF.md)

**Репа:** https://github.com/PavelNeyman/netductor  

Читать: [ARCHITECTURE-FREEZE](../ARCHITECTURE-FREEZE.md) · [RECOVER-DRILL](RECOVER-DRILL.md) · [RUNBOOK-INSTALL-RECOVER](../RUNBOOK-INSTALL-RECOVER.md) · [BREW](../BREW.md)

## Канон
primary + secondary · VLESS · SP/PS · mTLS :8789 · SSH **52222** на обеих VPS · один Mac-ключ · FHS · Mac op · redirect только **:8443**

## Baseline
**v0.9.71** — unattended recover; COMPONENTS; tg ELF; ufw IP secondary

## Тестовые VPS
- primary `2.27.118.70` · secondary `92.255.77.253`

## Сделано
- Live recover без ручных правок 2026-09-27
- Уборка docs: старые REVIEW/PLAN → `docs/archive/`

## Дальше
1. Полный прогон обеих нод
2. Hardware e2e
3. Фичи — после smoke

## Правило
Закрытый пункт → `[x]` в плане + handoff + CHANGELOG в том же изменении.
