**0.9.277:** channel log tar only on new alert (6h cooldown). Reality-invalid from secondary is not alerted while uplink is up (mux noise on public :443).
# AGENT Handoff

**Stop line: v0.9.261** (2026-10-06). New chat: AGENTS.md → this file → OPEN_ITEMS → docs/REVIEW-PROGRESS.md.

## Where we stopped

Pass 1 security review and R1–R15 refactor are in main. Last product change: doctor auto-diff of the install snapshot (listen ports and enabled units).

| Topic | State |
|--|--|
| Release | **v0.9.261** — node, tg, agent, op darwin/linux. Formula is **netductor-op only** |
| Tap | `Formula/netductor.rb` removed. `brew install netductor` was the same op binary and confused the node |
| VPN config | binary update does **not** rewrite sing-box. New JSON only after `vpn apply` or secondary sync |
| Open | dual-node smoke and hardware e2e (OpenWrt/Tapo/MikroTik). Not code |

## Review of 0.9.230–0.9.261

Checked, still present: session on addon update, stack apply/rollback, firewall apply, update token/apply, svc-paths, fleet digest. Release tags are digits and dots. Client file reads stay in the user dir. Legacy plaintext session filenames are not accepted. Snapshot diff is WARN, not FAIL.

Fixed in this pass: op `version` was hardcoded `0.9.128`. Release script wrote `Formula/netductor.rb` again; it now writes only `netductor-op.rb`.

Do not stack-apply 0.9.230–0.9.239 for the session guards; use **0.9.261**.


# AGENT Handoff
- **0.9.217:** service-net 10.88 from secondary dials primary as the real VLESS user (uplink-svc-<name>, vision, no mux). Shared relay-uplink is internet only. ACL no longer auto-allows relay-uplink.
- **0.9.208:** TG bot sets policy.ApplyHook → ApplyAccessPolicies (policy toggles were saving JSON only; sing-box mtime stayed old).
- **0.9.207:** TG/Web policy UI parity — full T() i18n, preset none, card template aligned with users/dns.
- **0.9.205:** service-net (`nd-svc` 198.18.88.0/24 + DNAT); `servicenet status|apply`; doctor; policy P4 docs; TG presets via ApplyPreset.
- **0.9.204:** `host-baseline status|apply` + doctor drift checks; policy presets CLI; doctor policy/catalog integrity (P4 partial).
- **0.9.201:** safer hoster purge (installed pkgs only, no pkill -f, port parse); apt Pin-Priority -1 block; host-audit --dry-run.
- **0.9.200:** install/stack apply/secondary upgrade auto EnsureHostBaseline (firewall role file + netductor-stack-watchdog.timer). Role detect fixed in 0.9.199.
- **0.9.175:** test release (version bump only) for TG/stack update smoke.

**Stop line: v0.9.199** (host-audit apt residual + fleet live CLEAN; policy P0–P3 at 0.9.196)

**Prev stop: v0.9.192** (2026-10-02). Guest stage no-reload; net/guest hard-fail; full edge path fixes.


## Docs added (2026-10-03)

| Doc | Purpose |
|--|--|
| [PLAN-SERVICE-ACCESS-POLICY.md](PLAN-SERVICE-ACCESS-POLICY.md) (+ ru/) | VPN user + OpenWrt edge service access policies (catalog + checkboxes); phased P0–P4 |
| [HOST-AUDIT.md](HOST-AUDIT.md) (+ ru/) | Hoster agent / listen surface audit; collect logs for offline analysis |
| `scripts/collect-host-audit.sh` | Full text audit bundle |
| `scripts/audit-hoster-agents.sh` | Expanded denylist (sync with `host_agents.go`) |

**Next implementer:** dual-node smoke (VPN/bot/policy routes) dual-node smoke (VPN/bot/policy routes) or OpenWrt e2e. HOST-AUDIT + service-net in 0.9.205.


**Prev stop: v0.9.172** (2026-10-02). New chats: AGENTS.md → this file → OPEN_ITEMS → code.

## Current baseline

