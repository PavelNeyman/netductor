
## 0.9.163 Edge DNS order + template CLI

- Remote DNS: **1.1.1.1** first, **9.9.9.9** second (RU paths often faster to CF).
- CLI: `edge template-get` / `template-set-vpn`. Web/TG: Template VPN/DNS → `GET|POST /api/edge/templates/vpn`.
## Docs EN/RU

**Hard rule:** full **semantic** parity `docs/` ↔ `docs/ru/` (not a RU digest). See AGENTS.md §7, docs/I18N.md.

## 0.9.162 Edge LAN via VLESS (soft WAN)

- Default template `vpn.enabled=true`, `mode=tun`, `fallback=wan`, `dns=vpn`
- Peer name `edge-<device_id>` — **hidden** from Users TG/API/Web (use `?include_edge=1`)
- Client: urltest proxy+direct; private IP direct; primary host direct (agent mTLS)
- DNS: hijack + resolve via proxy (blocky path on secondary uplink); `.ru` via 77.88.8.8
- Guest zone stays ISP-direct (not in TUN policy for guest iface isolation — guest firewall unchanged)

## Edge model (decided, partial code)

**Done (0.9.160+):** LuCI enable/disable/extend/status — agent + SSH LAN + TG Routers + Web + TUI + CLI.

**Implemented 0.9.162:** private Wi‑Fi → VLESS secondary + soft WAN fallback; edge peer; DNS via VPN path; agent→primary direct.

## 0.9.160 LuCI enable/disable (agent + SSH LAN)

- Agent actions: `luci_enable|disable|extend|status` (arg `hours=N`, default **1h** TTL; auto-stop on expire)
- Operator: `POST /v1/edge/luci` via **ssh** (same LAN, no internet) or **agent** (queue on primary)
- Web: Edge form section LuCI; presets 1/4/24/72h
- Package not removed — only uhttpd stop/start

## 0.9.159 Edge root password default-required; guest SSID visibility

- **New root password** required by default (LuCI). Checkbox/flag **skip** = leave empty (not recommended).
- Current SSH password still optional (factory empty).
- Guest SSID: default **hidden**; Web/TUI/CLI can set visible (`guest_hidden=0` / `--guest-visible`).
- Guest fields appear when «Guest Wi‑Fi» is checked.

## 0.9.158 Optional root password on edge provision

- Current router password may be **empty** (factory OpenWrt)
- Optional **new root password** (Web checkbox / TUI / `--new-root-password`) applied before SSH harden (password auth off, key only)
- Prefers password auth (incl. empty) for first contact; key fallback if already provisioned

## 0.9.157 Edge prefill key path

- Fleet localStorage stores SSH path as `key`; edge prefill looked for `primary_key` only → wrong/empty path
- Prefill order: fleet.primary_key → fleet.key → settings → `~/.ssh/netductor_primary`

## 0.9.156 Web edge form parity + prefill

- Web OpenWrt: 2.4/5 SSID, WAN dhcp|static|pppoe conditional fields, guest fields, reboot
- Prefill primary host/key/mTLS URL from Fleet localStorage + Settings
- Clarified net_configure vs guest_enable; TUI detail text
- Backend was already full; Web form was incomplete (user-visible gap)

## 0.9.155 Edge agent arch probe + multi-arch assets

- **No OpenWrt feed/ipk** (by design): first-boot = SSH probe + pure-Go binary SCP; day-2 = stack/agent_update
- `DeployEdge`: default arch **`auto`** → SSH `uname -m` / DISTRIB_ARCH before `EnsureAgentBinary`
- Assets: `amd64`, `arm64`, `arm` (GOARM=7), **`mipsle`** (GOMIPS=softfloat, Cudy TR1200), `riscv64`
- UI: Web/TUI/CLI arch field default `auto`; override only when needed
- Packages/repo for edge **not** planned; VPS stays stack binary, not .deb

## 0.9.154 Edge provision parity + reboot

- Web OpenWrt form matches TUI network fields; reboot checkbox
- DeployEdge step 9: optional router reboot after agent install

## 0.9.153 Force version list refresh

- TG Versions → 🔄 refresh GitHub bypasses 30m cache
- `GET /api/update/status|releases?force=1`, CLI `update list --refresh`
- Web Control update buttons append force=1

## 0.9.152 Hub force + secondary upgrade log

- `/menu force|new|reset` and Topics **Reset hub** drop `hub_msg.json` and send a new Control message
- Failed edit → clear singleton (fixes invisible hub after delete-for-me / clear chat)
- Topics recreate → ClearHubMsg + forceHub
- Secondary upgrade CmdLog includes HTTP code/size and arch asset names

