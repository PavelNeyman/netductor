# Agent handoff — netductor

**Baseline:** **v0.9.12** · https://github.com/PavelNeyman/netductor

## Read order

1. [AGENTS.md](../AGENTS.md)  
2. This file  
3. [PLAN-MAC-CLIENT.md](PLAN-MAC-CLIENT.md) · [ARCHITECTURE-OPERATOR.md](ARCHITECTURE-OPERATOR.md)  
4. [DOMAIN.md](DOMAIN.md) · [DEPLOY-MAC.md](DEPLOY-MAC.md) · [FLEET.md](FLEET.md) · [BACKUP.md](BACKUP.md) · [PORTS.md](PORTS.md)

## Product model

| Piece | Decision |
|-------|----------|
| **UI** | Only on Mac (`netductor-op` WebUI / TUI) |
| **Node** | API + VPN/agents — no product web admin |
| **VPS `/admin`** | Off unless `NETDUCTOR_LEGACY_ADMIN_UI=1` |
| **First deploy** | Password once → inject key → harden |
| **Day-2** | SSH key (+ agent); tunnel prefer VPN host then public |
| **Recovery** | Off until arm; WAN bind while armed |

## Binaries

| Binary | Role |
|--------|------|
| netductor-op | Mac operator (deploy + local WebUI Control) |
| netductor | Linux node |
| netductor-agent | OpenWrt |
| netductor-tg | TG addon |

## Closed (Mac client)

- Installer Fleet / Primary / Secondary / Credentials  
- Control: operator-session APIs (VPN, nodes, edge, NVR, git/registry, backup, probes) + Advanced generic path  
- Tables, row actions, session autofill, EN/RU chrome  
- Tunnel: health direct → VPN SSH → public SSH  

## Still open

- Hardware e2e · SMTP · mobile  
- TG/TUI: not every rare POST mirrored (use WebUI Advanced)

## Docs

Obsolete reviews/plans → [archive/](archive/). Prefer this handoff + PLAN-MAC-CLIENT.

## Rule for agents

Every completed work item: update docs + mark checklist; push release when version bumps.
