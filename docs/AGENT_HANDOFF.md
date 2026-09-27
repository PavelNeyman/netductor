# Agent handoff

**EN** · [RU](ru/AGENT_HANDOFF.md)

**Repo:** https://github.com/PavelNeyman/netductor  

Read: [ARCHITECTURE-FREEZE.md](ARCHITECTURE-FREEZE.md) · [ARCHITECTURE-PLAN.md](ARCHITECTURE-PLAN.md) · [RECOVER-DRILL.md](RECOVER-DRILL.md) · [RUNBOOK-INSTALL-RECOVER.md](RUNBOOK-INSTALL-RECOVER.md) · [BREW.md](BREW.md)

## Canon
primary + secondary · VLESS · SP/PS · mTLS :8789 · SSH **52222** both VPS · same Mac key · FHS · Mac op · redirect **:8443 only**

## Baseline
**v0.9.71** — unattended recover (continue-on-error + post-restore pass); COMPONENTS baseline; tg real binary; ufw secondary IP

## Live test VPS
- primary `2.27.118.70` · secondary `92.255.77.253` (rebuild OK)

## Done recently
- Live recover drill unattended 2026-09-27
- Docs cleanup: old REVIEW/PLAN snapshots → `docs/archive/`

## VPN after recover
See [RECOVER-DRILL.md](RECOVER-DRILL.md) § Reality/uplink. `vpn apply` must succeed; multiplex ON (inbound padding-only; outbound full). Baseline fix **0.9.73**.

## Next
1. Full dual-node smoke (primary + secondary together)
2. Hardware e2e when ready
3. Features only after that if desired

## Rule
Checklist item done → plan `[x]` + this handoff + CHANGELOG in the same change.
