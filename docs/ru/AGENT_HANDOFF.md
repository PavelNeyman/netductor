**RU** · [EN](../AGENT_HANDOFF.md)

# Handoff для агента

**Репо:** https://github.com/PavelNeyman/netductor  
**Версия:** **v0.9.75**

Читать: [ARCHITECTURE-FREEZE](ARCHITECTURE-FREEZE.md) · [PLAN](ARCHITECTURE-PLAN.md) · [RECOVER-DRILL](RECOVER-DRILL.md) · [SECURITY](SECURITY.md) · [REVIEW-0.9.75](REVIEW-0.9.75.md) · [BREW](BREW.md)

## Канон

primary + secondary · VLESS · SP/PS · mTLS :8789 · SSH 52222 · Mac op · redirect :8443 only · без plain agent / VPS admin / permanent public API

## Baseline

**0.9.75** — knobs убраны из кода · **0.9.74** — LE при recover · **0.9.73** — mux + ensure-relay-uplink · **0.9.71** — unattended recover

## Дальше

1. Dual-node smoke  
2. Hardware e2e  
3. Фичи  

Закрытый пункт → план + handoff + CHANGELOG в том же изменении.
