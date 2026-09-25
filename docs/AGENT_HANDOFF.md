# Agent handoff — netductor

**Start here in a new chat.**  
**Baseline:** **v0.9.1** · Repo: https://github.com/PavelNeyman/netductor

## Read order

1. [AGENTS.md](../AGENTS.md) — hard rules + doc discipline  
2. This file  
3. [ARCHITECTURE-OPERATOR.md](ARCHITECTURE-OPERATOR.md) · [OPERATOR-PLAN.md](OPERATOR-PLAN.md)  
4. [DOMAIN.md](DOMAIN.md) · [DEPLOY-MAC.md](DEPLOY-MAC.md) · [FLEET.md](FLEET.md) · [BACKUP.md](BACKUP.md) · [PORTS.md](PORTS.md) · [RESIDUAL_RISKS.md](RESIDUAL_RISKS.md)

## Binaries (physical split since 0.9.0)

| Binary | Where | Role |
|--------|--------|------|
| **netductor-op** | Mac / workstation | deploy, TUI, `operator serve`, credentials |
| **netductor** (`netductor-linux-*`) | VPS | node: install, serve, vpn, recovery, doctor |
| **netductor-agent** | OpenWrt / edge | agent plane |
| **netductor-tg** | VPS (addon) | Telegram bot |

Brew (Mac): install **netductor-op** from Formula. Deploy downloads **netductor-linux-*** onto VPS.

## Locked decisions

| Topic | Decision |
|-------|----------|
| Roles | Primary (abroad) = control; Secondary (RU) = VPN entry + agent |
| VPN | VLESS+Reality + HY2; prefer secondary under WL |
| SSH | Key on **Mac only**; password first login; port **52222** after harden |
| Secondary deploy | **Mac-direct** (no primary→secondary SSH) |
| Agent plane | mTLS **:8789** |
| Local-only | `:8787`, Lampac, registry, blocky → `127.0.0.1` |
| Redirect | LE **:8443**; **:443** = Reality |
| **Recovery** | **Off until `recovery arm`**. While armed: default bind **`0.0.0.0:8790`** (short TTL, Bearer, TLS). Loopback bind only for local tests. |
| Credentials | After deploy → `~/.netductor/credentials/` on Mac |
| DNS | External (Cloudflare); app `domain set` / deploy flags |
| Control plane | **Go-only** |
| Operator UI | Framed TUI + CLI + localhost `operator serve` |

## Where we stopped (2026-09-25)

- [x] Operator core (`internal/operator`), FleetDeploy, step events, localhost WebUI  
- [x] Physical split **cmd/netductor-op** vs **cmd/netductor**  
- [x] Security pass: path/filename, XFF, token SHA compares, host validation  
- [x] Recovery model clarified: WAN bind while armed is **by design** (doctor WARN, not FAIL)  
- [x] Richer operator WebUI (0.9.1): tabs + credentials + step chips
- [ ] Mobile day-2 client — deferred  
- [ ] Hardware e2e (OpenWrt / Tapo) — owner  

## Forbidden

- Always-on recovery / port-knock  
- Publish 8787/9118/5000 on `0.0.0.0`  
- Cloud-sync of `~/.netductor/credentials`  
- Mac private key stored on primary  
- Deploy logic only in TUI or only in CLI (use `internal/operator`)  

## AI progress rule

Every completed checklist item → mark `[x]` in OPERATOR-PLAN + update docs **in the same change**.
