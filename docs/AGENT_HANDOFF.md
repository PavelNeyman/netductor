
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