## 0.9.151 Control+Media hub

See docs/TG-MEDIA-TOPICS.md


## 0.9.150 — Versions hub + sub profiles

- TG **Versions** merges Fleet digest + Updates; multi-select primary/secondary Apply
- VPN **sub_profile**: `secondary` (default) | `primary` | `both` — subscription URL body replaced (client refresh)
- Operator unset profile → both; others → secondary
- Design: [TG-MEDIA-TOPICS.md](TG-MEDIA-TOPICS.md)


## Version policy (0.9.148+)

- **No auto upgrade/rollback** of node/tg/agent binaries.
- Operator chooses version (TG Updates / CLI `stack apply` / `stack rollback` / `stack heal`).
- Secondary: only explicit queue `upgrade` / `upgrade:vX` — not `desired_release` heartbeat.
- Watchdog: unit restart only, never swaps binaries.

**EN** · [RU](ru/AGENT_HANDOFF.md)

# Agent handoff

**Repo:** https://github.com/PavelNeyman/netductor  
**Version:** **v0.9.122** (`internal/version.Release` + GitHub Release assets)

Read first: [ARCHITECTURE-FREEZE](ARCHITECTURE-FREEZE.md) · [ARCHITECTURE-PLAN](ARCHITECTURE-PLAN.md) · [RECOVER-DRILL](RECOVER-DRILL.md) · [RUNBOOK-INSTALL-RECOVER](RUNBOOK-INSTALL-RECOVER.md) · [RUNBOOK-FORCE-UPDATE](RUNBOOK-FORCE-UPDATE.md) · [SECURITY](SECURITY.md) · [BREW](BREW.md)

## Canon (frozen)

primary + secondary · VLESS Reality · service SP/PS · mTLS **:8789** · SSH **52222** · one Mac operator key · FHS · **netductor-op** on Mac · redirect **:8443 only** · no plain agent / no VPS admin UI / no permanent public API

## Recent baseline (orchestrator)

| Ver | Note |
|-----|------|
| **0.9.122** | Web Control Stack; TG digest stack table |
| **0.9.120** | secondary `upgrade:vX` via agent; stack queues secondaries; secondary-local backup timer |
| **0.9.119** | secondary backup without primary API (local / recovery upload / SSH) |
| **0.9.118** | stack pre-backup, TG stack UI, apply/rollback/watchdog alerts |
| **0.9.116+** | stack orchestrator MVP, watchdog |
| **0.9.97–103** | update check/list/apply, multi-UI, pin version, secondary upgrade queue |

Always bump **`internal/version.Release`** with Release assets (not only `VERSION` / ldflags).

## Updates / stack

- CLI: `netductor update check|list|apply [vX]`, `netductor stack status|apply|rollback|watchdog-install`
- API: `/api/update/*`, `/api/stack/*`
- TG: Tools → Stack; Fleet digest
- Web op: Control → Overview → Stack
- Apply: pre-backup + wait backup_pull; queues secondary upgrade
- DR secondary: `backup secondary-local`, recovery upload, `push-ssh`

**Broken old node (e.g. 0.9.111):** do **not** rely on in-process update if bot/api OOM or broken — use [RUNBOOK-FORCE-UPDATE](RUNBOOK-FORCE-UPDATE.md).

## Live test VPS

- primary `2.27.118.70` · secondary `92.255.77.253` · SSH `-p 52222` · key `~/.ssh/netductor`

## Deploy

Mac **TUI + Web** → `internal/deploy`. TG does not deploy VPS.

## Next (owner)

1. Force-update live nodes to **0.9.122** if still on 111–115  
2. Dual-node smoke (VPN, bot, stack status, secondary heartbeat)  
3. Hardware e2e  

## Rule for agents

Closed checklist item → mark plan + update this handoff + CHANGELOG in the **same** change.

## Pending UX (low priority)

- Further Day-2 Web polish — [WEB-UI-FIXES.md](WEB-UI-FIXES.md)

## GitHub token (0.9.124)

- File: `/etc/netductor/secrets/github_token`
- CLI: `netductor update github-token status|set <tok>|clear`
- API: `GET/POST /api/update/github-token`
- TG: Tools → Updates → GitHub token
- Web Settings: GitHub token (on primary) — needs node session
- Env: `NETDUCTOR_GITHUB_TOKEN` / `GITHUB_TOKEN`


## 0.9.128
- Display version = `version.Running()` + `/etc/netductor/VERSION`
- After manual binary replace: write VERSION + restart units; Fleet/Stack follow file
