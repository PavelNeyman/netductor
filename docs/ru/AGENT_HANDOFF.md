**RU** · [EN](../AGENT_HANDOFF.md)

# Handoff для агента

**Репо:** https://github.com/PavelNeyman/netductor  

Читать: [ARCHITECTURE-FREEZE](ARCHITECTURE-FREEZE.md) · [ARCHITECTURE-PLAN](ARCHITECTURE-PLAN.md) · [RECOVER-DRILL](RECOVER-DRILL.md) · [RUNBOOK-INSTALL-RECOVER](RUNBOOK-INSTALL-RECOVER.md) · [BREW](BREW.md)

## Канон
primary + secondary · VLESS · SP/PS · mTLS :8789 · SSH **52222** обе VPS · один Mac key · FHS · Mac op · redirect **:8443 only**

## Baseline
- **v0.9.71** — unattended recover (continue-on-error + post-restore)
- **v0.9.73** — inbound mux schema; post-restore ensure-relay-uplink + vpn apply; Reality checklist

## Live test VPS
primary `2.27.118.70` · secondary `92.255.77.253` (rebuild OK)

## Сделано недавно
- Live recover drill; docs prune + RU parity
- VPN post-recover: secrets↔conf, users, multiplex ON

## Дальше
1. Full dual-node smoke  
2. Hardware e2e  
3. Features only  

## Правило
Закрытый пункт чеклиста → отметить план + обновить handoff + CHANGELOG в **том же** изменении.
