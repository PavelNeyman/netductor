**EN** · [RU](ru/AGENT_HANDOFF.md)

# Agent handoff

**Repo:** https://github.com/PavelNeyman/netductor  
**Version:** **v0.9.79**

Read first: [ARCHITECTURE-FREEZE](ARCHITECTURE-FREEZE.md) · [ARCHITECTURE-PLAN](ARCHITECTURE-PLAN.md) · [RECOVER-DRILL](RECOVER-DRILL.md) · [RUNBOOK-INSTALL-RECOVER](RUNBOOK-INSTALL-RECOVER.md) · [SECURITY](SECURITY.md) · [REVIEW-0.9.75](REVIEW-0.9.75.md) · [BREW](BREW.md)

## Canon (frozen)

primary + secondary · VLESS Reality · service SP/PS · mTLS **:8789** · SSH **52222** both VPS · one Mac operator key · FHS · **netductor-op** on Mac · redirect **:8443 only** · no plain agent / no VPS admin UI / no permanent public API

## Baseline releases

| Ver | Note |
|-----|------|
| **0.9.79** | Auto-heal redirect in collect; probes strip hy2; no ufw on agent heartbeat |
| **0.9.77** | Auto failover: probe→tick→apply (agent URL + uplink server) on secondary |
| **0.9.76** | SSH only :52222 (deny :22); :8789 no secondary WAN after SP; doctor healthz :8443 |
| **0.9.75** | Footgun knobs removed from code; `ndconfig` ignores obsolete conf keys |
| **0.9.74** | LE re-issue on recover when `DOMAIN`+`LE_EMAIL`; no certs in `.ndenc` |
| **0.9.73** | Inbound mux schema (padding only); post-restore `ensure-relay-uplink` + `vpn apply` |
| **0.9.71** | Unattended recover continue-on-error + post-restore pass |

## Live test VPS

- primary `2.27.118.70` · secondary `92.255.77.253`

## Deploy

Mac **TUI + Web** → shared `internal/deploy`. TG does not deploy VPS.

## After recover checklist

1. `vpn apply` OK · Reality secrets match · relay-uplink present · multiplex ON  
2. LE auto if conf has DOMAIN+LE_EMAIL; else `domain set --le` once  
3. See [RECOVER-DRILL.md](RECOVER-DRILL.md)

## Next (owner)

1. Full **dual-node smoke**  
2. Hardware e2e  
3. Features only after smoke  

## Rule for agents

Closed checklist item → mark plan + update this handoff + CHANGELOG in the **same** change.

## Service plane (SP/PS) — auto with secondary deploy (0.9.75+)

After `deploy secondary`, Mac runs:
1. `svc-paths bootstrap-primary --peer-ip <secondary>` on primary → material JSON
2. `svc-paths bootstrap-secondary --material` on secondary
3. Agent `secondary_core_url` → `https://10.87.10.1:8789`

Units: `nd-wss-sp-*`, `nd-wss-ps-*`, `wg-quick@nd-svc-sp|ps`. Health timer every 30s.
If missing on an old node: same two bootstrap commands (not optional in architecture).

## Pending UX (no code yet)

- Web **Control (Day-2)** simplify — [WEB-UI-FIXES.md](WEB-UI-FIXES.md) § Day-2 / Control UX.