| Item | State |
|--|--|
| **Release** | **v0.9.192** — 11 assets (mipsle+riscv64); Formula SHA OK |
| **Planes** | `netductor-op` (Mac) · `netductor` + `netductor-tg` (node) · `netductor-agent` (OpenWrt/secondary) |
| **Web UI** | **Only** `internal/operator/web` (embed in op). Legacy `runtime/api/admin` **removed** (0.9.171) |
| **Edge VPN template** | UI TG/Web + API `GET|POST /api/edge/templates/vpn`; CLI `template-get` / `template-set-vpn` |
| **Policy safety** | TemplateWithVPN does **not** overwrite fallback/dns/mode; SetTemplateVPN allowlist; merge POST templates |
| **Agent** | `enabled=false` / `mode=off` / missing vless → stop `netductor-vpn`; fallback=block ⇒ soft off |
| **/sub/** | Public token sub + **60 req/min/IP** rate limit (0.9.170) |
| **Tests** | `go test ./...` green (0.9.172); opcatalog unique IDs fixed |

## Recent versions (short)

- **0.9.189** — edge order: network/guest before harden; SSH prefer password when set
- **0.9.188** — OpenWrt provision without base64 (mTLS/pass via ssh stdin)
- **0.9.187** — operator token persist + app.js inject
- **0.9.186** — Dropbear ssh-pipe for agent; mipsle assets
- **0.9.186** — OpenWrt Dropbear: agent via ssh stdin; scp -O fallback; mipsle+riscv64 assets
- **0.9.185** — offline edge pubkey/~ expand; cert device match
- **0.9.184** — offline pack Web/CLI
- **0.9.182–183** — primary :52222; bootstrap-token offline path
- **0.9.181** — WAN DNS space/comma normalize + Web/TUI hints (recommend 1.1.1.1 9.9.9.9)
- **0.9.180** — guest UI hints; handoff; template bind needs device_id
- **0.9.179** — fix agent `guest enable --hidden=1` after default-visible
- **0.9.178** — guest SSID **visible by default** (optional hidden); Web/TUI/CLI/agent
- **0.9.177** — Web password toggle on Installer load; LuCI primary_key prefill + device_id help
- **0.9.176** — edge UCI stage without reload; force reboot after net/guest
- **0.9.173–175** — backup_pull mTLS; secondary upgrade node-only; stack smoke

- **0.9.172** — unit tests expansion; opcatalog duplicate IDs
- **0.9.171** — delete legacy VPS admin UI
- **0.9.170** — /sub/ rate-limit; template save mutex; github_token chmod
- **0.9.166–169** — VPN policy merge fixes; agent stop on disable; ValidName; merge validate
- **0.9.164–165** — Web+TG Template VPN card + legends
- **0.9.162–163** — edge LAN VLESS soft WAN; CF-first DNS; template CLI

## Operator apply on live nodes

```bash
# primary
netductor stack apply v0.9.185
# OpenWrt: agent_update to matching release + apply_template after template Save
# Mac
brew reinstall netductor   # or Formula netductor-op
netductor-op version       # expect 0.9.179
```

## Architecture reminders

- Save template on primary ≠ apply on router (explicit apply_template).
- Single operator session = full control (no read/destructive session split — optional later).
- EN/RU docs: **full semantic parity** (AGENTS §7).


## Verification plan (phased)

- EN: [VERIFICATION-PLAN.md](VERIFICATION-PLAN.md)
- RU: [ru/VERIFICATION-PLAN.md](ru/VERIFICATION-PLAN.md)
- Progress checkboxes live in those files. Phase **B code pass done** (B.1–B.10 in VERIFICATION-PLAN §9–§10). **Next: Phase C** live dual-VPS smoke (needs running nodes). Matrix §8 still open on Live column. Hardware D–F blocked until devices.

## Not done / owner

See [OPEN_ITEMS.md](OPEN_ITEMS.md): live VPS force-update if lagging; hardware e2e; optional CI Formula SHA.

---

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

## 0.9.193 host firewall mandatory
- ufw install; iptables fallback; doctor FAIL; TG/API

## 0.9.194 hoster agent audit
- host-audit CLI + scripts/audit-hoster-agents.sh
- doctor FAIL on salt/zabbix/telegraf/… units
