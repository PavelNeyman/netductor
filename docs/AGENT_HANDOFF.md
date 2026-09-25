# Agent handoff — netductor

**Start here.** **Baseline:** **v0.9.3** · https://github.com/PavelNeyman/netductor

## Read order

1. [AGENTS.md](../AGENTS.md)  
2. This file  
3. **[PLAN-MAC-CLIENT.md](PLAN-MAC-CLIENT.md)** — Mac UI · node API only  
4. [ARCHITECTURE-OPERATOR.md](ARCHITECTURE-OPERATOR.md) · [OPERATOR-PLAN.md](OPERATOR-PLAN.md)  
5. [DOMAIN.md](DOMAIN.md) · [DEPLOY-MAC.md](DEPLOY-MAC.md) · [FLEET.md](FLEET.md) · [BACKUP.md](BACKUP.md) · [PORTS.md](PORTS.md)

## Locked product model

| Piece | Decision |
|-------|----------|
| **UI** | **Only on Mac** (`netductor-op` WebUI/TUI) |
| **Node** | **API server** + vpn/agents — **no product web admin** |
| **Legacy `/admin` on VPS** | Not a goal; may exist until P3 removal |
| Installer + Control | Same local WebUI (Installer deploy · Control day-2 via API tunnel) |
| Recovery | Off until arm; WAN bind while armed by design |
| Deploy | Mac-direct; credentials on Mac |

## Binaries

| Binary | Where |
|--------|--------|
| netductor-op | Mac — UI + deploy |
| netductor (linux) | VPS node |
| netductor-agent | OpenWrt |
| netductor-tg | VPS addon |

## Where we stopped

- [x] Operator core + Fleet + operator WebUI installer tabs  
- [x] Physical split op / node  
- [x] Recovery WAN-while-armed clarified  
- [x] **Locked: Mac client UI, node API-only** ([PLAN-MAC-CLIENT.md](PLAN-MAC-CLIENT.md))  
- [x] **P1** Mac WebUI Installer | Control | Settings + tunnel + API status  
- [x] P2 Control parity (users/nodes/edge via /v1/node proxy + session)  
- [ ] P3 drop default VPS static admin  
- [ ] Mobile day-2 later  

## Forbidden

- New features as VPS HTML admin  
- Public 8787 / installer on primary  
- Always-on recovery  
- Mac private key on primary  

## AI rule

Every completed plan item → `[x]` in PLAN-MAC-CLIENT / OPERATOR-PLAN + docs in the **same** change.
