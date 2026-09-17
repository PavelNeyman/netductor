# Plan: Secondary = VPN entry only

**Status:** accepted 2026-09-17  
**Goal:** RU node is a thin data-plane entry, not a service mirror.

## Locked decisions

| Item | Decision |
|------|----------|
| Secondary role | **VLESS/Reality entry** + **thin agent** only |
| Primary role | Source of truth: users, API, TG bot, Blocky, edge enroll, backups, optional Lampac |
| Lampac | **Primary only** (optional component). Not deployed/synced to secondary |
| Bot failover / standby on RU | **Removed** |
| Fleet data sync (lampac dirs, hourly timer) | **Removed** |
| VPN user list on secondary | **Kept** — `ApplyConfig` → `config_ver` → agent `ExportRelayBundle` |
| Edge/OpenWrt enroll | **Primary only** (outbound agents) |
| Cross-node `.ndenc` peer backup | Optional later; not required for VPN-entry model |

## Implementation steps

1. **Docs** — FLEET, ARCHITECTURE, AGENT_HANDOFF, RELAY, this plan; strip “warm mirror / lampac prefer RU”.
2. **Policy** — `SyncEnabled=false` by default; no `lampac_node_id` on secondary; notes = VPN entry.
3. **provision-secondary** — only: `relay provision` + fleet secondary role + hostname. No lampac, bot-standby, SyncPaths, sync timer.
4. **Delete / gut** — `SyncPaths` hourly timer, `ApplyLampac` remote-to-secondary, `bot_failover` CLI/units install from provision.
5. **CLI** — `fleet sync` / `sync-timer` / `bot-failover` / `apply-lampac` (fleet) → print removed + tip; `install lampac` stays on local host.
6. **TUI** — drop “apply lampac on secondary” paths; wizard secondary text = VPN entry only.
7. **Keep** — secondary agent, `relay sync` / config_ver, user UUID push, provision SSH key lock-down.
8. **Live fleet** — disable `netductor-fleet-sync.timer`, bot-failover timer on nodes; stop/remove lampac container on secondary when operator confirms (ops, not code).

## Out of scope

- Changing VLESS/Reality settings or SNI presets
- Moving edge API to RU
- Full active-active control plane

## Done when

- [x] Plan in repo
- [x] Code: provision + policy + CLI/TUI cleaned
- [x] Docs match
- [x] New secondary provision does not install docker/lampac/bot standby
- [x] `vpn add` still updates secondary users via config_ver


## TUI follow-up (2026-09-17)

- Remote SSH ops from workstation; wizard integrated; chips/keys; menu regroup.
