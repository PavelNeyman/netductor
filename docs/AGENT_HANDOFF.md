# Agent handoff — netductor

**Start here in a new chat.**  
**Baseline:** v0.8.97 · Repo: https://github.com/PavelNeyman/netductor

## Read order

1. [AGENTS.md](../AGENTS.md) — hard rules  
2. This file  
3. **[ARCHITECTURE-OPERATOR.md](ARCHITECTURE-OPERATOR.md)** — Mac operator vs node; target backend  
4. **[OPERATOR-PLAN.md](OPERATOR-PLAN.md)** — checklist (mark done + update docs)  
5. [DOMAIN.md](DOMAIN.md) · [DEPLOY-MAC.md](DEPLOY-MAC.md) · [FLEET.md](FLEET.md) · [PORTS.md](PORTS.md)

## Locked decisions (do not reopen without owner)

| Topic | Decision |
|-------|----------|
| Roles | **Primary** (abroad) = control plane; **Secondary** (RU) = VPN entry + agent |
| VPN | VLESS+Reality + HY2; prefer secondary under whitelist |
| SSH | Key on **Mac only**; password first login only; default port **52222** after harden |
| Secondary deploy | **Mac-direct** (prepare-pack on primary; Mac SSHs secondary). No primary→secondary SSH |
| Agent plane | mTLS **:8789** |
| Local-only | `:8787` API, Lampac, registry, blocky → `127.0.0.1` |
| Redirect | LE on **:8443**; **:443** = Reality only |
| Recovery | `:8790` **off** until `recovery arm` (SSH); HTTPS self-signed; key offline |
| Credentials | After deploy → `~/.netductor/credentials/` on Mac |
| DNS | External (Cloudflare). App: `domain set` / deploy flags only |
| Control plane | **Go-only** |
| Deploy UI | **Framed TUI + CLI only** (huh deploy removed). Prefer **Fleet** wizard |

## Operator direction

**Done (0.8.95):**  Specs + FleetDeploy; CLI/TUI thin.
**Next:** step events; phase 3 operator serve + WebUI.

## Operator direction (active work)

**Goal:** one operator core; CLI / TUI / later WebUI are thin clients.

- Architecture: [ARCHITECTURE-OPERATOR.md](ARCHITECTURE-OPERATOR.md)  
- Checklist: [OPERATOR-PLAN.md](OPERATOR-PLAN.md) — **phase 1 next**  
- Do **not** add new deploy logic only in TUI or only in CLI.

## Current product surface (short)

- `netductor deploy primary|secondary|edge`  
- `netductor tui` → Wizard → **Fleet** / Primary / Secondary / OpenWrt / …  
- `netductor domain set --base … [--le --email …]`  
- `netductor credentials collect`  
- Node: `install`, `serve`, `vpn`, agent, tg  

Example DNS (operator’s zone): `p.nd.neyman.top`, `s.nd.neyman.top`, `i.nd.neyman.top` (grey cloud).

## Forbidden

- Publish 8787 / 9118 / 5000 / DNS on `0.0.0.0`  
- Recovery always-on or port-knock revival  
- Cloud-sync of `~/.netductor/credentials`  
- Deploy orchestration that requires Mac private key stored on primary  
- Public “installer” API on primary  

## AI progress rule (mandatory)

See **AGENTS.md → Documentation & checklist discipline**.  
Every completed plan item → update docs + mark `[x]` in OPERATOR-PLAN (or ROADMAP) in the **same** change set.
