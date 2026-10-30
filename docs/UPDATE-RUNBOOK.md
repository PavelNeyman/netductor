# Update runbook (operators)

## Prerequisites

1. Operator Mac on current `netductor-op` (Homebrew formula or release asset).
2. SSH TOFU pin for Primary (`netductor-op ssh-hosts trust …`) before Mac→Primary SSH.
3. GitHub token with release read if private; rate limits apply.
4. Backup window acceptable (stack apply runs pre-backup unless `--no-backup`).

## Primary node

```bash
netductor update check
netductor update list --limit 5
netductor update apply v0.9.612 --component node
# or: netductor stack apply v0.9.612
```

Downtime: API/VPN brief restart during stack apply. Confirm `netductor version` / doctor after.

## Telegram bot / secondary-agent on host

```bash
netductor update apply v0.9.612 --component tg
netductor update apply v0.9.612 --component agent   # secondary-agent unit only
netductor update jobs
```

## Edge OpenWrt

Do **not** use `--component agent`. Use edge `agent_update` with confirm and device id (see EDGE-AGENT-IDENTITY).

## Secondary VPS binary

Operator-driven: secondary upgrade commands / queue — not auto from primary `update agent` on the wrong host.

## Failure recovery

1. Read job state: `netductor update jobs`
2. [UPDATE-RECOVERY.md](UPDATE-RECOVERY.md) — rollback binary if health failed
3. SSH host key issues — [SSH-HOST-TRUST.md](SSH-HOST-TRUST.md)

## Trust model

Artifacts from GitHub Releases (or internal mirror) with SHA256SUMS. Skip-verify requires explicit confirm env. Never disable StrictHostKeyChecking for deploy SSH.
