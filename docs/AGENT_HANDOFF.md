# Agent handoff

**Repo:** https://github.com/PavelNeyman/netductor  

Read first: [ARCHITECTURE-FREEZE.md](ARCHITECTURE-FREEZE.md) · [ARCHITECTURE-PLAN.md](ARCHITECTURE-PLAN.md) · [PATHS.md](PATHS.md) · [BACKBONE-WG.md](BACKBONE-WG.md)

## Canon (summary)

primary + secondary · VLESS users · SP/PS service WG · :8789 mTLS (arm via TG only) · SSH 52222 · FHS · Mac op · backup data+config · redirect **:8443 only**

## Removed

Permanent `API_ALLOW_PUBLIC` / `PLAIN_AGENT` / `API_PUBLIC` / `TRUST_PROXY` · VPS admin · `/opt` · public :80

## Live ops

- TG Operator: table + toggles (API public 15m, Users→SP)
- Policy Users→SP syncs secondary via agent heartbeat
- VPS: primary `2.27.118.70`, secondary `92.255.77.253` (test; rebuild OK)

## Baseline

**v0.9.69+** (debts closed 2026-09-27) — A5 footgun doctor; B1 agent pin; B3/B4 posture; i18n audit

## Next work

Architecture debts closed. Next: **features** or owner **live recover / hardware**.

## Rule

Finished checklist item → mark plan `[x]` + update this handoff + CHANGELOG in the **same** change.
