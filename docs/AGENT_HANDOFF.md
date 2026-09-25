# Agent handoff — netductor

**Baseline:** **v0.9.19** · https://github.com/PavelNeyman/netductor/releases/tag/v0.9.19

## Read order (new chat)

1. [AGENTS.md](../AGENTS.md)  
2. **This file**  
3. [REVIEW-0.9.17.md](REVIEW-0.9.17.md) — latest full security/code review (covers 0.9.17–0.9.17)  
4. [PLAN-MAC-CLIENT.md](PLAN-MAC-CLIENT.md) · [ARCHITECTURE-OPERATOR.md](ARCHITECTURE-OPERATOR.md)  
5. [UI-PARITY.md](UI-PARITY.md) · [WEB-UI-NOTES.md](WEB-UI-NOTES.md)  
6. [DOMAIN.md](DOMAIN.md) · [DEPLOY-MAC.md](DEPLOY-MAC.md) · [FLEET.md](FLEET.md) · [BACKUP.md](BACKUP.md) · [PORTS.md](PORTS.md)

## Architecture (locked)

- TG Tools hub sections from opcatalog; day-2 action catalog: `internal/opcatalog` (`GET /v1/catalog`)
- Doctor: structured JSON via `/api/doctor` (`CollectDoctor`)
- Web Advanced: off by default (Settings → Show Advanced)


Thin UIs (Web, TUI, CLI) share **one** backend: `internal/operator` + `internal/deploy` + node API.
Parity is the rule; temporary gaps are debt (not intentional TUI-only features).

## Product model (locked)

| Piece | Rule |
|-------|------|
| UI | **Mac only** — `netductor-op` WebUI + TUI |
| Node | API + VPN + agents — **no product admin on VPS** |
| TG | Day-2 on node; **no VPS deploy** from bot |
| First deploy | Password once → inject key → harden (52222) |
| Day-2 access | SSH key; tunnel prefer VPN host then public |
| Recovery | Off until arm; WAN OK while armed (TTL) |

## Binaries

| Binary | Role |
|--------|------|
| `netductor-op` | Mac: serve WebUI, TUI, deploy, tunnel, session |
| `netductor` | Linux node |
| `netductor-agent` | OpenWrt |
| `netductor-tg` | Telegram addon on node |

## Mac deploy (both paths work)

Both call the **same** `internal/operator` → `internal/deploy` (Mac-direct SSH; no primary→secondary hop).

**Web Fleet/Primary Telegram:** checkbox + `tg_token` / `tg_admin` → same `DeployPrimary` secrets/install as TUI/CLI.

**Web:** `netductor-op operator serve` → Installer → Fleet / Primary / Secondary / Credentials  
**TUI:** `netductor-op` → Wizard → Fleet / Primary / Secondary (+ OpenWrt / MikroTik / NVR / Add-ons)

Shared backend: `internal/operator` + `internal/deploy`.

## Day-2

- **Web Control:** broad session API + Advanced  
- **TUI Tools:** `netductor …` local or `--remote`  
- **TG:** Tools (VPN, edge, NVR, git, DNS, backup, probes, …)

## Open / deferred

- Ideas: MikroTik API via RPi agent; Site rooms/photos (Web/TG) — see OPEN_ITEMS

- Hardware e2e (owner)  
- SMTP when mailbox exists  
- Mobile client  
- Web form-label full i18n  
- Optional Web thin links for OpenWrt wizard (TUI remains primary)

## Agent rules

1. Every completed item → update docs + checklist + version when shipping  
2. No new VPS HTML admin  
3. No password SSH for day-2 tunnel  
4. Do not enable PLAIN_AGENT / API_PUBLIC / CLAIM_FIRST in defaults  

## Obsolete docs

`docs/archive/` — old reviews/plans. Prefer this handoff + REVIEW-0.9.12.
